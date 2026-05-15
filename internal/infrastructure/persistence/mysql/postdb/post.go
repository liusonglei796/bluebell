package postdb

import (
	// 模型
	"bluebell/internal/infrastructure/persistence/mysql/model"

	// 领域层
	"bluebell/internal/domain"

	// 错误处理
	"bluebell/internal/domain/entity"

	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// postRepoStruct 帖子数据访问实现
type postRepoStruct struct {
	db *gorm.DB
}

// NewPostRepo 创建 postRepoStruct 实例
func NewPostRepo(db *gorm.DB) domain.PostRepository {
	return &postRepoStruct{db: db}
}

// toModelPost 将领域实体转换为数据库模型
func toModelPost(p *entity.Post) *model.Post {
	if p == nil {
		return nil
	}
	return &model.Post{
		PostID:      p.PostID,
		AuthorID:    p.AuthorID,
		CommunityID: p.CommunityID,
		PostTitle:   p.PostTitle,
		Content:     p.Content,
		Status:      p.Status,
	}
}

// fromModelPost 将数据库模型转换为领域实体
func fromModelPost(m *model.Post) *entity.Post {
	if m == nil {
		return nil
	}
	p := &entity.Post{
		PostID:      m.PostID,
		AuthorID:    m.AuthorID,
		CommunityID: m.CommunityID,
		PostTitle:   m.PostTitle,
		Content:     m.Content,
		Status:      m.Status,
		CreatedAt:   m.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	if m.Author != nil {
		p.Author = &entity.User{
			UserID:   m.Author.UserID,
			UserName: m.Author.UserName,
			Role:     m.Author.Role,
		}
	}

	if m.Community != nil {
		p.Community = &entity.Community{
			ID:            int64(m.Community.ID),
			CommunityName: m.Community.CommunityName,
			Introduction:  m.Community.Introduction,
		}
	}

	return p
}

// CreatePost 创建帖子
func (r *postRepoStruct) CreatePost(ctx context.Context, post *entity.Post) (err error) {
	m := toModelPost(post)
	err = r.db.WithContext(ctx).Create(m).Error
	if err != nil {
		return fmt.Errorf("创建帖子失败: %w", err)
	}
	return nil
}

// GetPostByID 根据帖子ID查询帖子详情（带预加载）
func (r *postRepoStruct) GetPostByID(ctx context.Context, pid int64) (*entity.Post, error) {
	m := new(model.Post)

	err := r.db.WithContext(ctx).Preload("Author").
		Preload("Community").
		Where("post_id = ?", pid).
		Where("status = ?", 1).
		First(m).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询帖子失败: %w", err)
	}
	return fromModelPost(m), nil
}

// GetPostListByIDsWithPreload 根据给定的ID列表查询帖子详情（带预加载）
func (r *postRepoStruct) GetPostListByIDsWithPreload(ctx context.Context, ids []string) (posts []*entity.Post, err error) {
	if len(ids) == 0 {
		return make([]*entity.Post, 0), nil
	}

	var mPosts []*model.Post

	err = r.db.WithContext(ctx).Preload("Author").
		Preload("Community").
		Where("post_id IN ?", ids).
		Where("status = ?", 1).
		Find(&mPosts).Error

	if err != nil {
		return nil, fmt.Errorf("批量查询帖子失败: %w", err)
	}

	// 按照传入的 ids 顺序排列结果
	postMap := make(map[string]*entity.Post, len(mPosts))
	for _, m := range mPosts {
		postMap[m.PostID] = fromModelPost(m)
	}

	orderedPosts := make([]*entity.Post, 0, len(ids))
	for _, id := range ids {
		if post, ok := postMap[id]; ok {
			orderedPosts = append(orderedPosts, post)
		}
	}

	return orderedPosts, nil
}

// DeletePostByAuthor 软删除帖子（带作者验证）
func (r *postRepoStruct) DeletePostByAuthor(ctx context.Context, postID, authorID int64) error {
	result := r.db.WithContext(ctx).Model(&model.Post{}).
		Where("post_id = ?", postID).
		Where("author_id = ?", authorID).
		Where("status = ?", 1).
		Update("status", 0)

	if result.Error != nil {
		return fmt.Errorf("删除帖子失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return entity.ErrNotFound
	}
	return nil
}

// UpdatePostStatus 更新帖子状态（用于审核不通过时隐藏帖子）
func (r *postRepoStruct) UpdatePostStatus(ctx context.Context, postID string, status int8) error {
	result := r.db.WithContext(ctx).Model(&model.Post{}).
		Where("post_id = ?", postID).
		Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("更新帖子状态失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return entity.ErrNotFound
	}
	return nil
}

// DB 返回底层 GORM DB 实例，用于事务操作
func (r *postRepoStruct) DB() *gorm.DB {
	return r.db
}
