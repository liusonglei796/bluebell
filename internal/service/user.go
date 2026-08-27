package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"bluebell/internal/config"
	"bluebell/internal/dao/mysql"
	"bluebell/internal/dao/redis"
	"bluebell/internal/dto/request/user"
	"bluebell/internal/jwt"
	"bluebell/internal/model"
	"bluebell/internal/snowflake"

	"go.uber.org/zap"
)

// UserService 用户业务逻辑服务
type UserService struct {
	userDao    *mysql.UserDao
	tokenCache *redis.UserTokenCache
	jwtCfg     *config.Config
}

// NewUserService 创建用户服务实例
func NewUserService(userDao *mysql.UserDao, tokenCache *redis.UserTokenCache, jwtCfg *config.Config) *UserService {
	return &UserService{
		userDao:    userDao,
		tokenCache: tokenCache,
		jwtCfg:     jwtCfg,
	}
}

// SignUp 处理用户注册业务逻辑
// 用户名唯一性由 user_name 唯一索引 + INSERT IGNORE 在数据库层兜底，
// InsertUser 内部通过 RowsAffected 判断重复并返回 ErrUserExist。
func (s *UserService) SignUp(ctx context.Context, p *userreq.SignUpRequest) error {
	// 1. 生成 UID
	userID := snowflake.GenID()

	// 2. 密码加密
	hashedPassword, err := model.HashPassword(p.Password)
	if err != nil {
		zap.L().Error("model.HashPassword failed", zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}

	// 3. 构造用户模型
	u := &model.User{
		UserID:   userID,
		UserName: p.Username,
		Passwd:   hashedPassword,
		Role:     model.RoleUser,
	}

	err = s.userDao.InsertUser(ctx, u)
	if err != nil {
		if errors.Is(err, model.ErrUserExist) {
			return err
		}
		zap.L().Error("userDao.InsertUser failed",
			zap.Int64("user_id", u.UserID),
			zap.String("username", p.Username),
			zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}

	return nil
}

// Login 处理用户登录业务逻辑（无状态发牌，不往 Redis 写入 AccessToken）
func (s *UserService) Login(ctx context.Context, p *userreq.LoginRequest) (string, string, error) {
	user := &model.User{
		UserName: p.Username,
		Passwd:   p.Password,
	}

	err := s.userDao.VerifyUser(ctx, user)
	if err != nil {
		if errors.Is(err, model.ErrUserNotExist) || errors.Is(err, model.ErrInvalidPassword) {
			return "", "", err
		}
		zap.L().Error("userDao.VerifyUser failed",
			zap.String("username", p.Username),
			zap.Error(err))
		return "", "", model.Wrap(model.ErrServerBusy, err)
	}

	aToken, rToken, err := jwt.GenToken(s.jwtCfg, user.UserID)
	if err != nil {
		zap.L().Error("jwt.GenToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", model.Wrap(model.ErrServerBusy, err)
	}

	return aToken, rToken, nil
}

// RefreshToken 刷新 Token（旧 Token 加入黑名单防重放，新 Token 无状态颁发）
func (s *UserService) RefreshToken(ctx context.Context, p *userreq.RefreshTokenRequest) (newAToken, newRToken string, err error) {
	// 1. 解析 Refresh Token 获取 Claims
	rClaims, err := jwt.ParseTokenClaims(s.jwtCfg, p.RefreshToken, jwt.RefreshTokenType)
	if err != nil {
		return "", "", model.ErrInvalidToken
	}

	// 2. 检查 Refresh Token 是否已被黑名单拦截
	if rClaims.ID != "" {
		isBlacklisted, err := s.tokenCache.IsBlacklisted(ctx, rClaims.ID)
		if err == nil && isBlacklisted {
			return "", "", model.ErrInvalidToken
		}
	}

	userID, err := strconv.ParseInt(rClaims.Subject, 10, 64)
	if err != nil {
		return "", "", model.ErrInvalidToken
	}

	// 3. 检查用户是否存在
	user, err := s.userDao.CheckUserExistsByID(ctx, userID)
	if err != nil || user == nil {
		zap.L().Error("userDao.CheckUserExistsByID failed",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return "", "", model.Wrap(model.ErrServerBusy, err)
	}

	// 4. 将旧 AccessToken 的 JTI 加入黑名单（若传递了 Authorization 且不同于 RefreshToken）
	if p.Authorization != "" {
		parts := strings.SplitN(p.Authorization, " ", 2)
		tokenStr := p.Authorization
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenStr = parts[1]
		}
		if tokenStr != p.RefreshToken {
			if oldAClaims, err := jwt.ParseTokenClaims(s.jwtCfg, tokenStr, jwt.AccessTokenType); err == nil && oldAClaims.ID != "" && oldAClaims.ExpiresAt != nil {
				_ = s.tokenCache.AddBlacklist(ctx, oldAClaims.ID, time.Until(oldAClaims.ExpiresAt.Time))
			}
		}
	}

	// 5. 旧 RefreshToken 也加入黑名单（Token Rotation 机制）
	if rClaims.ID != "" && rClaims.ExpiresAt != nil {
		_ = s.tokenCache.AddBlacklist(ctx, rClaims.ID, time.Until(rClaims.ExpiresAt.Time))
	}

	// 6. 生成全新 Access Token 与 Refresh Token
	newAToken, newRToken, err = jwt.GenToken(s.jwtCfg, user.UserID)
	if err != nil {
		zap.L().Error("jwt.GenToken failed in refresh",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", model.Wrap(model.ErrServerBusy, err)
	}

	return newAToken, newRToken, nil
}

// Logout 用户登出，将当前 Token 的 JTI 写入 Redis 黑名单（到期自动逐出）
func (s *UserService) Logout(ctx context.Context, jti string, expiresAt time.Time) error {
	if jti == "" {
		return nil
	}
	remainingTTL := time.Until(expiresAt)
	if remainingTTL > 0 {
		if err := s.tokenCache.AddBlacklist(ctx, jti, remainingTTL); err != nil {
			zap.L().Error("tokenCache.AddBlacklist failed",
				zap.String("jti", jti),
				zap.Error(err))
			return model.Wrap(model.ErrServerBusy, err)
		}
	}

	return nil
}

// GetUserByUsername 根据用户名获取用户信息
func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	user, err := s.userDao.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, model.Wrap(model.ErrServerBusy, err)
	}
	return user, nil
}
