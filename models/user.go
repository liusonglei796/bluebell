package models

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// BcryptCost bcrypt 加密成本参数
// DefaultCost = 10，每增加1，计算时间翻倍
// 性能考虑：生产环境使用 10 可以平衡安全性和性能
const BcryptCost = 10

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
// 为什么：将密码加密逻辑放在 Model 层，DAO 层保持纯粹的数据库操作
func (u *User) BeforeCreate(tx *gorm.DB) error {
	// 如果密码不为空，则进行加密
	if u.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), BcryptCost)
		if err != nil {
			return err
		}
		u.Password = string(hash)
	}
	return nil
}
