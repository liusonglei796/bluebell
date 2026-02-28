package mysql

import (
	"bluebell/models"
	"context"
	"fmt"

	"gorm.io/gorm"
)

// GetCommunityList 查询社区列表数据
func GetCommunityList(ctx context.Context) (data []*models.Community, err error) {
	// 初始化切片，防止查询为空时返回 nil
	data = make([]*models.Community, 0)

	// GORM 使用 Find 方法查询所有记录
	err = db.WithContext(ctx).Select("community_id", "community_name").Find(&data).Error
	if err != nil {
		return nil, fmt.Errorf("query community list failed: %w", err)
	}
	return data, nil
}

// GetCommunityDetailByID 根据ID查询社区详情
func GetCommunityDetailByID(ctx context.Context, id int64) (community *models.Community, err error) {
	community = new(models.Community)

	// GORM 使用 First 查询单条记录
	err = db.WithContext(ctx).Where("community_id = ?", id).First(community).Error
	if err != nil {
		// 特殊处理：如果没有查到数据
		if err == gorm.ErrRecordNotFound {
			return nil, nil // 返回nil而不是error，让上层决定如何处理
		}
		// 其他数据库错误（如连接断开、SQL语法错误等）
		return nil, fmt.Errorf("query community detail failed: %w", err)
	}
	return community, nil
}

// GetCommunitiesByIDs 根据社区ID列表批量获取社区信息
func GetCommunitiesByIDs(ctx context.Context, ids []int64) (communities []*models.Community, err error) {
	if len(ids) == 0 {
		return make([]*models.Community, 0), nil
	}

	// GORM 使用 Where IN 查询
	communities = make([]*models.Community, 0, len(ids))
	err = db.WithContext(ctx).Where("community_id IN ?", ids).Find(&communities).Error
	if err != nil {
		return nil, fmt.Errorf("query communities by ids failed: %w", err)
	}
	return communities, nil
}
