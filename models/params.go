package models

// ParamSignUp 注册请求参数
// 为什么：定义前端传递的 JSON 数据结构，使用 binding 标签进行参数校验
type ParamSignUp struct {
	// binding:"required" 表示该字段必填，如果为空 Gin 会报错
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	// re_password 用于确认密码，业务逻辑中会校验两次密码是否一致
	RePassword string `json:"re_password" binding:"required"`
}

// ParamLogin 登录请求参数
// 为什么：分离注册和登录的参数结构，虽然字段相似，但业务含义不同，解耦更灵活
type ParamLogin struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}



 //用于获取帖子列表时的分页和排序参数
//Page 和 Size 用于分页控制
//Order 用于指定排序方式（按时间或按分数）
//CommunityID 用于筛选特定社区的帖子
type ParamPostList struct{
Page int64 `json:"page" form:"page"`
Size int64 `json:"size" form:"size"`
Order string `json:"order" form:"order"`
// 新增的字段，用于区分是否按社区查询
// 'form:"community_id"' 标签允许 Gin 从 URL 查询参数中绑定
// (例如: /api/v1/posts?community_id=1)
CommunityID int64 `json:"community_id" form:"community_id"`
}

type ParamVoteData struct{
	PostID    int64 `json:"post_id" binding:"required"`
	Direction int8  `json:"direction" binding:"required,oneof=1 0 -1"`
}

// 排序规则常量
const (
	// OrderTime 按时间排序
	OrderTime = "time"
	// OrderScore 按分数排序
	OrderScore = "score"
)