package logic

import (
	"bluebell/models"
	"context"
)

// PostRepository 定义了关于帖子数据的访问接口
// 通过接口隔离，Logic 层不再直接依赖具体的 DAO 实现 (如 MySQL)
// 这使得我们可以轻松地为了单元测试 Mock 数据层，或者在未来将存储切换到其他数据库
type PostRepository interface {
	// CreatePost 创建帖子
	CreatePost(ctx context.Context, post *models.Post) error

	// GetPostByID 根据帖子ID查询帖子详情
	GetPostByID(ctx context.Context, pid int64) (*models.Post, error)

	// GetPostListByIDsWithPreload 根据给定的ID列表查询帖子详情（预加载作者和社区信息）
	GetPostListByIDsWithPreload(ctx context.Context, ids []string) ([]*models.Post, error)

	// DeletePost 软删除帖子
	DeletePost(ctx context.Context, postID int64) error

	// DeletePostByAuthor 软删除帖子（带作者验证）
	DeletePostByAuthor(ctx context.Context, postID, authorID int64) error
}

// CommunityRepository 定义了关于社区数据的访问接口
type CommunityRepository interface {
	// GetCommunityList 查询社区列表
	GetCommunityList(ctx context.Context) ([]*models.Community, error)

	// GetCommunityDetailByID 根据ID查询社区详情
	GetCommunityDetailByID(ctx context.Context, id int64) (*models.Community, error)

	// GetCommunitiesByIDs 根据社区ID列表批量获取社区信息
	GetCommunitiesByIDs(ctx context.Context, ids []int64) ([]*models.Community, error)
}

// UserRepository 定义了关于用户数据的访问接口
type UserRepository interface {
	// CheckUserExist 检查指定用户名的用户是否存在
	CheckUserExist(ctx context.Context, username string) error

	// InsertUser 插入新用户
	InsertUser(ctx context.Context, user *models.User) error

	// CheckLogin 登录验证
	CheckLogin(ctx context.Context, user *models.User) error

	// GetUserByID 根据用户ID查询用户信息
	GetUserByID(ctx context.Context, uid int64) (*models.User, error)

	// GetUsersByIDs 根据用户ID列表批量获取用户信息
	GetUsersByIDs(ctx context.Context, ids []int64) ([]*models.User, error)
}

// ====== 依赖注入的简易实现 ======
// 为了在这个循序渐进的优化中不破坏太多代码结构，我们先保留单例模式的雏形
// 但将原本直接调用 mysql 包的方法，改为调用接口实例。
// 这里的实现默认指向 mysql 的实现，可以通过 Init 方法或者针对测试修改这些变量。

var (
	postRepo      PostRepository
	communityRepo CommunityRepository
	userRepo      UserRepository
)

// SetRepositories 允许在系统初始化或单元测试中注入不同的实现
func SetRepositories(pr PostRepository, cr CommunityRepository, ur UserRepository) {
	postRepo = pr
	communityRepo = cr
	userRepo = ur
}
