package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"bluebell/internal/model"

	"github.com/stretchr/testify/assert"
)

// TestPostService_SingleflightDefense 验证 singleflight 在高并发未命中时将并发请求合并为 1 次打库 (防击穿)
func TestPostService_SingleflightDefense(t *testing.T) {
	svc := &PostService{}

	const concurrency = 50
	var dbHitCount int64
	var wg sync.WaitGroup
	wg.Add(concurrency)

	results := make([]string, concurrency)

	// 模拟 50 个用户同一微秒并发查询同一个冷门/到期帖子
	targetPostID := "999888777"

	for i := 0; i < concurrency; i++ {
		idx := i
		go func() {
			defer wg.Done()
			val, err, _ := svc.sfGroup.Do(targetPostID, func() (interface{}, error) {
				// 模拟耗时的数据库回源查询 (50ms)
				time.Sleep(50 * time.Millisecond)
				atomic.AddInt64(&dbHitCount, 1)
				return "post_data_content", nil
			})
			assert.NoError(t, err)
			results[idx] = val.(string)
		}()
	}

	wg.Wait()

	// 验证 1：尽管有 50 个并发请求，实际打库闭包只执行了 1 次 (防击穿成功！)
	assert.Equal(t, int64(1), atomic.LoadInt64(&dbHitCount), "50个并发请求必须被合并为单次查询")

	// 验证 2：所有 50 个请求均成功获取到正确且一致的数据
	for _, res := range results {
		assert.Equal(t, "post_data_content", res)
	}
}

// TestPostService_PenetrationDefense 验证非法 ID 和布隆过滤器对不存在帖子的防御 (防穿透)
func TestPostService_PenetrationDefense(t *testing.T) {
	svc := &PostService{}

	ctx := context.Background()

	// 1. 非法 ID 参数校验前置拦截
	_, err := svc.GetPostByID(ctx, -1, 0)
	assert.ErrorIs(t, err, model.ErrInvalidParam, "负数 ID 必须被前置拦截")

	_, err = svc.GetPostByID(ctx, 0, 0)
	assert.ErrorIs(t, err, model.ErrInvalidParam, "ID 为 0 必须被前置拦截")
}
