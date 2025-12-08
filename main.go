package main

import (
	"bluebell/controller"
	"bluebell/dao/mysql"
	"bluebell/dao/redis"
	_ "bluebell/docs" // 导入生成的 Swagger 文档包
	"bluebell/logger"
	"bluebell/pkg/snowflake"
	"bluebell/routers"
	"bluebell/settings"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// @title bluebell项目接口文档
// @version 1.0
// @description Go语言实战项目——社区web框架
// @termsOfService http://swagger.io/terms/

// @contact.name 这里写联系人姓名
// @contact.url http://www.swagger.io/support
// @contact.email 这里写联系人邮箱

// @host 127.0.0.1:8080
// @BasePath /api/v1

// main 是程序的入口函数
func main() {
	// 1. 加载配置
	// 使用 flag 包处理命令行参数，更标准，允许用户在启动时指定配置文件路径
	var confFile string
	// 绑定命令行参数 -conf 到 confFile 变量，默认值为 ./config.yaml
	flag.StringVar(&confFile, "conf", "./config.yaml", "配置文件路径")
	// 解析命令行参数
	flag.Parse()

	// 初始化配置模块，加载配置文件
	// 为什么：配置是程序运行的基础，必须最先加载，否则后续组件无法获取必要的参数（如数据库地址、端口等）
	if err := settings.Init(confFile); err != nil {
		fmt.Printf("init settings failed, err:%v\n", err)
		return
	}
	// 初始化雪花算法
	// 为什么：用于生成全局唯一的 ID，通常用于数据库主键或分布式系统中的唯一标识
	// 需要在业务逻辑开始前就绪
	if err := snowflake.Init(settings.Conf.Snowflake.StartTime, settings.Conf.Snowflake.MachineID); err != nil {
		fmt.Printf("init snowflake failed, err:%v\n", err)
		return
	}

	// 2. 初始化日志
	// 为什么：日志是排查问题的关键，尽早初始化以便记录后续的启动过程
	// 传入配置中的日志级别和应用模式（开发/生产），决定日志的输出格式和详细程度
	if err := logger.Init(settings.Conf.Log, settings.Conf.App.Mode); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	// 确保缓冲区日志刷新到磁盘
	// 为什么：zap 库使用缓冲区提高性能，程序退出前必须 Sync 确保所有日志都写入文件，防止丢失
	defer zap.L().Sync()

	// 3. 初始化 MySQL
	// 这里的原则是：Application 启动时，核心依赖挂了必须 Fatal，不要带病运行
	// 为什么：数据库是核心依赖，如果连不上，服务无法正常提供功能，不如直接启动失败报警
	if err := mysql.Init(settings.Conf.Mysql); err != nil {
		zap.L().Fatal("Init MySQL failed", zap.Error(err))
	}
	// 退出时关闭资源
	// 为什么：释放数据库连接资源，避免资源泄露
	defer mysql.Close()

	// 4. 初始化 Redis
	// 为什么：Redis 通常用于缓存，也是重要组件，初始化失败也应视为严重错误
	if err := redis.Init(settings.Conf.Redis); err != nil {
		zap.L().Fatal("Init Redis failed", zap.Error(err))
	}
	// 退出时关闭资源
	defer redis.Close()

	// 初始化Validator
	// 为什么：用于请求参数校验，并配置翻译器使错误信息支持中文，提升用户体验
	if err := controller.InitTrans("zh"); err != nil {
		zap.L().Fatal("init validator trans failed", zap.Error(err))
	}

	// 5. 注册路由
	// 为什么：将 HTTP 请求路径映射到具体的处理函数
	// 传入 App.Mode 可能是为了根据模式决定是否开启某些调试路由（如 pprof）
	r := routers.SetupRouter(settings.Conf.App.Mode)

	// 6. 启动服务 (优雅关机模式)
	port := settings.Conf.App.Port // 假设你在 settings 里设置了默认值，这里直接读即可
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	// 开启一个 goroutine 启动服务
	// 为什么：ListenAndServe 是阻塞的，如果不放在 goroutine 里，后面的优雅关机代码永远执行不到
	go func() {
		zap.L().Info("Server is running...", zap.Int("port", port))
		// 当你调用 srv.Shutdown()（优雅关机）时，正在运行的 ListenAndServe() 会被强制打断，并且一定会返回一个错误，这个错误就是 http.ErrServerClosed
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// 注意：这里用 zap.L().Fatal，因为如果端口被占用，程序应该直接挂掉
			zap.L().Fatal("listen failed", zap.Error(err))
		}
	}()

	// 7. 监听信号 (等待中断信号来优雅地关闭服务器)
	// 为什么：直接 kill 进程会导致正在处理的请求中断，数据不一致。优雅关机允许服务处理完当前请求后再退出。
	quit := make(chan os.Signal, 1)
	// kill 默认会发送 syscall.SIGTERM 信号
	// kill -2 发送 syscall.SIGINT 信号 (Ctrl+C)
	// kill -9 发送 syscall.SIGKILL 信号，无法被捕获，强杀
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞在此，直到接收到信号
	<-quit
	zap.L().Info("Shutdown Server ...")

	// 创建一个 5 秒超时的 context
	// 只要用了 WithTimeout 或 WithCancel，下一行马上跟一个 defer cancel()。这是铁律。
	// 为什么：给服务器 5 秒钟的时间来处理完当前正在进行的请求。如果 5 秒还没处理完，就强制关闭。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 8. 执行关机
	//尝试立刻关机，但最多只等你 5 秒。
	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("Server Shutdown failed", zap.Error(err))
	}

	zap.L().Info("Server exiting")
}
