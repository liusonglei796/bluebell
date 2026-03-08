package votedb

import (
	"bluebell/internal/domain/dbdomain"
	"bluebell/internal/model"
	"bluebell/pkg/errorx"
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type voteRepoStruct struct {
	db *gorm.DB
}

func NewVoteRepo(db *gorm.DB) dbdomain.VoteRepository {
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
		return errorx.Wrap(err, errorx.CodeDBError, "保存投票数据失败")
	}
	return nil
}
