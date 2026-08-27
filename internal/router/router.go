// Package router 提供路由注册
package router

import (
	"fmt"
	"net/http"
	"time"

	"bluebell/internal/config"
	"bluebell/internal/controller"
	"bluebell/internal/dao/redis"
	"bluebell/internal/middleware"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// NewRouter 初始化路由配置
func NewRouter(
	mode string,
	userController *controller.UserController,
	postController *controller.PostController,
	communityController *controller.CommunityController,
	commentController *controller.CommentController,
	relationController *controller.RelationController,
	notifController *controller.NotificationController,
	bookmarkController *controller.BookmarkController,
	tagController *controller.TagController,
	cfg *config.Config,
	tokenCache *redis.UserTokenCache,
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
	apiV1.Use(middleware.JWTOptionalAuthMiddleware(cfg, tokenCache))
	// 公共路由（无需登录即可访问）
	{
		apiV1.POST("/signup", userController.SignUpHandler)
		apiV1.POST("/login", userController.LoginHandler)
		apiV1.POST("/refresh_token", userController.RefreshTokenHandler)

		// 社区与标签
		apiV1.GET("/community", communityController.GetCommunityListHandler)
		apiV1.GET("/community/:id", communityController.GetCommunityDetailHandler)
		apiV1.GET("/tags", tagController.GetCommunityTagsHandler)

		// 帖子浏览（公开）
		apiV1.GET("/posts", postController.GetPostListHandler)
		apiV1.GET("/post/:id", postController.GetPostDetailHandler)

		// 二级评论浏览（公开）
		apiV1.GET("/comments", commentController.GetCommentListHandler)
		apiV1.GET("/comment/replies", commentController.GetSubRepliesHandler)

		// 关注与粉丝列表公开查看
		apiV1.GET("/user/:id/following", relationController.GetFollowingListHandler)
		apiV1.GET("/user/:id/followers", relationController.GetFollowerListHandler)
	}

	// 认证路由（需要 JWT 认证）
	authGroup := apiV1.Group("")
	authGroup.Use(middleware.JWTAuthMiddleware(cfg, tokenCache))
	{
		// 社区管理（需管理员权限）
		authGroup.POST("/community", communityController.CreateCommunityHandler)

		// 用户登出
		authGroup.POST("/logout", userController.LogoutHandler)

		// 帖子操作（需登录）
		authGroup.POST("/post", postController.CreatePostHandler)
		authGroup.DELETE("/post/:id", postController.DeletePostHandler)
		authGroup.POST("/post/pin", postController.PinPostHandler)
		authGroup.GET("/feed", postController.GetFeedListHandler)
		authGroup.POST("/vote", postController.PostVoteHandler)

		// 二级楼中楼评论操作（需登录）
		authGroup.POST("/comment", commentController.CreateCommentHandler)
		authGroup.DELETE("/comment/:id", commentController.DeleteCommentHandler)

		// 社交关注操作
		authGroup.POST("/user/follow", relationController.FollowHandler)

		// 消息通知中心
		authGroup.GET("/notifications", notifController.GetNotificationListHandler)
		authGroup.GET("/notifications/unread", notifController.GetUnreadCountHandler)
		authGroup.POST("/notifications/read", notifController.MarkReadHandler)

		// 收藏夹管理
		authGroup.POST("/bookmark/folder", bookmarkController.CreateFolderHandler)
		authGroup.GET("/bookmark/folders", bookmarkController.GetUserFoldersHandler)
		authGroup.POST("/bookmark", bookmarkController.AddBookmarkHandler)
		authGroup.DELETE("/bookmark", bookmarkController.RemoveBookmarkHandler)
		authGroup.GET("/bookmarks", bookmarkController.GetBookmarkPostListHandler)

		// 标签创建
		authGroup.POST("/tag", tagController.CreateTagHandler)
	}

	// 404
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "404 page not found",
		})
	})

	return r, nil
}
