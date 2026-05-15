package entity

import (
	"strings"
	"time"
)

// Remark 评论领域实体
type Remark struct {
	ID        uint
	PostID    int64
	Content   string
	AuthorID  int64
	CreatedAt time.Time
	Author    *User
}

// Validate 校验评论内容是否合法
func (r *Remark) Validate() error {
	if strings.TrimSpace(r.Content) == "" {
		return ErrInvalidParam
	}
	return nil
}
