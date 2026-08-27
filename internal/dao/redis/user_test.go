package redis

import (
	"context"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserTokenCache_Blacklist(t *testing.T) {
	rdb := goredis.NewClient(&goredis.Options{
		Addr: "127.0.0.1:6379",
		DB:   15,
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("跳过 Redis 测试：无法连接本地 Redis")
	}
	defer rdb.Close()
	_ = rdb.FlushDB(ctx).Err()

	cache := NewUserTokenCache(rdb)
	jti := "test-jti-uuid-999"

	// 1. 初始状态：不在黑名单
	blacklisted, err := cache.IsBlacklisted(ctx, jti)
	require.NoError(t, err)
	assert.False(t, blacklisted)

	// 2. 加入黑名单（设置 1 秒过期）
	err = cache.AddBlacklist(ctx, jti, 1*time.Second)
	require.NoError(t, err)

	// 3. 再次查询：应在黑名单中
	blacklisted, err = cache.IsBlacklisted(ctx, jti)
	require.NoError(t, err)
	assert.True(t, blacklisted)

	// 4. 等待 1.2 秒后自动过期逐出
	time.Sleep(1200 * time.Millisecond)
	blacklisted, err = cache.IsBlacklisted(ctx, jti)
	require.NoError(t, err)
	assert.False(t, blacklisted, "到期后应自动从 Redis 逐出")
}
