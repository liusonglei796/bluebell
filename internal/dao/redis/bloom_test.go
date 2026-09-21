package redis

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculateJitterTTL(t *testing.T) {
	baseTTL := 24 * time.Hour
	maxJitter := 30 * time.Minute

	// 测试 100 次随机采样
	for i := 0; i < 100; i++ {
		ttl := CalculateJitterTTL(baseTTL, maxJitter)
		assert.True(t, ttl >= baseTTL, "TTL 必须大于等于基准 TTL")
		assert.True(t, ttl <= baseTTL+maxJitter, "TTL 必须小于等于基准 TTL + 最大抖动窗口")
	}

	// 边界测试：参数非法时返回默认安全值
	assert.Equal(t, defaultPostDetailTTL, CalculateJitterTTL(0, 0))
	assert.Equal(t, 10*time.Minute, CalculateJitterTTL(10*time.Minute, 0))
}

func TestPostBloomFilter_NilClientHandling(t *testing.T) {
	bf := NewPostBloomFilter(nil, "test:bloom")
	ctx := context.Background()

	// 验证在 Client 为 nil 时的容错与安全退出 (Fail-open)
	assert.NoError(t, bf.EnsureReserved(ctx, 0.01, 1000000))
	assert.NoError(t, bf.Add(ctx, "123456"))
	exists, err := bf.Exists(ctx, "123456")
	assert.NoError(t, err)
	assert.True(t, exists, "Client 为 nil 时应容错放行 (Fail-open)")
}

func TestPostBloomFilter_RedisIntegration(t *testing.T) {
	rdb, cache, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	postID := "8888001"
	nonExistentID := "9999999"

	// 探测当前 Redis 实例是否加载了 RedisBloom 扩展模块
	err := rdb.BFAdd(ctx, "test:probe", "ping").Err()
	if err != nil && (strings.Contains(err.Error(), "unknown command") || strings.Contains(err.Error(), "ERR unknown")) {
		t.Skip("跳过 Redis 集成测试：当前连接的 Redis 为纯净版，未挂载 RedisBloom 模块（生产请使用 redis/redis-stack-server）")
	}

	// 1. 初始化布隆过滤器
	err = cache.bloom.EnsureReserved(ctx, 0.01, 10000)
	require.NoError(t, err)

	// 2. 添加 postID 到布隆过滤器
	err = cache.AddPostBloom(ctx, postID)
	require.NoError(t, err)

	// 3. 检查已添加的 postID -> 必须判定存在 (true)
	exists, err := cache.CheckPostInBloom(ctx, postID)
	require.NoError(t, err)
	assert.True(t, exists, "已写入布隆过滤器的 postID 必须存在")

	// 4. 检查未添加的 nonExistentID -> 必须判定不存在 (false, 拦截穿透)
	notExists, err := cache.CheckPostInBloom(ctx, nonExistentID)
	require.NoError(t, err)
	assert.False(t, notExists, "未写入布隆过滤器的 postID 必须被判定为不存在")
}
