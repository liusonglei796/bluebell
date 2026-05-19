package usersvc

import (
	// 配置
	"bluebell/internal/config"

	// 领域层 - Repository 接口
	"bluebell/internal/domain"

	// 领域层 - Service 接口
	"bluebell/internal/application"

	// DTO
	userreq "bluebell/internal/interfaces/http/dto/request/user"

	// 基础设施
	"bluebell/internal/infrastructure/jwt"
	"bluebell/internal/infrastructure/snowflake"

	// 错误处理
	"bluebell/internal/domain/entity"

	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	// 日志
	"bluebell/internal/infrastructure/logger"
)

// userServiceStruct 用户业务逻辑服务
type userServiceStruct struct {
	userRepo   domain.UserRepository
	tokenCache domain.UserTokenCacheRepository
	jwtCfg     *config.Config
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo domain.UserRepository, tokenCache domain.UserTokenCacheRepository, jwtCfg *config.Config) application.UserService {
	return &userServiceStruct{
		userRepo:   userRepo,
		tokenCache: tokenCache,
		jwtCfg:     jwtCfg,
	}
}

// SignUp 处理用户注册业务逻辑
func (s *userServiceStruct) SignUp(ctx context.Context, p *userreq.SignUpRequest) (err error) {
	if err = s.userRepo.CheckUserExist(ctx, p.Username); err != nil {
		if errors.Is(err, entity.ErrUserExist) {
			return err
		}
		logger.WithContext(ctx).Error("userRepo.CheckUserExist failed",
			zap.String("username", p.Username),
			zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}

	// 1. 生成 UID
	userID := snowflake.GenID()

	// 2. 密码加密 (下沉到领域层)
	hashedPassword, err := entity.HashPassword(p.Password)
	if err != nil {
		logger.WithContext(ctx).Error("entity.HashPassword failed", zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}

	// 3. 构造 User 领域实体
	u := &entity.User{
		UserID:   userID,
		UserName: p.Username,
		Password: hashedPassword,
		Role:     entity.RoleUser,
	}

	err = s.userRepo.CreateUser(ctx, u)
	if err != nil {
		logger.WithContext(ctx).Error("userRepo.CreateUser failed",
			zap.Int64("user_id", u.UserID),
			zap.String("username", p.Username),
			zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}

	return nil
}

// Login 处理用户登录业务逻辑
func (s *userServiceStruct) Login(ctx context.Context, p *userreq.LoginRequest) (string, string, error) {
	user := &entity.User{
		UserName: p.Username,
		Password: p.Password,
	}

	err := s.userRepo.VerifyUser(ctx, user)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotExist) || errors.Is(err, entity.ErrInvalidPassword) {
			return "", "", err
		}
		logger.WithContext(ctx).Error("userRepo.CheckLogin failed",
			zap.String("username", p.Username),
			zap.Error(err))
		return "", "", entity.Wrap(entity.ErrServerBusy, err)
	}

	aToken, rToken, err := jwt.GenToken(s.jwtCfg, user.UserID)
	if err != nil {
		logger.WithContext(ctx).Error("jwt.GenToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", entity.Wrap(entity.ErrServerBusy, err)
	}

	accessTokenExp, err := time.ParseDuration(s.jwtCfg.JWT.AccessExpiry)
	if err != nil {
		logger.WithContext(ctx).Error("parse access token expiry failed", zap.Error(err))
		accessTokenExp = 2 * time.Hour // 默认 2 小时
	}
	refreshTokenExp, err := time.ParseDuration(s.jwtCfg.JWT.RefreshExpiry)
	if err != nil {
		logger.WithContext(ctx).Error("parse refresh token expiry failed", zap.Error(err))
		refreshTokenExp = 7 * 24 * time.Hour // 默认 7 天
	}

	err = s.tokenCache.SetUserToken(ctx, user.UserID, aToken, rToken, accessTokenExp, refreshTokenExp)
	if err != nil {
		logger.WithContext(ctx).Error("tokenCache.SetUserToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", entity.Wrap(entity.ErrServerBusy, err)
	}

	return aToken, rToken, nil
}

// RefreshToken 刷新 Token
func (s *userServiceStruct) RefreshToken(ctx context.Context, p *userreq.RefreshTokenRequest) (newAToken, newRToken string, err error) {
	// 1. 解析 Authorization Header 获取 Access Token
	parts := strings.SplitN(p.Authorization, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		return "", "", fmt.Errorf("%w: Token格式错误", entity.ErrInvalidToken)
	}
	// aToken := parts[1] // aToken 暂时没有使用，但可以在此验证或者做其他逻辑

	// 2. 解析 Refresh Token 获取 UserID
	userID, err := jwt.ParseToken(s.jwtCfg, p.RefreshToken, jwt.RefreshTokenType)
	if err != nil {
		return "", "", entity.ErrInvalidToken
	}

	// 3. 检查用户是否存在
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		logger.WithContext(ctx).Error("userRepo.GetUserByID failed",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return "", "", entity.Wrap(entity.ErrServerBusy, err)
	}

	newAToken, newRToken, err = jwt.GenToken(s.jwtCfg, user.UserID)
	if err != nil {
		logger.WithContext(ctx).Error("jwt.GenToken failed in refresh",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", entity.Wrap(entity.ErrServerBusy, err)
	}

	accessTokenExp, err := time.ParseDuration(s.jwtCfg.JWT.AccessExpiry)
	if err != nil {
		logger.WithContext(ctx).Error("parse access token expiry failed in refresh", zap.Error(err))
		accessTokenExp = 2 * time.Hour // 默认 2 小时
	}
	refreshTokenExp, err := time.ParseDuration(s.jwtCfg.JWT.RefreshExpiry)
	if err != nil {
		logger.WithContext(ctx).Error("parse refresh token expiry failed in refresh", zap.Error(err))
		refreshTokenExp = 7 * 24 * time.Hour // 默认 7 天
	}

	err = s.tokenCache.SetUserToken(ctx, user.UserID, newAToken, newRToken, accessTokenExp, refreshTokenExp)
	if err != nil {
		logger.WithContext(ctx).Error("tokenCache.SetUserToken failed in refresh",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", entity.Wrap(entity.ErrServerBusy, err)
	}

	return newAToken, newRToken, nil
}

// Logout 用户登出，清除 Redis 中的 Token
func (s *userServiceStruct) Logout(ctx context.Context, userID int64) error {
	if err := s.tokenCache.DeleteUserToken(ctx, userID); err != nil {
		logger.WithContext(ctx).Error("tokenCache.DeleteUserToken failed",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}

	return nil
}

// GetUserByUsername 根据用户名获取用户信息
func (s *userServiceStruct) GetUserByUsername(ctx context.Context, username string) (*entity.User, error) {
	user, err := s.userRepo.GetUserByName(ctx, username)
	if err != nil {
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}
	return user, nil
}
