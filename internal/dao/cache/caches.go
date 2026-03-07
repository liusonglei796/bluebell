// Package caches 提供 Redis 缓存层的聚合与导出
package cache

import (
	postcache "bluebell/internal/dao/cache/post"
	usercache "bluebell/internal/dao/cache/user"
	"bluebell/internal/domain/cachedomain"

	"github.com/redis/go-redis/v9"
)

// Repositories 聚合所有缓存仓储实例
type Repositories struct {
	PostCache  cachedomain.PostRepository
	TokenCache cachedomain.UserTokenCacheRepository
}

// NewRepositories 创建缓存仓储聚合实例
func NewRepositories(rdb *redis.Client) *Repositories {
	return &Repositories{
		PostCache:  postcache.NewCache(rdb),
		TokenCache: usercache.NewUserTokenCache(rdb),
	}
}
