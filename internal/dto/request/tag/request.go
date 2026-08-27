package tagreq

// CreateTagRequest 社区创建标签
type CreateTagRequest struct {
	CommunityID int64  `json:"community_id" binding:"required"`
	Name        string `json:"name" binding:"required,min=1,max=32"`
}

// TagListRequest 社区标签列表
type TagListRequest struct {
	CommunityID int64 `form:"community_id" binding:"required"`
}
