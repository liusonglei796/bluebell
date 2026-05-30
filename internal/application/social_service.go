package application

import (
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/mq"
	socialResp "bluebell/internal/application/dto/response/social"
	"context"
	"fmt"
	"time"
)

type SocialService struct {
	socialRepo domain.SocialRepository
	userRepo   domain.UserRepository
	publisher  *mq.Publisher
}

func NewSocialService(socialRepo domain.SocialRepository, userRepo domain.UserRepository, publisher *mq.Publisher) *SocialService {
	return &SocialService{
		socialRepo: socialRepo,
		userRepo:   userRepo,
		publisher:  publisher,
	}
}

func (s *SocialService) GetProfile(ctx context.Context, userID, currentUserID int64) (*socialResp.ProfileResponse, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, entity.ErrNotFound
	}

	profile, err := s.socialRepo.GetUserProfile(ctx, userID)
	if err != nil || profile == nil {
		profile = &entity.UserProfile{UserID: userID}
	}

	followerCount, _ := s.socialRepo.GetFollowerCount(ctx, userID)
	followingCount, _ := s.socialRepo.GetFollowingCount(ctx, userID)
	
	isFollowing := false
	if currentUserID > 0 {
		isFollowing, _ = s.socialRepo.IsFollowing(ctx, currentUserID, userID)
	}

	return &socialResp.ProfileResponse{
		UserID:         user.UserID,
		Username:       user.UserName,
		AvatarURL:      profile.AvatarURL,
		Bio:            profile.Bio,
		GitHubURL:      profile.GitHubURL,
		FollowerCount:  followerCount,
		FollowingCount: followingCount,
		IsFollowing:    isFollowing,
	}, nil
}

func (s *SocialService) FollowUser(ctx context.Context, followerID, followingID int64) error {
	if followerID == followingID {
		return fmt.Errorf("cannot follow yourself")
	}

	err := s.socialRepo.FollowUser(ctx, followerID, followingID)
	if err != nil {
		return err
	}

	_ = s.publisher.PublishActivity(ctx, &mq.ActivityMessage{
		UserID:    followerID,
		Type:      "follow",
		TargetID:  fmt.Sprintf("%d", followingID),
		Timestamp: time.Now().Unix(),
	})

	return nil
}

func (s *SocialService) UnfollowUser(ctx context.Context, followerID, followingID int64) error {
	err := s.socialRepo.UnfollowUser(ctx, followerID, followingID)
	if err != nil {
		return err
	}

	_ = s.publisher.PublishActivity(ctx, &mq.ActivityMessage{
		UserID:    followerID,
		Type:      "unfollow",
		TargetID:  fmt.Sprintf("%d", followingID),
		Timestamp: time.Now().Unix(),
	})

	return nil
}

func (s *SocialService) GetActivities(ctx context.Context, userID int64, page, size int) ([]*socialResp.ActivityResponse, error) {
	activities, err := s.socialRepo.GetActivitiesByUserID(ctx, userID, page, size)
	if err != nil {
		return nil, err
	}

	resp := make([]*socialResp.ActivityResponse, 0, len(activities))
	for _, a := range activities {
		resp = append(resp, &socialResp.ActivityResponse{
			ID:          a.ID,
			UserID:      a.UserID,
			Type:        a.Type,
			TargetID:    a.TargetID,
			TargetName:  a.TargetName,
			Description: a.Description,
			CreatedAt:   a.CreatedAt.Unix(),
		})
	}

	return resp, nil
}
