package community

import (
	"bluebell/internal/domain/repointerface"
	"bluebell/internal/dto/response"
	"bluebell/internal/model"
	"bluebell/pkg/errorx"
	"context"
	"strconv"

	"go.uber.org/zap"
)

// communityServiceStruct 社区业务逻辑服务
type communityServiceStruct struct {
	communityRepo repointerface.CommunityRepository
}

// NewCommunityService 创建社区服务实例
func NewCommunityService(communityRepo repointerface.CommunityRepository) *communityServiceStruct {
	return &communityServiceStruct{communityRepo: communityRepo}
}

// toResponse 将 model.Community 转换为 response.CommunityResponse
func toResponse(c *model.Community) *response.CommunityResponse {
	return &response.CommunityResponse{
		ID:           strconv.FormatUint(uint64(c.ID), 10),
		Name:         c.CommunityName,
		Introduction: c.Introduction,
		CreateTime:   c.CreatedAt,
	}
}

// GetCommunityList 获取社区列表
func (s *communityServiceStruct) GetCommunityList(ctx context.Context) ([]*response.CommunityResponse, error) {
	data, err := s.communityRepo.GetCommunityList(ctx)
	if err != nil {
		zap.L().Error("communityRepo.GetCommunityList failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	result := make([]*response.CommunityResponse, 0, len(data))
	for _, c := range data {
		result = append(result, toResponse(c))
	}
	return result, nil
}

// GetCommunityDetail 根据ID获取社区详情
func (s *communityServiceStruct) GetCommunityDetail(ctx context.Context, id int64) (*response.CommunityResponse, error) {
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

	return toResponse(data), nil
}
