package handler

import (
	domainService "bluebell/internal/domain/serviceinterface"
)

// userHandlerStruct 用户相关处理器
type userHandlerStruct struct {
	userService domainService.UserService
}

// NewUserHandler 创建 userHandlerStruct 实例
// 通过构造函数进行依赖注入
func NewUserHandler(userService domainService.UserService) *userHandlerStruct {
	if userService == nil {
		panic("userService cannot be nil")
	}
	return &userHandlerStruct{
		userService: userService,
	}
}

// postHandlerStruct 帖子相关处理器
type postHandlerStruct struct {
	postService domainService.PostService
}

// NewPostHandler 创建 postHandlerStruct 实例
// 通过构造函数进行依赖注入
func NewPostHandler(postService domainService.PostService) *postHandlerStruct {
	if postService == nil {
		panic("postService cannot be nil")
	}
	return &postHandlerStruct{
		postService: postService,
	}
}

// communityHandlerStruct 社区相关处理器
type communityHandlerStruct struct {
	communityService domainService.CommunityService
}

// NewCommunityHandler 创建 communityHandlerStruct 实例
// 通过构造函数进行依赖注入
func NewCommunityHandler(communityService domainService.CommunityService) *communityHandlerStruct {
	if communityService == nil {
		panic("communityService cannot be nil")
	}
	return &communityHandlerStruct{
		communityService: communityService,
	}
}

// voteHandlerStruct 投票相关处理器
type voteHandlerStruct struct {
	voteService domainService.VoteService
}

// NewVoteHandler 创建 voteHandlerStruct 实例
// 通过构造函数进行依赖注入
func NewVoteHandler(voteService domainService.VoteService) *voteHandlerStruct {
	if voteService == nil {
		panic("voteService cannot be nil")
	}
	return &voteHandlerStruct{
		voteService: voteService,
	}
}

// HandlerProvider 处理器提供者（DI容器）
// 聚合所有 Handler 实例，作为依赖注入的入口点
type HandlerProvider struct {
	UserHandler      *userHandlerStruct
	PostHandler      *postHandlerStruct
	CommunityHandler *communityHandlerStruct
	VoteHandler      *voteHandlerStruct
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
