package model

import "gorm.io/gorm"

// UserProfile 用户资料模型
type UserProfile struct {
	gorm.Model
	UserID    int64  `gorm:"column:user_id;uniqueIndex"`
	AvatarURL string `gorm:"column:avatar_url"`
	Bio       string `gorm:"column:bio;type:text"`
	GitHubID  string `gorm:"column:github_id;index"`
	GitHubURL string `gorm:"column:github_url"`
}

func (UserProfile) TableName() string {
	return "user_profile"
}

// Follow 关注模型
type Follow struct {
	gorm.Model
	FollowerID  int64 `gorm:"column:follower_id;index:idx_follow,unique"`
	FollowingID int64 `gorm:"column:following_id;index:idx_follow,unique"`
}

func (Follow) TableName() string {
	return "follow"
}

// Activity 动态模型
type Activity struct {
	gorm.Model
	UserID      int64  `gorm:"column:user_id;index"`
	Type        string `gorm:"column:type"` // post, vote, follow, comment
	TargetID    string `gorm:"column:target_id"`
	TargetName  string `gorm:"column:target_name"`
	Description string `gorm:"column:description"`
}

func (Activity) TableName() string {
	return "activity"
}
