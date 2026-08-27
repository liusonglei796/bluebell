package model

import "time"

// BookmarkFolder 用户收藏夹目录模型
type BookmarkFolder struct {
	ID        int64     `gorm:"primaryKey;column:id" json:"id"`
	UserID    int64     `gorm:"column:user_id;not null;index:idx_user_folder,priority:1" json:"user_id"`
	Name      string    `gorm:"column:name;size:64;not null" json:"name"`
	IsDefault int8      `gorm:"column:is_default;not null;default:0" json:"is_default"`
	IsPublic  int8      `gorm:"column:is_public;not null;default:0" json:"is_public"`
	PostCount int       `gorm:"column:post_count;not null;default:0" json:"post_count"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3);index:idx_user_folder,priority:2" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3)" json:"updated_at"`
}

func (BookmarkFolder) TableName() string {
	return "bookmark_folder"
}

// PostBookmark 帖子收藏关联模型
type PostBookmark struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	UserID    int64     `gorm:"column:user_id;not null;uniqueIndex:uk_user_folder_post,priority:1;index:idx_user_posts,priority:1" json:"user_id"`
	FolderID  int64     `gorm:"column:folder_id;not null;uniqueIndex:uk_user_folder_post,priority:2;index:idx_folder_posts,priority:1" json:"folder_id"`
	PostID    int64     `gorm:"column:post_id;not null;uniqueIndex:uk_user_folder_post,priority:3" json:"post_id"`
	CreatedAt time.Time `gorm:"column:created_at;type:datetime(3);not null;default:CURRENT_TIMESTAMP(3);index:idx_folder_posts,priority:2;index:idx_user_posts,priority:2" json:"created_at"`
}

func (PostBookmark) TableName() string {
	return "post_bookmark"
}
