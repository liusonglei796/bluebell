package aireq

type RemarkSummaryReq struct {
	PostID int64 `json:"post_id" binding:"required"`

	// 可选：限制评论数量，防止token超限
	MaxComments int `json:"max_comments"`
}
