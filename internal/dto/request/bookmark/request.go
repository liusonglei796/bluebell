package bookmarkreq

// CreateFolderRequest 创建收藏夹
type CreateFolderRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=64"`
	IsPublic int8   `json:"is_public"` // 0:私密, 1:公开
}

// AddBookmarkRequest 添加帖子到收藏夹
type AddBookmarkRequest struct {
	PostID   int64 `json:"post_id" binding:"required"`
	FolderID int64 `json:"folder_id"` // 0则添加至默认收藏夹
}

// RemoveBookmarkRequest 从收藏夹移除帖子
type RemoveBookmarkRequest struct {
	PostID   int64 `json:"post_id" binding:"required"`
	FolderID int64 `json:"folder_id"`
}

// BookmarkListRequest 收藏夹内帖子列表
type BookmarkListRequest struct {
	FolderID int64 `form:"folder_id"` // 0为全量收藏列表
	Page     int64 `form:"page"`
	Size     int64 `form:"size"`
}
