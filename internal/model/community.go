package model

import "gorm.io/gorm"

// Community 社区结构体
// 为什么：使用 GORM ORM 映射数据库 community 表
type Community struct {
	gorm.Model
	CommunityName string `gorm:"column:community_name;not null;size:255"`
	Introduction  string `gorm:"column:introduction;not null;type:text"`
}

// TableName 自定义表名
// 为什么：GORM 默认使用复数形式表名(communities)，需要显式指定为 community
func (Community) TableName() string {
	return "community"
}
