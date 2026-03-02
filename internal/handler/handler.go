package handler

import (
	"bluebell/internal/service"
	"bluebell/pkg/errorx"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CtxUserIDKey 定义 Context 中 UserID 的 Key
const CtxUserIDKey = "userID"

// Handlers 聚合所有 Handler 方法
// 持有 Services 引用，作为 Handler 层的入口
type Handlers struct {
	Services *service.Services
}

// NewHandlers 创建 Handlers 实例
func NewHandlers(services *service.Services) *Handlers {
	return &Handlers{Services: services}
}

// ResponseData 统一响应结构体 (用于 Swagger 文档生成)
type ResponseData struct {
	Code int `json:"code"`           // 业务响应状态码
	Msg  any `json:"msg"`            // 提示信息
	Data any `json:"data,omitempty"` // 数据
}

// CodeSuccess 成功状态码
const CodeSuccess = 1000

// ResponseError 返回错误响应 (接受 CodeError 实例)
func ResponseError(c *gin.Context, err *errorx.CodeError) {
	c.JSON(http.StatusOK, gin.H{
		"code": err.Code,
		"msg":  err.Msg,
		"data": nil,
	})
}

// ResponseErrorWithMsg 返回带自定义消息的错误响应
func ResponseErrorWithMsg(c *gin.Context, code int, msg any) {
	c.JSON(http.StatusOK, gin.H{
		"code": code,
		"msg":  msg,
		"data": nil,
	})
}

// ResponseSuccess 返回成功响应
func ResponseSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{
		"code": CodeSuccess,
		"msg":  "success",
		"data": data,
	})
}

// HandleError 通用错误处理方法
func HandleError(c *gin.Context, err error) {
	var codeErr *errorx.CodeError
	if errors.As(err, &codeErr) {
		c.JSON(http.StatusOK, gin.H{
			"code": codeErr.Code,
			"msg":  codeErr.Msg,
			"data": nil,
		})
		return
	}

	zap.L().Error("system error occurred",
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.Error(err),
	)
	c.JSON(http.StatusOK, gin.H{
		"code": errorx.ErrServerBusy.Code,
		"msg":  errorx.ErrServerBusy.Msg,
		"data": nil,
	})
}

// ErrorUserNotLogin 用户未登录错误
var ErrorUserNotLogin = errors.New("用户未登录")

// GetCurrentUser 从 Gin 上下文中获取当前登录的用户ID
func GetCurrentUser(c *gin.Context) (userID int64, err error) {
	uid, ok := c.Get(CtxUserIDKey)
	if !ok {
		err = ErrorUserNotLogin
		return
	}

	userID, ok = uid.(int64)
	if !ok {
		err = ErrorUserNotLogin
		return
	}

	return userID, nil
}

// stringToInt64 将字符串转换为int64
func stringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// getPageInfo 从Gin上下文中获取分页参数
func getPageInfo(c *gin.Context) (page, size int64) {
	pageStr := c.Query("page")
	if pageStr == "" {
		page = 1
	} else {
		page, _ = strconv.ParseInt(pageStr, 10, 64)
		if page <= 0 {
			page = 1
		}
	}

	sizeStr := c.Query("size")
	if sizeStr == "" {
		size = 10
	} else {
		size, _ = strconv.ParseInt(sizeStr, 10, 64)
		if size <= 0 || size > 100 {
			size = 10
		}
	}

	return page, size
}
