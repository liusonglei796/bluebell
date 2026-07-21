package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"bluebell/internal/config"
	"bluebell/internal/infrastructure/es"
	"bluebell/internal/infrastructure/logger"
	"bluebell/internal/infrastructure/mq"
	"bluebell/internal/infrastructure/snowflake"

	"go.uber.org/zap"
)

func main() {
	// 1. 获取命令行参数：配置文件路径
	var confFile string
	flag.StringVar(&confFile, "conf", "./config.yaml", "配置文件路径")
	flag.Parse()

	// 2. 加载配置：读取 yaml 文件并映射到配置结构体
	cfg, err := config.Init(confFile)
	if err != nil {
		fmt.Printf("init config failed, err:%v\n", err)
		return
	}

	// 3. 初始化日志组件：根据配置设定日志级别和输出位置
	if err := logger.Init(cfg, cfg.App.Mode); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	// 程序退出前将缓冲区日志刷入磁盘
	defer zap.L().Sync()

	// 4. 初始化分布式 ID 生成器 Snowflake
	if err := snowflake.Init(cfg); err != nil {
		zap.L().Fatal("init snowflake failed", zap.Error(err))
	}

	// 6. 创建上下文，用于控制后续消费者的生命周期
	ctx, cancel := context.WithCancel(context.Background())
	// 优雅关机时通知所有依赖此 context 的 goroutine 退出
	defer cancel()

	// 7. 初始化 Elasticsearch 客户端
	esClient, err := es.NewClient(cfg)
	if err != nil {
		zap.L().Fatal("init ES client failed", zap.Error(err))
	}

	// 8. 建立 RabbitMQ 物理连接 (TCP)
	conn, err := mq.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		zap.L().Fatal("init MQ failed", zap.Error(err))
	}
	// 程序退出前关闭物理连接
	defer conn.Close()

	// 9. 装修资源：声明必要的交换机 (Exchange) 和队列 (Queue)
	// 使用临时信道进行声明，确保消费者启动前队列已存在
	setupCh, err := conn.Channel()
	if err != nil {
		zap.L().Fatal("create setup channel failed", zap.Error(err))
	}
	if err := mq.SetupResources(setupCh); err != nil {
		zap.L().Fatal("setup rabbitmq resources failed", zap.Error(err))
	}
	// 声明任务完成后立即关闭临时信道
	setupCh.Close()

	// 10. 创建专门用于消息消费的信道
	// 遵循“发布/消费信道分离”的最佳实践
	ch, err := conn.Channel()
	if err != nil {
		zap.L().Fatal("create sync consumer channel failed", zap.Error(err))
	}
	// 程序退出前关闭消费信道
	defer ch.Close()

	// 11. 初始化搜索同步消费者实例
	// 注入信道和 ES 客户端
	consumer := mq.NewSyncConsumer(ch, esClient)
	
	zap.L().Info("Starting Sync Consumer (ES)...")
	// 12. 在独立的协程中启动消费者监听
	go func() {
		if err := consumer.Start(ctx); err != nil {
			zap.L().Error("sync consumer exited with error", zap.Error(err))
		}
	}()

	// 13. 优雅关机：监听操作系统退出信号 (Ctrl+C 或 kill)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// 阻塞直到接收到退出信号
	<-quit

	zap.L().Info("Shutting down Sync Consumer...")
	// 此处函数结束，触发 defer 执行，依次关闭信道、连接和上下文
}
