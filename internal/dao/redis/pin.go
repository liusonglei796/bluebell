package redis

import (
	"context"
	"strconv"

	goredis "github.com/redis/go-redis/v9"
)

const (
	keyCommunityPinnedPrefix = "community:pinned:" // bluebell:community:pinned:{community_id}
)

// PinCache 社区置顶贴缓存
type PinCache struct {
	rdb *goredis.Client
}

// NewPinCache 创建置顶缓存实例
func NewPinCache(rdb *goredis.Client) *PinCache {
	return &PinCache{rdb: rdb}
}

// SetPinned 设置或取消帖子置顶
func (c *PinCache) SetPinned(ctx context.Context, communityID, postID int64, isPinned bool) error {
	key := redisKey(keyCommunityPinnedPrefix + strconv.FormatInt(communityID, 10))
	pidStr := strconv.FormatInt(postID, 10)
	if isPinned {
		return c.rdb.SAdd(ctx, key, pidStr).Err()
	}
	return c.rdb.SRem(ctx, key, pidStr).Err()
}

// GetCommunityPinned 获取社区的所有置顶帖子ID
func (c *PinCache) GetCommunityPinned(ctx context.Context, communityID int64) ([]string, error) {
	key := redisKey(keyCommunityPinnedPrefix + strconv.FormatInt(communityID, 10))
	return c.rdb.SMembers(ctx, key).Result()
}
