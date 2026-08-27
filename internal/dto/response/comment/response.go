package commentresp

import "time"

// CommentResponse 评论详情返回结构
type CommentResponse struct {
	ID           string             `json:"id"`
	PostID       string             `json:"post_id"`
	RootID       string             `json:"root_id"`
	ParentID     string             `json:"parent_id"`
	AuthorID     string             `json:"author_id"`
	AuthorName   string             `json:"author_name"`
	AuthorAvatar string             `json:"author_avatar,omitempty"`
	ReplyToUID   string             `json:"reply_to_uid,omitempty"`
	ReplyToName  string             `json:"reply_to_name,omitempty"`
	Content      string             `json:"content"`
	LikeCount    int                `json:"like_count"`
	ReplyCount   int                `json:"reply_count"`
	IsLiked      bool               `json:"is_liked"`
	IsDeleted    bool               `json:"is_deleted"`
	CreatedAt    time.Time          `json:"created_at"`
	SubReplies   []*CommentResponse `json:"sub_replies,omitempty"` // 前3条预览子评论
}

// CommentListResponse 根评论列表及统计
type CommentListResponse struct {
	Total    int64              `json:"total"`
	Comments []*CommentResponse `json:"comments"`
}
