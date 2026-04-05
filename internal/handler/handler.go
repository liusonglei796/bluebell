// Package handler 提供所有 HTTP 处理器的聚合与导出
// 聚合各个子包中的处理器，并通过 Provider 统一管理依赖注入
package handler

import (
	// 领域层 - Service 接口
	"bluebell/internal/domain/svcdomain"

	// Handler 层 - HTTP 处理器
	"bluebell/internal/handler/community_handler"
	"bluebell/internal/handler/post_handler"
	"bluebell/internal/handler/user_handler"

	// 基础设施 - MQ
	"bluebell/internal/infrastructure/mq"
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
	userService svcdomain.UserService,
	postService svcdomain.PostService,
	communityService svcdomain.CommunityService,
	publisher *mq.MQPublisher,
) *Provider {
	return &Provider{
		UserHandler:      user_handler.New(userService),
		PostHandler:      post_handler.New(postService, publisher),
		CommunityHandler: community_handler.New(communityService),
	}
}
