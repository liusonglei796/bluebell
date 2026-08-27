package bookmarkresp

import (
	postResp "bluebell/internal/dto/response/post"
	"time"
)

// FolderResponse 收藏夹详情
type FolderResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"is_default"`
	IsPublic  bool      `json:"is_public"`
	PostCount int       `json:"post_count"`
	CreatedAt time.Time `json:"created_at"`
}

// BookmarkListResponse 收藏帖子分页响应
type BookmarkListResponse struct {
	Total int64                      `json:"total"`
	Posts []*postResp.DetailResponse `json:"posts"`
}
