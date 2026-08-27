package model

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// bcryptCost bcrypt 加密成本参数
// DefaultCost = 10，每增加1，计算时间翻倍
const bcryptCost = 10

// 用户角色常量
const (
	RoleUser  = 1 // 普通用户
	RoleAdmin = 2 // 管理员
)

// User 用户模型
// 对应数据库 user 表结构
type User struct {
	gorm.Model
	UserID         int64  `gorm:"column:user_id"`
	// user_name 唯一索引：并发注册同名用户时由数据库兜底，防止 check-then-insert 竞态产生重复数据
	UserName       string `gorm:"column:user_name;size:64;not null;uniqueIndex"`
	Passwd         string `gorm:"column:passwd;size:255;not null"`
	Role           int    `gorm:"column:role;default:1;not null"`
	FollowingCount int    `gorm:"column:following_count;not null;default:0"`
	FollowerCount  int    `gorm:"column:follower_count;not null;default:0"`
}

// TableName 自定义表名
func (User) TableName() string {
	return "user"
}

// IsAdmin 判断用户是否为管理员
func (u *User) IsAdmin() bool {
	if u == nil {
		return false
	}
	return u.Role == RoleAdmin
}

// IsValid 检查用户数据是否合法（基本校验）
func (u *User) IsValid() bool {
	return u != nil && u.UserName != ""
}

// HashPassword 对明文密码进行 bcrypt 加密，返回密文
func HashPassword(raw string) (string, error) {
	if raw == "" {
		return "", ErrInvalidParam
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 校验明文密码是否与密文匹配
func CheckPassword(raw, hashed string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(raw)) == nil
}
