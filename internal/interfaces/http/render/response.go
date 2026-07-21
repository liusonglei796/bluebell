package render

import (
	"errors"
	"net/http"

	"bluebell/internal/domain/entity"
	"github.com/gin-gonic/gin"
)

// classifyError 将领域错误映射为 HTTP 状态码和 Prometheus 错误分类标签
func classifyError(err error) (int, string) {
	switch {
	case errors.Is(err, entity.ErrInvalidParam):
		return http.StatusBadRequest, "validation"
	case errors.Is(err, entity.ErrNotFound):
		return http.StatusNotFound, "not_found"
	case errors.Is(err, entity.ErrUnauthorized), errors.Is(err, entity.ErrNeedLogin), errors.Is(err, entity.ErrInvalidToken), errors.Is(err, entity.ErrNotLogin):
		return http.StatusUnauthorized, "auth"
	case errors.Is(err, entity.ErrForbidden):
		return http.StatusForbidden, "forbidden"
	case errors.Is(err, entity.ErrDuplicate), errors.Is(err, entity.ErrUserExist), errors.Is(err, entity.ErrVoteRepeated), errors.Is(err, entity.ErrVoteTimeExpire):
		return http.StatusConflict, "conflict"
	case errors.Is(err, entity.ErrRateLimitExceeded):
		return http.StatusTooManyRequests, "rate_limit"
	case errors.Is(err, entity.ErrServerBusy):
		return http.StatusServiceUnavailable, "server_busy"
	default:
		return http.StatusInternalServerError, "system"
	}
}

// HandleError 根据领域层的错误类型，返回合适的 HTTP 状态码和原生的 JSON 响应
func HandleError(c *gin.Context, err error) {
	status, _ := classifyError(err)


	c.JSON(status, gin.H{"error": err.Error()})
}

// HandleSuccess 快速返回成功，无额外包装
func HandleSuccess(c *gin.Context, data interface{}) {
	if data == nil {
		c.Status(http.StatusOK)
		return
	}
	c.JSON(http.StatusOK, data)
}
