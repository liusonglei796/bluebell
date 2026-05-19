package social

import (
	"bluebell/internal/application"
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/mq"
	socialResp "bluebell/internal/interfaces/http/dto/response/social"
	"context"
	"fmt"
	"time"
)

type socialService struct {
	socialRepo domain.SocialRepository
	userRepo   domain.UserRepository
	publisher  *mq.Publisher
}

func NewSocialService(socialRepo domain.SocialRepository, userRepo domain.UserRepository, publisher *mq.Publisher) application.SocialService {
	return &socialService{
		socialRepo: socialRepo,
		userRepo:   userRepo,
		publisher:  publisher,
	}
}

func (s *socialService) GetProfile(ctx context.Context, userID, currentUserID int64) (*socialResp.ProfileResponse, error) {
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	profile, err := s.socialRepo.GetUserProfile(ctx, userID)
	if err != nil {
		// If profile doesn't exist, return basic user info
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

func (s *socialService) FollowUser(ctx context.Context, followerID, followingID int64) error {
	if followerID == followingID {
		return fmt.Errorf("cannot follow yourself")
	}

	err := s.socialRepo.FollowUser(ctx, followerID, followingID)
	if err != nil {
		return err
	}

	// Emit activity message
	_ = s.publisher.PublishActivity(ctx, &mq.ActivityMessage{
		UserID:    followerID,
		Type:      "follow",
		TargetID:  fmt.Sprintf("%d", followingID),
		Timestamp: time.Now().Unix(),
	})

	return nil
}

func (s *socialService) UnfollowUser(ctx context.Context, followerID, followingID int64) error {
	err := s.socialRepo.UnfollowUser(ctx, followerID, followingID)
	if err != nil {
		return err
	}

	// Emit activity message
	_ = s.publisher.PublishActivity(ctx, &mq.ActivityMessage{
		UserID:    followerID,
		Type:      "unfollow",
		TargetID:  fmt.Sprintf("%d", followingID),
		Timestamp: time.Now().Unix(),
	})

	return nil
}

func (s *socialService) GetActivities(ctx context.Context, userID int64, page, size int) ([]*socialResp.ActivityResponse, error) {
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
