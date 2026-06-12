package bookmarkresp

type BookmarkListResponse struct {
	Bookmarks []*BookmarkResponse `json:"bookmarks"`
	Total     int                 `json:"total"`
	Page      int                 `json:"page"`
	Size      int                 `json:"size"`
}

type BookmarkResponse struct {
	PostID        string `json:"post_id"`
	CreatedAt     string `json:"created_at"`
	PostTitle     string `json:"post_title,omitempty"`
	AuthorName    string `json:"author_name,omitempty"`
	CommunityName string `json:"community_name,omitempty"`
	CommunityID   int64  `json:"community_id,omitempty"`
}

type BookmarkStatusResponse struct {
	Bookmarked bool `json:"bookmarked"`
	Count      int  `json:"count"`
}
