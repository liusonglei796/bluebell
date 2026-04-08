package middleware

import (
	"net/http"
	"time"

	"bluebell/internal/backfront"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
)

// RateLimitMiddleware 基于令牌桶的全局限流中间件
//
// 参数:
//   - fillInterval: 令牌填充间隔（如 20ms 表示每 20ms 生成 1 个令牌）
//   - capacity:     桶容量（最大令牌数，也是最大突发请求数）
//
// 中型项目推荐配置:
//   - fillInterval: "20ms"  → 50 请求/秒
//   - capacity: 100         → 允许 100 次突发请求
//
// 使用示例:
//
//	r.Use(middleware.RateLimitMiddleware(20*time.Millisecond, 100))
func RateLimitMiddleware(fillInterval time.Duration, capacity int64) gin.HandlerFunc {
	bucket := ratelimit.NewBucket(fillInterval, capacity)

	return func(c *gin.Context) {
		// 尝试获取 1 个令牌，如果桶中没有足够令牌则立即返回 429
		if !bucket.WaitMaxDuration(1, 0) {
			c.Header("Retry-After", "1")
			c.Header("X-RateLimit-Limit", "rate-limited")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, &backfront.ResponseData{
				Code: errorx.CodeRateLimitExceeded,
				Msg:  errorx.ErrRateLimitExceeded.Error(),
				Data: nil,
			})
			return
		}

		c.Next()
	}
}

// RateLimitMiddlewareWithHeader 带响应头信息的令牌桶限流中间件
// 会在响应头中添加限流相关信息
func RateLimitMiddlewareWithHeader(fillInterval time.Duration, capacity int64) gin.HandlerFunc {
	bucket := ratelimit.NewBucket(fillInterval, capacity)

	return func(c *gin.Context) {
		if !bucket.WaitMaxDuration(1, 0) {
			c.Header("Retry-After", "1")
			c.Header("X-RateLimit-Limit", "rate-limited")
			c.Header("X-RateLimit-Remaining", "0")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, &backfront.ResponseData{
				Code: errorx.CodeRateLimitExceeded,
				Msg:  errorx.ErrRateLimitExceeded.Error(),
				Data: nil,
			})
			return
		}

		c.Header("X-RateLimit-Limit", "token-bucket")
		remaining := bucket.Available()
		c.Header("X-RateLimit-Remaining", string(rune(remaining)))

		c.Next()
	}
}
