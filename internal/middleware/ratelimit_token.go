package middleware

import (
	"net/http"
	"sync/atomic"
	"time"

	"bluebell/internal/backfront"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
	"go.uber.org/zap"
)

var (
	totalRequests   int64
	limitedRequests int64
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

	zap.L().Info("RateLimitMiddleware initialized",
		zap.Duration("fillInterval", fillInterval),
		zap.Int64("capacity", capacity),
		zap.Float64("ratePerSecond", float64(time.Second)/float64(fillInterval)))

	return func(c *gin.Context) {
		atomic.AddInt64(&totalRequests, 1)

		// TakeAvailable 立即返回可获取的令牌数，不会阻塞
		// 如果返回值 < 1 说明桶中没有足够令牌，触发限流
		if bucket.TakeAvailable(1) < 1 {
			atomic.AddInt64(&limitedRequests, 1)
			limited := atomic.LoadInt64(&limitedRequests)
			total := atomic.LoadInt64(&totalRequests)
			zap.L().Warn("Rate limit triggered",
				zap.Int64("limited", limited),
				zap.Int64("total", total),
				zap.Float64("rate", float64(limited)/float64(total)*100))

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
