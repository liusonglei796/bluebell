package middlewares

import (
	"bluebell/pkg/errorx"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
)

// RateLimitMiddleware 创建一个令牌桶限流中间件
// fillInterval: 令牌填充间隔（例如 10ms = 每秒100个令牌）
// capacity: 令牌桶的总容量（允许的突发请求数量）
//
// 工作原理：
// 1. 系统以固定速率往桶里放入令牌（fillInterval 控制速率）
// 2. 每个请求必须获取1个令牌才能通过
// 3. 如果桶空了，请求被拒绝并返回 429 Too Many Requests
// 4. 桶的容量决定了允许的最大突发流量
func RateLimitMiddleware(fillInterval time.Duration, capacity int64) gin.HandlerFunc {
	// 创建一个令牌桶
	// fillInterval: 每隔多久往桶里放一个令牌
	// capacity: 桶的最大容量
	bucket := ratelimit.NewBucket(fillInterval, capacity)

	return func(c *gin.Context) {
		// 尝试从桶中获取 1 个令牌 (非阻塞)
		// TakeAvailable(1) 返回实际获取到的令牌数
		// 如果返回值 < 1，说明桶空了，需要限流
		if bucket.TakeAvailable(1) < 1 {
			// 限流：返回 429 Too Many Requests
			// 直接使用 JSON 返回，而不是 ResponseError (因为 ResponseError 总是返回 200)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code": errorx.CodeRateLimitExceeded,
				"msg":  "请求过于频繁，请稍后再试",
				"data": nil,
			})
			c.Abort() // 终止后续处理
			return
		}

		// 获取到令牌，放行请求
		c.Next()
	}
}
