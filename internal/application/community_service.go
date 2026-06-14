package application

import (
	communityResp "bluebell/internal/application/dto/response/community"
	"bluebell/internal/application/port"
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	"context"
	"strconv"
)

type CommunityService struct {
	communityRepo domain.CommunityRepository
	userRepo      domain.UserRepository
	logger        port.Logger
}

func NewCommunityService(
	communityRepo domain.CommunityRepository,
	userRepo domain.UserRepository,
	logger port.Logger,
) *CommunityService {
	return &CommunityService{
		communityRepo: communityRepo,
		userRepo:      userRepo,
		logger:        logger,
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
		s.logger.Error(ctx, "communityRepo.GetCommunityList failed", port.Err(err))
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
		s.logger.Error(ctx, "communityRepo.GetCommunityDetailByID failed",
			port.Int64("community_id", id),
			port.Err(err))
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
		s.logger.Error(ctx, "userRepo.GetUserByID failed",
			port.Int64("user_id", userID),
			port.Err(err))
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
		s.logger.Error(ctx, "communityRepo.CreateCommunity failed",
			port.String("community_name", name),
			port.Err(err))
		return entity.Wrap(entity.ErrServerBusy, err)
	}

	return nil
}
