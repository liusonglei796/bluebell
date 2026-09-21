package mysql

import (
	"context"
	"fmt"

	"bluebell/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RelationDao 关注关系数据访问对象
type RelationDao struct {
	db *gorm.DB
}

// NewRelationDao 创建关注 DAO 实例
func NewRelationDao(db *gorm.DB) *RelationDao {
	return &RelationDao{db: db}
}

// Follow 关注用户 (UPSERT 幂等写入)
func (d *RelationDao) Follow(ctx context.Context, followerID, followingID int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		relation := &model.UserRelation{
			FollowerID:  followerID,
			FollowingID: followingID,
			Status:      model.RelationStatusFollowing,
		}

		// UPSERT: ON DUPLICATE KEY UPDATE status = 1
		err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "follower_id"}, {Name: "following_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"status", "updated_at"}),
		}).Create(relation).Error
		if err != nil {
			return fmt.Errorf("upsert follow relation failed: %w", err)
		}

		// 严格按 user_id 升序更新计数器，消除并发关注时的 AB-BA 循环死锁 (Deadlock 1213)
		firstID, secondID := followerID, followingID
		firstCol, secondCol := "following_count", "follower_count"
		if firstID > secondID {
			firstID, secondID = followingID, followerID
			firstCol, secondCol = "follower_count", "following_count"
		}

		if err := tx.Model(&model.User{}).Where("user_id = ?", firstID).Update(firstCol, gorm.Expr(firstCol+" + 1")).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("user_id = ?", secondID).Update(secondCol, gorm.Expr(secondCol+" + 1")).Error; err != nil {
			return err
		}

		return nil
	})
}

// Unfollow 取消关注
func (d *RelationDao) Unfollow(ctx context.Context, followerID, followingID int64) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.UserRelation{}).
			Where("follower_id = ? AND following_id = ? AND status = ?", followerID, followingID, model.RelationStatusFollowing).
			Update("status", model.RelationStatusUnfollow)
		if res.Error != nil {
			return fmt.Errorf("update unfollow failed: %w", res.Error)
		}

		if res.RowsAffected > 0 {
			// 严格按 user_id 升序更新计数器，消除死锁
			firstID, secondID := followerID, followingID
			firstCol, secondCol := "following_count", "follower_count"
			if firstID > secondID {
				firstID, secondID = followingID, followerID
				firstCol, secondCol = "follower_count", "following_count"
			}

			if err := tx.Model(&model.User{}).Where("user_id = ? AND "+firstCol+" > 0", firstID).Update(firstCol, gorm.Expr(firstCol+" - 1")).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.User{}).Where("user_id = ? AND "+secondCol+" > 0", secondID).Update(secondCol, gorm.Expr(secondCol+" - 1")).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// IsFollowing 查询是否关注
func (d *RelationDao) IsFollowing(ctx context.Context, followerID, followingID int64) (bool, error) {
	var count int64
	err := d.db.WithContext(ctx).Model(&model.UserRelation{}).
		Where("follower_id = ? AND following_id = ? AND status = ?", followerID, followingID, model.RelationStatusFollowing).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check is following failed: %w", err)
	}
	return count > 0, nil
}

// GetFollowingIDs 获取用户所有关注的人的 ID 列表 (用于 Feed 聚合)
func (d *RelationDao) GetFollowingIDs(ctx context.Context, userID int64) ([]int64, error) {
	var ids []int64
	err := d.db.WithContext(ctx).Model(&model.UserRelation{}).
		Where("follower_id = ? AND status = ?", userID, model.RelationStatusFollowing).
		Pluck("following_id", &ids).Error
	if err != nil {
		return nil, fmt.Errorf("pluck following ids failed: %w", err)
	}
	return ids, nil
}

// GetFollowingList 分页获取我关注的人
func (d *RelationDao) GetFollowingList(ctx context.Context, userID int64, page, size int64) ([]int64, int64, error) {
	var ids []int64
	var total int64

	db := d.db.WithContext(ctx).Model(&model.UserRelation{}).
		Where("follower_id = ? AND status = ?", userID, model.RelationStatusFollowing)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := db.Offset(int(offset)).Limit(int(size)).Order("created_at DESC").Pluck("following_id", &ids).Error; err != nil {
		return nil, 0, err
	}

	return ids, total, nil
}

// GetFollowerList 分页获取我的粉丝
func (d *RelationDao) GetFollowerList(ctx context.Context, userID int64, page, size int64) ([]int64, int64, error) {
	var ids []int64
	var total int64

	db := d.db.WithContext(ctx).Model(&model.UserRelation{}).
		Where("following_id = ? AND status = ?", userID, model.RelationStatusFollowing)

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * size
	if err := db.Offset(int(offset)).Limit(int(size)).Order("created_at DESC").Pluck("follower_id", &ids).Error; err != nil {
		return nil, 0, err
	}

	return ids, total, nil
}
