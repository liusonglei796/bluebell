package entity

// Remark 评论领域实体
type Remark struct {
	ID       uint
	PostID   int64
	Content  string
	AuthorID int64
	Author   *User
}
