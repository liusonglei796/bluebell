package postResp

import (
	"time"

	communityResp "bluebell/internal/dto/response/community"
)

// DetailResponse 返回给客户端的帖子详情结构
type DetailResponse struct {
	ID            string                  `json:"id"`
	AuthorIDs     []string                `json:"author_ids"`
	AuthorNames   []string                `json:"author_names"`
	AuthorName    string                  `json:"author_name"`
	CommunityID   int64                   `json:"community_id"`
	CommunityName string                  `json:"community_name,omitempty"`
	Community     *communityResp.Response `json:"community,omitempty"`
	Status        int8                    `json:"status"`
	Title         string                  `json:"title"`
	Content       string                  `json:"content"`
	CreateTime    time.Time               `json:"create_time"`
	VoteNum       int64                   `json:"vote_num"`
	Score         int64                   `json:"score"`
	IsPinned      bool                    `json:"is_pinned"`
	IsHighlighted bool                    `json:"is_highlighted"`
	BookmarkCount int                     `json:"bookmark_count"`
	CommentCount  int                     `json:"comment_count"`
	IsBookmarked  bool                    `json:"is_bookmarked"`
	Tags          []string                `json:"tags,omitempty"`
}

