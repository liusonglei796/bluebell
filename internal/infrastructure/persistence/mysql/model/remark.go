package model

import (
	"gorm.io/gorm"
)

// Remark 评论模型
type Remark struct {
	gorm.Model
	PostID   int64  `gorm:"column:post_id;not null;index"`
	Content  string `gorm:"column:content;type:text;not null"`
	AuthorID int64  `gorm:"column:author_id;not null"`
	Author   *User  `gorm:"foreignKey:AuthorID;references:UserID"`
}

// TableName 自定义表名
func (Remark) TableName() string {
	return "remark"
}
