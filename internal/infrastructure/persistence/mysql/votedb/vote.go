package votedb

import (
	"bluebell/internal/domain"
	"bluebell/internal/infrastructure/persistence/mysql/model"
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type voteRepoStruct struct {
	db *gorm.DB
}

func NewVoteRepo(db *gorm.DB) domain.VoteRepository {
	return &voteRepoStruct{db: db}
}

func (r *voteRepoStruct) SaveVote(ctx context.Context, userID, postID int64, direction int8) error {
	vote := &model.Vote{
		UserID:    userID,
		PostID:    postID,
		Direction: direction,
	}

	// 使用 Upsert (ON DUPLICATE KEY UPDATE)
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "post_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"direction", "updated_at"}),
	}).Create(vote).Error

	if err != nil {
		return fmt.Errorf("保存投票数据失败: %w", err)
	}
	return nil
}
