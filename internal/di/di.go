// Package di 提供依赖注入容器，负责组装各层实例
package di

import (
	"bluebell/internal/application"
	"bluebell/internal/application/community"
	"bluebell/internal/application/post"
	"bluebell/internal/application/user"
	"bluebell/internal/config"
	"bluebell/internal/infrastructure/es"
	"bluebell/internal/infrastructure/mq"
	mysqlrepo "bluebell/internal/infrastructure/persistence/mysql"
	redisrepo "bluebell/internal/infrastructure/persistence/redis"
)

// Services 聚合所有 Service 实例
type Services struct {
	Post      application.PostService
	Community application.CommunityService
	User      application.UserService
}

// NewServices 创建并注入所有 Service 实例
func NewServices(
	dbRepos *mysqlrepo.Repositories,
	cacheRepos *redisrepo.Repositories,
	publisher *mq.Publisher,
	esClient *es.Client,
	cfg *config.Config,
) *Services {
	return &Services{
		Post:      postsvc.NewPostService(dbRepos.Post, cacheRepos.PostCache, dbRepos.Vote, dbRepos.Remark, publisher, esClient),
		Community: communitysvc.NewCommunityService(dbRepos.Community, dbRepos.User),
		User:      usersvc.NewUserService(dbRepos.User, cacheRepos.TokenCache, cfg),
	}
}
