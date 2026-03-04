package router

import (
	"bluebell/internal/config"
	"bluebell/internal/handler"
	"bluebell/internal/infrastructure/logger"
	"bluebell/internal/middleware"
	"bluebell/pkg/errorx"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// NewRouter 初始化路由配置
func NewRouter(
	mode string,
	h *handler.Handlers,
	rateLimitCfg *config.RateLimitConfig,
	jwtCfg *config.JWTConfig,
) *gin.Engine {
	if mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// 全局中间件
	var fillInterval time.Duration
	var capacity int64

	if rateLimitCfg != nil {
		var err error
		fillInterval, err = time.ParseDuration(rateLimitCfg.FillInterval)
		if err != nil {
			fillInterval = 10 * time.Millisecond
		}
		capacity = rateLimitCfg.Capacity
	} else {
		fillInterval = 10 * time.Millisecond
		capacity = 200
	}

	r.Use(
		logger.GinLogger(),
		logger.GinRecovery(true),
		middleware.RateLimitMiddleware(fillInterval, capacity),
		middleware.TimeoutMiddleware(10*time.Second),
	)

	// Swagger (仅在非生产环境)
	if mode != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// 路由组
	v1 := r.Group("/api/v1")

	// 公共路由
	{
		v1.POST("/signup", h.SignUpHandler)
		v1.POST("/login", h.LoginHandler)
		v1.POST("/refresh_token", h.RefreshTokenHandler)
	}

	// 认证路由
	authGroup := v1.Group("")
	authGroup.Use(middleware.JWTAuthMiddleware(jwtCfg))
	{
		// 社区相关
		authGroup.GET("/community", h.CommunityHandler)
		authGroup.GET("/community/:id", h.CommunityHandlerByID)

		// 帖子相关
		authGroup.POST("/post", h.CreatePostHandler)
		authGroup.GET("/post/:id", h.GetPostDetailHandler)
		authGroup.DELETE("/post/:id", h.DeletePostHandler)
		authGroup.GET("/posts", h.GetPostListHandler)

		// 投票相关
		authGroup.POST("/vote", h.PostVoteHandler)

		// Ping
		authGroup.GET("/ping", func(c *gin.Context) {
			userID, exists := c.Get(handler.CtxUserIDKey)
			if !exists {
				handler.ResponseError(c, errorx.ErrServerBusy)
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"msg":     "pong",
				"user_id": userID,
			})
		})
	}

	// 404
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"msg": "404 page not found",
		})
	})

	return r
}
