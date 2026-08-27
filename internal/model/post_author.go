package model

import "gorm.io/gorm"

// PostAuthor 帖子-作者关联表（多对多中间表）
type PostAuthor struct {
	gorm.Model
	PostID string `gorm:"column:post_id;not null;uniqueIndex:idx_post_user"`
	UserID int64  `gorm:"column:user_id;not null;uniqueIndex:idx_post_user"`
}

func (PostAuthor) TableName() string {
	return "post_author"
}
