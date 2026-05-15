package userdb

import (
	// 模型
	"bluebell/internal/infrastructure/persistence/mysql/model"

	// 领域层
	"bluebell/internal/domain"

	// 错误处理
	"bluebell/internal/domain/entity"

	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// userRepoStruct 用户数据访问实现
type userRepoStruct struct {
	db *gorm.DB
}

// NewUserRepo 创建 userRepoStruct 实例
func NewUserRepo(db *gorm.DB) domain.UserRepository {
	return &userRepoStruct{db: db}
}

// CheckUserExist 检查指定用户名的用户是否存在
func (r *userRepoStruct) CheckUserExist(ctx context.Context, username string) (err error) {
	var count int64
	err = r.db.WithContext(ctx).Model(&model.User{}).Where("user_name = ?", username).Count(&count).Error
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if count > 0 {
		return entity.ErrUserExist
	}
	return nil
}

// InsertUser 插入新用户
// 密码加密已移至 Model 的 BeforeCreate 钩子中自动处理
func (r *userRepoStruct) InsertUser(ctx context.Context, user *model.User) (err error) {
	err = r.db.WithContext(ctx).Create(user).Error
	if err != nil {
		return fmt.Errorf("插入用户失败: %w", err)
	}
	return nil
}

func (r *userRepoStruct) VerifyUser(ctx context.Context, user *model.User) (err error) {
	oPassword := user.Passwd
	err = r.db.WithContext(ctx).Where("user_name = ?", user.UserName).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrUserNotExist
		}
		return fmt.Errorf("登录失败: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Passwd), []byte(oPassword))
	if err != nil {
		return entity.ErrInvalidPassword
	}
	return nil
}

// CheckUserExistsByID 根据用户ID查询用户信息
func (r *userRepoStruct) CheckUserExistsByID(ctx context.Context, uid int64) (*model.User, error) {
	user := &model.User{}
	err := r.db.WithContext(ctx).Where("user_id = ?", uid).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return user, nil
}

// GetUsersByIDs 根据用户ID列表批量获取用户信息
func (r *userRepoStruct) GetUsersByIDs(ctx context.Context, ids []int64) (users []*model.User, err error) {
	if len(ids) == 0 {
		return make([]*model.User, 0), nil
	}

	users = make([]*model.User, 0, len(ids))
	err = r.db.WithContext(ctx).Where("user_id IN ?", ids).Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("批量查询用户失败: %w", err)
	}
	return users, nil
}

// GetUserRoleByID 根据用户ID查询用户角色
func (r *userRepoStruct) GetUserRoleByID(ctx context.Context, uid int64) (int, error) {
	user := &model.User{}
	err := r.db.WithContext(ctx).Select("role").Where("user_id = ?", uid).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, entity.ErrUserNotExist
		}
		return 0, fmt.Errorf("查询用户角色失败: %w", err)
	}
	return user.Role, nil
}

// GetUserByUsername 根据用户名获取用户信息
func (r *userRepoStruct) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	user := &model.User{}
	err := r.db.WithContext(ctx).Where("user_name = ?", username).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrUserNotExist
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return user, nil
}
