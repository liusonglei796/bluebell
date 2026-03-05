package middleware

import (
	"bluebell/internal/response"
	"bluebell/pkg/errorx"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
)

// RateLimitMiddleware 创建一个令牌桶限流中间件
func RateLimitMiddleware(fillInterval time.Duration, capacity int64) gin.HandlerFunc {
	bucket := ratelimit.NewBucket(fillInterval, capacity)
	return func(c *gin.Context) {
		if bucket.TakeAvailable(1) < 1 {
			response.HandleError(c, errorx.ErrRateLimitExceeded)
			c.Abort()
			return
		}
		c.Next()
	}
}
