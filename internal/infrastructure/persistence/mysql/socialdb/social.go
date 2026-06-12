package socialdb

import (
	"context"
	"errors"
	"fmt"

	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/persistence/mysql/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (r *socialRepoStruct) GetProfileByGitHubID(ctx context.Context, githubID string) (*entity.UserProfile, error) {
	var m model.UserProfile
	err := r.db.WithContext(ctx).Where("github_id = ?", githubID).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get profile by github id: %w", err)
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

	// 使用 ON DUPLICATE KEY UPDATE (Upsert) 保证原子性
	// 针对 user_id 冲突时，更新除 ID 外的所有字段
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"avatar_url", "bio", "github_id", "github_url", "updated_at"}),
	}).Create(&m).Error

	if err != nil {
		return fmt.Errorf("failed to save user profile: %w", err)
	}
	return nil
}

func (r *socialRepoStruct) FollowUser(ctx context.Context, followerID, followingID int64) error {
	m := model.Follow{
		FollowerID:  followerID,
		FollowingID: followingID,
	}
	// 使用 OnConflict DoNothing 保证幂等性
	// 针对 (follower_id, following_id) 联合唯一键冲突时，不做任何操作，不返回错误
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(&m).Error

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
