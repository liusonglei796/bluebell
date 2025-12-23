package redis

import (
	"fmt"
	"time"
)

// SetUserToken 存储用户的 Access Token 和 Refresh Token
func SetUserToken(userID int64, aToken, rToken string, aExp, rExp time.Duration) error {
	// 使用 Pipeline 保证原子性（尽可能）和减少网络 RTT
	pipe := rdb.Pipeline()

	// 存储 Access Token
	pipe.Set(ctx, getRedisKey(KeyUserAccessToken+fmt.Sprint(userID)), aToken, aExp)

	// 存储 Refresh Token
	pipe.Set(ctx, getRedisKey(KeyUserRefreshToken+fmt.Sprint(userID)), rToken, rExp)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("set user token pipeline exec failed (user_id: %d): %w", userID, err)
	}
	return nil
}

// GetUserAccessToken 获取用户的 Access Token
func GetUserAccessToken(userID int64) (string, error) {
	return rdb.Get(ctx, getRedisKey(KeyUserAccessToken+fmt.Sprint(userID))).Result()
}

// GetUserRefreshToken 获取用户的 Refresh Token
func GetUserRefreshToken(userID int64) (string, error) {
	return rdb.Get(ctx, getRedisKey(KeyUserRefreshToken+fmt.Sprint(userID))).Result()
}

// DeleteUserToken 删除用户的 Token (用于登出)
func DeleteUserToken(userID int64) error {
	pipe := rdb.Pipeline()
	pipe.Del(ctx, getRedisKey(KeyUserAccessToken+fmt.Sprint(userID)))
	pipe.Del(ctx, getRedisKey(KeyUserRefreshToken+fmt.Sprint(userID)))
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete user token pipeline exec failed (user_id: %d): %w", userID, err)
	}
	return nil
}
