package communitysvc

import (
	// 领域层 - Repository 接口
	"bluebell/internal/domain"

	// 领域层 - Service 接口
	"bluebell/internal/application"

	// DTO 响应
	communityResp "bluebell/internal/interfaces/http/dto/response/community"

	// 错误处理
	"bluebell/internal/domain/entity"

	"context"
	"strconv"

	"go.uber.org/zap"

	// 日志
	"bluebell/internal/infrastructure/logger"
)

// communityServiceStruct 社区业务逻辑服务
type communityServiceStruct struct {
	communityRepo domain.CommunityRepository
	userRepo      domain.UserRepository
}

// NewCommunityService 创建社区服务实例
func NewCommunityService(communityRepo domain.CommunityRepository, userRepo domain.UserRepository) application.CommunityService {
	return &communityServiceStruct{
		communityRepo: communityRepo,
		userRepo:      userRepo,
	}
}

// toResponse 将 entity.Community 转换为 communityResponse.Response
func toResponse(c *entity.Community) *communityResp.Response {
	return &communityResp.Response{
		ID:           strconv.FormatInt(c.ID, 10),
		Name:         c.CommunityName,
		Introduction: c.Introduction,
	}
}

// GetCommunityList 获取社区列表
func (s *communityServiceStruct) GetCommunityList(ctx context.Context) ([]*communityResp.Response, error) {
	data, err := s.communityRepo.GetCommunityList(ctx)
	if err != nil {
		logger.WithContext(ctx).Error("communityRepo.GetCommunityList failed", zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
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
		logger.WithContext(ctx).Error("communityRepo.GetCommunityDetailByID failed",
			zap.Int64("community_id", id),
			zap.Error(err))
		return nil, entity.Wrap(entity.ErrServerBusy, err)
	}

	if data == nil {
		return nil, entity.ErrNotFound
	}

	return toResponse(data), nil
}

// CreateCommunity 创建社区（仅管理员）
func (s *communityServiceStruct) CreateCommunity(ctx context.Context, name, introduction string, userID int64) error {
	// 1. 校验用户角色是否为管理员
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		logger.WithContext(ctx).Error("userRepo.GetUserByID failed",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}
	if user == nil || !user.IsAdmin() {
		return entity.ErrForbidden
	}

	// 2. 创建社区
	community := &entity.Community{
		CommunityName: name,
		Introduction:  introduction,
	}
	if err := s.communityRepo.CreateCommunity(ctx, community); err != nil {
		logger.WithContext(ctx).Error("communityRepo.CreateCommunity failed",
			zap.String("community_name", name),
			zap.Error(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}

	return nil
}
