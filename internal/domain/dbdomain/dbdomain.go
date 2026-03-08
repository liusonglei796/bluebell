package dbdomain

import (
	"bluebell/internal/model"
	"context"
)

// PostRepository 定义了关于帖子数据的访问接口
type PostRepository interface {
	CreatePost(ctx context.Context, post *model.Post) error
	GetPostByID(ctx context.Context, pid int64) (*model.Post, error)
	GetPostListByIDsWithPreload(ctx context.Context, ids []string) ([]*model.Post, error)
	DeletePostByAuthor(ctx context.Context, postID, authorID int64) error
}

// CommunityRepository 定义了关于社区数据的访问接口
type CommunityRepository interface {
	GetCommunityList(ctx context.Context) ([]*model.Community, error)
	GetCommunityDetailByID(ctx context.Context, id int64) (*model.Community, error)
	CreateCommunity(ctx context.Context, community *model.Community) error
}

// UserRepository 定义了关于用户数据的访问接口
type UserRepository interface {
	CheckUserExist(ctx context.Context, username string) error
	InsertUser(ctx context.Context, user *model.User) error
	VerifyUser(ctx context.Context, user *model.User) error
	CheckUserExistsByID(ctx context.Context, uid int64) (*model.User, error)
	GetUsersByIDs(ctx context.Context, ids []int64) ([]*model.User, error)
	GetUserRoleByID(ctx context.Context, uid int64) (int, error)
}

// VoteRepository 定义了关于投票数据的访问接口
type VoteRepository interface {
	SaveVote(ctx context.Context, userID, postID int64, direction int8) error
}

// RemarkRepository 定义了关于评论数据的访问接口
type RemarkRepository interface {
	CreateRemark(ctx context.Context, remark *model.Remark) error
	GetRemarksByPostID(ctx context.Context, postID int64) ([]*model.Remark, error)
}
