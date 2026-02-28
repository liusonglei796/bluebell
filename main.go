package main

import (
	"bluebell/controller"
	"bluebell/dao/mysql"
	"bluebell/dao/redis"
	_ "bluebell/docs" // 导入生成的 Swagger 文档包
	"bluebell/logger"
	"bluebell/logic"
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
	var confFile string
	flag.StringVar(&confFile, "conf", "./config.yaml", "配置文件路径")
	flag.Parse()

	if err := settings.Init(confFile); err != nil {
		fmt.Printf("init settings failed, err:%v\n", err)
		return
	}

	if err := snowflake.Init(settings.Conf.Snowflake.StartTime, settings.Conf.Snowflake.MachineID); err != nil {
		fmt.Printf("init snowflake failed, err:%v\n", err)
		return
	}

	// 2. 初始化日志
	if err := logger.Init(settings.Conf.Log, settings.Conf.App.Mode); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	defer zap.L().Sync()

	// 3. 初始化 MySQL
	if err := mysql.Init(settings.Conf.Mysql); err != nil {
		zap.L().Fatal("Init MySQL failed", zap.Error(err))
	}
	defer mysql.Close()

	// 4. 初始化 Redis
	if err := redis.Init(settings.Conf.Redis); err != nil {
		zap.L().Fatal("Init Redis failed", zap.Error(err))
	}
	defer redis.Close()

	// 初始化 Validator
	if err := controller.InitTrans("zh"); err != nil {
		zap.L().Fatal("init validator trans failed", zap.Error(err))
	}

	// ====== 依赖注入：组装整个应用 ======

	// 1) 创建 Repository 实例（DAO 层）
	postRepo := &mysql.PostRepositoryImpl{}
	communityRepo := &mysql.CommunityRepositoryImpl{}
	userRepo := &mysql.UserRepositoryImpl{}

	// 2) 创建 Service 实例（Logic 层），注入 Repository
	postService := logic.NewPostService(postRepo)
	communityService := logic.NewCommunityService(communityRepo)
	userService := logic.NewUserService(userRepo)
	voteService := logic.NewVoteService(postRepo)

	// 3) 创建 Controller 实例，注入 Service
	postController := controller.NewPostController(postService)
	communityController := controller.NewCommunityController(communityService)
	userController := controller.NewUserController(userService)
	voteController := controller.NewVoteController(voteService)

	// 4) 注册路由，注入 Controller
	r := routers.SetupRouter(
		settings.Conf.App.Mode,
		userController,
		communityController,
		postController,
		voteController,
	)

	// 6. 启动服务 (优雅关机模式)
	port := settings.Conf.App.Port
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	go func() {
		zap.L().Info("Server is running...", zap.Int("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("listen failed", zap.Error(err))
		}
	}()

	// 7. 监听信号 (优雅关机)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zap.L().Info("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("Server Shutdown failed", zap.Error(err))
	}

	zap.L().Info("Server exiting")
}
