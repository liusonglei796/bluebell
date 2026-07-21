package router

import (
	"bluebell/internal/config"
	"bluebell/internal/domain"
	"bluebell/internal/interfaces/http/handler"
	"bluebell/internal/middleware"

	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// NewRouter 初始化路由配置
// 接收 Provider（DI 容器）作为参数
func NewRouter(
	mode string,
	hp *handler.Provider,
	cfg *config.Config,
	tokenCache domain.UserTokenCacheRepository,
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

	r.Use(
		middleware.GinLogger(),
		middleware.GinRecovery(true),
		middleware.Cors(), // 跨域中间件
		middleware.RateLimitMiddleware(fillInterval, cfg.RateLimit.Capacity), // 令牌桶限流
		middleware.TimeoutMiddleware(timeout),
	)

	// Swagger & PProf (仅在非生产环境)
	if mode != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		pprof.Register(r) // 注册 pprof 路由
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
	}

	// 认证路由（需要 JWT 认证）
	authGroup := apiV1.Group("")
	authGroup.Use(middleware.JWTAuthMiddleware(cfg, tokenCache))
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
	}

	// 404
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "404 page not found",
		})
	})

	return r, nil
}
