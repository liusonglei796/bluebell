package usersvc

import (
	// 配置
	"bluebell/internal/config"

	// 领域层 - Repository 接口
	"bluebell/internal/domain/cachedomain"
	"bluebell/internal/domain/dbdomain"

	// 领域层 - Service 接口
	"bluebell/internal/domain/svcdomain"

	// DTO
	 "bluebell/internal/dto/request/user"

	// 基础设施
	"bluebell/internal/infrastructure/jwt"
	"bluebell/internal/infrastructure/snowflake"

	// 模型
	"bluebell/internal/model"

	// 错误处理
	"bluebell/pkg/errorx"

	"context"
	"strings"
	"time"

	"go.uber.org/zap"
)

// userServiceStruct 用户业务逻辑服务
type userServiceStruct struct {
	userRepo   dbdomain.UserRepository
	tokenCache cachedomain.UserTokenCacheRepository
	jwtCfg     *config.Config
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo dbdomain.UserRepository, tokenCache cachedomain.UserTokenCacheRepository, jwtCfg *config.Config) svcdomain.UserService {
	return &userServiceStruct{
		userRepo:   userRepo,
		tokenCache: tokenCache,
		jwtCfg:     jwtCfg,
	}
}

// SignUp 处理用户注册业务逻辑
func (s *userServiceStruct) SignUp(ctx context.Context, p *userreq.SignUpRequest) (err error) {
	if err = s.userRepo.CheckUserExist(ctx, p.Username); err != nil {
		if errorx.GetCode(err) == errorx.CodeUserExist {
			return err
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
func (s *userServiceStruct) Login(ctx context.Context, p *userreq.LoginRequest) (string, string, error) {
	user := &model.User{
		UserName: p.Username,
		Passwd:   p.Password,
	}

	err := s.userRepo.VerifyUser(ctx, user)
	if err != nil {
		code := errorx.GetCode(err)
		if code == errorx.CodeUserNotExist || code == errorx.CodeInvalidPassword {
			return "", "", err
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
func (s *userServiceStruct) RefreshToken(ctx context.Context, p *userreq.RefreshTokenRequest) (newAToken, newRToken string, err error) {
	// 1. 解析 Authorization Header 获取 Access Token
	parts := strings.SplitN(p.Authorization, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		return "", "", errorx.Wrap(errorx.ErrInvalidToken, errorx.CodeInvalidToken, "Token格式错误")
	}
	// aToken := parts[1] // aToken 暂时没有使用，但可以在此验证或者做其他逻辑

	// 2. 解析 Refresh Token 获取 UserID
	userID, err := jwt.ParseToken(s.jwtCfg, p.RefreshToken, jwt.RefreshTokenType)
	if err != nil {
		return "", "", errorx.ErrInvalidToken
	}

	// 3. 检查用户是否存在
	user, err := s.userRepo.CheckUserExistsByID(ctx, userID)
	if err != nil || user == nil {
		zap.L().Error("userRepo.CheckUserExistsByID failed",
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
