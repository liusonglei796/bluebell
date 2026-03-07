package usercache

import (
	"context"
	"fmt"
	"time"

	"bluebell/internal/domain/cachedomain"
	"bluebell/pkg/errorx"
	"github.com/redis/go-redis/v9"
)

// Redis Keys 相关常量
const (
	keyPrefix           = "bluebell:"
	keyUserAccessToken  = "active_access_token:"  // bluebell:active_access_token:1001
	keyUserRefreshToken = "active_refresh_token:" // bluebell:active_refresh_token:1001
)

func getRedisKey(key string) string {
	return keyPrefix + key
}

// userTokenCacheStruct 用户Token缓存仓储实现
type userTokenCacheStruct struct {
	rdb *redis.Client
}

// NewUserTokenCache 创建 userTokenCacheStruct 实例
func NewUserTokenCache(rdb *redis.Client) cachedomain.UserTokenCacheRepository {
	return &userTokenCacheStruct{rdb: rdb}
}

// SetUserToken 存储用户的 Access Token 和 Refresh Token
func (c *userTokenCacheStruct) SetUserToken(ctx context.Context, userID int64, aToken, rToken string, aExp, rExp time.Duration) error {
	pipe := c.rdb.Pipeline()

	pipe.Set(ctx, getRedisKey(keyUserAccessToken+fmt.Sprint(userID)), aToken, aExp)
	pipe.Set(ctx, getRedisKey(keyUserRefreshToken+fmt.Sprint(userID)), rToken, rExp)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeCacheError, "set user token pipeline exec failed (user_id: %d)", userID)
	}
	return nil
}

// GetUserAccessToken 获取用户的 Access Token
func (c *userTokenCacheStruct) GetUserAccessToken(ctx context.Context, userID int64) (string, error) {
	return c.rdb.Get(ctx, getRedisKey(keyUserAccessToken+fmt.Sprint(userID))).Result()
}

// GetUserRefreshToken 获取用户的 Refresh Token
func (c *userTokenCacheStruct) GetUserRefreshToken(ctx context.Context, userID int64) (string, error) {
	return c.rdb.Get(ctx, getRedisKey(keyUserRefreshToken+fmt.Sprint(userID))).Result()
}

// DeleteUserToken 删除用户的 Token (用于登出)
func (c *userTokenCacheStruct) DeleteUserToken(ctx context.Context, userID int64) error {
	pipe := c.rdb.Pipeline()
	pipe.Del(ctx, getRedisKey(keyUserAccessToken+fmt.Sprint(userID)))
	pipe.Del(ctx, getRedisKey(keyUserRefreshToken+fmt.Sprint(userID)))
	_, err := pipe.Exec(ctx)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeCacheError, "delete user token pipeline exec failed (user_id: %d)", userID)
	}
	return nil
}
