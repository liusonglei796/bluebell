package router

import (
	"bluebell/internal/config"
	"bluebell/internal/handler"
	"bluebell/internal/middleware"
	"bluebell/pkg/errorx"

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

	r := gin.New()

	fillInterval, err := time.ParseDuration(cfg.RateLimit.FillInterval)
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
		middleware.RateLimitMiddleware(fillInterval, cfg.RateLimit.Capacity),
		middleware.TimeoutMiddleware(timeout),
	)

	// Swagger (仅在非生产环境)
	if mode != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// 路由组
	apiV1 := r.Group("")

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
