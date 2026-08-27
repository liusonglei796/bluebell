package model

import "time"

// Tag 社区标签模型
type Tag struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	CommunityID int64     `gorm:"column:community_id;not null;uniqueIndex:uk_community_tag,priority:1" json:"community_id"`
	Name        string    `gorm:"column:name;size:32;not null;uniqueIndex:uk_community_tag,priority:2" json:"name"`
	PostCount   int       `gorm:"column:post_count;not null;default:0" json:"post_count"`
	CreatedAt   time.Time `gorm:"column:created_at;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3)" json:"created_at"`
}

func (Tag) TableName() string {
	return "tag"
}

// PostTag 帖子-标签多对多关联模型
type PostTag struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	PostID      int64     `gorm:"column:post_id;not null;uniqueIndex:uk_post_tag,priority:1" json:"post_id"`
	TagID       uint64    `gorm:"column:tag_id;not null;uniqueIndex:uk_post_tag,priority:2;index:idx_tag_posts,priority:1" json:"tag_id"`
	CommunityID int64     `gorm:"column:community_id;not null" json:"community_id"`
	CreatedAt   time.Time `gorm:"column:created_at;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3);index:idx_tag_posts,priority:2" json:"created_at"`
}

func (PostTag) TableName() string {
	return "post_tag"
}
