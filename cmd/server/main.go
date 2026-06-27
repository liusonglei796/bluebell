package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bluebell/internal/application"
	"bluebell/internal/config"
	"bluebell/internal/infrastructure/es"
	"bluebell/internal/infrastructure/jwt"
	infralogger "bluebell/internal/infrastructure/logger"
	"bluebell/internal/infrastructure/metrics"
	"bluebell/internal/infrastructure/mq"
	bluebellotel "bluebell/internal/infrastructure/otel"
	database "bluebell/internal/infrastructure/persistence/mysql"
	redisrepo "bluebell/internal/infrastructure/persistence/redis"
	"bluebell/internal/infrastructure/profiler"
	"bluebell/internal/infrastructure/snowflake"
	"bluebell/internal/interfaces/http/handler"
	"bluebell/internal/interfaces/http/router"

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

	if err := infralogger.Init(cfg, cfg.App.Mode); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	defer zap.L().Sync()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	if cfg.Pyroscope != nil && cfg.Pyroscope.Enabled {
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

	// 初始化业务指标（必须在 InitOTEL 之后调用）
	if err := metrics.Init(cfg.App.Name); err != nil {
		zap.L().Fatal("init metrics failed", zap.Error(err))
	}

	if err := snowflake.Init(cfg); err != nil {
		zap.L().Fatal("init snowflake failed", zap.Error(err))
	}

	gormDB, err := database.Init(cfg, bluebellotel.GetPrometheusRegistry())
	if err != nil {
		zap.L().Fatal("Init MySQL failed", zap.Error(err))
	}
	defer database.Close(gormDB)

	rdb, err := redisrepo.Init(cfg)
	if err != nil {
		zap.L().Fatal("Init Redis failed", zap.Error(err))
	}
	defer redisrepo.Close(rdb)

	mqConn, err := mq.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		zap.L().Fatal("init MQ failed", zap.Error(err))
	}
	defer mqConn.Close()

	setupCh, err := mqConn.Channel()
	if err != nil {
		zap.L().Fatal("create setup channel failed", zap.Error(err))
	}
	if err := mq.SetupResources(setupCh); err != nil {
		zap.L().Fatal("setup rabbitmq resources failed", zap.Error(err))
	}
	setupCh.Close()

	publisherCh, err := mqConn.Channel()
	if err != nil {
		zap.L().Fatal("create publisher channel failed", zap.Error(err))
	}
	defer publisherCh.Close()

	publisher := mq.NewPublisher(publisherCh)

	searchClient, err := es.NewClient(cfg)
	if err != nil {
		zap.L().Warn("init elasticsearch failed, search will be unavailable", zap.Error(err))
		searchClient = nil
	}

	// ========== 创建基础设施适配器（将 infrastructure 实现适配为 port 接口） ==========
	appLogger := infralogger.New()                          // port.Logger 适配器
	eventPublisher := mq.NewEventPublisher(publisher)       // port.EventPublisher 适配器
	idGen := snowflake.NewIDGenerator()                     // port.IDGenerator 适配器

	// ========== 创建仓储 ==========
	dbRepos := database.NewRepositories(gormDB)
	cacheRepos := redisrepo.NewRepositories(rdb)
	tokenService := jwt.NewJWTService(cfg)

	searchRepo := es.NewPostSearch(searchClient)
	searchSyncRepo := es.NewPostSync(searchClient)

	// ========== 创建应用服务（注入 port 接口而非 infrastructure 包） ==========
	postService := application.NewPostService(
		dbRepos.Post, cacheRepos.PostCache, dbRepos.Remark,
		eventPublisher, searchRepo, searchSyncRepo,
		appLogger, idGen,
	)
	communityService := application.NewCommunityService(
		dbRepos.Community, dbRepos.User, appLogger,
	)
	userService := application.NewUserService(
		dbRepos.User, dbRepos.Social, cacheRepos.TokenCache, tokenService,
		appLogger, idGen,
	)
	socialService := application.NewSocialService(
		dbRepos.Social, dbRepos.User, eventPublisher,
	)
	bookmarkService := application.NewBookmarkService(
		dbRepos.Bookmark, dbRepos.Post, dbRepos.User, dbRepos.Community,
		appLogger,
	)

	// ========== 创建 Handler Provider（传入 port 接口） ==========
	hp := handler.NewProvider(
		userService,
		postService,
		communityService,
		socialService,
		bookmarkService,
		idGen,
		gormDB,
		rdb,
		searchClient,
		cfg.Upload.Dir,
	)

	r, err := router.NewRouter(cfg.App.Mode, hp, cfg, tokenService, cacheRepos.TokenCache, rdb)
	if err != nil {
		zap.L().Fatal("init router failed", zap.Error(err))
	}

	// 创建 HTTP 服务器实例
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.App.Port),
		Handler: r,
	}

	// 在 goroutine 中启动 HTTP 监听
	go func() {
		zap.L().Info("HTTP server starting", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("HTTP server listen failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zap.L().Info("Shutting down server...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		zap.L().Fatal("Server forced to shutdown:", zap.Error(err))
	}

	zap.L().Info("Server exited")
}
