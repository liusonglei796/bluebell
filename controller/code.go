package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CtxUserIDKey 定义 Context 中 UserID 的 Key
// 注意:将此常量定义在 controller 包中而非 middlewares 包中,是为了避免循环引用
const CtxUserIDKey = "userID"

// ResponseData 统一响应结构体 (用于 Swagger 文档生成)
type ResponseData struct {
	Code int         `json:"code"`           // 业务响应状态码
	Msg  interface{} `json:"msg"`            // 提示信息
	Data interface{} `json:"data,omitempty"` // 数据
}

// 响应码定义
const (
	CodeSuccess       = 1000
	CodeInvalidParam  = 1001
	CodeUserExist     = 1002
	CodeUserNotExist  = 1003
	CodeInvalidPassword = 1004
	CodeServerBusy    = 1005
	CodeNeedLogin     = 1006
	CodeInvalidToken  = 1007
	CodeNotFound      = 1008  // 添加新的错误码
)

// 响应消息映射
var MsgFlags = map[int]string{
	CodeSuccess:       "success",
	CodeInvalidParam:  "请求参数错误",
	CodeUserExist:     "用户名已存在",
	CodeUserNotExist:  "用户名不存在",
	CodeInvalidPassword: "用户名或密码错误",
	CodeServerBusy:    "服务繁忙",
	CodeNeedLogin:     "需要登录",
	CodeInvalidToken:  "无效的Token",
	CodeNotFound:      "资源不存在",  // 添加新的错误消息
}

// ResponseError 返回错误响应
func ResponseError(c *gin.Context, code int) {
	c.JSON(http.StatusOK, gin.H{
		"code": code,
		"msg":  MsgFlags[code],
		"data": nil,
	})
}

// ResponseErrorWithMsg 返回带自定义消息的错误响应
func ResponseErrorWithMsg(c *gin.Context, code int, msg interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": code,
		"msg":  msg,
		"data": nil,
	})
}

// ResponseSuccess 返回成功响应
func ResponseSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code": CodeSuccess,
		"msg":  MsgFlags[CodeSuccess],
		"data": data,
	})
}