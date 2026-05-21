package social

// ProfileResponse 用户资料响应
type ProfileResponse struct {
	UserID         int64  `json:"user_id"`
	Username       string `json:"username"`
	AvatarURL      string `json:"avatar_url"`
	Bio            string `json:"bio"`
	GitHubURL      string `json:"github_url"`
	FollowerCount  int64  `json:"follower_count"`
	FollowingCount int64  `json:"following_count"`
	IsFollowing    bool   `json:"is_following"`
}

// ActivityResponse 用户动态响应
type ActivityResponse struct {
	ID          uint   `json:"id"`
	UserID      int64  `json:"user_id"`
	Type        string `json:"type"`
	TargetID    string `json:"target_id"`
	TargetName  string `json:"target_name"`
	Description string `json:"description"`
	CreatedAt   int64  `json:"created_at"`
}
