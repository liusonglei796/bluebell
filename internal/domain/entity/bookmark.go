package entity

import "time"

type Bookmark struct {
	CreatedAt time.Time
	UserID    int64
	PostID    int64
}
