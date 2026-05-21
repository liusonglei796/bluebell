package communitydb

import (
	// 模型
	"bluebell/internal/infrastructure/persistence/mysql/model"

	// 领域层
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"

	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

// communityRepoStruct 社区数据访问实现
type communityRepoStruct struct {
	db *gorm.DB
}

// NewCommunityRepo 创建 communityRepoStruct 实例
func NewCommunityRepo(db *gorm.DB) domain.CommunityRepository {
	return &communityRepoStruct{db: db}
}

// toModelCommunity 将领域实体转换为数据库模型
func toModelCommunity(c *entity.Community) *model.Community {
	if c == nil {
		return nil
	}
	return &model.Community{
		Model:         gorm.Model{ID: uint(c.ID)},
		CommunityName: c.CommunityName,
		Introduction:  c.Introduction,
	}
}

// fromModelCommunity 将数据库模型转换为领域实体
func fromModelCommunity(m *model.Community) *entity.Community {
	if m == nil {
		return nil
	}
	return &entity.Community{
		ID:            int64(m.ID),
		CommunityName: m.CommunityName,
		Introduction:  m.Introduction,
	}
}

// GetCommunityList 查询社区列表数据
func (r *communityRepoStruct) GetCommunityList(ctx context.Context) (data []*entity.Community, err error) {
	var mList []*model.Community
	err = r.db.WithContext(ctx).Select("id", "community_name", "introduction").Find(&mList).Error
	if err != nil {
		return nil, fmt.Errorf("查询社区列表失败: %w", err)
	}

	data = make([]*entity.Community, 0, len(mList))
	for _, m := range mList {
		data = append(data, fromModelCommunity(m))
	}
	return data, nil
}

// GetCommunityDetailByID 根据ID查询社区详情
func (r *communityRepoStruct) GetCommunityDetailByID(ctx context.Context, id int64) (*entity.Community, error) {
	m := new(model.Community)
	err := r.db.WithContext(ctx).Where("id = ?", id).First(m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询社区详情失败: %w", err)
	}
	return fromModelCommunity(m), nil
}

// CreateCommunity 创建新社区
func (r *communityRepoStruct) CreateCommunity(ctx context.Context, community *entity.Community) error {
	m := toModelCommunity(community)
	err := r.db.WithContext(ctx).Create(m).Error
	if err != nil {
		if isDuplicateEntryError(err) {
			return entity.ErrCommunityExist
		}
		return fmt.Errorf("创建社区失败: %w", err)
	}
	return nil
}

// isDuplicateEntryError 检查错误是否为 MySQL 唯一键冲突
func isDuplicateEntryError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "1062") || strings.Contains(err.Error(), "Duplicate entry")
}

