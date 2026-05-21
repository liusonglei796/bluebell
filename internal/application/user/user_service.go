// Package usersvc 实现用户应用服务
//
// 应用层（Application Layer）充当“指挥官”角色：
// 它不包含核心业务规则，而是协调领域实体、仓储接口和外部基础设施来完成一个完整的用例。
package usersvc

import (
	// 领域层 - Repository 接口
	"bluebell/internal/domain"

	// 领域层 - Service 接口
	"bluebell/internal/application"

	// DTO
	userreq "bluebell/internal/application/dto/request/user"

	// 基础设施
	"bluebell/internal/infrastructure/snowflake"

	// 错误处理
	"bluebell/internal/domain/entity"

	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	// 日志
	"bluebell/internal/infrastructure/logger"
)

// userServiceStruct 用户应用服务 (Application Service)
// DDD 定义：应用服务不包含业务规则，它是“流程编排者”。
// 它持有多个领域仓储接口，负责在不同的领域对象（User, Social）之间进行协调，完成用户的完整用例。
type userServiceStruct struct {
	userRepo     domain.UserRepository
	socialRepo   domain.SocialRepository
	tokenCache   domain.UserTokenCacheRepository
	tokenService domain.TokenService
}

// NewUserService 创建用户服务实例
func NewUserService(userRepo domain.UserRepository, socialRepo domain.SocialRepository, tokenCache domain.UserTokenCacheRepository, tokenService domain.TokenService) application.UserService {
	return &userServiceStruct{
		userRepo:     userRepo,
		socialRepo:   socialRepo,
		tokenCache:   tokenCache,
		tokenService: tokenService,
	}
}

// SocialLogin 处理社交账号登录 (如 GitHub)
// 为什么这个逻辑在应用层？
// 这是一个典型的业务流程编排：
// 1. 查 Social Profile (Infrastructure)
// 2. 如果不存在，创建 User 和 Profile (Domain/Infrastructure)
// 3. 生成 Token (Infrastructure/Utility)
// 4. 存入缓存 (Infrastructure)
// 每一个步骤本身可能是简单的，但“把它们按顺序串起来”就是应用层的职责。
func (s *userServiceStruct) SocialLogin(ctx context.Context, githubID, username, email, avatarURL string) (string, string, error) {
	// 1. 检查是否存在该 GitHub 账号的 Profile
	profile, err := s.socialRepo.GetProfileByGitHubID(ctx, githubID)
	var userID int64

	if err != nil {
		// 2. 如果不存在，创建新用户和 Profile
		userID = snowflake.GenID()
		u := &entity.User{
			UserID:   userID,
			UserName: username,
			Password: "", // 社交登录用户没有初始密码
			Role:     entity.RoleUser,
		}
		if err := s.userRepo.CreateUser(ctx, u); err != nil {
			return "", "", entity.Wrap(entity.ErrServerBusy, err)
		}

		profile = &entity.UserProfile{
			UserID:    userID,
			AvatarURL: avatarURL,
			Bio:       "GitHub User",
			GitHubID:  githubID,
			GitHubURL: fmt.Sprintf("https://github.com/%s", username),
		}
		if err := s.socialRepo.SaveUserProfile(ctx, profile); err != nil {
			return "", "", entity.Wrap(entity.ErrServerBusy, err)
		}
	} else {
		userID = profile.UserID
	}

	// 3. 生成 Token
	aToken, rToken, err := s.tokenService.GenToken(userID)
	if err != nil {
		return "", "", entity.Wrap(entity.ErrServerBusy, err)
	}

	// 4. 缓存 Token (复用 Login 中的过期逻辑)
	accessTokenExp := s.tokenService.GetAccessExpiry()
	refreshTokenExp := s.tokenService.GetRefreshExpiry()
	_ = s.tokenCache.SetUserToken(ctx, userID, aToken, rToken, accessTokenExp, refreshTokenExp)

	return aToken, rToken, nil
}

// SignUp 处理用户注册业务逻辑
// 为什么不直接在 Handler 里写注册逻辑？
// 1. 保证复用性：未来如果有 CLI 或 gRPC 注册需求，可以复用此逻辑。
// 2. 保证测试性：我们可以在不启动 HTTP 服务的情况下测试注册流程。
func (s *userServiceStruct) SignUp(ctx context.Context, p *userreq.SignUpRequest) (err error) {
	// 1. 生成 UID (应用层职责：分配全局唯一 ID)
	userID := snowflake.GenID()

	// 2. 密码加密 (下沉到领域层)
	// 为什么 SignUp 调 HashPassword 而不是自己加密？
	// 遵守“厨师（Domain）定菜谱，服务员（Application）传菜”的原则。
	// 加密策略由 Domain 决定，Application 只负责调用。
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
		// 如果是用户已存在错误，直接透传业务错误
		if errors.Is(err, entity.ErrUserExist) {
			return err
		}
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
	u, err := s.userRepo.GetUserByName(ctx, p.Username)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotExist) {
			return "", "", err
		}
		logger.WithContext(ctx).Error("userRepo.GetUserByName failed",
			zap.String("username", p.Username),
			zap.Error(err))
		return "", "", entity.Wrap(entity.ErrServerBusy, err)
	}

	if !entity.CheckPassword(p.Password, u.Password) {
		return "", "", entity.ErrInvalidPassword
	}

	aToken, rToken, err := s.tokenService.GenToken(u.UserID)
	if err != nil {
		logger.WithContext(ctx).Error("jwt.GenToken failed",
			zap.Int64("user_id", u.UserID),
			zap.Error(err))
		return "", "", entity.Wrap(entity.ErrServerBusy, err)
	}

	accessTokenExp := s.tokenService.GetAccessExpiry()
	refreshTokenExp := s.tokenService.GetRefreshExpiry()

	err = s.tokenCache.SetUserToken(ctx, u.UserID, aToken, rToken, accessTokenExp, refreshTokenExp)
	if err != nil {
		logger.WithContext(ctx).Error("tokenCache.SetUserToken failed",
			zap.Int64("user_id", u.UserID),
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
	userID, err := s.tokenService.ParseToken(p.RefreshToken, "refresh")
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

	newAToken, newRToken, err = s.tokenService.GenToken(user.UserID)
	if err != nil {
		logger.WithContext(ctx).Error("jwt.GenToken failed in refresh",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", entity.Wrap(entity.ErrServerBusy, err)
	}

	accessTokenExp := s.tokenService.GetAccessExpiry()
	refreshTokenExp := s.tokenService.GetRefreshExpiry()

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
