package main

import (
	_ "bluebell/docs" // 导入生成的 Swagger 文档包
	"bluebell/internal/config"
	"bluebell/internal/dao/mysql"
	"bluebell/internal/dao/redis"
	"bluebell/internal/handler"
	"bluebell/internal/http_server"
	"bluebell/internal/infrastructure/logger"
	"bluebell/internal/router"
	"bluebell/internal/service"
	"bluebell/internal/snowflake"
	"flag"
	"fmt"
	"time"

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

	// 设置 Gin 运行模式
	switch cfg.App.Mode {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	// 2. 初始化日志
	if err := logger.Init(cfg, cfg.App.Mode); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	defer zap.L().Sync()

	// 解析 snowflake 起始时间
	startTime, err := time.Parse("2006-01-02", cfg.Snowflake.StartTime)
	if err != nil {
		// 尝试 RFC3339 格式
		startTime, err = time.Parse(time.RFC3339, cfg.Snowflake.StartTime)
		if err != nil {
			zap.L().Fatal("parse snowflake start_time failed", zap.Error(err))
		}
	}
	if err := snowflake.Init(startTime, cfg.Snowflake.MachineID); err != nil {
		zap.L().Fatal("init snowflake failed", zap.Error(err))
	}

	// 3. 初始化 MySQL
	gormDB, err := mysql.Init(cfg)
	if err != nil {
		zap.L().Fatal("Init MySQL failed", zap.Error(err))
	}
	defer mysql.Close(gormDB)

	// 4. 初始化 Redis
	if err := redis.Init(cfg); err != nil {
		zap.L().Fatal("Init Redis failed", zap.Error(err))
	}
	defer redis.Close()

	// 初始化 Validator
	if err := handler.InitTrans("zh"); err != nil {
		zap.L().Fatal("init validator trans failed", zap.Error(err))
	}

	// ====== 依赖注入：DDD 风格组装整个应用 ======

	// 1) 创建 UnitOfWork 实例（DAO 层）
	uow := mysql.NewRepositories(gormDB)

	// 2) 创建 Cache Repository 实例
	voteCache := redis.NewVoteCache()
	tokenCache := redis.NewUserTokenCache()

	// 3) 创建 Services 聚合器，注入 UnitOfWork 和 Cache
	// VoteCache 同时实现了 VoteCacheRepository 和 PostCacheRepository
	services := service.NewServices(uow, voteCache, voteCache, tokenCache, cfg)

	// 4) 创建 Handlers 聚合器，注入 Services
	handlers := handler.NewHandlers(services)

	// 5) 注册路由，注入 Handlers
	r, err := router.NewRouter(cfg.App.Mode, handlers, cfg)
	if err != nil {
		zap.L().Fatal("init router failed", zap.Error(err))
	}

	// 6. 启动 HTTP 服务（含优雅关机）
	http_server.Run(r, cfg.App.Port)
}
