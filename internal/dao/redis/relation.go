package redis

import (
	"context"
	"fmt"
	"strconv"

	goredis "github.com/redis/go-redis/v9"
)

const (
	keyUserFollowingPrefix = "user:following:" // bluebell:user:following:{uid}
)

// UserRelationCache 用户关系 Redis 缓存
type UserRelationCache struct {
	rdb *goredis.Client
}

// NewUserRelationCache 创建关系缓存实例
func NewUserRelationCache(rdb *goredis.Client) *UserRelationCache {
	return &UserRelationCache{rdb: rdb}
}

// AddFollowing 将目标用户添加到我的关注 Set
func (c *UserRelationCache) AddFollowing(ctx context.Context, followerID, followingID int64) error {
	key := redisKey(keyUserFollowingPrefix + strconv.FormatInt(followerID, 10))
	return c.rdb.SAdd(ctx, key, strconv.FormatInt(followingID, 10)).Err()
}

// RemoveFollowing 从关注 Set 移除
func (c *UserRelationCache) RemoveFollowing(ctx context.Context, followerID, followingID int64) error {
	key := redisKey(keyUserFollowingPrefix + strconv.FormatInt(followerID, 10))
	return c.rdb.SRem(ctx, key, strconv.FormatInt(followingID, 10)).Err()
}

// IsFollowing O(1) 判定是否关注
func (c *UserRelationCache) IsFollowing(ctx context.Context, followerID, followingID int64) (bool, error) {
	key := redisKey(keyUserFollowingPrefix + strconv.FormatInt(followerID, 10))
	return c.rdb.SIsMember(ctx, key, strconv.FormatInt(followingID, 10)).Result()
}

// GetMutualStatus 判断双方互关状态 (isFollowed, isMutual)
func (c *UserRelationCache) GetMutualStatus(ctx context.Context, viewerID, targetID int64) (isFollowed bool, isMutual bool, err error) {
	if viewerID == 0 || targetID == 0 || viewerID == targetID {
		return false, false, nil
	}

	keyA := redisKey(keyUserFollowingPrefix + strconv.FormatInt(viewerID, 10))
	keyB := redisKey(keyUserFollowingPrefix + strconv.FormatInt(targetID, 10))

	pipe := c.rdb.Pipeline()
	cmdA := pipe.SIsMember(ctx, keyA, strconv.FormatInt(targetID, 10))
	cmdB := pipe.SIsMember(ctx, keyB, strconv.FormatInt(viewerID, 10))
	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, false, fmt.Errorf("check mutual follow status failed: %w", err)
	}

	isFollowed = cmdA.Val()
	isMutual = cmdA.Val() && cmdB.Val()
	return isFollowed, isMutual, nil
}
