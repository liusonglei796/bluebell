// Package social 实现社交应用服务
//
// Why Application Layer?
// 社交服务的应用层负责处理“社交交互流”：
// 1. 聚合多维数据：获取用户信息、个人资料、关注状态等。
// 2. 协调异步行为：在执行关注/取消关注后，负责向消息队列（MQ）发送动态。
package social

import (
	"bluebell/internal/application"
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/mq"
	socialResp "bluebell/internal/application/dto/response/social"
	"context"
	"fmt"
	"time"
)

// socialService 社交业务逻辑服务
// 为什么它持有 Publisher？
// 应用层负责处理业务流程产生的副效应（Side Effects），如发送异步消息。
// 领域层只负责逻辑判断，而发送消息属于“流程”的一部分。
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

// GetProfile 获取用户个人资料
// 为什么这个逻辑在应用层？
// 这是一个数据聚合过程：它从 UserRepo 获取核心信息，从 SocialRepo 获取资料和统计数据。
// 应用层负责将这些散落在各处的数据拼装成一个完整的 Profile 视图返回。
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

// FollowUser 关注用户
// 为什么在这里处理 MQ 发送？
// 关注动作是一个完整的用例：
// 1. 持久化关系 (Infrastructure)
// 2. 发送动态消息 (MQ/Infrastructure)
// 应用层保证了这两个步骤的顺序执行。
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
