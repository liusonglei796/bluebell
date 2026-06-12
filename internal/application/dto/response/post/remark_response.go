package postResp

import "time"

// RemarkDetail 评论详情返回结构
type RemarkDetail struct {
	CreateTime time.Time `json:"create_time"`
	Content    string    `json:"content"`
	AuthorName string    `json:"author_name"`
	ID         uint      `json:"id"`
}
