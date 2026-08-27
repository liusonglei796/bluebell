package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	keyUserUnreadPrefix    = "user:unread:"         // bluebell:user:unread:{uid}
	keyNotifThrottlePrefix = "notif:throttle:"      // bluebell:notif:throttle:{actor}:{type}:{entity}
)

// NotificationCache 通知与红点缓存
type NotificationCache struct {
	rdb *goredis.Client
}

// NewNotificationCache 创建通知缓存实例
func NewNotificationCache(rdb *goredis.Client) *NotificationCache {
	return &NotificationCache{rdb: rdb}
}

// IncrUnread 增加分类未读数和总未读数
func (c *NotificationCache) IncrUnread(ctx context.Context, recipientID int64, category string) error {
	key := redisKey(keyUserUnreadPrefix + strconv.FormatInt(recipientID, 10))
	pipe := c.rdb.Pipeline()
	pipe.HIncrBy(ctx, key, "total", 1)
	if category != "" {
		pipe.HIncrBy(ctx, key, category, 1)
	}
	pipe.Expire(ctx, key, 30*24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

// GetUnreadCounts 获取用户各分类未读数
func (c *NotificationCache) GetUnreadCounts(ctx context.Context, recipientID int64) (map[string]int64, error) {
	key := redisKey(keyUserUnreadPrefix + strconv.FormatInt(recipientID, 10))
	res, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64, len(res))
	for k, v := range res {
		n, _ := strconv.ParseInt(v, 10, 64)
		counts[k] = n
	}
	return counts, nil
}

// ResetCategoryUnread 清零某分类未读数并核减 total
func (c *NotificationCache) ResetCategoryUnread(ctx context.Context, recipientID int64, category string) error {
	key := redisKey(keyUserUnreadPrefix + strconv.FormatInt(recipientID, 10))
	curr, err := c.rdb.HGet(ctx, key, category).Int64()
	if err != nil || curr <= 0 {
		return nil
	}

	pipe := c.rdb.Pipeline()
	pipe.HSet(ctx, key, category, 0)
	pipe.HIncrBy(ctx, key, "total", -curr)
	_, err = pipe.Exec(ctx)
	return err
}

// ResetAllUnread 全部标记已读，清零所有未读数
func (c *NotificationCache) ResetAllUnread(ctx context.Context, recipientID int64) error {
	key := redisKey(keyUserUnreadPrefix + strconv.FormatInt(recipientID, 10))
	return c.rdb.Del(ctx, key).Err()
}

// CheckThrottle 防刷防抖拦截: 1小时内重复互动只产生1次通知
func (c *NotificationCache) CheckThrottle(ctx context.Context, actorID int64, entityType int8, entityID int64, actionType int8, ttl time.Duration) (bool, error) {
	key := redisKey(fmt.Sprintf("%s%d:%d:%d:%d", keyNotifThrottlePrefix, actorID, entityType, entityID, actionType))
	return c.rdb.SetNX(ctx, key, "1", ttl).Result()
}
