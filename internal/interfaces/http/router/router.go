// Package router 实现 HTTP 接口层（Interface Layer）
//
// Why Interface Layer?
// 按照 DDD 原则，接口层负责与外部系统（用户、其他服务）进行通信。
// 1. 它负责具体的通信协议实现（此处为 HTTP/Gin）。
// 2. 它将外部输入（JSON/Query）转化为应用层需要的 DTO/参数。
// 3. 它将应用层的返回结果（Entity/Error）转化为外部需要的响应格式（JSON/HTTP Code）。
// 4. 关键：接口层隔离了具体的技术栈。如果我们要增加 gRPC 接口，
//    只需要新开一个 grpc/server.go，而 Application 和 Domain 层完全不用动。
package router

import (
	"bluebell/internal/config"
	"bluebell/internal/domain"
	"bluebell/internal/interfaces/http/handler"
	"bluebell/internal/middleware"

	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// NewRouter 初始化路由配置
// 为什么要把配置和 Handler 都传进来？
// 接口层不应该自己去构造业务对象，而是通过依赖注入（DI）接收。
// 这样我们在测试路由逻辑时，可以注入 Mock 的 Handler。
func NewRouter(
	mode string,
	hp *handler.Provider,
	cfg *config.Config,
	tokenService domain.TokenService,
	tokenCache domain.UserTokenCacheRepository,
	rdb *redis.Client,
) (*gin.Engine, error) {

	r := gin.New()

	fillInterval, err := time.ParseDuration(cfg.RateLimit.FillInterval)
	if err != nil {
		return nil, fmt.Errorf("parse rate limit fill interval failed: %w", err)
	}

	timeout, err := time.ParseDuration(cfg.Timeout.Timeout)
	if err != nil {
		return nil, fmt.Errorf("parse request timeout failed: %w", err)
	}

	var slidingWindow time.Duration
	if cfg.SlidingRateLimit != nil && cfg.SlidingRateLimit.Enabled {
		if rdb == nil {
			return nil, fmt.Errorf("redis client (rdb) is required when sliding rate limit is enabled")
		}
		slidingWindow, err = time.ParseDuration(cfg.SlidingRateLimit.Window)
		if err != nil {
			return nil, fmt.Errorf("parse sliding rate limit window failed: %w", err)
		}
	}

	r.Use(
		otelgin.Middleware("bluebell"), // OpenTelemetry 链路追踪中间件
		middleware.GinLogger(),
		middleware.GinRecovery(true),
		middleware.Cors(), // 跨域中间件
		middleware.RateLimitMiddleware(fillInterval, cfg.RateLimit.Capacity), // 令牌桶限流
	)

	if cfg.SlidingRateLimit != nil && cfg.SlidingRateLimit.Enabled {
		r.Use(middleware.SlidingRateLimitMiddleware(rdb, slidingWindow, cfg.SlidingRateLimit.Limit))
	}

	r.Use(middleware.TimeoutMiddleware(timeout))

	// Health check 端点（根路径，无需认证）
	r.GET("/healthz", hp.HealthHandler.Healthz)
	r.GET("/readyz", hp.HealthHandler.Readyz)

	// Swagger (仅在非生产环境)
	if mode != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// 路由组
	apiV1 := r.Group("/api/v1")

	// 公共路由（无需登录即可访问）
	{
		apiV1.POST("/signup", hp.UserHandler.SignUpHandler)
		apiV1.POST("/login", hp.UserHandler.LoginHandler)
		apiV1.POST("/refresh_token", hp.UserHandler.RefreshTokenHandler)

		// 社区列表
		apiV1.GET("/community", hp.CommunityHandler.GetCommunityListHandler)

		// 帖子浏览（公开）
		apiV1.GET("/posts", hp.PostHandler.GetPostListHandler)
		apiV1.GET("/post/:id", hp.PostHandler.GetPostDetailHandler)
		apiV1.GET("/post/:id/remarks", hp.PostHandler.GetPostRemarksHandler)
		apiV1.GET("/search", hp.SearchHandler.SearchHandler)

		// 社交与个人资料 (公开)
		apiV1.GET("/user/:id", hp.SocialHandler.GetProfileHandler)
		apiV1.GET("/user/:id/activities", hp.SocialHandler.GetActivitiesHandler)

		// GitHub OAuth
		apiV1.GET("/auth/github/login", hp.UserHandler.GitHubLoginHandler)
		apiV1.GET("/auth/github/callback", hp.UserHandler.GitHubCallbackHandler)
	}

	// 认证路由（需要 JWT 认证）
	// 为什么中间件也在这里？
	// 认证和限流属于“接入规则”，是接口层的一部分。
	authGroup := apiV1.Group("")
	authGroup.Use(middleware.JWTAuthMiddleware(tokenService, tokenCache))
	{
		// 社区管理
		authGroup.GET("/community/:id", hp.CommunityHandler.GetCommunityDetailHandler)
		authGroup.POST("/community", hp.CommunityHandler.CreateCommunityHandler)

		// 用户登出
		authGroup.POST("/logout", hp.UserHandler.LogoutHandler)

		// 帖子操作（需登录）
		authGroup.POST("/post", hp.PostHandler.CreatePostHandler)
		authGroup.DELETE("/post/:id", hp.PostHandler.DeletePostHandler)
		authGroup.POST("/vote", hp.PostHandler.PostVoteHandler)
		authGroup.POST("/remark", hp.PostHandler.PostRemarkHandler)

		// 关注/取消关注
		authGroup.POST("/follow/:id", hp.SocialHandler.FollowHandler)
		authGroup.DELETE("/follow/:id", hp.SocialHandler.UnfollowHandler)
	}

	// 404
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "404 page not found",
		})
	})

	return r, nil
}
