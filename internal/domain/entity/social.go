package entity

import "time"

// UserProfile 用户资料实体
type UserProfile struct {
	UserID    int64
	AvatarURL string
	Bio       string
	GitHubID  string
	GitHubURL string
}

// Activity 用户动态实体
type Activity struct {
	ID          uint
	UserID      int64
	Type        string // post, vote, follow, comment
	TargetID    string
	TargetName  string
	Description string
	CreatedAt   time.Time
}
