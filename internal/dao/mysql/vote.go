package mysql

import (
	"context"
	"fmt"

	"bluebell/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// VoteDao 投票数据访问对象
type VoteDao struct {
	db *gorm.DB
}

// NewVoteDao 创建投票 DAO 实例
func NewVoteDao(db *gorm.DB) *VoteDao {
	return &VoteDao{db: db}
}

// SaveVote 保存投票（UPSERT）
func (d *VoteDao) SaveVote(ctx context.Context, userID, postID int64, direction int8) error {
	vote := &model.Vote{
		UserID:    userID,
		PostID:    postID,
		Direction: direction,
	}

	// 使用 Upsert (ON DUPLICATE KEY UPDATE)
	err := d.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "post_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"direction", "updated_at"}),
	}).Create(vote).Error

	if err != nil {
		return fmt.Errorf("保存投票数据失败: %w", err)
	}
	return nil
}
