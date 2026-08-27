package model

import (
	"strings"
	"time"
)

// 评论状态常量
const (
	CommentStatusNormal  int8 = 1 // 正常可见
	CommentStatusDeleted int8 = 0 // 作者已删除
	CommentStatusBlocked int8 = 2 // 管理员屏蔽
)

// Comment 二级楼中楼评论模型
type Comment struct {
	ID           int64     `gorm:"primaryKey;column:id" json:"id"`
	PostID       int64     `gorm:"column:post_id;not null;index:idx_post_root_hot,priority:1;index:idx_post_root_new,priority:1;index:idx_post_root_dialog,priority:1" json:"post_id"`
	RootID       int64     `gorm:"column:root_id;not null;default:0;index:idx_post_root_hot,priority:2;index:idx_post_root_new,priority:2;index:idx_post_root_dialog,priority:2" json:"root_id"`
	ParentID     int64     `gorm:"column:parent_id;not null;default:0" json:"parent_id"`
	AuthorID     int64     `gorm:"column:author_id;not null;index:idx_author_created,priority:1" json:"author_id"`
	ReplyToUID   int64     `gorm:"column:reply_to_uid;not null;default:0" json:"reply_to_uid"`
	Content      string    `gorm:"column:content;type:varchar(1000);not null" json:"content"`
	LikeCount    int       `gorm:"column:like_count;not null;default:0;index:idx_post_root_hot,priority:3" json:"like_count"`
	ReplyCount   int       `gorm:"column:reply_count;not null;default:0" json:"reply_count"`
	Status       int8      `gorm:"column:status;not null;default:1" json:"status"`
	CreatedAt    time.Time `gorm:"column:created_at;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3);index:idx_post_root_hot,priority:4;index:idx_post_root_new,priority:3;index:idx_post_root_dialog,priority:3;index:idx_author_created,priority:2" json:"created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)" json:"updated_at"`

	// 关联对象 (查询时填充)
	Author      *User `gorm:"-" json:"author,omitempty"`
	ReplyToUser *User `gorm:"-" json:"reply_to_user,omitempty"`
}

func (Comment) TableName() string {
	return "post_comment"
}

// Validate 校验评论内容合法性
func (c *Comment) Validate() error {
	if strings.TrimSpace(c.Content) == "" {
		return ErrInvalidParam
	}
	if len([]rune(c.Content)) > 1000 {
		return ErrInvalidParam
	}
	return nil
}

// IsRoot 判断是否为根评论
func (c *Comment) IsRoot() bool {
	return c.RootID == 0
}

// IsNormal 判断是否正常可见
func (c *Comment) IsNormal() bool {
	return c.Status == CommentStatusNormal
}
