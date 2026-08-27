package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	keyEventDedupPrefix = "event:dedup:" // bluebell:event:dedup:{group}:{event_id}
)

// DedupCache 事件去重锁缓存
type DedupCache struct {
	rdb *goredis.Client
}

// NewDedupCache 创建去重缓存实例
func NewDedupCache(rdb *goredis.Client) *DedupCache {
	return &DedupCache{rdb: rdb}
}

// AcquireEventLock 尝试获取事件消费锁 (PROCESSING 状态, 60s 超时)
func (c *DedupCache) AcquireEventLock(ctx context.Context, consumerGroup, eventID string, ttl time.Duration) (bool, error) {
	key := redisKey(fmt.Sprintf("%s%s:%s", keyEventDedupPrefix, consumerGroup, eventID))
	return c.rdb.SetNX(ctx, key, "PROCESSING", ttl).Result()
}

// MarkEventDone 标记事件已完成处理 (DONE 状态, 24h 留存)
func (c *DedupCache) MarkEventDone(ctx context.Context, consumerGroup, eventID string, ttl time.Duration) error {
	key := redisKey(fmt.Sprintf("%s%s:%s", keyEventDedupPrefix, consumerGroup, eventID))
	return c.rdb.Set(ctx, key, "DONE", ttl).Err()
}

// ReleaseEventLock 释放事件锁 (用于处理失败重试)
func (c *DedupCache) ReleaseEventLock(ctx context.Context, consumerGroup, eventID string) error {
	key := redisKey(fmt.Sprintf("%s%s:%s", keyEventDedupPrefix, consumerGroup, eventID))
	return c.rdb.Del(ctx, key).Err()
}

// IsEventDone 检查事件是否已经处理完毕
func (c *DedupCache) IsEventDone(ctx context.Context, consumerGroup, eventID string) (bool, error) {
	key := redisKey(fmt.Sprintf("%s%s:%s", keyEventDedupPrefix, consumerGroup, eventID))
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return false, nil
	}
	return val == "DONE", nil
}
