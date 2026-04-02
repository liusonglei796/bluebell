package router

import (
	"bluebell/internal/config"
	"bluebell/internal/domain/cachedomain"
	"bluebell/internal/handler"
	"bluebell/internal/middleware"
	"bluebell/pkg/errorx"

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
	tokenCache cachedomain.UserTokenCacheRepository,
) (*gin.Engine, error) {

	r := gin.New()

	fillInterval, err := time.ParseDuration(cfg.RateLimit.FillInterval)
	_ = fillInterval
	if err != nil {
		return nil, errorx.Wrap(err, errorx.CodeConfigError, "parse rate limit fill interval failed")
	}

	timeout, err := time.ParseDuration(cfg.Timeout.Timeout)
	if err != nil {
		return nil, errorx.Wrap(err, errorx.CodeConfigError, "parse request timeout failed")
	}

	r.Use(
		middleware.GinLogger(),
		middleware.GinRecovery(true),
		// middleware.RateLimitMiddleware(fillInterval, cfg.RateLimit.Capacity),
		middleware.TimeoutMiddleware(timeout),
	)

	// Swagger & PProf (仅在非生产环境)
	if mode != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		pprof.Register(r) // 注册 pprof 路由
	}

	// 路由组
	apiV1 := r.Group("/api/v1")

	// 公共路由（用户认证相关）
	{
		apiV1.POST("/signup", hp.UserHandler.SignUpHandler)
		apiV1.POST("/login", hp.UserHandler.LoginHandler)
		apiV1.POST("/refresh_token", hp.UserHandler.RefreshTokenHandler)

		// 社区相关（公开接口）
		apiV1.GET("/community", hp.CommunityHandler.GetCommunityListHandler)
	}

	// 认证路由（需要 JWT 认证）
	authGroup := apiV1.Group("")
	authGroup.Use(middleware.JWTAuthMiddleware(cfg, tokenCache))
	{
		// 社区相关
		authGroup.GET("/community/:id", hp.CommunityHandler.GetCommunityDetailHandler)
		authGroup.POST("/community", hp.CommunityHandler.CreateCommunityHandler)

		// 帖子相关
		authGroup.POST("/post", hp.PostHandler.CreatePostHandler)
		authGroup.GET("/post/:id", hp.PostHandler.GetPostDetailHandler)
		authGroup.GET("/post/:id/remarks", hp.PostHandler.GetPostRemarksHandler)
		authGroup.DELETE("/post/:id", hp.PostHandler.DeletePostHandler)
		authGroup.GET("/posts", hp.PostHandler.GetPostListHandler)
		authGroup.POST("/vote", hp.PostHandler.PostVoteHandler)
		authGroup.POST("remark", hp.PostHandler.PostRemarkHandler)
	}

	// 404
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"msg": "404 page not found",
		})
	})

	return r, nil
}
