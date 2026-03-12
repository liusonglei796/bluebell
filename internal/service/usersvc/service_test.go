package usersvc

import (
	"context"
	"testing"
	"time"

	"bluebell/internal/config"
	"bluebell/internal/domain/cachedomain"
	"bluebell/internal/domain/dbdomain"
	userreq "bluebell/internal/dto/request/user"
	"bluebell/internal/infrastructure/snowflake"
	"bluebell/internal/model"
	"bluebell/pkg/errorx"
)

// ==================== 1. 定义 UserRepository 的 Mock ====================
type MockUserRepo struct {
	CheckUserExistFunc      func(ctx context.Context, username string) error
	InsertUserFunc          func(ctx context.Context, user *model.User) error
	VerifyUserFunc          func(ctx context.Context, user *model.User) error
	CheckUserExistsByIDFunc func(ctx context.Context, uid int64) (*model.User, error)
	GetUsersByIDsFunc       func(ctx context.Context, ids []int64) ([]*model.User, error)
	GetUserRoleByIDFunc     func(ctx context.Context, uid int64) (int, error)
}

func (m *MockUserRepo) CheckUserExist(ctx context.Context, username string) error {
	if m.CheckUserExistFunc != nil {
		return m.CheckUserExistFunc(ctx, username)
	}
	return nil
}

func (m *MockUserRepo) InsertUser(ctx context.Context, user *model.User) error {
	if m.InsertUserFunc != nil {
		return m.InsertUserFunc(ctx, user)
	}
	return nil
}

func (m *MockUserRepo) VerifyUser(ctx context.Context, user *model.User) error { return nil }
func (m *MockUserRepo) CheckUserExistsByID(ctx context.Context, uid int64) (*model.User, error) { return nil, nil }
func (m *MockUserRepo) GetUsersByIDs(ctx context.Context, ids []int64) ([]*model.User, error) { return nil, nil }
func (m *MockUserRepo) GetUserRoleByID(ctx context.Context, uid int64) (int, error) { return 0, nil }

// 编译期类型检查，确保实现了接口
var _ dbdomain.UserRepository = (*MockUserRepo)(nil)

// ==================== 2. 定义 TokenCache 的 Mock ====================
type MockTokenCache struct {
	SetUserTokenFunc        func(ctx context.Context, userID int64, aToken, rToken string, aExp, rExp time.Duration) error
}

func (m *MockTokenCache) SetUserToken(ctx context.Context, userID int64, aToken, rToken string, aExp, rExp time.Duration) error { return nil }
func (m *MockTokenCache) GetUserAccessToken(ctx context.Context, userID int64) (string, error) { return "", nil }
func (m *MockTokenCache) GetUserRefreshToken(ctx context.Context, userID int64) (string, error) { return "", nil }
func (m *MockTokenCache) DeleteUserToken(ctx context.Context, userID int64) error { return nil }

var _ cachedomain.UserTokenCacheRepository = (*MockTokenCache)(nil)

// ==================== 3. 编写 Service 层的单元测试 ====================
func TestSignUp(t *testing.T) {
	// 初始化 snowflake 算法 (因为 Service 内部依赖了这个外部库)
	_ = snowflake.Init(&config.Config{
		Snowflake: &config.SnowflakeConfig{
			StartTime: time.Now(),
			MachineID: 1,
		},
	})

	// 测试用例 1: 成功注册
	t.Run("Success", func(t *testing.T) {
		// 1. 准备 Mock 对象
		mockRepo := &MockUserRepo{
			CheckUserExistFunc: func(ctx context.Context, username string) error {
				return nil // 模拟数据库："该用户名还没人注册"
			},
			InsertUserFunc: func(ctx context.Context, user *model.User) error {
				return nil // 模拟数据库："插入新用户成功"
			},
		}
		mockCache := &MockTokenCache{}
		mockCfg := &config.Config{} // 对于注册来说不需要具体配置

		// 2. 注入 Mock 创建 Service
		svc := NewUserService(mockRepo, mockCache, mockCfg)

		// 3. 执行业务逻辑
		req := &userreq.SignUpRequest{Username: "newuser", Password: "pwd"}
		err := svc.SignUp(context.Background(), req)

		// 4. 验证结果
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	// 测试用例 2: 用户名已存在
	t.Run("User Already Exists", func(t *testing.T) {
		// 1. 准备 Mock 对象
		mockRepo := &MockUserRepo{
			CheckUserExistFunc: func(ctx context.Context, username string) error {
				// 模拟数据库报错："该用户名已经存在啦！"
				return errorx.ErrUserExist
			},
		}

		// 2. 注入 Mock 创建 Service
		svc := NewUserService(mockRepo, &MockTokenCache{}, &config.Config{})

		// 3. 执行业务逻辑
		req := &userreq.SignUpRequest{Username: "existuser", Password: "pwd"}
		err := svc.SignUp(context.Background(), req)

		// 4. 验证结果
		if err != errorx.ErrUserExist {
			t.Errorf("expected ErrUserExist, got %v", err)
		}
	})
}


