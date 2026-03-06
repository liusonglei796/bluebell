package router

import (
	"bluebell/internal/config"
	"bluebell/internal/handler"
	"bluebell/internal/middleware"

	"bluebell/pkg/errorx"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// NewRouter 初始化路由配置
// 接收 HandlerProvider（DI 容器）作为参数
func NewRouter(
	mode string,
	hp *handler.HandlerProvider,
	cfg *config.Config,
) (*gin.Engine, error) {
	if mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	if cfg.RateLimit == nil {
		return nil, errorx.New(errorx.CodeConfigError, "missing rate limit configuration")
	}

	fillInterval, err := time.ParseDuration(cfg.RateLimit.FillInterval)
	if err != nil {
		return nil, fmt.Errorf("parse rate limit fill_interval failed: %w", err)
	}
	capacity := cfg.RateLimit.Capacity

	r.Use(
		middleware.GinLogger(),
		middleware.GinRecovery(true),
		middleware.RateLimitMiddleware(fillInterval, capacity),
		middleware.TimeoutMiddleware(10*time.Second),
	)

	// Swagger (仅在非生产环境)
	if mode != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// 路由组
	apiV1 := r.Group("/api/v1")

	// 公共路由（用户认证相关）
	{
		apiV1.POST("/signup", hp.UserHandler.SignUpHandler)
		apiV1.POST("/login", hp.UserHandler.LoginHandler)
		apiV1.POST("/refresh_token", hp.UserHandler.RefreshTokenHandler)
	}

	// 认证路由（需要 JWT 认证）
	authGroup := apiV1.Group("")
	authGroup.Use(middleware.JWTAuthMiddleware(cfg))
	{
		// 社区相关
		authGroup.GET("/community", hp.CommunityHandler.GetCommunityListHandler)
		authGroup.GET("/community/:id", hp.CommunityHandler.GetCommunityDetailHandler)

		// 帖子相关
		authGroup.POST("/post", hp.PostHandler.CreatePostHandler)
		authGroup.GET("/post/:id", hp.PostHandler.GetPostDetailHandler)
		authGroup.DELETE("/post/:id", hp.PostHandler.DeletePostHandler)
		authGroup.GET("/posts", hp.PostHandler.GetPostListHandler)

		// 投票相关
		authGroup.POST("/vote", hp.VoteHandler.PostVoteHandler)
	}

	// 404
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"msg": "404 page not found",
		})
	})

	return r, nil
}
