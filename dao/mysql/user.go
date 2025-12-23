package mysql

import (
	"bluebell/models"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 定义包级别的错误变量
// 为什么：预定义错误，方便 logic 层进行错误类型判断 (errors.Is)
var (
	ErrorUserExist       = errors.New("用户已存在")
	ErrorUserNotExist    = errors.New("用户不存在")
	ErrorInvalidPassword = errors.New("密码错误")
)

// CheckUserExist 检查指定用户名的用户是否存在
func CheckUserExist(username string) (err error) {
	var count int64
	// GORM 使用 Count 方法统计记录数
	err = db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrorUserExist
	}
	return nil
}

// encryptPassword 对密码进行加密 (使用 bcrypt)
// 为什么：数据库不能明文存储密码，bcrypt 是一种安全的哈希算法，自带盐值
func encryptPassword(oPassword string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(oPassword), bcrypt.DefaultCost)
	return string(hash), err
}

// InsertUser 插入新用户
func InsertUser(user *models.User) (err error) {
	// 1. 密码加密
	// 为什么：安全第一，存储加密后的密码
	user.Password, err = encryptPassword(user.Password)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	// 2. 使用 GORM 创建记录
	// Create 方法会自动插入数据
	err = db.Create(user).Error
	if err != nil {
		return fmt.Errorf("插入用户失败: %w", err)
	}

	return nil
}

// CheckLogin 登录验证
func CheckLogin(user *models.User) (err error) {
	// 记录用户输入的原始密码（明文）
	oPassword := user.Password

	// 使用 GORM 查询用户
	// Where 条件查询，First 查询单条记录
	err = db.Where("username = ?", user.Username).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 为什么：区分"查询出错"和"查不到数据"
			return ErrorUserNotExist
		}
		return fmt.Errorf("login failed: %w", err)
	}

	// 验证密码
	// 为什么：比较用户输入的明文密码和数据库中的哈希密码是否匹配
	err = verifyPassword(user.Password, oPassword)
	if err != nil {
		// 这里的 err 可能是 bcrypt.ErrMismatchedHashAndPassword
		return ErrorInvalidPassword
	}
	return nil
}

// verifyPassword 验证密码
func verifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// GetUserByID 根据用户ID查询用户信息
// DAO层只返回错误，不打印日志，由上层统一处理
func GetUserByID(uid int64) (*models.User, error) {
	user := &models.User{}

	// GORM 使用 First 查询单条记录
	err := db.Where("user_id = ?", uid).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 查不到数据返回nil，不是错误
		}
		return nil, fmt.Errorf("query user by id failed: %w", err)
	}
	return user, nil
}

// GetUsersByIDs 根据用户ID列表批量获取用户信息
func GetUsersByIDs(ids []int64) (users []*models.User, err error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// GORM 使用 Where IN 查询
	// Find 方法会自动查询多条记录
	users = make([]*models.User, 0, len(ids))
	err = db.Where("user_id IN ?", ids).Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("query users by ids failed: %w", err)
	}

	return users, nil
}
