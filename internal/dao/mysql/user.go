package mysql

import (
	"context"
	"errors"
	"fmt"

	"bluebell/internal/model"

	"gorm.io/gorm"
)

// UserDao 用户数据访问对象
type UserDao struct {
	db *gorm.DB
}

// NewUserDao 创建用户 DAO 实例
func NewUserDao(db *gorm.DB) *UserDao {
	return &UserDao{db: db}
}

// InsertUser 插入新用户
// 并发注册同名用户时，唯一索引会让数据库直接返回 1062 重复键错误（TranslateError 下为 gorm.ErrDuplicatedKey），
// 这里将其映射为 ErrUserExist，由 controller 映射为 409。单条 INSERT 由数据库兜底，不存在 check-then-insert 的竞态窗口。
func (d *UserDao) InsertUser(ctx context.Context, user *model.User) error {
	err := d.db.WithContext(ctx).Create(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return model.ErrUserExist
		}
		return fmt.Errorf("插入用户失败: %w", err)
	}
	return nil
}

// VerifyUser 校验用户名密码，验证通过后将查询到的信息填回 user
func (d *UserDao) VerifyUser(ctx context.Context, user *model.User) error {
	m := &model.User{}
	err := d.db.WithContext(ctx).Where("user_name = ?", user.UserName).First(m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ErrUserNotExist
		}
		return fmt.Errorf("登录失败: %w", err)
	}

	if !model.CheckPassword(user.Passwd, m.Passwd) {
		return model.ErrInvalidPassword
	}

	// 将查询到的信息填回
	user.UserID = m.UserID
	user.Role = m.Role
	return nil
}

// CheckUserExistsByID 根据用户ID查询用户信息
func (d *UserDao) CheckUserExistsByID(ctx context.Context, uid int64) (*model.User, error) {
	m := &model.User{}
	err := d.db.WithContext(ctx).Where("user_id = ?", uid).First(m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return m, nil
}

// GetUsersByIDs 根据用户ID列表批量获取用户信息
func (d *UserDao) GetUsersByIDs(ctx context.Context, ids []int64) (users []*model.User, err error) {
	if len(ids) == 0 {
		return make([]*model.User, 0), nil
	}

	err = d.db.WithContext(ctx).Where("user_id IN ?", ids).Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("批量查询用户失败: %w", err)
	}
	return users, nil
}

// GetUserRoleByID 根据用户ID查询用户角色
func (d *UserDao) GetUserRoleByID(ctx context.Context, uid int64) (int, error) {
	m := &model.User{}
	err := d.db.WithContext(ctx).Select("role").Where("user_id = ?", uid).First(m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, model.ErrUserNotExist
		}
		return 0, fmt.Errorf("查询用户角色失败: %w", err)
	}
	return m.Role, nil
}

// GetUserByUsername 根据用户名获取用户信息
func (d *UserDao) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	m := &model.User{}
	err := d.db.WithContext(ctx).Where("user_name = ?", username).First(m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrUserNotExist
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return m, nil
}
