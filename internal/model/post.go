package model

import (
	"time"

	"gorm.io/gorm"
)

// Post 内存对齐优化建议：把相同类型的字段放在一起，宽字段（如 int64, string）放在前面
// 这个结构体是对数据库表结构的直接映射，使用 GORM ORM
type Post struct {
	// 8 字节字段 (int64)
	ID          int64 `json:"id,string" gorm:"column:post_id;primaryKey"`
	AuthorID    int64 `json:"author_id,string" gorm:"column:author_id;index;not null"`
	CommunityID int64 `json:"community_id,string" gorm:"column:community_id;index;not null"`

	// 4 字节字段 (int32)
	Status int32 `json:"status" gorm:"column:status;default:1"`

	// 16 字节字段 (string)
	Title   string `json:"title" gorm:"column:title;size:128;not null"`
	Content string `json:"content" gorm:"column:content;type:text;not null"`

	// Time 类型
	CreateTime time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`

	// GORM 关联字段 (用于 Preload 预加载，解决 N+1 问题)
	// 为什么添加：使用 GORM 的 Preload 功能可以自动批量查询关联数据
	// gorm:"-" 表示不映射到数据库字段，只用于内存中的关联
	Author    *User      `json:"author,omitempty" gorm:"foreignKey:AuthorID;references:UserID"`
	Community *Community `json:"community,omitempty" gorm:"foreignKey:CommunityID;references:CommunityID"`
}

// TableName 自定义表名
// 为什么：GORM 默认使用复数形式表名(posts)，需要显式指定为 post
func (Post) TableName() string {
	return "post"
}

// BeforeCreate GORM 钩子
func (p *Post) BeforeCreate(tx *gorm.DB) error {
	return nil
}
