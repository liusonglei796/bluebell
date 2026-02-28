package logic

import (
	"bluebell/dao/mysql"
	"bluebell/dao/redis"
	"bluebell/dto/request"
	"bluebell/models"
	"bluebell/pkg/errorx"
	"bluebell/pkg/jwt"
	"bluebell/pkg/snowflake"
	"context"
	"errors"

	"go.uber.org/zap"
)

// UserService 用户业务逻辑服务
type UserService struct {
	userRepo UserRepository
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// SignUp 处理用户注册业务逻辑
func (s *UserService) SignUp(ctx context.Context, p *request.SignUpRequest) (err error) {
	if err = s.userRepo.CheckUserExist(ctx, p.Username); err != nil {
		if errors.Is(err, mysql.ErrorUserExist) {
			return errorx.ErrUserExist
		}
		zap.L().Error("userRepo.CheckUserExist failed",
			zap.String("username", p.Username),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	userID := snowflake.GetID()

	u := &models.User{
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
	user := &models.User{
		Username: p.Username,
		Password: p.Password,
	}

	err := s.userRepo.CheckLogin(ctx, user)
	if err != nil {
		if errors.Is(err, mysql.ErrorUserNotExist) {
			return "", "", errorx.ErrUserNotExist
		}
		if errors.Is(err, mysql.ErrorInvalidPassword) {
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

	err = redis.SetUserToken(ctx, user.UserID, aToken, rToken, jwt.AccessTokenExpireDuration, jwt.RefreshTokenExpireDuration)
	if err != nil {
		zap.L().Error("redis.SetUserToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	return aToken, rToken, nil
}

// RefreshToken 刷新 Token
func (s *UserService) RefreshToken(ctx context.Context, aToken, rToken string) (newAToken, newRToken string, err error) {
	user, err := jwt.ValidateRefreshToken(ctx, rToken)
	if err != nil {
		return "", "", errorx.ErrInvalidToken
	}

	newAToken, newRToken, err = jwt.GenToken(user.UserID, user.Username)
	if err != nil {
		zap.L().Error("jwt.GenToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	err = redis.SetUserToken(ctx, user.UserID, newAToken, newRToken, jwt.AccessTokenExpireDuration, jwt.RefreshTokenExpireDuration)
	if err != nil {
		zap.L().Error("redis.SetUserToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	return newAToken, newRToken, nil
}
