package postdb

import (
	// 模型
	"bluebell/internal/model"

	// 领域层
	"bluebell/internal/domain/dbdomain"

	// 错误处理
	"bluebell/pkg/errorx"

	"context"
	"errors"

	"gorm.io/gorm"
)

// postRepoStruct 帖子数据访问实现
type postRepoStruct struct {
	db *gorm.DB
}

// NewPostRepo 创建 postRepoStruct 实例
func NewPostRepo(db *gorm.DB) dbdomain.PostRepository {
	return &postRepoStruct{db: db}
}

// CreatePost 创建帖子
func (r *postRepoStruct) CreatePost(ctx context.Context, post *model.Post) (err error) {
	err = r.db.WithContext(ctx).Create(post).Error
	if err != nil {
		return errorx.Wrap(err, errorx.CodeDBError, "创建帖子失败")
	}
	return nil
}

// GetPostByID 根据帖子ID查询帖子详情（带预加载）
// 使用 Preload 自动加载关联的作者和社区信息，避免 N+1 查询问题
// 自动过滤已删除帖子（status = 0）
func (r *postRepoStruct) GetPostByID(ctx context.Context, pid int64) (post *model.Post, err error) {
	post = new(model.Post)

	err = r.db.WithContext(ctx).Preload("Author").
		Preload("Community").
		Where("post_id = ?", pid).
		Where("status = ?", 1).
		First(post).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, errorx.Wrap(err, errorx.CodeDBError, "查询帖子失败")
	}
	return
}

// GetPostListByIDsWithPreload 根据给定的ID列表查询帖子详情（带预加载）
// 自动过滤已删除帖子（status = 0）
func (r *postRepoStruct) GetPostListByIDsWithPreload(ctx context.Context, ids []string) (posts []*model.Post, err error) {
	if len(ids) == 0 {
		return make([]*model.Post, 0), nil
	}

	posts = make([]*model.Post, 0, len(ids))

	err = r.db.WithContext(ctx).Preload("Author").
		Preload("Community").
		Where("post_id IN ?", ids).
		Where("status = ?", 1).
		Find(&posts).Error

	if err != nil {
		return nil, errorx.Wrap(err, errorx.CodeDBError, "批量查询帖子失败")
	}

	// 按照传入的 ids 顺序排列结果
	postMap := make(map[string]*model.Post, len(posts))
	for _, post := range posts {
		postMap[post.PostID] = post
	}

	orderedPosts := make([]*model.Post, 0, len(ids))
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
		return errorx.Wrap(result.Error, errorx.CodeDBError, "删除帖子失败")
	}
	if result.RowsAffected == 0 {
		return errorx.ErrNotFound
	}
	return nil
}

// UpdatePostStatus 更新帖子状态（用于审核不通过时隐藏帖子）
func (r *postRepoStruct) UpdatePostStatus(ctx context.Context, postID string, status int8) error {
	result := r.db.WithContext(ctx).Model(&model.Post{}).
		Where("post_id = ?", postID).
		Update("status", status)

	if result.Error != nil {
		return errorx.Wrap(result.Error, errorx.CodeDBError, "更新帖子状态失败")
	}
	if result.RowsAffected == 0 {
		return errorx.ErrNotFound
	}
	return nil
}
