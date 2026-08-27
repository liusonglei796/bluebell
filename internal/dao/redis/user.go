package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Redis Keys 相关常量
const (
	keyJWTBlacklistPrefix = "jwt:blacklist:" // bluebell:jwt:blacklist:{jti}
)

// UserTokenCache 用户 Token 黑名单数据访问对象
type UserTokenCache struct {
	rdb *goredis.Client
}

// NewUserTokenCache 创建用户 Token 缓存 DAO 实例
func NewUserTokenCache(rdb *goredis.Client) *UserTokenCache {
	return &UserTokenCache{rdb: rdb}
}

// AddBlacklist 将指定的 JTI 加入黑名单，过期时间为 Token 剩余有效时长（到期自动从 Redis 逐出）
func (c *UserTokenCache) AddBlacklist(ctx context.Context, jti string, remainingTTL time.Duration) error {
	if jti == "" || remainingTTL <= 0 {
		return nil
	}
	err := c.rdb.Set(ctx, redisKey(keyJWTBlacklistPrefix+jti), "1", remainingTTL).Err()
	if err != nil {
		return fmt.Errorf("add jwt blacklist failed (jti: %s): %w", jti, err)
	}
	return nil
}

// IsBlacklisted 检查 JTI 是否存在于黑名单中
func (c *UserTokenCache) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	if jti == "" {
		return false, nil
	}
	val, err := c.rdb.Exists(ctx, redisKey(keyJWTBlacklistPrefix+jti)).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("check jwt blacklist failed (jti: %s): %w", jti, err)
	}
	return val > 0, nil
}
