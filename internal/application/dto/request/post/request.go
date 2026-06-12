package postreq

// CreatePostRequest 用于接收前端请求的参数
type CreatePostRequest struct {
	Title       string `json:"title" binding:"required"`
	Content     string `json:"content" binding:"required"`
	CommunityID int64  `json:"community_id" binding:"required"`
}

// PostListRequest 用于获取帖子列表时的分页和排序参数
type PostListRequest struct {
	Order       string `form:"order"`
	Page        int64  `form:"page"`
	Size        int64  `form:"size"`
	CommunityID int64  `form:"community_id"`
}
type VoteRequest struct {
	PostID    int64 `json:"post_id,string" binding:"required"`
	Direction int8  `json:"direction" binding:"required,oneof=1 0 -1"`
}

// 排序规则常量
const (
	OrderTime  = "time"
	OrderScore = "score"
)

type RemarkRequest struct {
	Content string `json:"content" binding:"required"`
	PostID  int64  `json:"post_id,string" binding:"required"`
	ReplyTo int64  `json:"reply_to,string"`
}
