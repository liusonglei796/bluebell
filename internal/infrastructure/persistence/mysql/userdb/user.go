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

// toModelUser 将领域实体转换为数据库模型
func toModelUser(u *entity.User) *model.User {
	if u == nil {
		return nil
	}
	return &model.User{
		UserID:   u.UserID,
		UserName: u.UserName,
		Passwd:   u.Password,
		Role:     u.Role,
	}
}

// fromModelUser 将数据库模型转换为领域实体
func fromModelUser(m *model.User) *entity.User {
	if m == nil {
		return nil
	}
	return &entity.User{
		UserID:   m.UserID,
		UserName: m.UserName,
		Password: m.Passwd,
		Role:     m.Role,
	}
}

// InsertUser 插入新用户
func (r *userRepoStruct) InsertUser(ctx context.Context, user *entity.User) (err error) {
	m := toModelUser(user)
	err = r.db.WithContext(ctx).Create(m).Error
	if err != nil {
		return fmt.Errorf("插入用户失败: %w", err)
	}
	return nil
}

func (r *userRepoStruct) VerifyUser(ctx context.Context, user *entity.User) (err error) {
	m := &model.User{}
	err = r.db.WithContext(ctx).Where("user_name = ?", user.UserName).First(m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrUserNotExist
		}
		return fmt.Errorf("登录失败: %w", err)
	}

	if !entity.CheckPassword(user.Password, m.Passwd) {
		return entity.ErrInvalidPassword
	}

	// 将查询到的信息填回 entity
	user.UserID = m.UserID
	user.Role = m.Role
	return nil
}

// CheckUserExistsByID 根据用户ID查询用户信息
func (r *userRepoStruct) CheckUserExistsByID(ctx context.Context, uid int64) (*entity.User, error) {
	m := &model.User{}
	err := r.db.WithContext(ctx).Where("user_id = ?", uid).First(m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return fromModelUser(m), nil
}

// GetUsersByIDs 根据用户ID列表批量获取用户信息
func (r *userRepoStruct) GetUsersByIDs(ctx context.Context, ids []int64) (users []*entity.User, err error) {
	if len(ids) == 0 {
		return make([]*entity.User, 0), nil
	}

	var mUsers []*model.User
	err = r.db.WithContext(ctx).Where("user_id IN ?", ids).Find(&mUsers).Error
	if err != nil {
		return nil, fmt.Errorf("批量查询用户失败: %w", err)
	}

	users = make([]*entity.User, 0, len(mUsers))
	for _, m := range mUsers {
		users = append(users, fromModelUser(m))
	}
	return users, nil
}

// GetUserRoleByID 根据用户ID查询用户角色
func (r *userRepoStruct) GetUserRoleByID(ctx context.Context, uid int64) (int, error) {
	m := &model.User{}
	err := r.db.WithContext(ctx).Select("role").Where("user_id = ?", uid).First(m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, entity.ErrUserNotExist
		}
		return 0, fmt.Errorf("查询用户角色失败: %w", err)
	}
	return m.Role, nil
}

// GetUserByUsername 根据用户名获取用户信息
func (r *userRepoStruct) GetUserByUsername(ctx context.Context, username string) (*entity.User, error) {
	m := &model.User{}
	err := r.db.WithContext(ctx).Where("user_name = ?", username).First(m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, entity.ErrUserNotExist
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return fromModelUser(m), nil
}
