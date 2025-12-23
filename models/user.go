package models

import "gorm.io/gorm"

// User 用户模型
// 为什么：对应数据库中的 user 表结构，使用 GORM ORM 映射
type User struct {
	// gorm:"column:user_id;primaryKey" 指定列名和主键
	UserID   int64  `json:"user_id,string" gorm:"column:user_id;primaryKey"`
	Username string `json:"username" gorm:"column:username;uniqueIndex;size:64;not null"`
	Password string `json:"-" gorm:"column:password;size:255;not null"` // json:"-" 防止密码序列化到 JSON
}

// TableName 自定义表名
// 为什么：GORM 默认使用复数形式表名(users)，需要显式指定为 user
func (User) TableName() string {
	return "user"
}

// BeforeCreate GORM 钩子，创建前自动处理
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// 可以在这里添加创建前的逻辑（如时间戳等）
	return nil
}
