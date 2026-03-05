package response

import (
	"bluebell/pkg/errorx"
	"errors"
	"net/http"
	"github.com/gin-gonic/gin"
)

// ResponseData 统一响应结构体 (用于 Swagger 文档生成)
type ResponseData struct {
	Code int  `json:"code"`           // 业务响应状态码
	Msg  any  `json:"msg"`            // 提示信息
	Data any  `json:"data,omitempty"` // 数据
}

// ResponseSuccess 返回成功响应
func ResponseSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, &ResponseData{
		Code: errorx.CodeSuccess,
		Msg:  "success",
		Data: data,
	})
}

// HandleError 通用错误处理方法
func HandleError(c *gin.Context, err error) {
	var codeErr *errorx.CodeError
	if errors.As(err, &codeErr) {
		// 如果是业务错误，只返回 Msg，不带底层 cause (避免泄露敏感信息)
		c.JSON(http.StatusOK, &ResponseData{
			Code: codeErr.Code,
			Msg:  codeErr.Msg,
			Data: nil,
		})
		return
	}

	// 如果是非业务错误，返回统一的“服务繁忙”
	c.JSON(http.StatusOK, &ResponseData{
		Code: errorx.CodeServerBusy,
		Msg:  errorx.ErrServerBusy.Error(),
		Data: nil,
	})
}

// HandleErrorWithMsg 返回带自定义消息的错误响应
func HandleErrorWithMsg(c *gin.Context, code int, msg any) {
	c.JSON(http.StatusOK, &ResponseData{
		Code: code,
		Msg:  msg,
		Data: nil,
	})
}
