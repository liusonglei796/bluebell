package cache

import (
	"bluebell/internal/config"
	"bluebell/pkg/errorx"
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Init 初始化 Redis 连接
func Init(cfg *config.Config) (*redis.Client, error) {
	if cfg == nil {
		return nil, errorx.New(errorx.CodeConfigError, "redis config is nil")
	}

	redisCfg := cfg.Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port),
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
		PoolSize: redisCfg.PoolSize,
	})
	
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return nil, errorx.Wrap(err, errorx.CodeInfraError, "connect to redis failed")
	}
	zap.L().Info("init redis success",
		zap.String("addr", fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port)),
		zap.Int("db", redisCfg.DB),
	)
	return rdb, nil
}

// Close 关闭 Redis 连接
func Close(rdb *redis.Client) {
	if rdb != nil {
		_ = rdb.Close()
	}
}
