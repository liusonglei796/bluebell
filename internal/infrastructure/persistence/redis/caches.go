// Package caches 提供 Redis 缓存层的聚合与导出
package cache

import (
	postcache "bluebell/internal/infrastructure/persistence/redis/post"
	usercache "bluebell/internal/infrastructure/persistence/redis/user"
	"bluebell/internal/domain"

	"github.com/redis/go-redis/v9"
)

// Repositories 聚合所有缓存仓储实例
type Repositories struct {
	PostCache         domain.PostCacheRepository
	TokenCache        domain.UserTokenCacheRepository
	HotScoreRefresher *postcache.HotScoreRefresher
}

// NewRepositories 创建缓存仓储聚合实例
func NewRepositories(rdb *redis.Client) *Repositories {
	postCache, refresher := postcache.NewCacheWithRefresher(rdb)
	return &Repositories{
		PostCache:         postCache,
		TokenCache:        usercache.NewUserTokenCache(rdb),
		HotScoreRefresher: refresher,
	}
}
