package mysql

import (
	"bluebell/models"
	"database/sql"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"

	"go.uber.org/zap"
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
	sqlStr := "select count(user_id) from user where username = ?"
	var count int
	// db.Get 用于查询单行数据
	if err := db.Get(&count, sqlStr, username); err != nil {
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
	// 2. 构建 SQL
	// 使用命名参数 (:user_id, :username, :password)，sqlx 会自动映射结构体字段
	sqlStr := `INSERT INTO user (user_id, username, password) VALUES (:user_id, :username, :password)`

	// 3. 执行插入
	_, err = db.NamedExec(sqlStr, user)
	if err != nil {
		return fmt.Errorf("插入用户失败: %w", err)
	}

	return nil
}

// Login 登录验证
func CheckLogin(user *models.User) (err error) {
	// 记录用户输入的原始密码（明文）
	oPassword := user.Password
	sqlStr := `select user_id,username,password from user where username=?`
	// 查询数据库中的用户信息（包含加密后的密码）
	err = db.Get(user, sqlStr, user.Username)
	if err == sql.ErrNoRows {
		// 为什么：区分“查询出错”和“查不到数据”
		return ErrorUserNotExist
	}
	if err != nil {
		return fmt.Errorf("login failed %w", err)
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
// 注意函数名：FindUserByUserID (明确是查业务ID)
func GetUserByID(uid int64) (*models.User, error) {
    user := &models.User{}
    sqlStr := `SELECT user_id, username FROM user WHERE user_id = ?`
    
    err := db.Get(user, sqlStr, uid)
    if err != nil {
        if err == sql.ErrNoRows {
            zap.L().Warn("there is no user in db", zap.Int64("user_id", uid))
            return nil, nil // 返回nil而不是error，让上层决定如何处理
        }
        zap.L().Error("query user failed", zap.Int64("user_id", uid), zap.Error(err))
        return nil, err
    }
    return user, nil
}