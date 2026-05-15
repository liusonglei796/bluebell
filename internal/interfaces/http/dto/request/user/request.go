package userreq

// SignUpRequest 注册请求参数
type SignUpRequest struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	RePassword string `json:"re_password" binding:"required,eqfield=Password"`
}

// LoginRequest 登录请求参数
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest 刷新Token请求参数
type RefreshTokenRequest struct {
	Authorization string `header:"Authorization" binding:"required"`
	RefreshToken  string `form:"refresh_token" binding:"required"`
}
