package render

import (
	"errors"
	"net/http"

	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/metrics"
	"github.com/gin-gonic/gin"
)

// ────────────────────────────────────────────────────────────
//  统一 API 响应体（泛型）
// ────────────────────────────────────────────────────────────

// Response 是所有 API 返回的统一信封。
// 泛型参数 T 让 Data 字段保持强类型，序列化后结构与直接返回业务对象一致。
//
// 示例 JSON:
//
//	成功: {"code":0,"message":"ok","data":{...}}
//	失败: {"code":10001,"message":"参数错误"}
type Response[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
}

// ────────────────────────────────────────────────────────────
//  业务错误码（与 HTTP 状态码解耦）
// ────────────────────────────────────────────────────────────

const (
	CodeOK                = 0
	CodeInvalidParam      = 10001
	CodeNotFound          = 10002
	CodeUnauthorized      = 10003
	CodeForbidden         = 10004
	CodeDuplicate         = 10005
	CodeRateLimitExceeded = 10006
	CodeServerBusy        = 10007
	CodeInternal          = 19999
)

// ────────────────────────────────────────────────────────────
//  错误分类 & 映射
// ────────────────────────────────────────────────────────────

// classifyError 将领域错误映射为 HTTP 状态码、业务码和 Prometheus 标签
func classifyError(err error) (httpStatus int, bizCode int, tag string) {
	switch {
	case errors.Is(err, entity.ErrInvalidParam):
		return http.StatusBadRequest, CodeInvalidParam, "validation"
	case errors.Is(err, entity.ErrNotFound):
		return http.StatusNotFound, CodeNotFound, "not_found"
	case errors.Is(err, entity.ErrUnauthorized), errors.Is(err, entity.ErrNeedLogin), errors.Is(err, entity.ErrInvalidToken), errors.Is(err, entity.ErrNotLogin):
		return http.StatusUnauthorized, CodeUnauthorized, "auth"
	case errors.Is(err, entity.ErrForbidden):
		return http.StatusForbidden, CodeForbidden, "forbidden"
	case errors.Is(err, entity.ErrDuplicate), errors.Is(err, entity.ErrUserExist), errors.Is(err, entity.ErrVoteRepeated), errors.Is(err, entity.ErrVoteTimeExpire):
		return http.StatusConflict, CodeDuplicate, "conflict"
	case errors.Is(err, entity.ErrRateLimitExceeded):
		return http.StatusTooManyRequests, CodeRateLimitExceeded, "rate_limit"
	case errors.Is(err, entity.ErrServerBusy):
		return http.StatusServiceUnavailable, CodeServerBusy, "server_busy"
	default:
		return http.StatusInternalServerError, CodeInternal, "system"
	}
}

// ────────────────────────────────────────────────────────────
//  Handler 层调用的响应函数
// ────────────────────────────────────────────────────────────

// HandleError 根据领域层的错误类型，返回统一格式的 JSON 响应
func HandleError(c *gin.Context, err error) {
	status, bizCode, tag := classifyError(err)

	ctx := c.Request.Context()
	metrics.RecordError(ctx, tag)

	c.JSON(status, Response[any]{
		Code:    bizCode,
		Message: err.Error(),
	})
}

// HandleSuccess 返回统一格式的成功响应。
// data 为 nil 时只返回 200 状态码（无 body），适用于创建/删除等无返回数据的场景。
func HandleSuccess(c *gin.Context, data any) {
	if data == nil {
		c.JSON(http.StatusOK, Response[any]{
			Code:    CodeOK,
			Message: "ok",
		})
		return
	}
	c.JSON(http.StatusOK, Response[any]{
		Code:    CodeOK,
		Message: "ok",
		Data:    data,
	})
}

// HandleValidationError 返回参数校验失败的统一响应，
// 消除各 handler 中重复的 "validator → translate → c.JSON" 模板代码。
func HandleValidationError(c *gin.Context, detail any) {
	c.JSON(http.StatusBadRequest, Response[any]{
		Code:    CodeInvalidParam,
		Message: "参数校验失败",
		Data:    detail,
	})
}
