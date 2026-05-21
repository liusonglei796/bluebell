package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"bluebell/internal/config"
	"bluebell/internal/di"
	"bluebell/internal/domain"
	"bluebell/internal/http_server"
	"bluebell/internal/infrastructure/jwt"
	"bluebell/internal/infrastructure/es"
	"bluebell/internal/infrastructure/logger"
	"bluebell/internal/infrastructure/metrics"
	"bluebell/internal/infrastructure/mq"
	bluebellotel "bluebell/internal/infrastructure/otel"
	"bluebell/internal/infrastructure/profiler"
	database "bluebell/internal/infrastructure/persistence/mysql"
	redisrepo "bluebell/internal/infrastructure/persistence/redis"
	"bluebell/internal/infrastructure/snowflake"
	"bluebell/internal/infrastructure/translate"
	"bluebell/internal/interfaces/http/handler"
	"bluebell/internal/interfaces/http/router"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
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

	// 设置 Gin 运行模式 — 强制 ReleaseMode（禁用调试日志和 Gin 默认日志）
	gin.SetMode(gin.ReleaseMode)

	// ====== 基础设施层：OpenTelemetry ======
	// 初始化 OTel SDK（Traces + Metrics + Logs），必须在 Logger 之前
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

	// 初始化自定义业务指标（MeterProvider 已设置，使用 no-op 回退）
	serviceName := "bluebell"
	if cfg.Otel != nil {
		serviceName = cfg.Otel.ServiceName
	}
	if err := metrics.Init(serviceName); err != nil {
		zap.L().Error("init custom metrics failed", zap.Error(err))
	}

	// 2. 初始化日志（集成 OTel 自动关联）
	if err := logger.Init(cfg, cfg.App.Mode); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	defer zap.L().Sync()

	// 启动 Go 运行时指标采集（goroutine 数、内存使用等）
	if err := runtime.Start(); err != nil {
		zap.L().Error("failed to start runtime instrumentation", zap.Error(err))
	}

	// 初始化 snowflake
	if err := snowflake.Init(cfg); err != nil {
		zap.L().Fatal("init snowflake failed", zap.Error(err))
	}

	// 3. 初始化 MySQL
	gormDB, err := database.Init(cfg)
	if err != nil {
		zap.L().Fatal("Init MySQL failed", zap.Error(err))
	}
	defer database.Close(gormDB)

	// 4. 初始化 Redis
	rdb, err := redisrepo.Init(cfg)
	if err != nil {
		zap.L().Fatal("Init Redis failed", zap.Error(err))
	}
	defer redisrepo.Close(rdb)


	// 初始化 Validator
	if err := translate.InitTrans(); err != nil {
		zap.L().Fatal("init validator trans failed", zap.Error(err))
	}

	// ====== 完整的 DI 流程 ======
	// 按照分层架构从下往上依次注入依赖

	// 1) 基础设施层：创建 Repository 实例（数据访问层）
	repositoriesUOW := database.NewRepositories(gormDB)
	cacheRepos := redisrepo.NewRepositories(rdb)

	// 启动 Gravity 热度分数定时刷新任务
	cacheRepos.HotScoreRefresher.Start()
	defer cacheRepos.HotScoreRefresher.Stop()

	// ====== 基础设施层：ES / MQ ======

	// 初始化 Elasticsearch 客户端（带重试，最多等 30 秒）
	esClient, err := initESWithRetry(ctx, cfg, 30*time.Second, 3*time.Second)
	if err != nil {
		zap.L().Warn("init ES client failed after retry, running without ES", zap.Error(err))
		esClient = nil
	} else {
		if err := esClient.CreatePostIndex(ctx); err != nil {
			zap.L().Error("create ES post index failed", zap.Error(err))
		}
	}

	// ====== RabbitMQ 手动挡初始化 ======
	// (1) 建立物理连接
	conn, err := mq.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		zap.L().Error("init MQ failed", zap.Error(err))
		conn = nil
	}

	var publisher *mq.Publisher
	if conn != nil {
		// (2) 装修资源：临时信道声明 Exchange 和 Queue
		setupCh, err := conn.Channel()
		if err != nil {
			zap.L().Fatal("create setup channel failed", zap.Error(err))
		}
		if err := mq.SetupResources(setupCh); err != nil {
			zap.L().Fatal("setup rabbitmq resources failed", zap.Error(err))
		}
		setupCh.Close()

		// (3) 发布信道：给生产者用
		pubCh, err := conn.Channel()
		if err != nil {
			zap.L().Fatal("create publish channel failed", zap.Error(err))
		}
		defer pubCh.Close()
		publisher = mq.NewPublisher(pubCh)
	}

	// 2) 业务逻辑层：创建 Service 实例
	tokenService := jwt.NewJWTService(cfg)
	var searchRepo domain.PostSearchRepository
	if esClient != nil {
		searchRepo = esClient
	}
	var searchSyncRepo domain.PostSearchSyncRepository
	if publisher != nil {
		searchSyncRepo = publisher
	}
	services := di.NewServices(repositoriesUOW, cacheRepos, tokenService, searchRepo, searchSyncRepo, publisher, cfg)

	// 3) 表现层：创建 Handler 实例
	handlerProvider := handler.NewProvider(
	        services.User,
	        services.Post,
	        services.Community,
	        services.Social,
	        publisher,
	        gormDB,
	        rdb,
	        esClient,
	)
	// 4) 创建并启动 MQ 消费者
	if conn != nil {
		// 消费信道：给投票消费者用
		voteCh, err := conn.Channel()
		if err != nil {
			zap.L().Fatal("create vote consumer channel failed", zap.Error(err))
		}
		defer voteCh.Close()
		voteConsumer := mq.NewVoteConsumer(voteCh, repositoriesUOW.Vote, rdb)
		go func() {
			if err := voteConsumer.Start(ctx); err != nil {
				zap.L().Error("vote consumer exited", zap.Error(err))
			}
		}()

		if esClient != nil {
			// 消费信道：给搜索消费者用
			searchCh, err := conn.Channel()
			if err != nil {
				zap.L().Fatal("create search consumer channel failed", zap.Error(err))
			}
			defer searchCh.Close()
			esConsumer := mq.NewSyncConsumer(searchCh, esClient)
			go func() {
				if err := esConsumer.Start(ctx); err != nil {
					zap.L().Error("sync consumer exited", zap.Error(err))
				}
			}()
		}

		// 消费信道：给用户动态消费用
		activityCh, err := conn.Channel()
		if err != nil {
			zap.L().Fatal("create activity consumer channel failed", zap.Error(err))
		}
		defer activityCh.Close()
		activityConsumer := mq.NewActivityConsumer(activityCh, repositoriesUOW.Social)
		go func() {
			if err := activityConsumer.Start(ctx); err != nil {
				zap.L().Error("activity consumer exited", zap.Error(err))
			}
		}()
	}

	// 5) 路由层：初始化路由，注入 Handler
	r, err := router.NewRouter(cfg.App.Mode, handlerProvider, cfg, tokenService, cacheRepos.TokenCache)
	if err != nil {
		zap.L().Fatal("init router failed", zap.Error(err))
	}

	// 6. 启动 HTTP 服务（含优雅关机）
	http_server.Run(r, cfg.App.Port)

	// HTTP 服务已关闭，关闭 MQ 连接
	if conn != nil {
		conn.Close()
	}
}

// initESWithRetry 初始化 ES 客户端，失败时按间隔重试，最多持续 maxDuration
func initESWithRetry(ctx context.Context, cfg *config.Config, maxDuration, interval time.Duration) (*es.Client, error) {
	deadline := time.Now().Add(maxDuration)
	var lastErr error

	for time.Now().Before(deadline) {
		client, err := es.NewClient(cfg)
		if err == nil {
			return client, nil
		}
		lastErr = err
		zap.L().Warn("ES client init failed, retrying...", zap.Error(err), zap.Time("deadline", deadline))

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
	return nil, fmt.Errorf("ES init failed after %v retries: %w", maxDuration, lastErr)
}
