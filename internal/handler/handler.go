package handler

import (
	domainService "bluebell/internal/domain/service"
)

// UserHandler 用户相关处理器
type UserHandler struct {
	userService domainService.UserService
}

// NewUserHandler 创建 UserHandler 实例
// 通过构造函数进行依赖注入
func NewUserHandler(userService domainService.UserService) *UserHandler {
	if userService == nil {
		panic("userService cannot be nil")
	}
	return &UserHandler{
		userService: userService,
	}
}
// PostHandler 帖子相关处理器
type PostHandler struct {
	postService domainService.PostService
}

// NewPostHandler 创建 PostHandler 实例
// 通过构造函数进行依赖注入
func NewPostHandler(postService domainService.PostService) *PostHandler {
	if postService == nil {
		panic("postService cannot be nil")
	}
	return &PostHandler{
		postService: postService,
	}
}

// CommunityHandler 社区相关处理器
type CommunityHandler struct {
	communityService domainService.CommunityService
}

// NewCommunityHandler 创建 CommunityHandler 实例
// 通过构造函数进行依赖注入
func NewCommunityHandler(communityService domainService.CommunityService) *CommunityHandler {
	if communityService == nil {
		panic("communityService cannot be nil")
	}
	return &CommunityHandler{
		communityService: communityService,
	}
}

// VoteHandler 投票相关处理器
type VoteHandler struct {
	voteService domainService.VoteService
}

// NewVoteHandler 创建 VoteHandler 实例
// 通过构造函数进行依赖注入
func NewVoteHandler(voteService domainService.VoteService) *VoteHandler {
	if voteService == nil {
		panic("voteService cannot be nil")
	}
	return &VoteHandler{
		voteService: voteService,
	}
}

// HandlerProvider 处理器提供者（DI容器）
// 聚合所有 Handler 实例，作为依赖注入的入口点
type HandlerProvider struct {
	UserHandler      *UserHandler
	PostHandler      *PostHandler
	CommunityHandler *CommunityHandler
	VoteHandler      *VoteHandler
}

// NewHandlerProvider 创建 HandlerProvider 实例
// 通过 Services 进行完整的依赖注入和装配
func NewHandlerProvider(
	userService domainService.UserService,
	postService domainService.PostService,
	communityService domainService.CommunityService,
	voteService domainService.VoteService,
) *HandlerProvider {
	return &HandlerProvider{
		UserHandler:      NewUserHandler(userService),
		PostHandler:      NewPostHandler(postService),
		CommunityHandler: NewCommunityHandler(communityService),
		VoteHandler:      NewVoteHandler(voteService),
	}
}

