package response

import (
	"time"
)

// PostDetailResponse 返回给客户端的帖子详情结构
type PostDetailResponse struct {
	ID          string    `json:"id"`
	AuthorID    string    `json:"author_id"`
	CommunityID int64     `json:"community_id"`
	Status      int8      `json:"status"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	CreateTime  time.Time `json:"create_time"`
	AuthorName  string    `json:"author_name"` // 作者名称
	VoteNum     int64     `json:"vote_num"`    // 投票数（赞成票数）
}


