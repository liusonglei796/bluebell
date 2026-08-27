package tagresp

import "time"

// TagResponse 标签详情
type TagResponse struct {
	ID          string    `json:"id"`
	CommunityID string    `json:"community_id"`
	Name        string    `json:"name"`
	PostCount   int       `json:"post_count"`
	CreatedAt   time.Time `json:"created_at"`
}
