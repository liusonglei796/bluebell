package redis

import (
	"bluebell/internal/config"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// rdb 全局 Redis 客户端
// redis.Client 是线程安全的，整个应用共享一个连接池即可
var rdb *redis.Client

// Init 初始化 Redis 连接
func Init(cfg *config.RedisConfig) error {
	if cfg == nil {
		return fmt.Errorf("redis config is nil")
	}

	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("connect to redis failed: %w", err)
	}

	zap.L().Info("init redis success",
		zap.String("addr", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
		zap.Int("db", cfg.DB),
	)

	return nil
}

// Close 关闭 Redis 连接
func Close() {
	_ = rdb.Close()
}
