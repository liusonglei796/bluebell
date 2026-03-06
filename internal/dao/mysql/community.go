package mysql

import (
	"bluebell/internal/model"
	"context"
	"fmt"

	"gorm.io/gorm"
)

// communityRepoStruct 社区数据访问实现
type communityRepoStruct struct {
	db *gorm.DB
}

// NewCommunityRepo 创建 communityRepoStruct 实例
func NewCommunityRepo(db *gorm.DB) *communityRepoStruct {
	return &communityRepoStruct{db: db}
}

// GetCommunityList 查询社区列表数据
func (r *communityRepoStruct) GetCommunityList(ctx context.Context) (data []*model.Community, err error) {
	data = make([]*model.Community, 0)
	err = r.db.WithContext(ctx).Select("community_id", "community_name").Find(&data).Error
	if err != nil {
		return nil, fmt.Errorf("query community list failed: %w", err)
	}
	return data, nil
}

// GetCommunityDetailByID 根据ID查询社区详情
func (r *communityRepoStruct) GetCommunityDetailByID(ctx context.Context, id int64) (community *model.Community, err error) {
	community = new(model.Community)
	err = r.db.WithContext(ctx).Where("community_id = ?", id).First(community).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("query community detail failed: %w", err)
	}
	return community, nil
}

// GetCommunitiesByIDs 根据社区ID列表批量获取社区信息
func (r *communityRepoStruct) GetCommunitiesByIDs(ctx context.Context, ids []int64) (communities []*model.Community, err error) {
	if len(ids) == 0 {
		return make([]*model.Community, 0), nil
	}

	communities = make([]*model.Community, 0, len(ids))
	err = r.db.WithContext(ctx).Where("community_id IN ?", ids).Find(&communities).Error
	if err != nil {
		return nil, fmt.Errorf("query communities by ids failed: %w", err)
	}
	return communities, nil
}
