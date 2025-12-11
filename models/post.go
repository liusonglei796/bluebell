package models

import "time"

// Post 内存对齐优化建议：把相同类型的字段放在一起，宽字段（如 int64, string）放在前面
// 这个结构体是对数据库表结构的直接映射。
type Post struct {
	// 8 字节字段 (int64)
	ID          int64 `json:"id" db:"post_id"`
	AuthorID    int64 `json:"author_id" db:"author_id"`
	CommunityID int64 `json:"community_id" db:"community_id"`

	// 4 字节字段 (int32)
	Status int32 `json:"status" db:"status"`

	// 16 字节字段 (string) - 虽然string是指针+长度，但在结构体布局中通常按指针对齐
	Title   string `json:"title" db:"title"`
	Content string `json:"content" db:"content"`

	// Time 类型
	CreateTime time.Time `json:"create_time" db:"create_time"`
}

// ParamPost 用于接收前端请求的参数
//这个结构体用于创建帖子的请求参数：
//作用：

//用于创建新帖子时接收前端传递的数据
//包含帖子的基本信息：标题(Title)、内容(Content)、所属社区ID(CommunityID)
//AuthorID 字段是从 JWT token 中提取的，不需要前端传递

type ParamPost struct {
	Title       string `json:"title" binding:"required"`
	Content     string `json:"content" binding:"required"`
	CommunityID int64  `json:"community_id" binding:"required"`
	AuthorID    int64  `json:"author_id"` // 从 Token 获取，不需要前端传
}
// ApiPostDetail 返回给客户端的帖子详情结构
type ApiPostDetail struct {
	*Post                                 // 内嵌帖子基本信息
	AuthorName      string `json:"author_name"` // 作者名称
	*CommunityDetail `json:"communitydetail"`   // 内嵌社区详情
	VoteNum         int64  `json:"vote_num"`    // 投票数（赞成票数）
}