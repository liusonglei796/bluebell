package socialdb

import (
	"context"
	"errors"
	"fmt"

	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/persistence/mysql/model"

	"gorm.io/gorm"
)

type socialRepoStruct struct {
	db *gorm.DB
}

func NewSocialRepo(db *gorm.DB) domain.SocialRepository {
	return &socialRepoStruct{db: db}
}

func (r *socialRepoStruct) GetUserProfile(ctx context.Context, userID int64) (*entity.UserProfile, error) {
	var m model.UserProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	return &entity.UserProfile{
		UserID:    m.UserID,
		AvatarURL: m.AvatarURL,
		Bio:       m.Bio,
		GitHubID:  m.GitHubID,
		GitHubURL: m.GitHubURL,
	}, nil
}

func (r *socialRepoStruct) SaveUserProfile(ctx context.Context, profile *entity.UserProfile) error {
	m := model.UserProfile{
		UserID:    profile.UserID,
		AvatarURL: profile.AvatarURL,
		Bio:       profile.Bio,
		GitHubID:  profile.GitHubID,
		GitHubURL: profile.GitHubURL,
	}

	// Use Save to update or create
	var existing model.UserProfile
	err := r.db.WithContext(ctx).Where("user_id = ?", profile.UserID).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return r.db.WithContext(ctx).Create(&m).Error
		}
		return err
	}

	m.ID = existing.ID // Keep existing ID
	return r.db.WithContext(ctx).Save(&m).Error
}

func (r *socialRepoStruct) FollowUser(ctx context.Context, followerID, followingID int64) error {
	m := model.Follow{
		FollowerID:  followerID,
		FollowingID: followingID,
	}
	err := r.db.WithContext(ctx).Create(&m).Error
	if err != nil {
		return fmt.Errorf("failed to follow user: %w", err)
	}
	return nil
}

func (r *socialRepoStruct) UnfollowUser(ctx context.Context, followerID, followingID int64) error {
	err := r.db.WithContext(ctx).Where("follower_id = ? AND following_id = ?", followerID, followingID).Delete(&model.Follow{}).Error
	if err != nil {
		return fmt.Errorf("failed to unfollow user: %w", err)
	}
	return nil
}

func (r *socialRepoStruct) IsFollowing(ctx context.Context, followerID, followingID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Follow{}).Where("follower_id = ? AND following_id = ?", followerID, followingID).Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check follow status: %w", err)
	}
	return count > 0, nil
}

func (r *socialRepoStruct) GetFollowerCount(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Follow{}).Where("following_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *socialRepoStruct) GetFollowingCount(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Follow{}).Where("follower_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *socialRepoStruct) CreateActivity(ctx context.Context, activity *entity.Activity) error {
	m := model.Activity{
		UserID:      activity.UserID,
		Type:        activity.Type,
		TargetID:    activity.TargetID,
		TargetName:  activity.TargetName,
		Description: activity.Description,
	}
	err := r.db.WithContext(ctx).Create(&m).Error
	if err != nil {
		return fmt.Errorf("failed to create activity: %w", err)
	}
	activity.ID = m.ID
	activity.CreatedAt = m.CreatedAt
	return nil
}

func (r *socialRepoStruct) GetActivitiesByUserID(ctx context.Context, userID int64, page, size int) ([]*entity.Activity, error) {
	var models []model.Activity
	offset := (page - 1) * size
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Offset(offset).Limit(size).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get activities: %w", err)
	}

	activities := make([]*entity.Activity, len(models))
	for i, m := range models {
		activities[i] = &entity.Activity{
			ID:          m.ID,
			UserID:      m.UserID,
			Type:        m.Type,
			TargetID:    m.TargetID,
			TargetName:  m.TargetName,
			Description: m.Description,
			CreatedAt:   m.CreatedAt,
		}
	}
	return activities, nil
}
