package model

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// bcryptCost bcrypt 加密成本参数
// DefaultCost = 10，每增加1，计算时间翻倍
// 性能考虑：生产环境使用 10 可以平衡安全性和性能
const bcryptCost = 10

// User 用户模型
// 为什么：对应数据库中的 user 表结构，使用 GORM ORM 映射

type User struct {
	gorm.Model
	UserID   int64  `gorm:"column:user_id"`
	UserName string `gorm:"column:user_name;size:64;not null"`
	Passwd   string `gorm:"column:passwd;size:255;not null"`
}

// TableName 自定义表名
// 为什么：GORM 默认使用复数形式表名(users)，需要显式指定为 user
func (User) TableName() string {
	return "user"
}

// BeforeCreate GORM 钩子，创建前自动处理
// 为什么：将密码加密逻辑放在 Model 层，DAO 层保持纯粹的数据库操作
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// 如果密码不为空，则进行加密
	if u.Passwd != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.Passwd), bcryptCost)
		if err != nil {
			return err
		}
		u.Passwd = string(hash)
	}
	return nil
}
