// Package communitysvc 实现社区应用服务
//
// Why Application Layer?
// 应用层服务（Application Service）作为“编排者”：
// 1. 协调 CommunityRepo 和 UserRepo 两个不同的领域资源。
// 2. 将领域模型（entity.Community）转换为接口层需要的 DTO（communityResp.Response）。
// 3. 处理跨实体的业务流程（如：创建社区前先验证用户权限）。
package communitysvc

import (
	// 领域层 - Repository 接口
	"bluebell/internal/domain"

	// 领域层 - Service 接口
	"bluebell/internal/application"

	// DTO 响应
	communityResp "bluebell/internal/application/dto/response/community"

	// 错误处理
	"bluebell/internal/domain/entity"

	"context"
	"strconv"

	"go.uber.org/zap"

	// 日志
	"bluebell/internal/infrastructure/logger"
)

// communityServiceStruct 社区业务逻辑服务
// 为什么持有多个 Repository？
// 即使是简单的社区创建，也可能需要检查用户状态（UserRepo）并操作社区数据（CommunityRepo）。
// 应用层负责协调这些不同的领域模型。
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
// 为什么在这里转换？
// 接口层 DTO 包含字符串 ID（为了解决前端大整数精度问题）和特定的展示格式，
// 这些是“外部展现细节”，不应污染纯净的领域实体。
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
// 为什么这个逻辑在应用层？
// 这是一个典型的流程编排：
// 1. 获取用户信息 (Infrastructure)
// 2. 调用领域方法 user.IsAdmin() 判定权限 (Domain)
// 3. 构造并保存社区实体 (Infrastructure)
// 注意：权限判断的“规则”在 entity.User，但“先查用户再判断”的“流程”在应用层。
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
