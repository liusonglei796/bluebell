package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestUserRateLimitMiddleware_UserIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	// 每个用户每秒补充 1 个令牌，桶容量为 3（最多允许 3 次突发）
	limiter := UserRateLimitMiddleware(1*time.Second, 3)

	r.GET("/api/v1/vote", func(c *gin.Context) {
		// 模拟由上游 JWTAuthMiddleware 注入 UserID
		if uidStr := c.Query("uid"); uidStr != "" {
			if uidStr == "1001" {
				c.Set("UserIDKey", int64(1001))
			} else if uidStr == "1002" {
				c.Set("UserIDKey", int64(1002))
			}
		}
		c.Next()
	}, limiter, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "vote success"})
	})

	// 1. 用户 1001 连续发送 3 次投票 -> 应当全部成功 (200)
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/vote?uid=1001", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "用户1001在容量内的请求应成功")
	}

	// 2. 用户 1001 发送第 4 次投票 -> 超过容量，应被拦截 (429)
	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/vote?uid=1001", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code, "用户1001超出容量后应返回 429")
		assert.Equal(t, "user-rate-limited", w.Header().Get("X-RateLimit-Limit"))
	}

	// 3. 用户 1002 发起第 1 次投票 -> 应当不受 1001 影响，正常成功 (200)！
	{
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/vote?uid=1002", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "用户1002应当完全独立，不受用户1001被限流的影响")
	}
}

func TestUserRateLimitMiddleware_IPFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	limiter := UserRateLimitMiddleware(1*time.Second, 2)

	r.GET("/api/v1/vote/anonymous", limiter, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "ok"})
	})

	// 无 UserID 场景下降级按 IP 限流
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/vote/anonymous", nil)
		req.RemoteAddr = "192.168.1.50:12345"
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// 第 3 次触发限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/vote/anonymous", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestUserLimiter_Cleanup(t *testing.T) {
	// 空闲 TTL 设置为 50ms
	limiter := NewUserLimiter(1*time.Second, 5, 50*time.Millisecond)
	defer limiter.Close()

	// 注入一个用户桶
	b := limiter.getBucket("user:9999")
	assert.NotNil(t, b)

	limiter.mu.RLock()
	assert.Equal(t, 1, len(limiter.buckets))
	limiter.mu.RUnlock()

	// 等待超过空闲 TTL
	time.Sleep(80 * time.Millisecond)

	// 执行清理
	limiter.cleanup()

	limiter.mu.RLock()
	assert.Equal(t, 0, len(limiter.buckets), "超过空闲时间的用户桶应被自动回收")
	limiter.mu.RUnlock()
}

func TestUserRateLimitMiddleware_Concurrency(t *testing.T) {
	limiter := NewUserLimiter(100*time.Millisecond, 10, 1*time.Minute)
	defer limiter.Close()

	var wg sync.WaitGroup
	// 50 个并发 goroutine 同时获取桶并消耗令牌，验证 race condition
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(uid int) {
			defer wg.Done()
			bucket := limiter.getBucket("user:concurrency")
			_ = bucket.TakeAvailable(1)
		}(i)
	}
	wg.Wait()
}
