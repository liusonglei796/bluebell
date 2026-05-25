package bookmarkreq

type CreateBookmarkRequest struct {
	PostID int64 `json:"post_id,string" binding:"required"`
}

type BookmarkListRequest struct {
	Page int `form:"page"`
	Size int `form:"size"`
}