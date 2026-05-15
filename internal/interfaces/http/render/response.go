package render

import (
	"errors"
	"net/http"

	"bluebell/internal/domain/entity"
	"github.com/gin-gonic/gin"
)

// HandleError 根据领域层的错误类型，返回合适的 HTTP 状态码和原生的 JSON 响应
func HandleError(c *gin.Context, err error) {
	status := http.StatusInternalServerError

	if errors.Is(err, entity.ErrInvalidParam) {
		status = http.StatusBadRequest
	} else if errors.Is(err, entity.ErrNotFound) {
		status = http.StatusNotFound
	} else if errors.Is(err, entity.ErrUnauthorized) || errors.Is(err, entity.ErrNeedLogin) || errors.Is(err, entity.ErrInvalidToken) || errors.Is(err, entity.ErrNotLogin) {
		status = http.StatusUnauthorized
	} else if errors.Is(err, entity.ErrForbidden) {
		status = http.StatusForbidden
	} else if errors.Is(err, entity.ErrDuplicate) || errors.Is(err, entity.ErrUserExist) || errors.Is(err, entity.ErrVoteRepeated) || errors.Is(err, entity.ErrVoteTimeExpire) {
		status = http.StatusConflict
	} else if errors.Is(err, entity.ErrRateLimitExceeded) {
		status = http.StatusTooManyRequests
	} else if errors.Is(err, entity.ErrServerBusy) {
		status = http.StatusServiceUnavailable
	}

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
