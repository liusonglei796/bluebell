package routers

import (
	"bluebell/controller"
	"bluebell/logger"
	"bluebell/middlewares"
	"bluebell/pkg/errorx"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter 初始化路由配置
// mode: 运行模式 (debug, release, test)
func SetupRouter(mode string) *gin.Engine {
	// 1. 设置 Gin 的运行模式
	if mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode) // 生产模式
	}

	// 2. 创建引擎 (使用 New 而不是 Default，以便自定义中间件)
	r := gin.New()

	// 3. 注册全局中间件
	// Logger: 使用我们自定义的 zap 日志库
	// Recovery: 捕获 panic 防止程序崩溃，并记录堆栈
	r.Use(logger.GinLogger(), logger.GinRecovery(true))

	// 4. 添加Swagger路由 (仅在非生产环境中)
	if mode != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// 5. 注册路由组
	v1 := r.Group("/api/v1")

	// ----------------------------------------------------------------
	// A. 公共路由 (Public Routes)
	// 不需要 JWT 认证即可访问
	// ----------------------------------------------------------------
	{
		v1.POST("/signup", controller.SignUpHandler)   // 注册
		v1.POST("/login", controller.LoginHandler)     // 登录
		v1.POST("/refresh_token", controller.RefreshTokenHandler) // 刷新token
	}

	// ----------------------------------------------------------------
	// B. 认证路由 (Protected Routes)
	// 需要 Header 中携带 Authorization: Bearer <token>
	// ----------------------------------------------------------------
	// 严谨做法：使用 v1.Group("") 创建一个新的路由组实例，专门用于鉴权
	// 这样可以确保 authGroup 下的所有路由 100% 经过 JWT 中间件
	authGroup := v1.Group("") 
	authGroup.Use(middlewares.JWTAuthMiddleware()) 
	{
		// 1. 社区相关
		authGroup.GET("/community", controller.CommunityHandler) // 获取社区列表
		authGroup.GET("/community/:id", controller.CommunityDetailHandler) // 获取社区详情

		// 2. 帖子相关
		authGroup.POST("/post", controller.CreatePostHandler)         // 创建帖子
		authGroup.GET("/post/:id", controller.GetPostDetailHandler)   // 获取帖子
		authGroup.GET("/posts", controller.GetPostListHandler)      // 获取帖子列表（升级版，支持按时间/分数排序
		// 3. 投票相关
		authGroup.POST("/vote", controller.PostVoteHandler)           // 帖子投票

		// 4. 系统检测 (Ping)
		authGroup.GET("/ping", func(c *gin.Context) {
			// 这里演示如何从上下文获取经过中间件解析的 UserID
			userID, exists := c.Get(controller.CtxUserIDKey)
			if !exists {
				// 理论上经过中间件不应该出现这种情况，但为了严谨处理异常
				controller.ResponseError(c, errorx.ErrServerBusy)
				return 
			}
			
			c.JSON(http.StatusOK, gin.H{
				"msg":     "pong",
				"user_id": userID,
			})
		})
 

	}

	// 6. 处理 404 (可选，严谨的 API 服务通常会返回 JSON 格式的 404)
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"msg": "404 page not found",
		})
	})

	return r
}