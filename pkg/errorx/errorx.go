package errorx

import (
	"errors"
	"fmt"
)

//内部实例化返回指针（&CodeError），但在函数签名上对外暴露 error 接口，是 Go 语言里最安全、最地道的写法！

// CodeError 带业务错误码的自定义错误
// 实现了 error 接口，支持 %w 包装底层错误，且能被 errors.Is/errors.As 识别
type CodeError struct {
	Code  int    // 业务错误码
	Msg   string // 错误消息
	cause error  // 被包装的底层错误
}

// Error 实现 Go 标准 error 接口
// 当存在底层错误时，返回格式为 "消息: 底层错误"；否则仅返回消息
func (e *CodeError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Msg, e.cause)
	}
	return e.Msg
}

// Unwrap 实现 errors.Unwrap 接口，支持 errors.Is/errors.As 向下追溯
func (e *CodeError) Unwrap() error {
	return e.cause
}

// New 创建一个新的 CodeError
func New(code int, msg string) error {
	return &CodeError{
		Code: code,
		Msg:  msg,
	}
}

// Newf 创建一个带格式化消息的 CodeError
func Newf(code int, format string, args ...any) error {
	return &CodeError{
		Code: code,
		Msg:  fmt.Sprintf(format, args...),
	}
}

// Wrap 包装底层错误，添加业务错误码和消息
// 用法: errorx.Wrap(err, CodeNotFound, "用户不存在")
func Wrap(err error, code int, msg string) error {
	return &CodeError{
		Code:  code,
		Msg:   msg,
		cause: err,
	}
}

// Wrapf 包装底层错误，支持格式化消息
// 用法: errorx.Wrapf(err, CodeNotFound, "用户 %s 不存在", userId)
func Wrapf(err error, code int, format string, args ...any) error {
	return &CodeError{
		Code:  code,
		Msg:   fmt.Sprintf(format, args...),
		cause: err,
	}
}

// GetCode 从错误中提取业务错误码，如果不是 CodeError 则返回默认码
func GetCode(err error) int {
	var codeErr *CodeError
	if errors.As(err, &codeErr) {
		return codeErr.Code
	}
	return CodeServerBusy
}

// IsNotFound 检查错误是否为"未找到"类型
func IsNotFound(err error) bool {
	var codeErr *CodeError
	if errors.As(err, &codeErr) && codeErr.Code == CodeNotFound {
		return true
	}
	return false
}

// ==================== 全局业务错误码 ====================
// 规范统一后端和前端交互的业务级错误
const (
	CodeSuccess           = 1000 // 成功
	CodeInvalidParam      = 1001 // 请求参数错误
	CodeUserExist         = 1002 // 用户已存在
	CodeUserNotExist      = 1003 // 用户不存在
	CodeInvalidPassword   = 1004 // 密码错误
	CodeServerBusy        = 1005 // 服务繁忙
	CodeNeedLogin         = 1006 // 未登录
	CodeInvalidToken      = 1007 // 无效Token
	CodeNotFound          = 1008 // 资源不存在
	CodeVoteTimeExpire    = 1009 // 投票时间已过
	CodeVoteRepeated      = 1010 // 重复投票
	CodeRateLimitExceeded = 1011 // 请求过于频繁
	CodeForbidden         = 1012 // 无权限操作
	CodeDBError           = 1013 // 数据库错误
	CodeCacheError        = 1014 // 缓存错误
	CodeConfigError       = 1015 // 配置错误
	CodeInfraError        = 1016 // 基础设施初始化错误
	CodeRequestTimeout    = 1017 // 请求超时
)

// 预定义常用错误实例
var (
	ErrInvalidParam      = New(CodeInvalidParam, "请求参数错误")
	ErrUserExist         = New(CodeUserExist, "用户名已存在")
	ErrUserNotExist      = New(CodeUserNotExist, "用户名不存在")
	ErrInvalidPassword   = New(CodeInvalidPassword, "用户名或密码错误")
	ErrServerBusy        = New(CodeServerBusy, "服务繁忙")
	ErrNeedLogin         = New(CodeNeedLogin, "需要登录")
	ErrInvalidToken      = New(CodeInvalidToken, "无效的Token")
	ErrNotFound          = New(CodeNotFound, "资源不存在")
	ErrVoteTimeExpire    = New(CodeVoteTimeExpire, "投票时间已过")
	ErrVoteRepeated      = New(CodeVoteRepeated, "不允许重复投票")
	ErrRateLimitExceeded = New(CodeRateLimitExceeded, "请求过于频繁，请稍后再试")
	ErrForbidden         = New(CodeForbidden, "无权限操作")
	ErrConfigInit        = New(CodeConfigError, "配置初始化失败")
	ErrInfraInit         = New(CodeInfraError, "基础设施初始化失败")
	ErrRequestTimeout    = New(CodeRequestTimeout, "请求超时")
)

/*User{}.TableName()（GORM 模型读取属性）： GORM 内部经常操作切片 []User 的值对象来进行反射。为了防止 GORM 反射不到表名，必须用值接收者。
User{}.BeforeCreate()（GORM 模型修改属性）： 为了保证密码修改能真正生效并落库，必须用指针接收者。
CodeError{}.Error()（自定义错误接口）： 错误对象在程序里跑来跑去，为了性能极致和规避复制带来的不可控行为，必须用指针传递及指针接收者*/
