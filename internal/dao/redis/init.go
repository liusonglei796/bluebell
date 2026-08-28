// Package redis 提供 Redis 数据访问层（DAO）
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"bluebell/internal/config"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Init 初始化 Redis 连接
func Init(cfg *config.Config) (*goredis.Client, error) {
	if cfg == nil {
		return nil, errors.New("redis config is nil")
	}

	redisCfg := cfg.Redis
	rdb := goredis.NewClient(&goredis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port),
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
		PoolSize: redisCfg.PoolSize,
		// 原生客户端缓存（Client-Side Caching）：RESP3 + 库内分片近似-LRU 本地缓存。
		// 命中走进程内 L1（0 RTT），失效由 Redis CLIENT TRACKING 自动精准推送，
		// 取代原先手搓的 LocalPostCache + BcastTracker。
		Protocol: 3,
		ClientSideCache: goredis.NewLocalCache(goredis.CacheConfig{
			MaxEntries:    10000,
			MaxStaleness: 30 * time.Second,
		}),
	})

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("connect to redis failed: %w", err)
	}
	zap.L().Info("init redis success",
		zap.String("addr", fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port)),
		zap.Int("db", redisCfg.DB),
	)
	return rdb, nil
}

// Close 关闭 Redis 连接
func Close(rdb *goredis.Client) {
	if rdb != nil {
		_ = rdb.Close()
	}
}

var keyPrefix = "bluebell:"

func redisKey(key string) string {
	return keyPrefix + key
}
