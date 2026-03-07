package svcdomain

import (
	// DTO
	"bluebell/internal/dto/request/post"
	"bluebell/internal/dto/request/user"
	"bluebell/internal/dto/response/community"
	"bluebell/internal/dto/response/post"

	"context"
)

// ========== Community Service 接口 ==========

// CommunityService 社区业务逻辑服务接口
type CommunityService interface {
	// GetCommunityList 获取社区列表
	GetCommunityList(ctx context.Context) ([]*communityResp.Response, error)
	// GetCommunityDetail 根据社区ID获取社区详情
	GetCommunityDetail(ctx context.Context, communityID int64) (*communityResp.Response, error)
}

// ========== Post Service 接口 ==========

// PostService 帖子业务逻辑服务接口
type PostService interface {
	// CreatePost 创建帖子
	CreatePost(ctx context.Context, p *postreq.CreatePostRequest, authorID int64) error

	// GetPostByID 查询单个帖子详情
	GetPostByID(ctx context.Context, pid int64) (*postResp.DetailResponse, error)

	// GetPostList 获取帖子列表
	GetPostList(ctx context.Context, p *postreq.PostListRequest) ([]*postResp.DetailResponse, error)

	// GetCommunityPostList 根据社区ID获取帖子列表
	GetCommunityPostList(ctx context.Context, p *postreq.PostListRequest) ([]*postResp.DetailResponse, error)

	// DeletePost 删除帖子（软删除）
	DeletePost(ctx context.Context, postID int64, userID int64) error

	// VoteForPost 为帖子投票
	VoteForPost(ctx context.Context, userID int64, p *postreq.VoteRequest) error
}

// ========== User Service 接口 ==========

// UserService 用户业务逻辑服务接口
type UserService interface {
	// SignUp 处理用户注册业务逻辑
	SignUp(ctx context.Context, p *userreq.SignUpRequest) error

	// Login 处理用户登录业务逻辑，返回访问令牌和刷新令牌
	Login(ctx context.Context, p *userreq.LoginRequest) (accessToken, refreshToken string, err error)

	// RefreshToken 使用刷新令牌获取新的访问令牌
	RefreshToken(ctx context.Context, p *userreq.RefreshTokenRequest) (newAccessToken, newRefreshToken string, err error)
}
