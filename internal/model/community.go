package model

import "gorm.io/gorm"

// Community 社区结构体
// 对应数据库 community 表
type Community struct {
	gorm.Model
	// community_name 唯一索引：并发创建同名社区时由数据库兜底，防止 check-then-insert 竞态产生重复数据
	CommunityName string `gorm:"column:community_name;size:128;not null;uniqueIndex"`
	Introduction  string `gorm:"column:introduction;not null;type:text"`
}

// TableName 自定义表名
func (Community) TableName() string {
	return "community"
}
