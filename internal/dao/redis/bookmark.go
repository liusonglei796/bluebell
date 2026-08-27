package redis

import (
	"context"
	"strconv"

	goredis "github.com/redis/go-redis/v9"
)

const (
	keyUserBookmarksPrefix = "user:bookmarks:" // bluebell:user:bookmarks:{uid}
)

// BookmarkCache 收藏夹缓存
type BookmarkCache struct {
	rdb *goredis.Client
}

// NewBookmarkCache 创建收藏夹缓存实例
func NewBookmarkCache(rdb *goredis.Client) *BookmarkCache {
	return &BookmarkCache{rdb: rdb}
}

// AddBookmark 将帖子加入用户的已收藏 Set
func (c *BookmarkCache) AddBookmark(ctx context.Context, userID, postID int64) error {
	key := redisKey(keyUserBookmarksPrefix + strconv.FormatInt(userID, 10))
	return c.rdb.SAdd(ctx, key, strconv.FormatInt(postID, 10)).Err()
}

// RemoveBookmark 从用户的已收藏 Set 移除
func (c *BookmarkCache) RemoveBookmark(ctx context.Context, userID, postID int64) error {
	key := redisKey(keyUserBookmarksPrefix + strconv.FormatInt(userID, 10))
	return c.rdb.SRem(ctx, key, strconv.FormatInt(postID, 10)).Err()
}

// IsBookmarked O(1) 判定是否已收藏
func (c *BookmarkCache) IsBookmarked(ctx context.Context, userID, postID int64) (bool, error) {
	if userID == 0 || postID == 0 {
		return false, nil
	}
	key := redisKey(keyUserBookmarksPrefix + strconv.FormatInt(userID, 10))
	return c.rdb.SIsMember(ctx, key, strconv.FormatInt(postID, 10)).Result()
}

// BatchIsBookmarked 批量判定是否已收藏
func (c *BookmarkCache) BatchIsBookmarked(ctx context.Context, userID int64, postIDs []string) (map[string]bool, error) {
	resMap := make(map[string]bool, len(postIDs))
	if userID == 0 || len(postIDs) == 0 {
		for _, pid := range postIDs {
			resMap[pid] = false
		}
		return resMap, nil
	}

	key := redisKey(keyUserBookmarksPrefix + strconv.FormatInt(userID, 10))
	pipe := c.rdb.Pipeline()
	cmds := make([]*goredis.BoolCmd, len(postIDs))
	for i, pid := range postIDs {
		cmds[i] = pipe.SIsMember(ctx, key, pid)
	}

	if _, err := pipe.Exec(ctx); err != nil && err != goredis.Nil {
		return nil, err
	}

	for i, pid := range postIDs {
		resMap[pid] = cmds[i].Val()
	}

	return resMap, nil
}
