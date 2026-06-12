package entity

import (
	"strings"
	"time"
)

// Remark 评论领域实体
type Remark struct {
	CreatedAt time.Time
	Author    *User
	Content   string
	ID        uint
	PostID    int64
	AuthorID  int64
	ReplyTo   int64
}

// Validate 校验评论内容是否合法
func (r *Remark) Validate() error {
	if strings.TrimSpace(r.Content) == "" {
		return ErrInvalidParam
	}
	return nil
}
