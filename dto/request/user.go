package request

// SignUpRequest 注册请求参数
// 为什么：定义前端传递的 JSON 数据结构，使用 binding 标签进行参数校验
type SignUpRequest struct {
	// binding:"required" 表示该字段必填，如果为空 Gin 会报错
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	// re_password 用于确认密码，业务逻辑中会校验两次密码是否一致
	RePassword string `json:"re_password" binding:"required"`
}

// LoginRequest 登录请求参数
// 为什么：分离注册和登录的参数结构，虽然字段相似，但业务含义不同，解耦更灵活
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
