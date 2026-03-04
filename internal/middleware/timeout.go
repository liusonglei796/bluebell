package middleware

import (
	"bluebell/internal/handler"
	"bluebell/pkg/errorx"
	"context"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutMiddleware 请求超时中间件
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		finished := make(chan struct{})
		go func() {
			c.Next()
			close(finished)
		}()

		select {
		case <-finished:
			return
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				handler.ResponseErrorWithMsg(c, errorx.CodeServerBusy, "请求超时")
				c.Abort()
			}
			return
		}
	}
}
