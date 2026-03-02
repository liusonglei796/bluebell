package user

import (
	"bluebell/internal/domain/repository"
	"bluebell/internal/dto/request"
	"bluebell/internal/model"
	"bluebell/pkg/errorx"
	"bluebell/pkg/jwt"
	"bluebell/pkg/snowflake"
	"context"
	"errors"

	"go.uber.org/zap"
)

// UserService 用户业务逻辑服务
type UserService struct {
	userRepo   repository.UserRepository
	tokenCache repository.UserTokenCacheRepository
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo repository.UserRepository, tokenCache repository.UserTokenCacheRepository) *UserService {
	return &UserService{
		userRepo:   userRepo,
		tokenCache: tokenCache,
	}
}

// SignUp 处理用户注册业务逻辑
func (s *UserService) SignUp(ctx context.Context, p *request.SignUpRequest) (err error) {
	if err = s.userRepo.CheckUserExist(ctx, p.Username); err != nil {
		if errors.Is(err, repository.ErrUserExist) {
			return errorx.ErrUserExist
		}
		zap.L().Error("userRepo.CheckUserExist failed",
			zap.String("username", p.Username),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	userID := snowflake.GetID()

	u := &model.User{
		UserID:   userID,
		Username: p.Username,
		Password: p.Password,
	}

	err = s.userRepo.InsertUser(ctx, u)
	if err != nil {
		zap.L().Error("userRepo.InsertUser failed",
			zap.Int64("user_id", userID),
			zap.String("username", p.Username),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	return nil
}

// Login 处理用户登录业务逻辑
func (s *UserService) Login(ctx context.Context, p *request.LoginRequest) (string, string, error) {
	user := &model.User{
		Username: p.Username,
		Password: p.Password,
	}

	err := s.userRepo.CheckLogin(ctx, user)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotExist) {
			return "", "", errorx.ErrUserNotExist
		}
		if errors.Is(err, repository.ErrInvalidPassword) {
			return "", "", errorx.ErrInvalidPassword
		}

		zap.L().Error("userRepo.CheckLogin failed",
			zap.String("username", p.Username),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	aToken, rToken, err := jwt.GenToken(user.UserID, user.Username)
	if err != nil {
		zap.L().Error("jwt.GenToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	err = s.tokenCache.SetUserToken(ctx, user.UserID, aToken, rToken, jwt.AccessTokenExpireDuration, jwt.RefreshTokenExpireDuration)
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
	userID, err := jwt.ValidateRefreshToken(rToken)
	if err != nil {
		return "", "", errorx.ErrInvalidToken
	}

	// 查询用户信息以获取 username（GenToken 需要）
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil || user == nil {
		zap.L().Error("userRepo.GetUserByID failed",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	newAToken, newRToken, err = jwt.GenToken(user.UserID, user.Username)
	if err != nil {
		zap.L().Error("jwt.GenToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	err = s.tokenCache.SetUserToken(ctx, user.UserID, newAToken, newRToken, jwt.AccessTokenExpireDuration, jwt.RefreshTokenExpireDuration)
	if err != nil {
		zap.L().Error("tokenCache.SetUserToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	return newAToken, newRToken, nil
}
