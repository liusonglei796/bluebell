package mysql

import (
	"bluebell/models"
	"context"
)

// ========= 具体的 Repository 实现 =========
// 在这里提供对 logic 层接口的具体实现

// PostRepositoryImpl 是对 logic.PostRepository 接口的 MySQL 实现
type PostRepositoryImpl struct{}

func (pr *PostRepositoryImpl) CreatePost(ctx context.Context, post *models.Post) error {
	return CreatePost(ctx, post)
}

func (pr *PostRepositoryImpl) GetPostByID(ctx context.Context, pid int64) (*models.Post, error) {
	return GetPostByID(ctx, pid)
}

func (pr *PostRepositoryImpl) GetPostListByIDsWithPreload(ctx context.Context, ids []string) ([]*models.Post, error) {
	return GetPostListByIDsWithPreload(ctx, ids)
}

func (pr *PostRepositoryImpl) DeletePost(ctx context.Context, postID int64) error {
	return DeletePost(ctx, postID)
}

func (pr *PostRepositoryImpl) DeletePostByAuthor(ctx context.Context, postID, authorID int64) error {
	return DeletePostByAuthor(ctx, postID, authorID)
}

// CommunityRepositoryImpl 是对 logic.CommunityRepository 接口的 MySQL 实现
type CommunityRepositoryImpl struct{}

func (cr *CommunityRepositoryImpl) GetCommunityList(ctx context.Context) ([]*models.Community, error) {
	return GetCommunityList(ctx)
}

func (cr *CommunityRepositoryImpl) GetCommunityDetailByID(ctx context.Context, id int64) (*models.Community, error) {
	return GetCommunityDetailByID(ctx, id)
}

func (cr *CommunityRepositoryImpl) GetCommunitiesByIDs(ctx context.Context, ids []int64) ([]*models.Community, error) {
	return GetCommunitiesByIDs(ctx, ids)
}

// UserRepositoryImpl 是对 logic.UserRepository 接口的 MySQL 实现
type UserRepositoryImpl struct{}

func (ur *UserRepositoryImpl) CheckUserExist(ctx context.Context, username string) error {
	return CheckUserExist(ctx, username)
}

func (ur *UserRepositoryImpl) InsertUser(ctx context.Context, user *models.User) error {
	return InsertUser(ctx, user)
}

func (ur *UserRepositoryImpl) CheckLogin(ctx context.Context, user *models.User) error {
	return CheckLogin(ctx, user)
}

func (ur *UserRepositoryImpl) GetUserByID(ctx context.Context, uid int64) (*models.User, error) {
	return GetUserByID(ctx, uid)
}

func (ur *UserRepositoryImpl) GetUsersByIDs(ctx context.Context, ids []int64) ([]*models.User, error) {
	return GetUsersByIDs(ctx, ids)
}
