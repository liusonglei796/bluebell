// Package controller 提供 HTTP 控制器（MVC 的 C 层）
package controller

import (
	"errors"
	"net/http"

	"bluebell/internal/model"

	"github.com/gin-gonic/gin"
)

// 业务响应码定义
const (
	CodeSuccess           = 1000
	CodeInvalidParam      = 1001
	CodeUserExist         = 1002
	CodeUserNotExist      = 1003
	CodeInvalidPassword   = 1004
	CodeServerBusy        = 1005
	CodeNeedLogin         = 1006
	CodeInvalidToken      = 1007
	CodeVoteTimeExpire    = 1008
	CodeVoteRepeated      = 1009
	CodeDuplicateSubmit   = 1010
	CodeForbidden         = 1011
	CodeNotFound          = 1012
	CodeDuplicate         = 1013
	CodeRateLimitExceeded = 1014
)

// classifyErrorCode 将业务错误映射为业务状态码
func classifyErrorCode(err error) int {
	switch {
	case errors.Is(err, model.ErrInvalidParam):
		return CodeInvalidParam
	case errors.Is(err, model.ErrUserExist):
		return CodeUserExist
	case errors.Is(err, model.ErrUserNotExist):
		return CodeUserNotExist
	case errors.Is(err, model.ErrInvalidPassword):
		return CodeInvalidPassword
	case errors.Is(err, model.ErrNeedLogin), errors.Is(err, model.ErrNotLogin), errors.Is(err, model.ErrUnauthorized):
		return CodeNeedLogin
	case errors.Is(err, model.ErrInvalidToken):
		return CodeInvalidToken
	case errors.Is(err, model.ErrVoteTimeExpire):
		return CodeVoteTimeExpire
	case errors.Is(err, model.ErrVoteRepeated):
		return CodeVoteRepeated
	case errors.Is(err, model.ErrDuplicateSubmit):
		return CodeDuplicateSubmit
	case errors.Is(err, model.ErrForbidden):
		return CodeForbidden
	case errors.Is(err, model.ErrNotFound):
		return CodeNotFound
	case errors.Is(err, model.ErrDuplicate):
		return CodeDuplicate
	case errors.Is(err, model.ErrRateLimitExceeded):
		return CodeRateLimitExceeded
	case errors.Is(err, model.ErrServerBusy):
		return CodeServerBusy
	default:
		return CodeServerBusy
	}
}

// classifyError 将业务错误映射为 HTTP 状态码
func classifyError(err error) int {
	switch {
	case errors.Is(err, model.ErrInvalidParam):
		return http.StatusBadRequest
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, model.ErrUnauthorized), errors.Is(err, model.ErrNeedLogin), errors.Is(err, model.ErrInvalidToken), errors.Is(err, model.ErrNotLogin):
		return http.StatusUnauthorized
	case errors.Is(err, model.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, model.ErrDuplicate), errors.Is(err, model.ErrUserExist), errors.Is(err, model.ErrVoteRepeated), errors.Is(err, model.ErrVoteTimeExpire):
		return http.StatusConflict
	case errors.Is(err, model.ErrRateLimitExceeded), errors.Is(err, model.ErrDuplicateSubmit):
		return http.StatusTooManyRequests
	case errors.Is(err, model.ErrServerBusy):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// HandleError 根据错误的类型，返回合适的 HTTP 状态码和 JSON 响应
func HandleError(c *gin.Context, err error) {
	c.JSON(classifyError(err), gin.H{
		"code":  classifyErrorCode(err),
		"msg":   err.Error(),
		"error": err.Error(),
	})
}

// HandleSuccess 返回标准成功响应
func HandleSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": CodeSuccess,
		"msg":  "success",
		"data": data,
	})
}

