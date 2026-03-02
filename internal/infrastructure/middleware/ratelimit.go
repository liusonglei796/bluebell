package middleware

import (
	"bluebell/pkg/errorx"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
)

// RateLimitMiddleware 创建一个令牌桶限流中间件
func RateLimitMiddleware(fillInterval time.Duration, capacity int64) gin.HandlerFunc {
	bucket := ratelimit.NewBucket(fillInterval, capacity)

	return func(c *gin.Context) {
		if bucket.TakeAvailable(1) < 1 {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code": errorx.CodeRateLimitExceeded,
				"msg":  "请求过于频繁，请稍后再试",
				"data": nil,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
