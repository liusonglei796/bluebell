package model

import "gorm.io/gorm"

// Vote 投票数据模型
// 对应数据库中的 vote 表
type Vote struct {
	gorm.Model
	PostID    int64 `gorm:"column:post_id;not null;index:idx_post_user,unique"`
	UserID    int64 `gorm:"column:user_id;not null;index:idx_post_user,unique"`
	Direction int8  `gorm:"column:direction;not null"` // 1: 赞成, -1: 反对, 0: 取消
}

// TableName 自定义表名
func (Vote) TableName() string {
	return "vote"
}
