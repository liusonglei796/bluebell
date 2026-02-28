package routers

import (
	"bluebell/controller"
	"bluebell/logger"
	"bluebell/middlewares"
	"bluebell/pkg/errorx"
	"bluebell/settings"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter 初始化路由配置
// mode: 运行模式 (debug, release, test)
func SetupRouter(
	mode string,
	uc *controller.UserController,
	cc *controller.CommunityController,
	pc *controller.PostController,
	vc *controller.VoteController,
) *gin.Engine {
	// 1. 设置 Gin 的运行模式
	if mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	// 2. 创建引擎
	r := gin.New()

	// 3. 注册全局中间件
	var fillInterval time.Duration
	var capacity int64

	if settings.Conf.RateLimit != nil {
		var err error
		fillInterval, err = time.ParseDuration(settings.Conf.RateLimit.FillInterval)
		if err != nil {
			fillInterval = 10 * time.Millisecond
		}
		capacity = settings.Conf.RateLimit.Capacity
	} else {
		fillInterval = 10 * time.Millisecond
		capacity = 200
	}

	r.Use(
		logger.GinLogger(),
		logger.GinRecovery(true),
		middlewares.RateLimitMiddleware(fillInterval, capacity),
		middlewares.TimeoutMiddleware(10*time.Second),
	)

	// 4. 添加Swagger路由 (仅在非生产环境中)
	if mode != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// 5. 注册路由组
	v1 := r.Group("/api/v1")

	// ----------------------------------------------------------------
	// A. 公共路由 (Public Routes)
	// ----------------------------------------------------------------
	{
		v1.POST("/signup", uc.SignUpHandler)
		v1.POST("/login", uc.LoginHandler)
		v1.POST("/refresh_token", uc.RefreshTokenHandler)
	}

	// ----------------------------------------------------------------
	// B. 认证路由 (Protected Routes)
	// ----------------------------------------------------------------
	authGroup := v1.Group("")
	authGroup.Use(middlewares.JWTAuthMiddleware())
	{
		// 1. 社区相关
		authGroup.GET("/community", cc.CommunityHandler)
		authGroup.GET("/community/:id", cc.CommunityHandlerByID)

		// 2. 帖子相关
		authGroup.POST("/post", pc.CreatePostHandler)
		authGroup.GET("/post/:id", pc.GetPostDetailHandler)
		authGroup.DELETE("/post/:id", pc.DeletePostHandler)
		authGroup.GET("/posts", pc.GetPostListHandler)

		// 3. 投票相关
		authGroup.POST("/vote", vc.PostVoteHandler)

		// 4. 系统检测 (Ping)
		authGroup.GET("/ping", func(c *gin.Context) {
			userID, exists := c.Get(controller.CtxUserIDKey)
			if !exists {
				controller.ResponseError(c, errorx.ErrServerBusy)
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"msg":     "pong",
				"user_id": userID,
			})
		})
	}

	// 6. 处理 404
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"msg": "404 page not found",
		})
	})

	return r
}
