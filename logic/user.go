package logic

import (
	"bluebell/dao/mysql"
	"bluebell/dao/redis"
	"bluebell/models"
	"bluebell/pkg/jwt"
	"bluebell/pkg/snowflake"
)

// SignUp 处理用户注册业务逻辑
func SignUp(p *models.ParamSignUp) (err error) {
	// 1. 判断用户存不存在
	// 为什么：用户名必须唯一，防止重复注册
	if err = mysql.CheckUserExist(p.Username); err != nil {
		return err
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
	return err
}

// Login 处理用户登录业务逻辑
func Login(p *models.ParamLogin) (string, string, error) {
	user := &models.User{
		Username: p.Username,
		Password: p.Password,
	}
	// 传递的是指针，Login 内部会修改 user 对象（例如填充 UserID 和加密后的密码），虽然这里没用到 UserID
	if err := mysql.CheckLogin(user); err != nil {
		return "", "", err
	}
	//登录成功，为该用户生成JWT
	aToken, rToken, err := jwt.GenToken(user.UserID, user.Username)
	if err != nil {
		return "", "", err
	}

	// 将 Token 存入 Redis，实现单点登录
	err = redis.SetUserToken(user.UserID, aToken, rToken, jwt.AccessTokenExpireDuration, jwt.RefreshTokenExpireDuration)
	if err != nil {
		return "", "", err
	}

	return aToken, rToken, nil
}

// RefreshToken 刷新 Token
func RefreshToken(aToken, rToken string) (newAToken, newRToken string, err error) {
    // 1. 验证 RefreshToken 并获取 UserID
    claims, err := jwt.ValidateRefreshToken(rToken)
    if err != nil {
        return "", "", err
    }
    userID := claims.UserID

    // 2. 【关键修复】从数据库重新查询用户最新信息
    // 确保使用的是最新的用户名、权限等状态
    user, err := mysql.GetUserByID(userID)
    if err != nil {
        return "", "", err
    }

    // 3. 使用最新用户信息生成新 Token
    newAToken, newRToken, err = jwt.GenToken(user.UserID, user.Username)
    if err != nil {
        return "", "", err
    }

    // 4. 更新 Redis 中的 Token
    err = redis.SetUserToken(user.UserID, newAToken, newRToken, jwt.AccessTokenExpireDuration, jwt.RefreshTokenExpireDuration)
    if err != nil {
        return "", "", err
    }

    return newAToken, newRToken, nil
}