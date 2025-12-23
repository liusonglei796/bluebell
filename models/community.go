package models

import "time"

// CommunityDetail 社区详情结构体
// 为什么：使用 GORM ORM 映射数据库 community 表
type CommunityDetail struct {
	ID           int64     `json:"id,string" gorm:"column:community_id;primaryKey"`
	Name         string    `json:"name" gorm:"column:community_name;uniqueIndex;size:128;not null"`
	Introduction string    `json:"introduction" gorm:"column:introduction;type:text"`
	CreateTime   time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`
}

// TableName 自定义表名
// 为什么：GORM 默认使用复数形式表名(community_details)，需要显式指定为 community
func (CommunityDetail) TableName() string {
	return "community"
}