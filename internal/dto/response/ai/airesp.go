package ai_resp

type RemarkSummaryResp struct {
	Summary      string `json:"summary"`
	SummaryType  string `json:"summary_type"`
	Language     string `json:"language"`
	PostID       int64  `json:"post_id"`
	CommentCount int    `json:"comment_count"`
	CreatedAt    string `json:"created_at"`
}
