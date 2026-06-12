package searchresp

// SearchPostDoc 搜索结果文档结构体
type SearchPostDoc struct {
	PostID           string   `json:"post_id"`
	PostTitle        string   `json:"post_title"`
	Content          string   `json:"content"`
	CreatedAt        string   `json:"created_at"`
	HighlightTitle   []string `json:"highlight_title,omitempty"`
	HighlightContent []string `json:"highlight_content,omitempty"`
	AuthorID         int64    `json:"author_id"`
	CommunityID      int64    `json:"community_id"`
	Status           int8     `json:"status"`
}

// SearchResponse 搜索响应结构体
type SearchResponse struct {
	Posts    []SearchPostDoc `json:"posts"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}
