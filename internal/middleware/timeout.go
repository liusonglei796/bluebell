package middleware

import (
	"bluebell/internal/model"
	"net/http"
	"time"

	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
)

func TimeoutMiddleware(timeoutDuration time.Duration) gin.HandlerFunc {
	return timeout.New(
		timeout.WithTimeout(timeoutDuration),
		timeout.WithResponse(customTimeoutResponse),
	)
}

// customTimeoutResponse 自定义超时响应
func customTimeoutResponse(c *gin.Context) {
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": model.ErrServerBusy.Error()})
}
