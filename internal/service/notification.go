package service

import (
	"context"
	"encoding/json"
	"strconv"

	"bluebell/internal/dao/mysql"
	"bluebell/internal/dao/redis"
	notifreq "bluebell/internal/dto/request/notification"
	notifresp "bluebell/internal/dto/response/notification"
	"bluebell/internal/model"

	"go.uber.org/zap"
)

// NotificationService 通知中心服务
type NotificationService struct {
	notifDao   *mysql.NotificationDao
	notifCache *redis.NotificationCache
	userDao    *mysql.UserDao
}

// NewNotificationService 创建通知服务实例
func NewNotificationService(
	notifDao *mysql.NotificationDao,
	notifCache *redis.NotificationCache,
	userDao *mysql.UserDao,
) *NotificationService {
	return &NotificationService{
		notifDao:   notifDao,
		notifCache: notifCache,
		userDao:    userDao,
	}
}

// GetNotificationList 获取通知列表
func (s *NotificationService) GetNotificationList(ctx context.Context, userID int64, p *notifreq.NotificationListRequest) (*notifresp.NotificationListResponse, error) {
	if p.Size <= 0 || p.Size > 50 {
		p.Size = 20
	}
	if p.Page <= 0 {
		p.Page = 1
	}

	notifs, total, err := s.notifDao.GetNotifications(ctx, userID, p.ActionType, p.Page, p.Size)
	if err != nil {
		zap.L().Error("notifDao.GetNotifications failed", zap.Error(err))
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	unreadCount, _ := s.notifDao.GetUnreadCount(ctx, userID)

	actorIDs := make([]int64, len(notifs))
	for i, n := range notifs {
		actorIDs[i] = n.ActorID
	}
	actors, _ := s.userDao.GetUsersByIDs(ctx, actorIDs)
	actorMap := make(map[int64]string, len(actors))
	for _, a := range actors {
		actorMap[a.UserID] = a.UserName
	}

	items := make([]*notifresp.NotificationItem, len(notifs))
	for i, n := range notifs {
		var extraMap map[string]interface{}
		if n.ExtraInfo != "" {
			_ = json.Unmarshal([]byte(n.ExtraInfo), &extraMap)
		}

		actorName := actorMap[n.ActorID]
		if actorName == "" {
			actorName = "用户"
		}

		items[i] = &notifresp.NotificationItem{
			ID:         strconv.FormatInt(n.ID, 10),
			ActorID:    strconv.FormatInt(n.ActorID, 10),
			ActorName:  actorName,
			ActionType: n.ActionType,
			EntityType: n.EntityType,
			EntityID:   strconv.FormatInt(n.EntityID, 10),
			ExtraInfo:  extraMap,
			IsRead:     n.IsRead == 1,
			CreatedAt:  n.CreatedAt,
		}
	}

	return &notifresp.NotificationListResponse{
		Total:         total,
		UnreadCount:   unreadCount,
		Notifications: items,
	}, nil
}

// GetUnreadCount 获取未读红点数
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID int64) (*notifresp.UnreadCountResponse, error) {
	counts, err := s.notifCache.GetUnreadCounts(ctx, userID)
	if err != nil || len(counts) == 0 {
		// 回退查 DB
		total, _ := s.notifDao.GetUnreadCount(ctx, userID)
		return &notifresp.UnreadCountResponse{Total: total}, nil
	}

	return &notifresp.UnreadCountResponse{
		Total:  counts["total"],
		Reply:  counts["reply"],
		Like:   counts["like"],
		Follow: counts["follow"],
		System: counts["system"],
	}, nil
}

// MarkAsRead 标记已读
func (s *NotificationService) MarkAsRead(ctx context.Context, userID int64, p *notifreq.MarkReadRequest) error {
	if err := s.notifDao.MarkAsRead(ctx, userID, p.NotificationIDs, p.ActionType, p.All); err != nil {
		return model.Wrap(model.ErrServerBusy, err)
	}

	if p.All {
		_ = s.notifCache.ResetAllUnread(ctx, userID)
	} else if p.ActionType == model.ActionTypeReplyPost || p.ActionType == model.ActionTypeReplyComment {
		_ = s.notifCache.ResetCategoryUnread(ctx, userID, "reply")
	} else if p.ActionType == model.ActionTypeLikePost || p.ActionType == model.ActionTypeLikeComment {
		_ = s.notifCache.ResetCategoryUnread(ctx, userID, "like")
	} else if p.ActionType == model.ActionTypeFollow {
		_ = s.notifCache.ResetCategoryUnread(ctx, userID, "follow")
	}

	return nil
}
