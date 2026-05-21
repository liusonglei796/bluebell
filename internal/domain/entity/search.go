package entity

// SearchPostDoc 搜索结果文档结构体
type SearchPostDoc struct {
	PostID           string   `json:"post_id"`
	AuthorID         int64    `json:"author_id"`
	CommunityID      int64    `json:"community_id"`
	PostTitle        string   `json:"post_title"`
	Content          string   `json:"content"`
	Status           int8     `json:"status"`
	CreatedAt        string   `json:"created_at"`
	HighlightTitle   []string `json:"highlight_title,omitempty"`
	HighlightContent []string `json:"highlight_content,omitempty"`
}

// SearchResponse 搜索响应结构体
type SearchResponse struct {
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
	Posts    []SearchPostDoc `json:"posts"`
}
