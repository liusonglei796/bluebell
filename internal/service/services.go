// Package service 提供 Service 层聚合与构造
package service

import (
	"bluebell/internal/config"
	"bluebell/internal/domain/repointerface"
	domainService "bluebell/internal/domain/serviceinterface"
	"bluebell/internal/service/community"
	"bluebell/internal/service/post"
	"bluebell/internal/service/user"
	"bluebell/internal/service/vote"
)

// Services 聚合所有 Service 实例
// 作为依赖注入的入口，Handler 层通过 service.Services 访问各个 Service
type Services struct {
	Post      domainService.PostService      // 帖子 Service 接口
	Community domainService.CommunityService // 社区 Service 接口
	User      domainService.UserService      // 用户 Service 接口
	Vote      domainService.VoteService      // 投票 Service 接口
}

// NewServices 创建并注入所有 Service 实例
// uow: UnitOfWork 接口，提供事务支持和 Repository 访问
// voteCache: 投票缓存仓储
// postCache: 帖子缓存仓储
// tokenCache: 用户Token缓存仓储
// jwtCfg: JWT 配置
func NewServices(
	uow repointerface.UnitOfWork,
	voteCache repointerface.VoteCacheRepository,
	postCache repointerface.PostCacheRepository,
	tokenCache repointerface.UserTokenCacheRepository,
	jwtCfg *config.Config,
) *Services {
	postSvc := post.NewPostService(uow.PostRepo(), postCache, voteCache)
	communitySvc := community.NewCommunityService(uow.CommunityRepo())
	userSvc := user.NewUserService(uow.UserRepo(), tokenCache, jwtCfg)
	voteSvc := vote.NewVoteService(uow.PostRepo(), voteCache)

	return &Services{
		Post:      postSvc,
		Community: communitySvc,
		User:      userSvc,
		Vote:      voteSvc,
	}
}
