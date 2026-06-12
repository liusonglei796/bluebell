package render

// 业务状态码
// 200: 成功
// 1xxx: 通用错误
// 2xxx: 认证授权错误
// 3xxx: 资源错误
const (
	CodeSuccess      = 200  // 成功
	CodeInvalidParam = 1001 // 请求参数错误
	CodeServerBusy   = 1002 // 服务繁忙
	CodeRateLimit    = 1003 // 请求限流
	CodeNeedLogin    = 2001 // 需要登录
	CodeInvalidToken = 2002 // 令牌无效或过期
	CodeUnauthorized = 2003 // 未授权
	CodeForbidden    = 2004 // 无权限
	CodeNotFound     = 3001 // 资源不存在
	CodeConflict     = 3002 // 资源冲突（重复等）
)

// codeMsgMap 业务状态码到默认消息的映射
// 被 GetMsg 调用，用于 HandleSuccess/HandleError 填充 msg 字段
var codeMsgMap = map[int]string{
	CodeSuccess:      "success",
	CodeInvalidParam: "invalid parameters",
	CodeServerBusy:   "server is busy",
	CodeRateLimit:    "rate limit exceeded",
	CodeNeedLogin:    "need login",
	CodeInvalidToken: "invalid token",
	CodeUnauthorized: "unauthorized",
	CodeForbidden:    "forbidden operation",
	CodeNotFound:     "entity not found",
	CodeConflict:     "entity already exists",
}

// GetMsg 根据业务状态码返回默认消息
// 被 HandleSuccess 和 HandleError 内部调用
func GetMsg(code int) string {
	msg, ok := codeMsgMap[code]
	if !ok {
		return "unknown error"
	}
	return msg
}
