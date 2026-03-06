package serviceinterface

import (
	"bluebell/internal/dto/request"
	"context"
)

// UserService 用户业务逻辑服务接口
type UserService interface {
	// SignUp 处理用户注册业务逻辑
	SignUp(ctx context.Context, p *request.SignUpRequest) error

	// Login 处理用户登录业务逻辑，返回访问令牌和刷新令牌
	Login(ctx context.Context, p *request.LoginRequest) (accessToken, refreshToken string, err error)

	// RefreshToken 使用刷新令牌获取新的访问令牌
	RefreshToken(ctx context.Context, accessToken, refreshToken string) (newAccessToken, newRefreshToken string, err error)
}
