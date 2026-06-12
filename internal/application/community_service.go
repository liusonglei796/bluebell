package application

import (
	communityResp "bluebell/internal/application/dto/response/community"
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/logger"
	"context"
	"go.uber.org/zap"
	"strconv"
)

type CommunityService struct {
	communityRepo domain.CommunityRepository
	userRepo      domain.UserRepository
}

func NewCommunityService(communityRepo domain.CommunityRepository, userRepo domain.UserRepository) *CommunityService {
	return &CommunityService{
		communityRepo: communityRepo,
		userRepo:      userRepo,
	}
}

func toResponse(c *entity.Community) *communityResp.Response {
	return &communityResp.Response{
		ID:           strconv.FormatInt(c.ID, 10),
		Name:         c.CommunityName,
		Introduction: c.Introduction,
	}
}

func (s *CommunityService) GetCommunityList(ctx context.Context) ([]*communityResp.Response, error) {
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

func (s *CommunityService) GetCommunityDetail(ctx context.Context, id int64) (*communityResp.Response, error) {
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

func (s *CommunityService) CreateCommunity(ctx context.Context, name, introduction string, userID int64) error {
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
