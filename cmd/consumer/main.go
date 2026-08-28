package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"bluebell/internal/config"
	"bluebell/internal/consumer"
	mysqldao "bluebell/internal/dao/mysql"
	redisdao "bluebell/internal/dao/redis"
	"bluebell/internal/logger"
	"bluebell/internal/mq"
	"bluebell/internal/snowflake"

	"go.uber.org/zap"
)

func main() {
	// 1. 加载配置
	var confFile string
	flag.StringVar(&confFile, "conf", "./config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Init(confFile)
	if err != nil {
		fmt.Printf("init config failed, err:%v\n", err)
		return
	}

	// 2. 初始化日志
	if err := logger.Init(cfg, cfg.App.Mode); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	defer zap.L().Sync()

	// 3. 初始化雪花算法
	if err := snowflake.Init(cfg); err != nil {
		zap.L().Fatal("init snowflake failed", zap.Error(err))
	}

	// 4. 初始化 MySQL 与 Redis
	gormDB, err := mysqldao.Init(cfg)
	if err != nil {
		zap.L().Fatal("Init MySQL failed", zap.Error(err))
	}
	defer mysqldao.Close(gormDB)

	rdb, err := redisdao.Init(cfg)
	if err != nil {
		zap.L().Fatal("Init Redis failed", zap.Error(err))
	}
	defer redisdao.Close(rdb)

	// 5. 初始化 DAO 与 Cache
	postDao := mysqldao.NewPostDao(gormDB)
	userDao := mysqldao.NewUserDao(gormDB)
	relationDao := mysqldao.NewRelationDao(gormDB)
	notifDao := mysqldao.NewNotificationDao(gormDB)
	eventLogDao := mysqldao.NewEventLogDao(gormDB)

	notifCache := redisdao.NewNotificationCache(rdb)
	dedupCache := redisdao.NewDedupCache(rdb)
	feedCache := redisdao.NewFeedCache(rdb)

	// 6. 初始化 MQ 事件总线（消费端）
	amqpURL := ""
	if cfg.RabbitMQ != nil {
		amqpURL = cfg.RabbitMQ.URL
	}
	eventBus, err := mq.NewEventBus(amqpURL)
	if err != nil {
		zap.L().Fatal("init event bus failed", zap.Error(err))
	}
	defer eventBus.Close()

	// 7. 启动后台异步消费 Worker 容器
	workers := consumer.NewWorkersContainer(
		eventBus,
		notifDao,
		notifCache,
		dedupCache,
		eventLogDao,
		userDao,
		postDao,
		feedCache,
		relationDao,
	)
	if err := workers.Start(context.Background()); err != nil {
		zap.L().Fatal("start consumer workers failed", zap.Error(err))
	}

	zap.L().Info("Bluebell Consumer worker process started successfully, listening for events...")
	// 8. 监听退出信号实现优雅停机
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zap.L().Info("Shutting down Consumer worker process...")
}
