// Package caches 提供 Redis 缓存层的聚合与导出
package cache

import (
	"bluebell/internal/domain"
	communitycache "bluebell/internal/infrastructure/persistence/redis/community"
	postcache "bluebell/internal/infrastructure/persistence/redis/post"
	socialcache "bluebell/internal/infrastructure/persistence/redis/social"
	usercache "bluebell/internal/infrastructure/persistence/redis/user"

	"github.com/redis/go-redis/v9"
)

// Repositories 聚合所有缓存仓储实例
type Repositories struct {
	PostCache         domain.PostCacheRepository
	TokenCache        domain.UserTokenCacheRepository
	RemarkCache       domain.RemarkCacheRepository
	CommunityCache    domain.CommunityCacheRepository
	UserInfoCache     domain.UserInfoCacheRepository
	SocialCache       domain.SocialCacheRepository
	HotScoreRefresher *postcache.HotScoreRefresher
}

// NewRepositories 创建缓存仓储聚合实例
func NewRepositories(rdb *redis.Client) *Repositories {
	postCache, refresher := postcache.NewCacheWithRefresher(rdb)
	return &Repositories{
		PostCache:         postCache,
		TokenCache:        usercache.NewUserTokenCache(rdb),
		RemarkCache:       postcache.NewRemarkCache(rdb),
		CommunityCache:    communitycache.NewCommunityCache(rdb),
		UserInfoCache:     usercache.NewUserInfoCache(rdb),
		SocialCache:       socialcache.NewSocialCache(rdb),
		HotScoreRefresher: refresher,
	}
}
