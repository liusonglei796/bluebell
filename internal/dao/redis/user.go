package redis

import (
	"context"
	"fmt"
	"time"
)

// userTokenCacheStruct 用户Token缓存仓储实现
type userTokenCacheStruct struct{}

// NewUserTokenCache 创建 userTokenCacheStruct 实例
func NewUserTokenCache() *userTokenCacheStruct {
	return &userTokenCacheStruct{}
}

// SetUserToken 存储用户的 Access Token 和 Refresh Token
func (c *userTokenCacheStruct) SetUserToken(ctx context.Context, userID int64, aToken, rToken string, aExp, rExp time.Duration) error {
	pipe := rdb.Pipeline()

	pipe.Set(ctx, getRedisKey(KeyUserAccessToken+fmt.Sprint(userID)), aToken, aExp)
	pipe.Set(ctx, getRedisKey(KeyUserRefreshToken+fmt.Sprint(userID)), rToken, rExp)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("set user token pipeline exec failed (user_id: %d): %w", userID, err)
	}
	return nil
}

// GetUserAccessToken 获取用户的 Access Token
func (c *userTokenCacheStruct) GetUserAccessToken(ctx context.Context, userID int64) (string, error) {
	return rdb.Get(ctx, getRedisKey(KeyUserAccessToken+fmt.Sprint(userID))).Result()
}

// GetUserRefreshToken 获取用户的 Refresh Token
func (c *userTokenCacheStruct) GetUserRefreshToken(ctx context.Context, userID int64) (string, error) {
	return rdb.Get(ctx, getRedisKey(KeyUserRefreshToken+fmt.Sprint(userID))).Result()
}

// DeleteUserToken 删除用户的 Token (用于登出)
func (c *userTokenCacheStruct) DeleteUserToken(ctx context.Context, userID int64) error {
	pipe := rdb.Pipeline()
	pipe.Del(ctx, getRedisKey(KeyUserAccessToken+fmt.Sprint(userID)))
	pipe.Del(ctx, getRedisKey(KeyUserRefreshToken+fmt.Sprint(userID)))
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete user token pipeline exec failed (user_id: %d): %w", userID, err)
	}
	return nil
}
