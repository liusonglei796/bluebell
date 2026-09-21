package redis

import (
	"context"
	"strings"

	goredis "github.com/redis/go-redis/v9"
)

const (
	// Redis Key 常量
	keyPostBloom = "post:bloom"
	// 默认预估容量 100 万帖子
	defaultBloomCapacity int64 = 1000000
	// 默认假阳性误判率 1%
	defaultBloomErrorRate float64 = 0.01
)

// PostBloomFilter 基于 go-redis 原生 API 与 RedisBloom 模块的分布式布隆过滤器
type PostBloomFilter struct {
	rdb *goredis.Client
	key string
}

// NewPostBloomFilter 创建布隆过滤器实例
func NewPostBloomFilter(rdb *goredis.Client, key string) *PostBloomFilter {
	if key == "" {
		key = keyPostBloom
	}
	return &PostBloomFilter{
		rdb: rdb,
		key: redisKey(key),
	}
}

// EnsureReserved 初始化并预分配布隆过滤器的容量与误报率 (BF.RESERVE)
func (bf *PostBloomFilter) EnsureReserved(ctx context.Context, errorRate float64, capacity int64) error {
	if bf == nil || bf.rdb == nil {
		return nil
	}
	if errorRate <= 0 {
		errorRate = defaultBloomErrorRate
	}
	if capacity <= 0 {
		capacity = defaultBloomCapacity
	}

	err := bf.rdb.BFReserve(ctx, bf.key, errorRate, capacity).Err()
	// 如果过滤器已存在，Redis 返回 "ERR item exists"，直接忽略
	if err != nil && (strings.Contains(err.Error(), "ERR item exists") || strings.Contains(err.Error(), "item exists")) {
		return nil
	}
	return err
}

// Add 将元素添加到布隆过滤器中 (调用原生 BF.ADD 指令)
func (bf *PostBloomFilter) Add(ctx context.Context, item string) error {
	if bf == nil || bf.rdb == nil || item == "" {
		return nil
	}
	return bf.rdb.BFAdd(ctx, bf.key, item).Err()
}

// Exists 判定元素是否存在于布隆过滤器中 (调用原生 BF.EXISTS 指令)
// 返回 false 表示绝对不存在 (100% 可信，拦截穿透)
// 返回 true 表示可能存在 (允许小概率假阳性)
func (bf *PostBloomFilter) Exists(ctx context.Context, item string) (bool, error) {
	if bf == nil || bf.rdb == nil || item == "" {
		return true, nil // 容错放行 (Fail-open)
	}

	// 冷启动/键尚未创建时容错放行，回源自愈预热
	if bf.rdb.Exists(ctx, bf.key).Val() == 0 {
		return true, nil
	}

	exists, err := bf.rdb.BFExists(ctx, bf.key, item).Result()
	if err != nil {
		// 若 Redis 异常或未加载模块，容错放行，避免阻断核心读链路
		return true, err
	}
	return exists, nil
}
