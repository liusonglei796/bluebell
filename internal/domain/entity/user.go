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

// User 用户领域实体
type User struct {
	UserID   int64
	UserName string
	Password string // 明文或密文，取决于使用场景
	Role     int
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
// 这是领域层的核心业务逻辑：密码策略由领域层决定，不依赖 ORM 钩子
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
