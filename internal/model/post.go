package model

import (
	"gorm.io/gorm"
)

// PostStatus 帖子状态
const (
	PostStatusPublished = 1 // 已发布
)

// Post 内存对齐优化建议：把相同类型的字段放在一起，宽字段（如 int64, string）放在前面
// 这个结构体是对数据库表结构的直接映射，使用 GORM ORM
type Post struct {
	gorm.Model
	PostID      string     `gorm:"column:post_id;not null;primaryKey;size:255"`
	AuthorID    int64      `gorm:"column:author_id"`
	CommunityID int64      `gorm:"column:community_id"`
	PostTitle   string     `gorm:"column:post_title;not null;type:text"`
	Author      *User      `gorm:"foreignKey:AuthorID;references:UserID"`
	Community   *Community `gorm:"foreignKey:CommunityID;references:ID"`
	Content     string     `gorm:"column:content;type:text;not null"`
	Status      int8       `gorm:"column:status"`
}

// TableName 自定义表名
// 为什么：GORM 默认使用复数形式表名(posts)，需要显式指定为 post
func (Post) TableName() string {
	return "post"
}
