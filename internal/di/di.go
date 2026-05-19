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
	"bluebell/internal/infrastructure/snowflake"
	mysqlrepo "bluebell/internal/infrastructure/persistence/mysql"
	redisrepo "bluebell/internal/infrastructure/persistence/redis"
	"context"
	"strconv"
	"strings"
	"time"
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
	// 初始化投票缓冲区 (100ms 聚合窗口)
	flushFunc := func(ctx context.Context, votes map[string]int8) error {
		// 1. 批量更新 Redis (Pipeline)
		if err := cacheRepos.PostCache.BatchVoteForPost(ctx, votes); err != nil {
			return err
		}
		// 2. 批量发送 MQ 消息用于持久化
		for key, direction := range votes {
			parts := strings.Split(key, ":")
			if len(parts) != 2 {
				continue
			}
			msg := &mq.VoteMessage{
				MsgID:  strconv.FormatInt(snowflake.GenID(), 10),
				PostID: parts[0],
				UserID: parts[1],
				Action: int(direction),
			}
			_ = publisher.PublishVote(ctx, msg)
		}
		return nil
	}
	voteBuffer := mq.NewVoteBuffer(100*time.Millisecond, flushFunc)
	// 启动缓冲区刷新协程 (生产环境应接入生命周期管理)
	go voteBuffer.Start(context.Background())

	return &Services{
		Post:      postsvc.NewPostService(dbRepos.Post, cacheRepos.PostCache, dbRepos.Vote, dbRepos.Remark, publisher, esClient, voteBuffer),
		Community: communitysvc.NewCommunityService(dbRepos.Community, dbRepos.User),
		User:      usersvc.NewUserService(dbRepos.User, cacheRepos.TokenCache, cfg),
	}
}
