package service

import (
	"context"
	"strconv"
	"time"

	"bluebell/internal/dao/mysql"
	"bluebell/internal/dao/redis"
	relationresp "bluebell/internal/dto/response/relation"
	"bluebell/internal/model"
	"bluebell/internal/mq"
	"bluebell/pkg/event"

	"go.uber.org/zap"
)

// RelationService 关注与社交关系服务
type RelationService struct {
	relationDao   *mysql.RelationDao
	userDao       *mysql.UserDao
	relationCache *redis.UserRelationCache
	eventBus      *mq.EventBus
}

// NewRelationService 创建关系服务实例
func NewRelationService(
	relationDao *mysql.RelationDao,
	userDao *mysql.UserDao,
	relationCache *redis.UserRelationCache,
	eventBus *mq.EventBus,
) *RelationService {
	return &RelationService{
		relationDao:   relationDao,
		userDao:       userDao,
		relationCache: relationCache,
		eventBus:      eventBus,
	}
}

// FollowUser 关注用户
func (s *RelationService) FollowUser(ctx context.Context, followerID, targetUserID int64) error {
	if followerID == targetUserID {
		return model.ErrInvalidParam
	}

	target, err := s.userDao.CheckUserExistsByID(ctx, targetUserID)
	if err != nil || target == nil {
		return model.ErrUserNotExist
	}

	if err := s.relationDao.Follow(ctx, followerID, targetUserID); err != nil {
		zap.L().Error("relationDao.Follow failed", zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}

	// 同步写 Redis Set 缓存
	_ = s.relationCache.AddFollowing(ctx, followerID, targetUserID)

	// 异步发布关注事件
	_ = s.eventBus.Publish(ctx, event.EventTypeUserFollowed, strconv.FormatInt(time.Now().UnixNano(), 10), followerID, event.UserFollowedEvent{
		FollowerID:  followerID,
		FollowingID: targetUserID,
		Action:      "follow",
	})

	return nil
}

// UnfollowUser 取消关注
func (s *RelationService) UnfollowUser(ctx context.Context, followerID, targetUserID int64) error {
	if err := s.relationDao.Unfollow(ctx, followerID, targetUserID); err != nil {
		zap.L().Error("relationDao.Unfollow failed", zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}

	_ = s.relationCache.RemoveFollowing(ctx, followerID, targetUserID)

	_ = s.eventBus.Publish(ctx, event.EventTypeUserFollowed, strconv.FormatInt(time.Now().UnixNano(), 10), followerID, event.UserFollowedEvent{
		FollowerID:  followerID,
		FollowingID: targetUserID,
		Action:      "unfollow",
	})

	return nil
}

// GetFollowingList 获取用户的关注列表
func (s *RelationService) GetFollowingList(ctx context.Context, targetUID, viewerUID int64, page, size int64) (*relationresp.RelationListResponse, error) {
	if size <= 0 || size > 50 {
		size = 20
	}
	if page <= 0 {
		page = 1
	}

	followingIDs, total, err := s.relationDao.GetFollowingList(ctx, targetUID, page, size)
	if err != nil {
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	return s.buildUserSummaries(ctx, followingIDs, total, viewerUID)
}

// GetFollowerList 获取用户的粉丝列表
func (s *RelationService) GetFollowerList(ctx context.Context, targetUID, viewerUID int64, page, size int64) (*relationresp.RelationListResponse, error) {
	if size <= 0 || size > 50 {
		size = 20
	}
	if page <= 0 {
		page = 1
	}

	followerIDs, total, err := s.relationDao.GetFollowerList(ctx, targetUID, page, size)
	if err != nil {
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	return s.buildUserSummaries(ctx, followerIDs, total, viewerUID)
}

func (s *RelationService) buildUserSummaries(ctx context.Context, userIDs []int64, total int64, viewerUID int64) (*relationresp.RelationListResponse, error) {
	if len(userIDs) == 0 {
		return &relationresp.RelationListResponse{Total: total, Users: make([]*relationresp.UserSummary, 0)}, nil
	}

	users, err := s.userDao.GetUsersByIDs(ctx, userIDs)
	if err != nil {
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	summaries := make([]*relationresp.UserSummary, len(users))
	for i, u := range users {
		isFollowed, isMutual, _ := s.relationCache.GetMutualStatus(ctx, viewerUID, u.UserID)
		summaries[i] = &relationresp.UserSummary{
			UserID:         strconv.FormatInt(u.UserID, 10),
			UserName:       u.UserName,
			FollowingCount: u.FollowingCount,
			FollowerCount:  u.FollowerCount,
			IsFollowed:     isFollowed,
			IsMutual:       isMutual,
			CreatedAt:      u.CreatedAt,
		}
	}

	return &relationresp.RelationListResponse{
		Total: total,
		Users: summaries,
	}, nil
}
