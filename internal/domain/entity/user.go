// Package entity 定义领域实体
//
// 领域层是 DDD 的核心，包含了最稳定的业务规则。
// 这些规则（如加密成本、角色定义）不随技术栈或外部框架的改变而改变。
package entity

import "golang.org/x/crypto/bcrypt"

// bcryptCost bcrypt 加密成本参数
// DefaultCost = 10，每增加1，计算时间翻倍
const bcryptCost = 10

// 用户角色常量
const (
	RoleUser  = 1 // 普通用户
	RoleAdmin = 2 // 管理员
)

// User 用户领域实体 (Domain Entity)
// DDD 定义：实体是有唯一标识（UserID）且包含业务行为的对象。
// 它承载了用户最核心的属性和必须遵守的规则（如密码加密、管理员判定）。
type User struct {
	UserID   int64
	UserName string
	Password string // 明文或密文，取决于使用场景
	Role     int
}

// IsAdmin 判断用户是否为管理员
// 这是一个领域规则：管理员权限的判定逻辑被封装在实体内部，与外部鉴权框架解耦。
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
// 这是领域层的核心业务逻辑：密码策略（加密算法、成本）由领域规则决定。
// 将其放在领域层可确保：无论用户是通过 API 注册还是后台导入，其密码安全策略都是强制一致的。
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
