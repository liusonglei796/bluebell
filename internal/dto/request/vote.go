package request

type VoteRequest struct {
	PostID    int64 `json:"post_id" binding:"required"`
	Direction int8  `json:"direction" binding:"required,oneof=1 0 -1"`
}
