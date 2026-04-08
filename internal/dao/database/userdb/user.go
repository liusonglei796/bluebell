package userdb

import (
	// 模型
	"bluebell/internal/model"

	// 领域层
	"bluebell/internal/domain/dbdomain"

	// 错误处理
	"bluebell/pkg/errorx"

	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// userRepoStruct 用户数据访问实现
type userRepoStruct struct {
	db *gorm.DB
}

// NewUserRepo 创建 userRepoStruct 实例
func NewUserRepo(db *gorm.DB) dbdomain.UserRepository {
	return &userRepoStruct{db: db}
}

// CheckUserExist 检查指定用户名的用户是否存在
func (r *userRepoStruct) CheckUserExist(ctx context.Context, username string) (err error) {
	var count int64
	err = r.db.WithContext(ctx).Model(&model.User{}).Where("user_name = ?", username).Count(&count).Error
	if err != nil {
		return errorx.Wrap(err, errorx.CodeDBError, "查询用户失败")
	}
	if count > 0 {
		return errorx.ErrUserExist
	}
	return nil
}

// InsertUser 插入新用户
// 密码加密已移至 Model 的 BeforeCreate 钩子中自动处理
func (r *userRepoStruct) InsertUser(ctx context.Context, user *model.User) (err error) {
	err = r.db.WithContext(ctx).Create(user).Error
	if err != nil {
		return errorx.Wrap(err, errorx.CodeDBError, "插入用户失败")
	}
	return nil
}

func (r *userRepoStruct) VerifyUser(ctx context.Context, user *model.User) (err error) {
	oPassword := user.Passwd
	err = r.db.WithContext(ctx).Where("user_name = ?", user.UserName).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorx.ErrUserNotExist
		}
		return errorx.Wrap(err, errorx.CodeDBError, "登录失败")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Passwd), []byte(oPassword))
	if err != nil {
		return errorx.ErrInvalidPassword
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
		return nil, errorx.Wrap(err, errorx.CodeDBError, "查询用户失败")
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
		return nil, errorx.Wrap(err, errorx.CodeDBError, "批量查询用户失败")
	}
	return users, nil
}

// GetUserRoleByID 根据用户ID查询用户角色
func (r *userRepoStruct) GetUserRoleByID(ctx context.Context, uid int64) (int, error) {
	user := &model.User{}
	err := r.db.WithContext(ctx).Select("role").Where("user_id = ?", uid).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errorx.ErrUserNotExist
		}
		return 0, errorx.Wrap(err, errorx.CodeDBError, "查询用户角色失败")
	}
	return user.Role, nil
}

// GetUserByUsername 根据用户名获取用户信息
func (r *userRepoStruct) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	user := &model.User{}
	err := r.db.WithContext(ctx).Where("user_name = ?", username).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorx.ErrUserNotExist
		}
		return nil, errorx.Wrap(err, errorx.CodeDBError, "查询用户失败")
	}
	return user, nil
}
