package postdb

import (
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/persistence/mysql/model"
	"context"
	"fmt"

	"gorm.io/gorm"
)

// toModelRemark 将领域实体转换为数据库模型
func toModelRemark(r *entity.Remark) *model.Remark {
	if r == nil {
		return nil
	}
	return &model.Remark{
		Model:    gorm.Model{ID: r.ID},
		PostID:   r.PostID,
		Content:  r.Content,
		AuthorID: r.AuthorID,
	}
}

// fromModelRemark 将数据库模型转换为领域实体
func fromModelRemark(m *model.Remark) *entity.Remark {
	if m == nil {
		return nil
	}
	r := &entity.Remark{
		ID:        m.ID,
		PostID:    m.PostID,
		Content:   m.Content,
		AuthorID:  m.AuthorID,
		CreatedAt: m.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	if m.Author != nil {
		r.Author = &entity.User{
			UserID:   m.Author.UserID,
			UserName: m.Author.UserName,
			Role:     m.Author.Role,
		}
	}
	return r
}

// CreateRemark 实现 dbdomain.RemarkRepository 接口
func (r *postRepoStruct) CreateRemark(ctx context.Context, remark *entity.Remark) error {
	m := toModelRemark(remark)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("create remark failed: %w", err)
	}
	remark.ID = m.ID
	return nil
}

// GetRemarksByPostID 获取帖子的评论列表
func (r *postRepoStruct) GetRemarksByPostID(ctx context.Context, postID int64) ([]*entity.Remark, error) {
	var mRemarks []*model.Remark
	if err := r.db.WithContext(ctx).
		Where("post_id = ?", postID).
		Preload("Author"). // 预加载作者，以便获取作者名
		Order("created_at DESC").
		Find(&mRemarks).Error; err != nil {
		return nil, fmt.Errorf("get remarks failed: %w", err)
	}

	remarks := make([]*entity.Remark, 0, len(mRemarks))
	for _, m := range mRemarks {
		remarks = append(remarks, fromModelRemark(m))
	}
	return remarks, nil
}

// DeleteRemarkByID 根据评论ID删除评论（软删除，利用 gorm.Model 的 DeletedAt 字段）
func (r *postRepoStruct) DeleteRemarkByID(ctx context.Context, remarkID uint) error {
	if err := r.db.WithContext(ctx).Delete(&model.Remark{}, remarkID).Error; err != nil {
		return fmt.Errorf("delete remark failed: %w", err)
	}
	return nil
}

// DeleteRemarksByPostID 删除指定帖子的所有评论（用于级联删除）
func (r *postRepoStruct) DeleteRemarksByPostID(ctx context.Context, postID int64) error {
	if err := r.db.WithContext(ctx).Where("post_id = ?", postID).Delete(&model.Remark{}).Error; err != nil {
		return fmt.Errorf("delete remarks by post_id failed: %w", err)
	}
	return nil
}

// NewRemarkRepo 返回 RemarkRepository 接口实现
// 实际上 postRepoStruct 已经实现了该接口，为了保持一致性这里可以返回它
func NewRemarkRepo(db *gorm.DB) *postRepoStruct {
	return &postRepoStruct{db: db}
}
