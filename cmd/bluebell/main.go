package main

import (
	"context"
	"flag"
	"fmt"

	"bluebell/internal/config"
	"bluebell/internal/dao/cache"
	"bluebell/internal/dao/database"
	"bluebell/internal/handler"
	"bluebell/internal/http_server"
	"bluebell/internal/infrastructure/ai"
	"bluebell/internal/infrastructure/es"
	"bluebell/internal/infrastructure/logger"
	"bluebell/internal/infrastructure/metrics"
	"bluebell/internal/infrastructure/mq"
	"bluebell/internal/infrastructure/otel"
	"bluebell/internal/infrastructure/snowflake"
	"bluebell/internal/infrastructure/translate"
	"bluebell/internal/middleware"
	"bluebell/internal/router"
	"bluebell/internal/service"

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

	// 设置 Gin 运行模式 — 强制 ReleaseMode（禁用调试日志和 Gin 默认日志）
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

	// 3. 初始化 MySQL
	gormDB, err := database.Init(cfg)
	if err != nil {
		zap.L().Fatal("Init MySQL failed", zap.Error(err))
	}
	defer database.Close(gormDB)

	// 4. 初始化 Redis
	rdb, err := cache.Init(cfg)
	if err != nil {
		zap.L().Fatal("Init Redis failed", zap.Error(err))
	}
	defer cache.Close(rdb)

	// ====== 基础设施层：OpenTelemetry & Prometheus ======
	// 初始化 OpenTelemetry TracerProvider
	otelShutdown := otel.InitTracerProvider(cfg)
	defer otelShutdown()

	// 注册 Prometheus 指标收集器
	middleware.RegisterPrometheusMetrics()

	// 注册自定义业务指标（Metrics 已自动注册到 Prometheus 全局注册表）
	metrics.RegisterCustomMetrics()

	// 初始化 Validator
	if err := translate.InitTrans(); err != nil {
		zap.L().Fatal("init validator trans failed", zap.Error(err))
	}

	// ====== 完整的 DI 流程 ======
	// 按照分层架构从下往上依次注入依赖

	// 1) 基础设施层：创建 Repository 实例（数据访问层）
	repositoriesUOW := database.NewRepositories(gormDB)
	cacheRepos := cache.NewRepositories(rdb)

	// 启动 Gravity 热度分数定时刷新任务
	cacheRepos.HotScoreRefresher.Start()
	defer cacheRepos.HotScoreRefresher.Stop()

	// ====== 基础设施层：ES / AI / MQ ======
	ctx := context.Background()

	// 初始化 RabbitMQ (必须在 services 之前，因为 PostService 需要 publisher)
	conn, publisher, err := mq.InitMQ(ctx, cfg)
	if err != nil {
		zap.L().Error("init MQ failed", zap.Error(err))
		conn = nil
		publisher = nil
	}

	// 2) 业务逻辑层：创建 Service 实例
	services := service.NewServices(repositoriesUOW, cacheRepos, publisher, cfg)

	// 初始化 Elasticsearch 客户端
	esClient, err := es.NewClient(cfg)
	if err != nil {
		zap.L().Error("init ES client failed", zap.Error(err))
		// ES 非关键依赖，记录错误后继续启动
		esClient = nil
	} else {
		// 确保 post 索引存在
		if err := esClient.CreatePostIndex(ctx); err != nil {
			zap.L().Error("create ES post index failed", zap.Error(err))
		}
	}

	// 初始化 AI Auditor
	auditor, err := ai.NewAuditor(ctx, cfg)
	if err != nil {
		zap.L().Error("init AI auditor failed", zap.Error(err))
		auditor = nil
	}

	// 3) 表现层：创建 Handler 实例（通过 DI 注入 Service 接口 + MQ Publisher）
	handlerProvider := handler.NewProvider(
		services.User,      // 注入 UserService 接口
		services.Post,      // 注入 PostService 接口
		services.Community, // 注入 CommunityService 接口
		publisher,          // 注入 MQ Publisher（可为 nil）
	)

	// 创建并启动所有 MQ 消费者（非阻塞 goroutine）
	// VoteConsumer: 投票异步计数（始终启动）
	// AuditConsumer: AI 内容审核（auditor 不为空时启动）
	// SyncConsumer: ES 数据同步（esClient 不为空时启动）
	// 各消费者在独立 goroutine 中并行运行，互不干扰
	if conn != nil {
		// VoteConsumer: 投票异步落盘 MySQL
		voteConsumer := mq.NewVoteConsumer(conn, repositoriesUOW.Vote)
		go func() {
			if err := voteConsumer.Start(ctx); err != nil {
				zap.L().Error("vote consumer exited", zap.Error(err))
			}
		}()

		// AuditConsumer: AI 内容审核
		if auditor != nil {
			auditConsumer := mq.NewAuditConsumer(conn, auditor, func(ctx context.Context, msgType string, postID string, remarkID uint, violations []string, reason string) {
				switch msgType {
				case "post":
					// 审核不通过：将帖子状态设为 -1（审核失败隐藏）
					if err := services.Post.UpdatePostStatus(ctx, postID, -1); err != nil {
						zap.L().Error("hide post on audit failure failed",
							zap.String("post_id", postID),
							zap.Error(err))
					} else {
						zap.L().Info("post hidden due to audit failure",
							zap.String("post_id", postID),
							zap.Strings("violations", violations),
							zap.String("reason", reason))
					}
				case "remark":
					// 审核不通过：删除违规评论
					if remarkID == 0 {
						zap.L().Warn("remark audit failure with no remarkID, skipping")
						return
					}
					if err := services.Post.DeleteRemark(ctx, remarkID); err != nil {
						zap.L().Error("delete remark on audit failure failed",
							zap.Uint("remark_id", remarkID),
							zap.Error(err))
					} else {
						zap.L().Info("remark deleted due to audit failure",
							zap.Uint("remark_id", remarkID),
							zap.Strings("violations", violations),
							zap.String("reason", reason))
					}
				}
			})
			go func() {
				if err := auditConsumer.Start(ctx); err != nil {
					zap.L().Error("audit consumer exited", zap.Error(err))
				}
			}()
		}

		// SyncConsumer: ES 数据同步
		if esClient != nil {
			esConsumer := mq.NewSyncConsumer(conn, esClient)
			go func() {
				if err := esConsumer.Start(ctx); err != nil {
					zap.L().Error("sync consumer exited", zap.Error(err))
				}
			}()
		}
	}

	// 4) 路由层：初始化路由，注入 Handler
	r, err := router.NewRouter(cfg.App.Mode, handlerProvider, cfg, cacheRepos.TokenCache)
	if err != nil {
		zap.L().Fatal("init router failed", zap.Error(err))
	}

	// 5. 启动 HTTP 服务（含优雅关机）
	// http_server.Run 内部已处理信号监听和优雅关闭，此处阻塞等待其返回
	http_server.Run(r, cfg.App.Port)

	// HTTP 服务已关闭，关闭 MQ 连接
	if conn != nil {
		conn.Close()
	}
}
