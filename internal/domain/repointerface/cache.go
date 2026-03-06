package repointerface

import (
	"context"
	"time"
)

// PostCacheRepository 帖子缓存仓储接口
// 封装帖子在 Redis 中的创建、排序、分页等操作
type PostCacheRepository interface {
	// CreatePost 创建帖子时初始化 Redis 数据（时间排序、分数排序）
	CreatePost(ctx context.Context, postID, communityID int64) error
	// GetPostIDsInOrder 按照指定顺序获取帖子ID列表
	GetPostIDsInOrder(ctx context.Context, orderKey string, page, size int64) ([]string, error)
	// GetCommunityPostIDsInOrder 按社区获取帖子ID列表
	GetCommunityPostIDsInOrder(ctx context.Context, communityID int64, orderKey string, page, size int64) ([]string, error)
}

// VoteCacheRepository 投票缓存仓储接口
type VoteCacheRepository interface {
	// VoteForPost 为帖子投票
	VoteForPost(ctx context.Context, userID, postID, communityID string, value float64) error
	// GetPostsVoteData 批量获取多个帖子的投票数（赞成票数）
	GetPostsVoteData(ctx context.Context, ids []string) ([]int64, error)
	// GetPostVoteData 获取单个帖子的投票数据
	GetPostVoteData(ctx context.Context, postID string) (upVotes, downVotes int64, err error)
	// GetPostScore 获取帖子的当前分数
	GetPostScore(ctx context.Context, postID string) (float64, error)
	// GetPostVoteStatus 获取用户对某个帖子的投票状态
	GetPostVoteStatus(ctx context.Context, userID, postID string) (int8, error)
	// BatchGetPostVoteStatus 批量获取用户对多个帖子的投票状态
	BatchGetPostVoteStatus(ctx context.Context, userID string, postIDs []string) (map[string]int8, error)
}

// UserTokenCacheRepository 用户Token缓存仓储接口
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
