// Package handler 提供所有 HTTP 处理器的聚合与导出
// 聚合各个子包中的处理器，并通过 Provider 统一管理依赖注入
package handler

import (
	"bluebell/internal/application"
	"bluebell/internal/infrastructure/es"
	"bluebell/internal/infrastructure/mq"
	"bluebell/internal/interfaces/http/handler/community_handler"
	"bluebell/internal/interfaces/http/handler/health"
	"bluebell/internal/interfaces/http/handler/post_handler"
	"bluebell/internal/interfaces/http/handler/search_handler"
	"bluebell/internal/interfaces/http/handler/social_handler"
	"bluebell/internal/interfaces/http/handler/user_handler"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// ========== Handler Provider ==========

// Provider 处理器提供者（DI容器）
// 聚合所有 Handler 实例，作为依赖注入的入口点
type Provider struct {
	UserHandler      *user_handler.Handler
	PostHandler      *post_handler.Handler
	CommunityHandler *community_handler.Handler
	SocialHandler    *social_handler.Handler
	SearchHandler    *search_handler.Handler
	HealthHandler    *health.Handler
}

// NewProvider 创建 Provider 实例
// 通过 Services 进行完整的依赖注入和装配
func NewProvider(
	userService application.UserService,
	postService application.PostService,
	communityService application.CommunityService,
	socialService application.SocialService,
	publisher *mq.Publisher,
	db *gorm.DB,
	rdb *redis.Client,
	esClient *es.Client,
) *Provider {
	return &Provider{
		UserHandler:      user_handler.New(userService),
		PostHandler:      post_handler.New(postService),
		CommunityHandler: community_handler.New(communityService),
		SocialHandler:    social_handler.New(socialService),
		SearchHandler:    search_handler.New(postService),
		HealthHandler:    health.New(db, rdb, esClient),
	}
}
