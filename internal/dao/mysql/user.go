package mysql

import (
	"bluebell/internal/domain/repository"
	"bluebell/internal/model"
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserRepo 用户数据访问实现
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo 创建 UserRepo 实例
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// CheckUserExist 检查指定用户名的用户是否存在
func (r *UserRepo) CheckUserExist(ctx context.Context, username string) (err error) {
	var count int64
	err = r.db.WithContext(ctx).Model(&model.User{}).Where("user_name = ?", username).Count(&count).Error
	if err != nil {
		return err
	}
	if count > 0 {
		return repository.ErrUserExist
	}
	return nil
}

// InsertUser 插入新用户
// 密码加密已移至 Model 的 BeforeCreate 钩子中自动处理
func (r *UserRepo) InsertUser(ctx context.Context, user *model.User) (err error) {
	err = r.db.WithContext(ctx).Create(user).Error
	if err != nil {
		return fmt.Errorf("插入用户失败: %w", err)
	}
	return nil
}

// CheckLogin 登录验证
func (r *UserRepo) CheckLogin(ctx context.Context, user *model.User) (err error) {
	oPassword := user.Passwd

	err = r.db.WithContext(ctx).Where("user_name = ?", user.UserName).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repository.ErrUserNotExist
		}
		return fmt.Errorf("login failed: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Passwd), []byte(oPassword))
	if err != nil {
		return repository.ErrInvalidPassword
	}
	return nil
}

// GetUserByID 根据用户ID查询用户信息
func (r *UserRepo) GetUserByID(ctx context.Context, uid int64) (*model.User, error) {
	user := &model.User{}
	err := r.db.WithContext(ctx).Where("user_id = ?", uid).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("query user by id failed: %w", err)
	}
	return user, nil
}

// GetUsersByIDs 根据用户ID列表批量获取用户信息
func (r *UserRepo) GetUsersByIDs(ctx context.Context, ids []int64) (users []*model.User, err error) {
	if len(ids) == 0 {
		return make([]*model.User, 0), nil
	}

	users = make([]*model.User, 0, len(ids))
	err = r.db.WithContext(ctx).Where("user_id IN ?", ids).Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("query users by ids failed: %w", err)
	}
	return users, nil
}
