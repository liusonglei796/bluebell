package user

import (
	"bluebell/internal/config"
	"bluebell/internal/domain/repointerface"
	"bluebell/internal/dto/request"
	"bluebell/internal/infrastructure/jwt"
	"bluebell/internal/infrastructure/snowflake"
	"bluebell/internal/model"
	"bluebell/pkg/errorx"
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

// UserService 用户业务逻辑服务
type UserService struct {
	userRepo   repointerface.UserRepository
	tokenCache repointerface.UserTokenCacheRepository
	jwtCfg     *config.Config
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo repointerface.UserRepository, tokenCache repointerface.UserTokenCacheRepository, jwtCfg *config.Config) *UserService {
	return &UserService{
		userRepo:   userRepo,
		tokenCache: tokenCache,
		jwtCfg:     jwtCfg,
	}
}

// SignUp 处理用户注册业务逻辑
func (s *UserService) SignUp(ctx context.Context, p *request.SignUpRequest) (err error) {
	if err = s.userRepo.CheckUserExist(ctx, p.Username); err != nil {
		if errors.Is(err, repointerface.ErrUserExist) {
			return errorx.ErrUserExist
		}
		zap.L().Error("userRepo.CheckUserExist failed",
			zap.String("username", p.Username),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	// 1. 生成 UID
	userID := snowflake.GenID()

	// 2. 构造 User 实例
	u := &model.User{
		UserID:   userID,
		UserName: p.Username,
		Passwd:   p.Password,
	}

	err = s.userRepo.InsertUser(ctx, u)
	if err != nil {
		zap.L().Error("userRepo.InsertUser failed",
			zap.Int64("user_id", u.UserID),
			zap.String("username", p.Username),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	return nil
}

// Login 处理用户登录业务逻辑
func (s *UserService) Login(ctx context.Context, p *request.LoginRequest) (string, string, error) {
	user := &model.User{
		UserName: p.Username,
		Passwd:   p.Password,
	}

	err := s.userRepo.CheckLogin(ctx, user)
	if err != nil {
		if errors.Is(err, repointerface.ErrUserNotExist) {
			return "", "", errorx.ErrUserNotExist
		}
		if errors.Is(err, repointerface.ErrInvalidPassword) {
			return "", "", errorx.ErrInvalidPassword
		}

		zap.L().Error("userRepo.CheckLogin failed",
			zap.String("username", p.Username),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	aToken, rToken, err := jwt.GenToken(s.jwtCfg, user.UserID)
	if err != nil {
		zap.L().Error("jwt.GenToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	accessTokenExp, _ := time.ParseDuration(s.jwtCfg.JWT.AccessExpiry)
	refreshTokenExp, _ := time.ParseDuration(s.jwtCfg.JWT.RefreshExpiry)

	err = s.tokenCache.SetUserToken(ctx, user.UserID, aToken, rToken, accessTokenExp, refreshTokenExp)
	if err != nil {
		zap.L().Error("tokenCache.SetUserToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	return aToken, rToken, nil
}

// RefreshToken 刷新 Token
func (s *UserService) RefreshToken(ctx context.Context, aToken, rToken string) (newAToken, newRToken string, err error) {
	// 1. 解析 Refresh Token 获取 UserID
	userID, err := jwt.ParseToken(s.jwtCfg, rToken)
	if err != nil {
		return "", "", errorx.ErrInvalidToken
	}

	// 2. 检查用户是否存在
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		zap.L().Error("userRepo.GetUserByID failed",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	newAToken, newRToken, err = jwt.GenToken(s.jwtCfg, user.UserID)
	if err != nil {
		zap.L().Error("jwt.GenToken failed in refresh",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	accessTokenExp, _ := time.ParseDuration(s.jwtCfg.JWT.AccessExpiry)
	refreshTokenExp, _ := time.ParseDuration(s.jwtCfg.JWT.RefreshExpiry)

	err = s.tokenCache.SetUserToken(ctx, user.UserID, newAToken, newRToken, accessTokenExp, refreshTokenExp)
	if err != nil {
		zap.L().Error("tokenCache.SetUserToken failed in refresh",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	return newAToken, newRToken, nil
}
