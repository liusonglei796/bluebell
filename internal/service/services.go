// Package service 提供 Service 层聚合与构造
package service

import (
	"bluebell/internal/domain/repository"
	"bluebell/internal/service/community"
	"bluebell/internal/service/post"
	"bluebell/internal/service/user"
	"bluebell/internal/service/vote"
)

// Services 聚合所有 Service 实例
// 作为依赖注入的入口，Handler 层通过 service.Services 访问各个 Service
type Services struct {
	Post      *post.PostService           // 帖子 Service
	Community *community.CommunityService // 社区 Service
	User      *user.UserService           // 用户 Service
	Vote      *vote.VoteService           // 投票 Service
}

// NewServices 创建并注入所有 Service 实例
// uow: UnitOfWork 接口，提供事务支持和 Repository 访问
// voteCache: 投票缓存仓储
// postCache: 帖子缓存仓储
// tokenCache: 用户Token缓存仓储
func NewServices(
	uow repository.UnitOfWork,
	voteCache repository.VoteCacheRepository,
	postCache repository.PostCacheRepository,
	tokenCache repository.UserTokenCacheRepository,
) *Services {
	postSvc := post.NewPostService(uow.PostRepo(), postCache, voteCache)
	communitySvc := community.NewCommunityService(uow.CommunityRepo())
	userSvc := user.NewUserService(uow.UserRepo(), tokenCache)
	voteSvc := vote.NewVoteService(uow.PostRepo(), voteCache)

	return &Services{
		Post:      postSvc,
		Community: communitySvc,
		User:      userSvc,
		Vote:      voteSvc,
	}
}
