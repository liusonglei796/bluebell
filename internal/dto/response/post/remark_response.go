package postResp

import "time"

// RemarkDetail 评论详情返回结构
type RemarkDetail struct {
	ID         uint      `json:"id"`
	Content    string    `json:"content"`
	AuthorName string    `json:"author_name"`
	CreateTime time.Time `json:"create_time"`
}
