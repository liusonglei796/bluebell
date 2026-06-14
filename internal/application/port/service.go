package port

import (
	postreq "bluebell/internal/application/dto/request/post"
	postResp "bluebell/internal/application/dto/response/post"
	searchResp "bluebell/internal/application/dto/response/search"
	userreq "bluebell/internal/application/dto/request/user"
	communityResp "bluebell/internal/application/dto/response/community"
	socialResp "bluebell/internal/application/dto/response/social"
	bookmarkResp "bluebell/internal/application/dto/response/bookmark"
	"bluebell/internal/domain/entity"
	"context"
)

// ========== 入站端口接口（Inbound Ports） ==========
//
// 这些接口定义了应用层对外暴露的用例（Use Cases）。
// 接口层（Handler / Controller）通过依赖这些接口来调用应用服务，
// 而不是直接持有具体的 *XxxService 结构体。
// 这使得：
// 1. Handler 可以通过 Mock 接口进行单元测试
// 2. 应用服务的实现可以被替换而不影响 Handler
// 3. 依赖方向正确：interfaces → application/port

// PostService 帖子相关用例接口
type PostService interface {
	CreatePost(ctx context.Context, p *postreq.CreatePostRequest, authorID int64) (string, error)
	GetPostByID(ctx context.Context, pid int64) (*postResp.DetailResponse, error)
	GetPostList(ctx context.Context, p *postreq.PostListRequest) ([]*postResp.DetailResponse, error)
	GetCommunityPostList(ctx context.Context, p *postreq.PostListRequest) ([]*postResp.DetailResponse, error)
	DeletePost(ctx context.Context, postID int64, userID int64) error
	VoteForPost(ctx context.Context, userID int64, p *postreq.VoteRequest) error
	RemarkPost(ctx context.Context, req *postreq.RemarkRequest, userID int64) (uint, error)
	GetPostRemarks(ctx context.Context, postID int64, replyTo int64) ([]*postResp.RemarkDetail, error)
	SearchPosts(ctx context.Context, keyword string, page, pageSize int) (*searchResp.SearchResponse, error)
}

// UserService 用户相关用例接口
type UserService interface {
	SignUp(ctx context.Context, p *userreq.SignUpRequest) error
	Login(ctx context.Context, p *userreq.LoginRequest) (string, string, error)
	RefreshToken(ctx context.Context, p *userreq.RefreshTokenRequest) (string, string, error)
	Logout(ctx context.Context, userID int64) error
	SocialLogin(ctx context.Context, githubID, username, email, avatarURL string) (string, string, error)
	GetUserByUsername(ctx context.Context, username string) (*entity.User, error)
	UploadAvatar(ctx context.Context, userID int64, avatarURL string) error
	GetAvatarURL(ctx context.Context, userID int64) (string, error)
}

// CommunityService 社区相关用例接口
type CommunityService interface {
	GetCommunityList(ctx context.Context) ([]*communityResp.Response, error)
	GetCommunityDetail(ctx context.Context, id int64) (*communityResp.Response, error)
	CreateCommunity(ctx context.Context, name, introduction string, userID int64) error
}

// SocialService 社交相关用例接口
type SocialService interface {
	GetProfile(ctx context.Context, userID, currentUserID int64) (*socialResp.ProfileResponse, error)
	FollowUser(ctx context.Context, followerID, followingID int64) error
	UnfollowUser(ctx context.Context, followerID, followingID int64) error
	GetActivities(ctx context.Context, userID int64, page, size int) ([]*socialResp.ActivityResponse, error)
}

// BookmarkService 收藏相关用例接口
type BookmarkService interface {
	CreateBookmark(ctx context.Context, userID, postID int64) error
	DeleteBookmark(ctx context.Context, userID, postID int64) error
	GetUserBookmarks(ctx context.Context, userID int64, page, size int) (*bookmarkResp.BookmarkListResponse, error)
	IsBookmarked(ctx context.Context, userID, postID int64) (*bookmarkResp.BookmarkStatusResponse, error)
}
