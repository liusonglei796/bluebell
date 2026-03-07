package communitysvc

import (
	// 领域层 - Repository 接口
	"bluebell/internal/domain/dbdomain"

	// 领域层 - Service 接口
	"bluebell/internal/domain/svcdomain"

	// DTO 响应
	communityResp "bluebell/internal/dto/response/community"

	// 模型
	"bluebell/internal/model"

	// 错误处理
	"bluebell/pkg/errorx"

	"context"
	"strconv"

	"go.uber.org/zap"
)

// communityServiceStruct 社区业务逻辑服务
type communityServiceStruct struct {
	communityRepo dbdomain.CommunityRepository
}

// NewCommunityService 创建社区服务实例
func NewCommunityService(communityRepo dbdomain.CommunityRepository) svcdomain.CommunityService {
	return &communityServiceStruct{communityRepo: communityRepo}
}

// toResponse 将 model.Community 转换为 communityResponse.Response
func toResponse(c *model.Community) *communityResp.Response {
	return &communityResp.Response{
		ID:           strconv.FormatUint(uint64(c.ID), 10),
		Name:         c.CommunityName,
		Introduction: c.Introduction,
	}
}

// GetCommunityList 获取社区列表
func (s *communityServiceStruct) GetCommunityList(ctx context.Context) ([]*communityResp.Response, error) {
	data, err := s.communityRepo.GetCommunityList(ctx)
	if err != nil {
		zap.L().Error("communityRepo.GetCommunityList failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	result := make([]*communityResp.Response, 0, len(data))
	for _, c := range data {
		result = append(result, toResponse(c))
	}
	return result, nil
}

// GetCommunityDetail 根据ID获取社区详情
func (s *communityServiceStruct) GetCommunityDetail(ctx context.Context, id int64) (*communityResp.Response, error) {
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
