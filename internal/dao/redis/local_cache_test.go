package redis

import (
	"strconv"
	"testing"
	"time"

	postResp "bluebell/internal/dto/response/post"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestLocalPostCache_GetSetDelete(t *testing.T) {
	c := NewLocalPostCache()

	p1 := &postResp.DetailResponse{
		ID:         "1001",
		Title:      "Local Cache Test",
		AuthorName: "Alice",
	}

	// 1. 写入 L1 缓存，TTL 500ms
	c.Set("1001", p1, 500*time.Millisecond)

	// 2. 立即读取 -> 命中
	item, ok := c.Get("1001")
	assert.True(t, ok)
	assert.Equal(t, "Local Cache Test", item.Title)
	assert.Equal(t, "Alice", item.AuthorName)

	// 3. 删除 -> 未命中
	c.Delete("1001")
	_, ok = c.Get("1001")
	assert.False(t, ok)

	// 4. 写入短 TTL，等待过期 -> 自动清理未命中
	c.Set("1002", p1, 50*time.Millisecond)
	time.Sleep(70 * time.Millisecond)
	_, ok = c.Get("1002")
	assert.False(t, ok)
}

func TestLocalPostCache_CapacityEviction(t *testing.T) {
	c := NewLocalPostCache()

	// 写入 5010 条短期过期的数据
	for i := 1; i <= 5010; i++ {
		id := strconv.Itoa(i)
		c.Set(id, &postResp.DetailResponse{ID: id}, 10*time.Millisecond)
	}

	time.Sleep(20 * time.Millisecond)

	// 再次 Set 触发自动清理过期 key
	c.Set("9999", &postResp.DetailResponse{ID: "9999"}, 1*time.Minute)

	item, ok := c.Get("9999")
	assert.True(t, ok)
	assert.Equal(t, "9999", item.ID)
}

func TestBcastTracker_HandleInvalidate(t *testing.T) {
	localCache := NewLocalPostCache()
	tracker := NewBcastTracker(nil, localCache, "bluebell:post:detail:")

	p1 := &postResp.DetailResponse{ID: "5001", Title: "Post 5001"}
	p2 := &postResp.DetailResponse{ID: "5002", Title: "Post 5002"}
	p3 := &postResp.DetailResponse{ID: "5003", Title: "Post 5003"}

	localCache.Set("5001", p1, 10*time.Minute)
	localCache.Set("5002", p2, 10*time.Minute)
	localCache.Set("5003", p3, 10*time.Minute)

	// 1. 模拟收到包含 5001 的失效广播 -> 只有 5001 被逐出，5002 和 5003 保留
	msg1 := &goredis.Message{
		Channel:      "__redis__:invalidate",
		PayloadSlice: []string{"bluebell:post:detail:5001"},
	}
	tracker.handleInvalidateMessage(msg1)

	_, ok1 := localCache.Get("5001")
	assert.False(t, ok1, "5001 收到广播后必须被从 L1 内存逐出")

	_, ok2 := localCache.Get("5002")
	assert.True(t, ok2, "5002 未失效，必须保留")

	// 2. 模拟收到全量失效通知 (nil keys) -> 清空所有 L1 缓存
	msgNil := &goredis.Message{
		Channel: "__redis__:invalidate",
	}
	tracker.handleInvalidateMessage(msgNil)

	_, ok2After := localCache.Get("5002")
	assert.False(t, ok2After, "收到全量广播后 5002 必须被清空")
	_, ok3After := localCache.Get("5003")
	assert.False(t, ok3After, "收到全量广播后 5003 必须被清空")
}
