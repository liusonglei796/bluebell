package controller

import (
	"bluebell/pkg/errorx"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CtxUserIDKey 定义 Context 中 UserID 的 Key
// 注意:将此常量定义在 controller 包中而非 middlewares 包中,是为了避免循环引用
const CtxUserIDKey = "userID"

// ResponseData 统一响应结构体 (用于 Swagger 文档生成)
type ResponseData struct {
	Code int `json:"code"`           // 业务响应状态码
	Msg  any `json:"msg"`            // 提示信息
	Data any `json:"data,omitempty"` // 数据
}

// CodeSuccess 成功状态码
// 注意：错误码统一使用 errorx.CodeXXX
const CodeSuccess = 1000

// ResponseError 返回错误响应 (接受 CodeError 实例)
// 推荐直接使用: HandleError(c, err)
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
// 自动识别 errorx.CodeError 类型的业务错误，或者将系统错误转换为 CodeServerBusy
// 使用示例：
//   if err := logic.Login(p); err != nil {
//       HandleError(c, err)
//       return
//   }
func HandleError(c *gin.Context, err error) {
	// 1. 尝试断言为 *errorx.CodeError 类型
	var codeErr *errorx.CodeError
	if errors.As(err, &codeErr) {
		// 业务错误：直接返回携带的错误码和消息
		c.JSON(http.StatusOK, gin.H{
			"code": codeErr.Code,
			"msg":  codeErr.Msg,
			"data": nil,
		})
		return
	}

	// 2. 系统错误或未知错误：记录日志并返回服务繁忙
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
