// Package handler 提供所有 HTTP 处理器的聚合与导出
// 聚合各个子包中的处理器，并通过 Provider 统一管理依赖注入
package handler

import (
	"bluebell/internal/application"
	"bluebell/internal/infrastructure/mq"
	"bluebell/internal/interfaces/http/handler/community_handler"
	"bluebell/internal/interfaces/http/handler/post_handler"
	"bluebell/internal/interfaces/http/handler/user_handler"
)

// ========== Handler Provider ==========

// Provider 处理器提供者（DI容器）
// 聚合所有 Handler 实例，作为依赖注入的入口点
type Provider struct {
	UserHandler      *user_handler.Handler
	PostHandler      *post_handler.Handler
	CommunityHandler *community_handler.Handler
}

// NewProvider 创建 Provider 实例
// 通过 Services 进行完整的依赖注入和装配
func NewProvider(
	userService application.UserService,
	postService application.PostService,
	communityService application.CommunityService,
	publisher *mq.Publisher,
) *Provider {
	return &Provider{
		UserHandler:      user_handler.New(userService),
		PostHandler:      post_handler.New(postService, publisher),
		CommunityHandler: community_handler.New(communityService),
	}
}
