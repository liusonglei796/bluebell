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
func Init(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("redis config is nil")
	}

	redisCfg := cfg.Redis
	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port),
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
		PoolSize: redisCfg.PoolSize,
	})

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("connect to redis failed: %w", err)
	}

	zap.L().Info("init redis success",
		zap.String("addr", fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port)),
		zap.Int("db", redisCfg.DB),
	)

	return nil
}

// Close 关闭 Redis 连接
func Close() {
	_ = rdb.Close()
}
