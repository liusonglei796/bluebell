package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"bluebell/internal/config"
	"bluebell/internal/infrastructure/logger"
	"bluebell/internal/infrastructure/mq"
	bluebellotel "bluebell/internal/infrastructure/otel"
	"bluebell/internal/infrastructure/profiler"
	database "bluebell/internal/infrastructure/persistence/mysql"
	redisrepo "bluebell/internal/infrastructure/persistence/redis"
	"bluebell/internal/infrastructure/snowflake"

	"go.uber.org/zap"
)

func main() {
	var confFile string
	flag.StringVar(&confFile, "conf", "./config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Init(confFile)
	if err != nil {
		fmt.Printf("init config failed, err:%v\n", err)
		return
	}

	if err := logger.Init(cfg, cfg.App.Mode); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	defer zap.L().Sync()

	// 初始化 OTel
	ctx := context.Background()
	otelShutdown, err := bluebellotel.InitOTEL(ctx, cfg.Otel)
	if err != nil {
		fmt.Printf("init otel failed, err:%v\n", err)
		return
	}
	defer func() {
		if err := otelShutdown(ctx); err != nil {
			fmt.Printf("otel shutdown error: %v\n", err)
		}
	}()

	// ====== 基础设施层：Pyroscope Profiler ======
	if cfg.Pyroscope != nil {
		pyroShutdown, err := profiler.Init(cfg.Pyroscope)
		if err != nil {
			fmt.Printf("init pyroscope failed, err:%v\n", err)
		} else {
			defer func() {
				if err := pyroShutdown(); err != nil {
					fmt.Printf("pyroscope shutdown error: %v\n", err)
				}
			}()
		}
	}

	if err := snowflake.Init(cfg); err != nil {
		zap.L().Fatal("init snowflake failed", zap.Error(err))
	}

	gormDB, err := database.Init(cfg)
	if err != nil {
		zap.L().Fatal("Init MySQL failed", zap.Error(err))
	}
	defer database.Close(gormDB)

	rdb, err := redisrepo.Init(cfg)
	if err != nil {
		zap.L().Fatal("Init Redis failed", zap.Error(err))
	}
	defer redisrepo.Close(rdb)


	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := mq.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		zap.L().Fatal("init MQ failed", zap.Error(err))
	}
	defer conn.Close()

	// Setup resources
	setupCh, err := conn.Channel()
	if err != nil {
		zap.L().Fatal("create setup channel failed", zap.Error(err))
	}
	if err := mq.SetupResources(setupCh); err != nil {
		zap.L().Fatal("setup rabbitmq resources failed", zap.Error(err))
	}
	setupCh.Close()

	// Start Vote Consumer
	ch, err := conn.Channel()
	if err != nil {
		zap.L().Fatal("create vote consumer channel failed", zap.Error(err))
	}
	defer ch.Close()

	repositoriesUOW := database.NewRepositories(gormDB)
	consumer := mq.NewVoteConsumer(ch, repositoriesUOW.Vote, rdb)
	
	zap.L().Info("Starting Vote Consumer...")
	go func() {
		if err := consumer.Start(ctx); err != nil {
			zap.L().Error("vote consumer exited with error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zap.L().Info("Shutting down Vote Consumer...")
}
