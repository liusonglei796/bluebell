package entity

import "time"

// UserProfile 用户资料实体
type UserProfile struct {
	AvatarURL string
	Bio       string
	GitHubID  string
	GitHubURL string
	UserID    int64
}

// Activity 用户动态实体
type Activity struct {
	CreatedAt   time.Time
	Type        string // post, vote, follow, comment
	TargetID    string
	TargetName  string
	Description string
	ID          uint
	UserID      int64
}
