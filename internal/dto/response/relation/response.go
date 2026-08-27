package relationresp

import "time"

// UserSummary 用户简要信息
type UserSummary struct {
	UserID         string    `json:"user_id"`
	UserName       string    `json:"username"`
	FollowingCount int       `json:"following_count"`
	FollowerCount  int       `json:"follower_count"`
	IsFollowed     bool      `json:"is_followed"`
	IsMutual       bool      `json:"is_mutual"`
	CreatedAt      time.Time `json:"created_at"`
}

// RelationListResponse 关系列表响应
type RelationListResponse struct {
	Total int64          `json:"total"`
	Users []*UserSummary `json:"users"`
}
