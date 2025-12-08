package models

import "time"


// CommunityDetail 社区详情结构体
type CommunityDetail struct {
	ID           int64     `json:"id,string" db:"community_id"`
	Name         string    `json:"name" db:"community_name"`
	Introduction string    `json:"introduction" db:"introduction"`
	CreateTime   time.Time `json:"create_time" db:"create_time"`
}