// Package userdb 实现 MySQL 基础设施层（Infrastructure Layer）
//
// Why Infrastructure Layer?
// 按照 DDD 原则，基础设施层负责具体的技术实现。
// 1. 它实现了领域层定义的 Repository 接口。
// 2. 它处理与具体技术栈（如 GORM, MySQL）相关的细节。
// 3. 它通过“数据模型（Model）”与“领域实体（Entity）”的转换，保持领域层的纯净。
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
	"strings"

	"gorm.io/gorm"
)

// userRepoStruct 用户数据访问实现
// 为什么持有 *gorm.DB？
// 基础设施层是允许直接依赖具体技术的。
// 这个结构体隐藏了所有 MySQL 操作细节，向上（领域/应用层）只暴露抽象接口。
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

// fromModelUser 将数据库模型转换为领域实体
// 为什么需要转换？
// 数据库模型（model.User）通常带有数据库标签（gorm:"..."），这属于技术细节。
// 领域实体（entity.User）应该只包含业务属性。
// 这种转换保证了即便数据库表结构发生细微调整，只要业务含义没变，领域层就无需修改。
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

// CreateUser 插入新用户
func (r *userRepoStruct) CreateUser(ctx context.Context, user *entity.User) error {
	m := &model.User{
		UserID:   user.UserID,
		UserName: user.UserName,
		Passwd:   user.Password,
		Role:     user.Role,
	}
	err := r.db.WithContext(ctx).Create(m).Error
	if err != nil {
		// 检查是否为唯一键冲突错误 (MySQL Error 1062)
		// 这样就不需要先 SELECT 检查是否存在，保证了高并发下的原子性
		if isDuplicateEntryError(err) {
			return entity.ErrUserExist
		}
		return fmt.Errorf("插入用户失败: %w", err)
	}
	return nil
}

// isDuplicateEntryError 检查错误是否为 MySQL 唯一键冲突
func isDuplicateEntryError(err error) bool {
	if err == nil {
		return false
	}
	// 常见的 MySQL 驱动错误检查方式
	return strings.Contains(err.Error(), "1062") || strings.Contains(err.Error(), "Duplicate entry")
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

// GetUserByID 根据用户ID查询用户信息
func (r *userRepoStruct) GetUserByID(ctx context.Context, uid int64) (*entity.User, error) {
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

// GetUserByName 根据用户名获取用户信息
func (r *userRepoStruct) GetUserByName(ctx context.Context, username string) (*entity.User, error) {
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
