package communitydb

import (
	// 模型
	"bluebell/internal/infrastructure/persistence/mysql/model"

	// 领域层
	"bluebell/internal/domain"

	"context"
	"errors"
	"fmt"

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

// GetCommunityList 查询社区列表数据
func (r *communityRepoStruct) GetCommunityList(ctx context.Context) (data []*model.Community, err error) {
	data = make([]*model.Community, 0)
	err = r.db.WithContext(ctx).Select("id", "community_name", "introduction").Find(&data).Error
	if err != nil {
		return nil, fmt.Errorf("查询社区列表失败: %w", err)
	}
	return data, nil
}

// GetCommunityDetailByID 根据ID查询社区详情
func (r *communityRepoStruct) GetCommunityDetailByID(ctx context.Context, id int64) (community *model.Community, err error) {
	community = new(model.Community)
	err = r.db.WithContext(ctx).Where("id = ?", id).First(community).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("查询社区详情失败: %w", err)
	}
	return community, nil
}

// CreateCommunity 创建新社区
func (r *communityRepoStruct) CreateCommunity(ctx context.Context, community *model.Community) error {
	err := r.db.WithContext(ctx).Create(community).Error
	if err != nil {
		return fmt.Errorf("创建社区失败: %w", err)
	}
	return nil
}
