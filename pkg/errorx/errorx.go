package errorx

import "fmt"

// CodeError 带业务错误码的自定义错误
// 实现了 error 接口，用于在 Logic 层返回包含业务错误码的错误
type CodeError struct {
	Code int    // 业务错误码
	Msg  string // 错误消息
}

// Error 实现 error 接口
func (e *CodeError) Error() string {
	return e.Msg
}

// New 创建一个新的 CodeError
func New(code int, msg string) *CodeError {
	return &CodeError{
		Code: code,
		Msg:  msg,
	}
}

// Newf 创建一个带格式化消息的 CodeError
func Newf(code int, format string, args ...any) *CodeError {
	return &CodeError{
		Code: code,
		Msg:  fmt.Sprintf(format, args...),
	}
}

// 业务错误码常量定义
const (
	CodeInvalidParam    = 1001
	CodeUserExist       = 1002
	CodeUserNotExist    = 1003
	CodeInvalidPassword = 1004
	CodeServerBusy      = 1005
	CodeNeedLogin       = 1006
	CodeInvalidToken    = 1007
	CodeNotFound        = 1008
	CodeVoteTimeExpire  = 1009 // 投票时间已过
	CodeVoteRepeated    = 1010 // 重复投票
)

// 预定义常用错误实例（Logic 层可直接返回）
var (
	ErrInvalidParam    = New(CodeInvalidParam, "请求参数错误")
	ErrUserExist       = New(CodeUserExist, "用户名已存在")
	ErrUserNotExist    = New(CodeUserNotExist, "用户名不存在")
	ErrInvalidPassword = New(CodeInvalidPassword, "用户名或密码错误")
	ErrServerBusy      = New(CodeServerBusy, "服务繁忙")
	ErrNeedLogin       = New(CodeNeedLogin, "需要登录")
	ErrInvalidToken    = New(CodeInvalidToken, "无效的Token")
	ErrNotFound        = New(CodeNotFound, "资源不存在")
	ErrVoteTimeExpire  = New(CodeVoteTimeExpire, "投票时间已过")
	ErrVoteRepeated    = New(CodeVoteRepeated, "不允许重复投票")
)
