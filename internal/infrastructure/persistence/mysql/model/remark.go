package model

import (
	"gorm.io/gorm"
)

// Remark 评论模型
type Remark struct {
	Author *User `gorm:"foreignKey:AuthorID;references:UserID"`
	gorm.Model
	Content  string `gorm:"column:content;type:text;not null"`
	PostID   int64  `gorm:"column:post_id;not null;index"`
	AuthorID int64  `gorm:"column:author_id;not null"`
	ReplyTo  int64  `gorm:"column:reply_to;not null;default:0"`
}

// TableName 自定义表名
func (Remark) TableName() string {
	return "remark"
}
