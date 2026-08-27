package commentreq

// CreateCommentRequest 创建评论请求参数
type CreateCommentRequest struct {
	PostID     int64  `json:"post_id" binding:"required"`
	RootID     int64  `json:"root_id"`      // 0为根评论, >0为所属根评论
	ParentID   int64  `json:"parent_id"`    // 0为根评论, >0为被回复的评论
	ReplyToUID int64  `json:"reply_to_uid"` // 被回复的用户ID
	Content    string `json:"content" binding:"required,min=1,max=1000"`
}

// CommentListRequest 根评论列表分页请求
type CommentListRequest struct {
	PostID int64  `form:"post_id" binding:"required"`
	Page   int64  `form:"page"`
	Size   int64  `form:"size"`
	Order  string `form:"order"` // "hot" | "new"
}

// SubReplyListRequest 展开某根评论下的子回复请求
type SubReplyListRequest struct {
	PostID int64 `form:"post_id" binding:"required"`
	RootID int64 `form:"root_id" binding:"required"`
	Page   int64 `form:"page"`
	Size   int64 `form:"size"`
}

// LikeCommentRequest 点赞评论请求
type LikeCommentRequest struct {
	CommentID int64 `json:"comment_id" binding:"required"`
}
