package response

import "bluebell/models"

// PostDetailResponse 返回给客户端的帖子详情结构
type PostDetailResponse struct {
	*models.Post                         // 内嵌帖子基本信息
	AuthorName        string             `json:"author_name"` // 作者名称
	*models.Community `json:"community"` // 内嵌社区详情
	VoteNum           int64              `json:"vote_num"` // 投票数（赞成票数）
}
