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


//这个结构体定义了前端调用接口时需要传递的参数。
  //  定义位置：models/params.go
  //  目的：用于 Controller 层 接收和校验前端的请求。
 //   字段含义：它的字段（如 Page, Size, Order）是为了控制查询行为（分页、排序），这些字段并不存在于数据库的 post 表中。

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