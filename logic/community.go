package logic

import (
	"bluebell/models"
	"bluebell/pkg/errorx"
	"context"

	"go.uber.org/zap"
)

// CommunityService 社区业务逻辑服务
type CommunityService struct {
	communityRepo CommunityRepository
}

// NewCommunityService 创建社区服务实例
func NewCommunityService(communityRepo CommunityRepository) *CommunityService {
	return &CommunityService{communityRepo: communityRepo}
}

// GetCommunityList 获取社区列表
func (s *CommunityService) GetCommunityList(ctx context.Context) ([]*models.Community, error) {
	data, err := s.communityRepo.GetCommunityList(ctx)
	if err != nil {
		zap.L().Error("communityRepo.GetCommunityList failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}
	return data, nil
}

// GetCommunityDetail 根据ID获取社区详情
func (s *CommunityService) GetCommunityDetail(ctx context.Context, id int64) (*models.Community, error) {
	data, err := s.communityRepo.GetCommunityDetailByID(ctx, id)
	if err != nil {
		zap.L().Error("communityRepo.GetCommunityDetailByID failed",
			zap.Int64("community_id", id),
			zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	if data == nil {
		return nil, errorx.ErrNotFound
	}

	return data, nil
}
