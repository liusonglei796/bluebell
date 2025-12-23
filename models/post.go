package models

import (
	"time"
	"gorm.io/gorm"
)

// Post 内存对齐优化建议：把相同类型的字段放在一起，宽字段（如 int64, string）放在前面
// 这个结构体是对数据库表结构的直接映射，使用 GORM ORM
type Post struct {
	// 8 字节字段 (int64)
	ID          int64 `json:"id" gorm:"column:post_id;primaryKey"`
	AuthorID    int64 `json:"author_id" gorm:"column:author_id;index;not null"`
	CommunityID int64 `json:"community_id" gorm:"column:community_id;index;not null"`

	// 4 字节字段 (int32)
	Status int32 `json:"status" gorm:"column:status;default:1"`

	// 16 字节字段 (string)
	Title   string `json:"title" gorm:"column:title;size:128;not null"`
	Content string `json:"content" gorm:"column:content;type:text;not null"`

	// Time 类型
	CreateTime time.Time `json:"create_time" gorm:"column:create_time;autoCreateTime"`

		// GORM 关联字段 (用于 Preload 预加载，解决 N+1 问题)
	// 为什么添加：使用 GORM 的 Preload 功能可以自动批量查询关联数据
	// gorm:"-" 表示不映射到数据库字段，只用于内存中的关联
	Author    *User             `json:"author,omitempty" gorm:"foreignKey:AuthorID;references:UserID"`
	Community *CommunityDetail  `json:"community,omitempty" gorm:"foreignKey:CommunityID;references:CommunityID"`
}

// TableName 自定义表名
// 为什么：GORM 默认使用复数形式表名(posts)，需要显式指定为 post
func (Post) TableName() string {
	return "post"
}

// BeforeCreate GORM 钩子
func (p *Post) BeforeCreate(tx *gorm.DB) error {
	return nil
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