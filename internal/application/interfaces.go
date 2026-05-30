package application

import (
	postreq "bluebell/internal/application/dto/request/post"
	userreq "bluebell/internal/application/dto/request/user"
	votereq "bluebell/internal/application/dto/request/vote"
	voteresp "bluebell/internal/application/dto/response/vote"
	"bluebell/internal/domain/entity"

	"context"
)



// ========== User Service 接口 ==========

// UserService 用户业务逻辑服务接口
type UserService interface {
	// SignUp 处理用户注册业务逻辑
	SignUp(ctx context.Context, p *userreq.SignUpRequest) error

	// Login 处理用户登录业务逻辑，返回访问令牌和刷新令牌
	Login(ctx context.Context, p *userreq.LoginRequest) (accessToken, refreshToken string, err error)

	// SocialLogin 处理社交登录业务逻辑
	SocialLogin(ctx context.Context, githubID, username, email, avatarURL string) (accessToken, refreshToken string, err error)

	// RefreshToken 使用刷新令牌获取新的访问令牌
	RefreshToken(ctx context.Context, p *userreq.RefreshTokenRequest) (newAccessToken, newRefreshToken string, err error)

	// GetUserByUsername 根据用户名获取用户信息
	GetUserByUsername(ctx context.Context, username string) (*entity.User, error)

	// UploadAvatar 上传头像，更新 user_profile 表的 avatar_url
	UploadAvatar(ctx context.Context, userID int64, avatarURL string) error

	// GetAvatarURL 获取用户头像 URL
	GetAvatarURL(ctx context.Context, userID int64) (string, error)

	// Logout 用户登出，清除 Redis 中存储的 Token
	Logout(ctx context.Context, userID int64) error
}

// ========== Vote Service 接口 ==========

// VoteService 投票与排行榜业务逻辑服务接口
type VoteService interface {
	// VoteForPost 为帖子投票
	VoteForPost(ctx context.Context, userID int64, p *postreq.VoteRequest) error
	// GetLeaderboard 获取排行榜
	GetLeaderboard(ctx context.Context, p *votereq.LeaderboardRequest) (*voteresp.LeaderboardResponse, error)
}

