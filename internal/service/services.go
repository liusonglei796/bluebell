// Package service 提供 Service 层聚合与构造
package service

import (
	// 配置
	"bluebell/internal/config"

	// 基础设施层 - Repository 聚合
	"bluebell/internal/dao/cache"
	"bluebell/internal/dao/database"

	// 领域层 - Service 接口
	"bluebell/internal/domain/svcdomain"

	// Service 层
	"bluebell/internal/service/communitysvc"
	"bluebell/internal/service/postsvc"
	"bluebell/internal/service/usersvc"
)

// Services 聚合所有 Service 实例
type Services struct {
	Post      svcdomain.PostService
	Community svcdomain.CommunityService
	User      svcdomain.UserService
}

// NewServices 创建并注入所有 Service 实例
func NewServices(
	dbRepos *database.Repositories,
	cacheRepos *cache.Repositories,
	cfg *config.Config,
) *Services {
	return &Services{
		Post:      postsvc.NewPostService(dbRepos.Post, cacheRepos.PostCache),
		Community: communitysvc.NewCommunityService(dbRepos.Community, dbRepos.User),
		User:      usersvc.NewUserService(dbRepos.User, cacheRepos.TokenCache, cfg),
	}
}
