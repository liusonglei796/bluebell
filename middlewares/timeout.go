package middlewares

import (
	"bluebell/controller"
	"bluebell/pkg/errorx"
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutMiddleware 请求超时中间件
// 防止慢请求占用资源，提高系统稳定性
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建带超时的 context
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		// 替换请求的 context
		c.Request = c.Request.WithContext(ctx)

		// 使用 channel 监听请求完成
		finished := make(chan struct{})
		go func() {
			c.Next()
			close(finished)
		}()

		// 等待请求完成或超时
		select {
		case <-finished:
			// 请求正常完成
			return
		case <-ctx.Done():
			// 请求超时
			if ctx.Err() == context.DeadlineExceeded {
				controller.ResponseErrorWithMsg(c, errorx.CodeServerBusy, "请求超时")
				c.Abort()
			}
			return
		}
	}
}
