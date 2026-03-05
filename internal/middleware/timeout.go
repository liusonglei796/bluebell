package middleware

import (
	"bluebell/internal/response"
	"bluebell/pkg/errorx"
	"context"
	"time"

	"github.com/gin-gonic/gin"
)
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		ch := make(chan struct{})
		go func() {
			c.Next()
			close(ch)
		}()
		select {
		case <-ch:
			return
		case <-c.Request.Context().Done():
			if c.Request.Context().Err() == context.DeadlineExceeded {
				response.HandleError(c, errorx.ErrServerBusy)
				c.Abort()
			}
			return
		}
	}
}
