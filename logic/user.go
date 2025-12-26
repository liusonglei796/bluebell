package logic

import (
	"bluebell/dao/mysql"
	"bluebell/dao/redis"
	"bluebell/models"
	"bluebell/pkg/errorx"
	"bluebell/pkg/jwt"
	"bluebell/pkg/snowflake"
	"errors"

	"go.uber.org/zap"
)

// SignUp 处理用户注册业务逻辑
func SignUp(p *models.ParamSignUp) (err error) {
	// 1. 判断用户存不存在
	// 为什么：用户名必须唯一，防止重复注册
	if err = mysql.CheckUserExist(p.Username); err != nil {
		// 区分业务错误和系统错误
		if errors.Is(err, mysql.ErrorUserExist) {
			return errorx.ErrUserExist
		}
		// 系统错误: 数据库查询失败
		zap.L().Error("mysql.CheckUserExist failed",
			zap.String("username", p.Username),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	// 2. 生成UID
	// 为什么：使用雪花算法生成全局唯一的 UserID，避免使用数据库自增 ID 带来的分库分表麻烦
	userID := snowflake.GetID()

	// 3. 构造User实例
	// 为什么：将业务数据封装成数据库模型
	u := &models.User{
		UserID:   userID,
		Username: p.Username,
		Password: p.Password,
	}

	// 4. 保存进数据库
	// 为什么：持久化用户数据
	err = mysql.InsertUser(u)
	if err != nil {
		// 系统错误: 数据库插入失败
		zap.L().Error("mysql.InsertUser failed",
			zap.Int64("user_id", userID),
			zap.String("username", p.Username),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	return nil
}

// Login 处理用户登录业务逻辑
// 返回值：aToken, rToken, error
// error 类型说明：
//   - *errorx.CodeError: 业务错误（密码错误、用户不存在）
//   - 系统错误: DB/Redis 错误，Controller 会自动转换为 CodeServerBusy
func Login(p *models.ParamLogin) (string, string, error) {
	user := &models.User{
		Username: p.Username,
		Password: p.Password,
	}

	// 1. 调用 DAO 层验证用户登录
	err := mysql.CheckLogin(user)
	if err != nil {
		// 判断是否是业务错误（用户不存在或密码错误）
		if errors.Is(err, mysql.ErrorUserNotExist) {
			// 业务错误：返回带错误码的 CodeError
			return "", "", errorx.ErrUserNotExist
		}
		if errors.Is(err, mysql.ErrorInvalidPassword) {
			// 业务错误：返回带错误码的 CodeError
			return "", "", errorx.ErrInvalidPassword
		}

		// 系统错误：记录详细日志并返回通用的服务繁忙错误
		zap.L().Error("mysql.CheckLogin failed",
			zap.String("username", p.Username),
			zap.Error(err),
		)
		return "", "", errorx.ErrServerBusy
	}

	// 2. 登录成功，为该用户生成 JWT Token
	aToken, rToken, err := jwt.GenToken(user.UserID, user.Username)
	if err != nil {
		// JWT 生成失败属于系统错误
		zap.L().Error("jwt.GenToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err),
		)
		return "", "", errorx.ErrServerBusy
	}

	// 3. 将 Token 存入 Redis，实现单点登录
	err = redis.SetUserToken(user.UserID, aToken, rToken, jwt.AccessTokenExpireDuration, jwt.RefreshTokenExpireDuration)
	if err != nil {
		// Redis 存储失败属于系统错误
		zap.L().Error("redis.SetUserToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err),
		)
		return "", "", errorx.ErrServerBusy
	}

	return aToken, rToken, nil
}

// RefreshToken 刷新 Token
func RefreshToken(aToken, rToken string) (newAToken, newRToken string, err error) {
	// 1. 验证 RefreshToken 并获取最新的用户信息
	// ValidateRefreshToken 内部已经查询了数据库，确保用户存在且未被封禁
	user, err := jwt.ValidateRefreshToken(rToken)
	if err != nil {
		// Token 验证失败属于业务错误 (Token 无效或过期)
		return "", "", errorx.ErrInvalidToken
	}

	// 3. 使用最新用户信息生成新 Token
	newAToken, newRToken, err = jwt.GenToken(user.UserID, user.Username)
	if err != nil {
		// 系统错误: Token 生成失败
		zap.L().Error("jwt.GenToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	// 4. 更新 Redis 中的 Token
	err = redis.SetUserToken(user.UserID, newAToken, newRToken, jwt.AccessTokenExpireDuration, jwt.RefreshTokenExpireDuration)
	if err != nil {
		// 系统错误: Redis 操作失败
		zap.L().Error("redis.SetUserToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	return newAToken, newRToken, nil
}