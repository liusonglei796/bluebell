package postreq

// CreatePostRequest 用于接收前端请求的参数
type CreatePostRequest struct {
	Title       string   `json:"title" binding:"required"`
	Content     string   `json:"content" binding:"required"`
	CommunityID int64    `json:"community_id" binding:"required"`
	TagIDs      []uint64 `json:"tag_ids"`
}

// PostListRequest 用于获取帖子列表时的分页和排序参数
type PostListRequest struct {
	Page        int64  `form:"page"`
	Size        int64  `form:"size"`
	Order       string `form:"order"`
	CommunityID int64  `form:"community_id"`
	TagID       uint64 `form:"tag_id"`
}
type VoteRequest struct {
	PostID    int64 `json:"post_id" binding:"required"`
	Direction int8  `json:"direction" binding:"required,oneof=1 0 -1"`
}

// 排序规则常量
const (
	OrderTime  = "time"
	OrderScore = "score"
)


// PinPostRequest 置顶/取消置顶帖子请求
type PinPostRequest struct {
	PostID   int64 `json:"post_id" binding:"required"`
	IsPinned int8  `json:"is_pinned" binding:"oneof=0 1"` // 1:置顶, 0:取消置顶
}

// FeedListRequest 关注者动态 Feed 流游标分页请求
type FeedListRequest struct {
	Cursor int64 `form:"cursor"` // 上一页最后一条时间戳毫秒数
	Size   int64 `form:"size"`
}
