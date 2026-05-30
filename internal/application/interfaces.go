package application

import (
	postreq "bluebell/internal/application/dto/request/post"
	userreq "bluebell/internal/application/dto/request/user"
	votereq "bluebell/internal/application/dto/request/vote"
	bookmarkresp "bluebell/internal/application/dto/response/bookmark"
	postResp "bluebell/internal/application/dto/response/post"
	searchResp "bluebell/internal/application/dto/response/search"
	socialResp "bluebell/internal/application/dto/response/social"
	voteresp "bluebell/internal/application/dto/response/vote"
	"bluebell/internal/domain/entity"

	"context"
)


// ========== Post Service 接口 ==========

// PostService 帖子业务逻辑服务接口
type PostService interface {
	// CreatePost 创建帖子
	CreatePost(ctx context.Context, p *postreq.CreatePostRequest, authorID int64) (postID string, err error)

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
	//发表评论
	RemarkPost(ctx context.Context, req *postreq.RemarkRequest, userID int64) (remarkID uint, err error)
	// GetPostRemarks 获取帖子评论列表
	GetPostRemarks(ctx context.Context, postID int64, replyTo int64) ([]*postResp.RemarkDetail, error)

	// SearchPosts 全文搜索帖子
	SearchPosts(ctx context.Context, keyword string, page, pageSize int) (*searchResp.SearchResponse, error)
}

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

// ========== Social Service 接口 ==========

// SocialService 社交业务逻辑服务接口
type SocialService interface {
	// GetProfile 获取用户资料
	GetProfile(ctx context.Context, userID, currentUserID int64) (*socialResp.ProfileResponse, error)
	// FollowUser 关注用户
	FollowUser(ctx context.Context, followerID, followingID int64) error
	// UnfollowUser 取消关注
	UnfollowUser(ctx context.Context, followerID, followingID int64) error
	// GetActivities 获取用户动态
	GetActivities(ctx context.Context, userID int64, page, size int) ([]*socialResp.ActivityResponse, error)
}

// BookmarkService 收藏帖子业务逻辑服务接口
type BookmarkService interface {
	// CreateBookmark 收藏帖子
	CreateBookmark(ctx context.Context, userID, postID int64) error
	// DeleteBookmark 取消收藏
	DeleteBookmark(ctx context.Context, userID, postID int64) error
	// GetUserBookmarks 获取用户收藏列表
	GetUserBookmarks(ctx context.Context, userID int64, page, size int) (*bookmarkresp.BookmarkListResponse, error)
	// IsBookmarked 检查帖子是否被收藏
	IsBookmarked(ctx context.Context, userID, postID int64) (*bookmarkresp.BookmarkStatusResponse, error)
}
