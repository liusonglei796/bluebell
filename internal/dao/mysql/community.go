package mysql

import (
	"context"
	"errors"
	"fmt"

	"bluebell/internal/model"

	"gorm.io/gorm"
)

// CommunityDao 社区数据访问对象
type CommunityDao struct {
	db *gorm.DB
}

// NewCommunityDao 创建社区 DAO 实例
func NewCommunityDao(db *gorm.DB) *CommunityDao {
	return &CommunityDao{db: db}
}

// GetCommunityList 查询社区列表数据
func (d *CommunityDao) GetCommunityList(ctx context.Context) (data []*model.Community, err error) {
	err = d.db.WithContext(ctx).Select("id", "community_name", "introduction").Find(&data).Error
	if err != nil {
		return nil, fmt.Errorf("查询社区列表失败: %w", err)
	}
	return data, nil
}

// GetCommunityDetailByID 根据ID查询社区详情
func (d *CommunityDao) GetCommunityDetailByID(ctx context.Context, id int64) (*model.Community, error) {
	m := new(model.Community)
	err := d.db.WithContext(ctx).Where("id = ?", id).First(m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询社区详情失败: %w", err)
	}
	return m, nil
}

// CreateCommunity 创建新社区
// 并发创建同名社区时，唯一索引会让数据库直接返回 1062 重复键错误（TranslateError 下为 gorm.ErrDuplicatedKey），
// 这里将其映射为 ErrDuplicate，由 controller 映射为 409。单条 INSERT 由数据库兜底，不存在 check-then-insert 的竞态窗口。
func (d *CommunityDao) CreateCommunity(ctx context.Context, community *model.Community) error {
	err := d.db.WithContext(ctx).Create(community).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return model.ErrDuplicate
		}
		return fmt.Errorf("创建社区失败: %w", err)
	}
	return nil
}
