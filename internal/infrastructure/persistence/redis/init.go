package cache

import (
	"bluebell/internal/config"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Init 初始化 Redis 连接
func Init(cfg *config.Config) (*redis.Client, error) {
	if cfg == nil {
		return nil, errors.New("redis config is nil")
	}

	redisCfg := cfg.Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisCfg.Host, redisCfg.Port),
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
		PoolSize: redisCfg.PoolSize,
	})

	// 启用 OpenTelemetry 追踪插件
	if err := redisotel.InstrumentTracing(rdb); err != nil {
		zap.L().Error("redisotel.InstrumentTracing failed", zap.Error(err))
	}
	// 启用 OpenTelemetry 指标插件
	if err := redisotel.InstrumentMetrics(rdb); err != nil {
		zap.L().Error("redisotel.InstrumentMetrics failed", zap.Error(err))
	}

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
func Close(rdb *redis.Client) {
	if rdb != nil {
		_ = rdb.Close()
	}
}
