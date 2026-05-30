package application

import (
	"bluebell/internal/domain"
	userreq "bluebell/internal/application/dto/request/user"
	"bluebell/internal/infrastructure/snowflake"
	"bluebell/internal/domain/entity"
	"context"
	"errors"
	"fmt"
	"strings"
	"go.uber.org/zap"
	"bluebell/internal/infrastructure/logger"
	"bluebell/internal/infrastructure/trace"
)

var tracerUser = trace.TracerForModule("service/user")

type UserService struct {
	userRepo     domain.UserRepository
	socialRepo   domain.SocialRepository
	tokenCache   domain.UserTokenCacheRepository
	tokenService domain.TokenService
}

func NewUserService(
	userRepo domain.UserRepository,
	socialRepo domain.SocialRepository,
	tokenCache domain.UserTokenCacheRepository,
	tokenService domain.TokenService,
) *UserService {
	return &UserService{
		userRepo:     userRepo,
		socialRepo:   socialRepo,
		tokenCache:   tokenCache,
		tokenService: tokenService,
	}
}

func (s *UserService) SocialLogin(ctx context.Context, githubID, username, email, avatarURL string) (string, string, error) {
	profile, err := s.socialRepo.GetProfileByGitHubID(ctx, githubID)
	var userID int64

	if err != nil {
		userID = snowflake.GenID()
		u := &entity.User{
			UserID:   userID,
			UserName: username,
			Password: "",
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

	aToken, rToken, err := s.tokenService.GenToken(userID)
	if err != nil {
		return "", "", entity.Wrap(entity.ErrServerBusy, err)
	}

	accessTokenExp := s.tokenService.GetAccessExpiry()
	refreshTokenExp := s.tokenService.GetRefreshExpiry()
	_ = s.tokenCache.SetUserToken(ctx, userID, aToken, rToken, accessTokenExp, refreshTokenExp)

	return aToken, rToken, nil
}

func (s *UserService) SignUp(ctx context.Context, p *userreq.SignUpRequest) (err error) {
	userID := snowflake.GenID()

	hashedPassword, err := entity.HashPassword(p.Password)
	if err != nil {
		logger.WithContext(ctx).Error("entity.HashPassword failed", zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}

	u := &entity.User{
		UserID:   userID,
		UserName: p.Username,
		Password: hashedPassword,
		Role:     entity.RoleUser,
	}

	err = s.userRepo.CreateUser(ctx, u)
	if err != nil {
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

func (s *UserService) Login(ctx context.Context, p *userreq.LoginRequest) (string, string, error) {
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

func (s *UserService) RefreshToken(ctx context.Context, p *userreq.RefreshTokenRequest) (newAToken, newRToken string, err error) {
	parts := strings.SplitN(p.RefreshToken, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		return "", "", fmt.Errorf("%w: Token格式错误", entity.ErrInvalidToken)
	}

	userID, err := s.tokenService.ParseToken(parts[1], "refresh")
	if err != nil {
		return "", "", entity.ErrInvalidToken
	}

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

func (s *UserService) Logout(ctx context.Context, userID int64) error {
	if err := s.tokenCache.DeleteUserToken(ctx, userID); err != nil {
		logger.WithContext(ctx).Error("tokenCache.DeleteUserToken failed",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}

	return nil
}

func (s *UserService) GetUserByUsername(ctx context.Context, username string) (*entity.User, error) {
	user, err := s.userRepo.GetUserByName(ctx, username)
	if err != nil {
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}
	return user, nil
}

func (s *UserService) UploadAvatar(ctx context.Context, userID int64, avatarURL string) error {
	profile := &entity.UserProfile{
		UserID:    userID,
		AvatarURL: avatarURL,
	}
	return s.socialRepo.SaveUserProfile(ctx, profile)
}

func (s *UserService) GetAvatarURL(ctx context.Context, userID int64) (string, error) {
	profile, err := s.socialRepo.GetUserProfile(ctx, userID)
	if err != nil || profile == nil {
		return "", nil
	}
	return profile.AvatarURL, nil
}
