package model

import "time"

type Bookmark struct {
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	UserID    int64     `gorm:"column:user_id;index:idx_user_post,unique"`
	PostID    int64     `gorm:"column:post_id;index:idx_user_post,unique"`
}

func (Bookmark) TableName() string {
	return "bookmark"
}
