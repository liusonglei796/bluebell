package main

import (
	"flag"
	"fmt"

	"bluebell/internal/config"
	"bluebell/internal/controller"
	mysqldao "bluebell/internal/dao/mysql"
	redisdao "bluebell/internal/dao/redis"
	"bluebell/internal/http_server"
	"bluebell/internal/logger"
	"bluebell/internal/mq"
	"bluebell/internal/router"
	"bluebell/internal/service"
	"bluebell/internal/snowflake"
	"bluebell/internal/translate"

	"github.com/gin-gonic/gin"
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

	// 设置 Gin 运行模式 — 强制 ReleaseMode
	gin.SetMode(gin.ReleaseMode)

	// 2. 初始化日志
	if err := logger.Init(cfg, cfg.App.Mode); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	defer zap.L().Sync()

	// 初始化 snowflake
	if err := snowflake.Init(cfg); err != nil {
		zap.L().Fatal("init snowflake failed", zap.Error(err))
	}

	// 3. 初始化 MySQL / Redis
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

	// 初始化 Validator
	if err := translate.InitTrans(); err != nil {
		zap.L().Fatal("init validator trans failed", zap.Error(err))
	}

	// ====== 依赖装配（自下而上） ======

	// 1) DAO 层
	postDao := mysqldao.NewPostDao(gormDB)
	communityDao := mysqldao.NewCommunityDao(gormDB)
	userDao := mysqldao.NewUserDao(gormDB)
	voteDao := mysqldao.NewVoteDao(gormDB)
	commentDao := mysqldao.NewCommentDao(gormDB)
	relationDao := mysqldao.NewRelationDao(gormDB)
	notifDao := mysqldao.NewNotificationDao(gormDB)
	bookmarkDao := mysqldao.NewBookmarkDao(gormDB)
	tagDao := mysqldao.NewTagDao(gormDB)

	postCache, refresher := redisdao.NewPostCacheWithRefresher(rdb)
	tokenCache := redisdao.NewUserTokenCache(rdb)
	relationCache := redisdao.NewUserRelationCache(rdb)
	notifCache := redisdao.NewNotificationCache(rdb)
	bookmarkCache := redisdao.NewBookmarkCache(rdb)
	feedCache := redisdao.NewFeedCache(rdb)
	pinCache := redisdao.NewPinCache(rdb)

	// 启动 Gravity 热度分数定时刷新任务
	refresher.Start()
	defer refresher.Stop()

	// 启动 Redis 6.0+ Client-Side Caching (BCAST 广播追踪模式)
	localPostCache := redisdao.NewLocalPostCache()
	bcastTracker := redisdao.NewBcastTracker(rdb, localPostCache)
	if err := bcastTracker.Start(); err != nil {
		zap.L().Warn("start redis bcast tracker warning", zap.Error(err))
	}
	defer bcastTracker.Stop()

	// 2) MQ 事件总线（发布端）
	amqpURL := ""
	if cfg.RabbitMQ != nil {
		amqpURL = cfg.RabbitMQ.URL
	}
	eventBus, err := mq.NewEventBus(amqpURL)
	if err != nil {
		zap.L().Warn("init event bus warning", zap.Error(err))
	}
	defer eventBus.Close()

	// 3) Service 层
	postSvc := service.NewPostService(
		postDao,
		postCache,
		localPostCache,
		voteDao,
		commentDao,
		tagDao,
		pinCache,
		bookmarkCache,
		relationDao,
		feedCache,
		eventBus,
		communityDao,
		userDao,
	)
	communitySvc := service.NewCommunityService(communityDao, userDao)
	userSvc := service.NewUserService(userDao, tokenCache, cfg)
	commentSvc := service.NewCommentService(commentDao, postDao, userDao, postCache, eventBus)
	relationSvc := service.NewRelationService(relationDao, userDao, relationCache, eventBus)
	notifSvc := service.NewNotificationService(notifDao, notifCache, userDao)
	bookmarkSvc := service.NewBookmarkService(bookmarkDao, postDao, bookmarkCache, postSvc)
	tagSvc := service.NewTagService(tagDao, communityDao, userDao)

	// 4) Controller 层
	userController := controller.NewUserController(userSvc)
	postController := controller.NewPostController(postSvc)
	communityController := controller.NewCommunityController(communitySvc)
	commentController := controller.NewCommentController(commentSvc)
	relationController := controller.NewRelationController(relationSvc)
	notifController := controller.NewNotificationController(notifSvc)
	bookmarkController := controller.NewBookmarkController(bookmarkSvc)
	tagController := controller.NewTagController(tagSvc)

	// 5) 路由层：初始化路由，注入 Controller
	r, err := router.NewRouter(
		cfg.App.Mode,
		userController,
		postController,
		communityController,
		commentController,
		relationController,
		notifController,
		bookmarkController,
		tagController,
		cfg,
		tokenCache,
	)
	if err != nil {
		zap.L().Fatal("init router failed", zap.Error(err))
	}

	// 6) 启动 HTTP 服务（含优雅关机）
	http_server.Run(r, cfg.App.Port)
}
