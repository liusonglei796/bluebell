package postResp

import (
	"time"
)

// DetailResponse 返回给客户端的帖子详情结构
type DetailResponse struct {
	CreateTime  time.Time `json:"create_time"`
	ID          string    `json:"id"`
	AuthorID    string    `json:"author_id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	AuthorName  string    `json:"author_name"` // 作者名称
	CommunityID int64     `json:"community_id"`
	VoteNum     int64     `json:"vote_num"` // 净投票数（vote_up - vote_down）
	Status      int8      `json:"status"`
}
