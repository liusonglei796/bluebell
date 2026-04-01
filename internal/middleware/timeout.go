package middleware

import (
	"bluebell/internal/backfront"
	"bluebell/pkg/errorx"
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
	backfront.HandleError(c, errorx.ErrServerBusy)
}

// GinLogger 日志中间件（确保 Recovery 在 timeout 之后执行）
// 注意：gin-contrib/timeout 内部包含 panic 恢复逻辑
// 如果需要全局 Recovery，应放在 timeout 中间件之前
