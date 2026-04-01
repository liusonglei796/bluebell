package main

import (
	_ "bluebell/docs" // 导入生成的 Swagger 文档包
	"bluebell/internal/config"
	"bluebell/internal/dao/cache"
	"bluebell/internal/dao/database"
	"bluebell/internal/handler"
	"bluebell/internal/http_server"
	"bluebell/internal/infrastructure/logger"
	"bluebell/internal/infrastructure/snowflake"
	"bluebell/internal/infrastructure/translate"
	"bluebell/internal/router"
	"bluebell/internal/service"
	"flag"
	"fmt"

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

	// 2) 业务逻辑层：创建 Service 实例
	services := service.NewServices(repositoriesUOW, cacheRepos, cfg)

	// 3) 表现层：创建 Handler 实例（通过 DI 注入 Service 接口）
	handlerProvider := handler.NewProvider(
		services.User,      // 注入 UserService 接口
		services.Post,      // 注入 PostService 接口
		services.Community, // 注入 CommunityService 接口
		services.AI,        // 注入 AiSerive 接口
	)

	// 4) 路由层：初始化路由，注入 Handler
	r, err := router.NewRouter(cfg.App.Mode, handlerProvider, cfg, cacheRepos.TokenCache)
	if err != nil {
		zap.L().Fatal("init router failed", zap.Error(err))
	}

	// 5. 启动 HTTP 服务（含优雅关机）
	http_server.Run(r, cfg.App.Port)
}
