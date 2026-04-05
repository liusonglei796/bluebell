package postdb

import (
	"bluebell/internal/model"
	"bluebell/pkg/errorx"
	"context"

	"gorm.io/gorm"
)

// CreateRemark 实现 dbdomain.RemarkRepository 接口
func (r *postRepoStruct) CreateRemark(ctx context.Context, remark *model.Remark) error {
	if err := r.db.WithContext(ctx).Create(remark).Error; err != nil {
		return errorx.Wrap(err, errorx.CodeDBError, "create remark failed")
	}
	return nil
}

// GetRemarksByPostID 获取帖子的评论列表
func (r *postRepoStruct) GetRemarksByPostID(ctx context.Context, postID int64) ([]*model.Remark, error) {
	var remarks []*model.Remark
	if err := r.db.WithContext(ctx).
		Where("post_id = ?", postID).
		Preload("Author"). // 预加载作者，以便获取作者名
		Order("created_at DESC").
		Find(&remarks).Error; err != nil {
		return nil, errorx.Wrap(err, errorx.CodeDBError, "get remarks failed")
	}
	return remarks, nil
}

// DeleteRemarkByID 根据评论ID删除评论（软删除，利用 gorm.Model 的 DeletedAt 字段）
func (r *postRepoStruct) DeleteRemarkByID(ctx context.Context, remarkID uint) error {
	if err := r.db.WithContext(ctx).Delete(&model.Remark{}, remarkID).Error; err != nil {
		return errorx.Wrap(err, errorx.CodeDBError, "delete remark failed")
	}
	return nil
}

// NewRemarkRepo 返回 RemarkRepository 接口实现
// 实际上 postRepoStruct 已经实现了该接口，为了保持一致性这里可以返回它
func NewRemarkRepo(db *gorm.DB) *postRepoStruct {
	return &postRepoStruct{db: db}
}
