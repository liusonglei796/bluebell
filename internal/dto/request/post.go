package request

// CreatePostRequest 用于接收前端请求的参数
type CreatePostRequest struct {
	Title       string `json:"title" binding:"required"`
	Content     string `json:"content" binding:"required"`
	CommunityID int64  `json:"community_id" binding:"required"`
	AuthorID    string `json:"author_id"` // 从 Token 获取，不需要前端传
}

// PostListRequest 用于获取帖子列表时的分页和排序参数
type PostListRequest struct {
	Page  int64  `json:"page" form:"page"`
	Size  int64  `json:"size" form:"size"`
	Order string `json:"order" form:"order"`
	// 新增的字段，用于区分是否按社区查询
	CommunityID int64 `json:"community_id" form:"community_id"`
}

// 排序规则常量
const (
	OrderTime  = "time"
	OrderScore = "score"
)
