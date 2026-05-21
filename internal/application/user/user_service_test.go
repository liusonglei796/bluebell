package usersvc

import (
	"bluebell/internal/config"
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/snowflake"
	userreq "bluebell/internal/application/dto/request/user"
	"context"
	"testing"
	"time"
)

// MockUserRepository is a manual mock for domain.UserRepository
type MockUserRepository struct {
	CheckUserExistFunc func(ctx context.Context, username string) error
	CreateUserFunc      func(ctx context.Context, user *entity.User) error
	VerifyUserFunc      func(ctx context.Context, user *entity.User) error
	GetUserByIDFunc     func(ctx context.Context, uid int64) (*entity.User, error)
	GetUsersByIDsFunc   func(ctx context.Context, ids []int64) ([]*entity.User, error)
	GetUserRoleByIDFunc func(ctx context.Context, uid int64) (int, error)
	GetUserByNameFunc   func(ctx context.Context, username string) (*entity.User, error)
}

func (m *MockUserRepository) CheckUserExist(ctx context.Context, username string) error {
	return m.CheckUserExistFunc(ctx, username)
}
func (m *MockUserRepository) CreateUser(ctx context.Context, user *entity.User) error {
	return m.CreateUserFunc(ctx, user)
}
func (m *MockUserRepository) VerifyUser(ctx context.Context, user *entity.User) error {
	return m.VerifyUserFunc(ctx, user)
}
func (m *MockUserRepository) GetUserByID(ctx context.Context, uid int64) (*entity.User, error) {
	return m.GetUserByIDFunc(ctx, uid)
}
func (m *MockUserRepository) GetUsersByIDs(ctx context.Context, ids []int64) ([]*entity.User, error) {
	return m.GetUsersByIDsFunc(ctx, ids)
}
func (m *MockUserRepository) GetUserRoleByID(ctx context.Context, uid int64) (int, error) {
	return m.GetUserRoleByIDFunc(ctx, uid)
}
func (m *MockUserRepository) GetUserByName(ctx context.Context, username string) (*entity.User, error) {
	return m.GetUserByNameFunc(ctx, username)
}

// MockTokenCacheRepository is a manual mock for domain.UserTokenCacheRepository
type MockTokenCacheRepository struct {
	SetUserTokenFunc       func(ctx context.Context, userID int64, aToken, rToken string, aExp, rExp time.Duration) error
	GetUserAccessTokenFunc func(ctx context.Context, userID int64) (string, error)
	GetUserRefreshTokenFunc func(ctx context.Context, userID int64) (string, error)
	DeleteUserTokenFunc    func(ctx context.Context, userID int64) error
}

func (m *MockTokenCacheRepository) SetUserToken(ctx context.Context, userID int64, aToken, rToken string, aExp, rExp time.Duration) error {
	return m.SetUserTokenFunc(ctx, userID, aToken, rToken, aExp, rExp)
}
func (m *MockTokenCacheRepository) GetUserAccessToken(ctx context.Context, userID int64) (string, error) {
	return m.GetUserAccessTokenFunc(ctx, userID)
}
func (m *MockTokenCacheRepository) GetUserRefreshToken(ctx context.Context, userID int64) (string, error) {
	return m.GetUserRefreshTokenFunc(ctx, userID)
}
func (m *MockTokenCacheRepository) DeleteUserToken(ctx context.Context, userID int64) error {
	return m.DeleteUserTokenFunc(ctx, userID)
}

// MockTokenService is a manual mock for domain.TokenService
type MockTokenService struct {
	GenTokenFunc func(userID int64) (string, string, error)
	ParseTokenFunc func(tokenString string, expectedType string) (int64, error)
}

func (m *MockTokenService) GenToken(userID int64) (string, string, error) {
	return m.GenTokenFunc(userID)
}
func (m *MockTokenService) ParseToken(tokenString string, expectedType string) (int64, error) {
	return m.ParseTokenFunc(tokenString, expectedType)
}
func (m *MockTokenService) GetAccessExpiry() time.Duration {
	return 2 * time.Hour
}
func (m *MockTokenService) GetRefreshExpiry() time.Duration {
	return 7 * 24 * time.Hour
}

func TestUserService_SignUp(t *testing.T) {
	// Initialize snowflake for tests to avoid nil pointer dereference
	snowflake.Init(&config.Config{
		Snowflake: &config.SnowflakeConfig{
			StartTime: 1775539200000,
			MachineID: 1,
		},
	})

	mockRepo := &MockUserRepository{
		CheckUserExistFunc: func(ctx context.Context, username string) error {
			return nil
		},
		CreateUserFunc: func(ctx context.Context, user *entity.User) error {
			if user.UserName != "testuser" {
				t.Errorf("expected username testuser, got %s", user.UserName)
			}
			if user.Password == "password123" {
				t.Error("expected hashed password, got plain text")
			}
			return nil
		},
	}
	mockTokenService := &MockTokenService{
		GenTokenFunc: func(userID int64) (string, string, error) {
			return "access", "refresh", nil
		},
	}
	s := NewUserService(mockRepo, nil, nil, mockTokenService)
	err := s.SignUp(context.Background(), &userreq.SignUpRequest{
		Username:   "testuser",
		Password:   "password123",
		RePassword: "password123",
	})
	if err != nil {
		t.Errorf("SignUp failed: %v", err)
	}
}
