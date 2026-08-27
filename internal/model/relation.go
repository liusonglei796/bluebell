package model

import "time"

// 关注关系状态
const (
	RelationStatusFollowing = 1 // 正常关注
	RelationStatusUnfollow  = 0 // 已取消关注
)

// UserRelation 用户关注关系模型
type UserRelation struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	FollowerID  int64     `gorm:"column:follower_id;not null;uniqueIndex:uk_follower_following,priority:1;index:idx_follower_list,priority:1" json:"follower_id"`
	FollowingID int64     `gorm:"column:following_id;not null;uniqueIndex:uk_follower_following,priority:2;index:idx_following_list,priority:1" json:"following_id"`
	Status      int8      `gorm:"column:status;not null;default:1;index:idx_follower_list,priority:2;index:idx_following_list,priority:2" json:"status"`
	CreatedAt   time.Time `gorm:"column:created_at;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3);index:idx_follower_list,priority:3;index:idx_following_list,priority:3" json:"created_at"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)" json:"updated_at"`
}

func (UserRelation) TableName() string {
	return "user_relation"
}
