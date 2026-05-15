package entity

import "strings"

// Remark 评论领域实体
type Remark struct {
	ID        uint
	PostID    int64
	Content   string
	AuthorID  int64
	CreatedAt string
	Author    *User
}

// Validate 校验评论内容是否合法
func (r *Remark) Validate() error {
	if strings.TrimSpace(r.Content) == "" {
		return ErrInvalidParam
	}
	return nil
}
