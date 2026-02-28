package request

// CreatePostRequest 用于接收前端请求的参数
//这个结构体用于创建帖子的请求参数：
//作用：

//用于创建新帖子时接收前端传递的数据
//包含帖子的基本信息：标题(Title)、内容(Content)、所属社区ID(CommunityID)
//UserID 字段是从 JWT token 中提取的，不需要前端传递

type CreatePostRequest struct {
	Title       string `json:"title" binding:"required"`
	Content     string `json:"content" binding:"required"`
	CommunityID int64  `json:"community_id" binding:"required"`
	AuthorID    int64  `json:"author_id"` // 从 Token 获取，不需要前端传
}

// 用于获取帖子列表时的分页和排序参数
// Page 和 Size 用于分页控制
// Order 用于指定排序方式（按时间或按分数）
// CommunityID 用于筛选特定社区的帖子
type PostListRequest struct {
	Page  int64  `json:"page" form:"page"`
	Size  int64  `json:"size" form:"size"`
	Order string `json:"order" form:"order"`
	// 新增的字段，用于区分是否按社区查询
	// 'form:"community_id"' 标签允许 Gin 从 URL 查询参数中绑定
	// (例如: /api/v1/posts?community_id=1)
	CommunityID int64 `json:"community_id" form:"community_id"`
}

// 排序规则常量
const (
	// OrderTime 按时间排序
	OrderTime = "time"
	// OrderScore 按分数排序
	OrderScore = "score"
)
