package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	keyUserFeedPrefix = "user:feed:" // bluebell:user:feed:{uid}
)

// FeedCache Timeline Feed 流缓存
type FeedCache struct {
	rdb *goredis.Client
}

// NewFeedCache 创建 Feed 缓存实例
func NewFeedCache(rdb *goredis.Client) *FeedCache {
	return &FeedCache{rdb: rdb}
}

// SetUserFeed 写入用户聚合出的 Feed 流 ZSet (带 10min TTL)
func (c *FeedCache) SetUserFeed(ctx context.Context, userID int64, postIDs []int64, timestamps []int64, ttl time.Duration) error {
	if len(postIDs) == 0 {
		return nil
	}

	key := redisKey(keyUserFeedPrefix + strconv.FormatInt(userID, 10))
	zList := make([]goredis.Z, len(postIDs))
	for i := range postIDs {
		zList[i] = goredis.Z{
			Score:  float64(timestamps[i]),
			Member: strconv.FormatInt(postIDs[i], 10),
		}
	}

	pipe := c.rdb.Pipeline()
	pipe.ZAdd(ctx, key, zList...)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// GetUserFeedPage 基于时间戳游标（Cursor）分页拉取 Feed 动态
func (c *FeedCache) GetUserFeedPage(ctx context.Context, userID int64, cursor int64, size int64) ([]string, error) {
	key := redisKey(keyUserFeedPrefix + strconv.FormatInt(userID, 10))

	maxScore := "+inf"
	if cursor > 0 {
		maxScore = fmt.Sprintf("(%d", cursor) // 不包含游标本身
	}

	return c.rdb.ZRevRangeByScore(ctx, key, &goredis.ZRangeBy{
		Max:    maxScore,
		Min:    "-inf",
		Offset: 0,
		Count:  size,
	}).Result()
}
