// Package domain 提供领域层仓储接口定义
//
// DDD: 仓储接口属于领域层，实现属于基础设施层
// 通过依赖倒置原则，领域层定义接口，基础设施层提供实现
package domain

import (
	"bluebell/internal/domain/entity"
	"context"
	"time"
)

// ========== 缓存层仓储接口 ==========

// PostCacheRepository 帖子缓存仓储接口（Redis）
type PostCacheRepository interface {
	// CreatePost 创建帖子时初始化 Redis 数据（时间排序、分数排序）
	CreatePost(ctx context.Context, postID, communityID int64) error
	// GetPostIDsInOrder 按照指定顺序获取帖子ID列表
	GetPostIDsInOrder(ctx context.Context, orderKey string, page, size int64) ([]string, error)
	// GetCommunityPostIDsInOrder 按社区获取帖子ID列表
	GetCommunityPostIDsInOrder(ctx context.Context, communityID int64, orderKey string, page, size int64) ([]string, error)
	// VoteForPost 为帖子投票
	VoteForPost(ctx context.Context, userID, postID, communityID string, value float64) error
	// BatchVoteForPost 批量为帖子投票 (用于高并发聚合)
	BatchVoteForPost(ctx context.Context, votes map[string]int8) error
	// GetPostsVoteData 批量获取多个帖子的投票数（赞成票数）
	GetPostsVoteData(ctx context.Context, ids []string) ([]int64, error)
	// DeletePost 删除帖子时清理 Redis 缓存（ZSet、Hash、投票记录）
	DeletePost(ctx context.Context, postID, communityID int64) error
	// GetPostCommunityID 从 Redis 缓存中获取帖子的社区 ID
	GetPostCommunityID(ctx context.Context, postID int64) (int64, error)
}

// UserTokenCacheRepository 用户 Token 缓存仓储接口（Redis）
type UserTokenCacheRepository interface {
	// SetUserToken 存储用户的 Access Token 和 Refresh Token
	SetUserToken(ctx context.Context, userID int64, aToken, rToken string, aExp, rExp time.Duration) error
	// GetUserAccessToken 获取用户的 Access Token
	GetUserAccessToken(ctx context.Context, userID int64) (string, error)
	// GetUserRefreshToken 获取用户的 Refresh Token
	GetUserRefreshToken(ctx context.Context, userID int64) (string, error)
	// DeleteUserToken 删除用户的 Token (用于登出)
	DeleteUserToken(ctx context.Context, userID int64) error
}

// ========== 数据库层仓储接口 ==========

// PostRepository 帖子数据库仓储接口（MySQL）
type PostRepository interface {
	CreatePost(ctx context.Context, post *entity.Post) error
	GetPostByID(ctx context.Context, pid int64) (*entity.Post, error)
	GetPostListByIDsWithPreload(ctx context.Context, ids []string) ([]*entity.Post, error)
	DeletePostByAuthor(ctx context.Context, postID, authorID int64) error
}

// CommunityRepository 社区数据库仓储接口
type CommunityRepository interface {
	GetCommunityList(ctx context.Context) ([]*entity.Community, error)
	GetCommunityDetailByID(ctx context.Context, id int64) (*entity.Community, error)
	CreateCommunity(ctx context.Context, community *entity.Community) error
}

// UserRepository 用户数据库仓储接口
type UserRepository interface {
	CheckUserExist(ctx context.Context, username string) error
	CreateUser(ctx context.Context, user *entity.User) error
	VerifyUser(ctx context.Context, user *entity.User) error
	GetUserByID(ctx context.Context, uid int64) (*entity.User, error)
	GetUsersByIDs(ctx context.Context, ids []int64) ([]*entity.User, error)
	GetUserRoleByID(ctx context.Context, uid int64) (int, error)
	GetUserByName(ctx context.Context, username string) (*entity.User, error)
}

// VoteRepository 投票数据库仓储接口
type VoteRepository interface {
	SaveVote(ctx context.Context, userID, postID int64, direction int8) error
}

// RemarkRepository 评论数据库仓储接口
type RemarkRepository interface {
	CreateRemark(ctx context.Context, remark *entity.Remark) error
	GetRemarksByPostID(ctx context.Context, postID int64) ([]*entity.Remark, error)
	DeleteRemarkByID(ctx context.Context, remarkID uint) error
	DeleteRemarksByPostID(ctx context.Context, postID int64) error
}
