package voteresp

import "time"

// LeaderboardItem 排行榜单项
type LeaderboardItem struct {
	Rank        int       `json:"rank"`         // 排名
	PostID      string    `json:"post_id"`      // 帖子ID
	Title       string    `json:"title"`        // 帖子标题
	AuthorName  string    `json:"author_name"`  // 作者名
	VoteCount   int64     `json:"vote_count"`   // 投票数
	Score       float64   `json:"score"`        // 分数
	CommunityID int64     `json:"community_id"` // 社区ID
	CreateTime  time.Time `json:"create_time"`  // 创建时间
}

// LeaderboardResponse 排行榜响应
type LeaderboardResponse struct {
	Items []*LeaderboardItem `json:"items"` // 排行榜项
	Total int64              `json:"total"` // 总数
}
