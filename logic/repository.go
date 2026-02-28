package logic

import (
	"bluebell/models"
	"context"
)

// PostRepository 定义了关于帖子数据的访问接口
type PostRepository interface {
	CreatePost(ctx context.Context, post *models.Post) error
	GetPostByID(ctx context.Context, pid int64) (*models.Post, error)
	GetPostListByIDsWithPreload(ctx context.Context, ids []string) ([]*models.Post, error)
	DeletePost(ctx context.Context, postID int64) error
	DeletePostByAuthor(ctx context.Context, postID, authorID int64) error
}

// CommunityRepository 定义了关于社区数据的访问接口
type CommunityRepository interface {
	GetCommunityList(ctx context.Context) ([]*models.Community, error)
	GetCommunityDetailByID(ctx context.Context, id int64) (*models.Community, error)
	GetCommunitiesByIDs(ctx context.Context, ids []int64) ([]*models.Community, error)
}

// UserRepository 定义了关于用户数据的访问接口
type UserRepository interface {
	CheckUserExist(ctx context.Context, username string) error
	InsertUser(ctx context.Context, user *models.User) error
	CheckLogin(ctx context.Context, user *models.User) error
	GetUserByID(ctx context.Context, uid int64) (*models.User, error)
	GetUsersByIDs(ctx context.Context, ids []int64) ([]*models.User, error)
}
