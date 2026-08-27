package consumer

import (
	"context"
	"encoding/json"

	"bluebell/internal/dao/mysql"
	"bluebell/internal/dao/redis"
	"bluebell/internal/model"
	"bluebell/internal/mq"
	"bluebell/internal/snowflake"
	"bluebell/pkg/event"

	"go.uber.org/zap"
)
// WorkersContainer 管理所有后台异步消费 Worker
type WorkersContainer struct {
	eventBus    *mq.EventBus
	notifDao    *mysql.NotificationDao
	notifCache  *redis.NotificationCache
	dedupCache  *redis.DedupCache
	eventLogDao *mysql.EventLogDao
	userDao     *mysql.UserDao
	postDao     *mysql.PostDao
	feedCache   *redis.FeedCache
	relationDao *mysql.RelationDao
}

// NewWorkersContainer 创建 Worker 容器
func NewWorkersContainer(
	eventBus *mq.EventBus,
	notifDao *mysql.NotificationDao,
	notifCache *redis.NotificationCache,
	dedupCache *redis.DedupCache,
	eventLogDao *mysql.EventLogDao,
	userDao *mysql.UserDao,
	postDao *mysql.PostDao,
	feedCache *redis.FeedCache,
	relationDao *mysql.RelationDao,
) *WorkersContainer {
	return &WorkersContainer{
		eventBus:    eventBus,
		notifDao:    notifDao,
		notifCache:  notifCache,
		dedupCache:  dedupCache,
		eventLogDao: eventLogDao,
		userDao:     userDao,
		postDao:     postDao,
		feedCache:   feedCache,
		relationDao: relationDao,
	}
}

// Start 启动所有业务队列的 RabbitMQ 监听消费
func (c *WorkersContainer) Start(ctx context.Context) error {
	if c.eventBus == nil {
		return nil
	}

	// 1. 监听通知业务队列 (q.notification.worker)
	if err := c.eventBus.StartQueueConsumer(ctx, mq.QueueNotification, c.handleNotificationQueue); err != nil {
		return err
	}

	// 2. 监听 Feed 业务队列 (q.feed.worker)
	if err := c.eventBus.StartQueueConsumer(ctx, mq.QueueFeed, c.handleFeedQueue); err != nil {
		return err
	}

	// 3. 监听计数统计业务队列 (q.counter.worker)
	if err := c.eventBus.StartQueueConsumer(ctx, mq.QueueCounter, c.handleCounterQueue); err != nil {
		return err
	}

	return nil
}

func (c *WorkersContainer) handleNotificationQueue(ctx context.Context, raw *event.RawEvent) error {
	switch raw.EventType {
	case event.EventTypeCommentCreated:
		return c.handleCommentCreated(ctx, raw)
	case event.EventTypeUserFollowed:
		return c.handleUserFollowed(ctx, raw)
	case event.EventTypePostPublished:
		return c.handlePostPublished(ctx, raw)
	}
	return nil
}

func (c *WorkersContainer) handleFeedQueue(ctx context.Context, raw *event.RawEvent) error {
	return nil
}

func (c *WorkersContainer) handleCounterQueue(ctx context.Context, raw *event.RawEvent) error {
	return nil
}

func (c *WorkersContainer) handleCommentCreated(ctx context.Context, raw *event.RawEvent) error {
	var payload event.CommentCreatedEvent
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return err
	}

	// 自身对自己互动无需发通知
	if payload.ReplyToUserID == 0 || payload.AuthorID == payload.ReplyToUserID {
		return nil
	}

	actor, _ := c.userDao.CheckUserExistsByID(ctx, payload.AuthorID)
	actorName := "某用户"
	if actor != nil {
		actorName = actor.UserName
	}

	extra, _ := json.Marshal(map[string]interface{}{
		"actor_name":      actorName,
		"content_preview": payload.ContentPreview,
		"post_id":         payload.PostID,
		"comment_id":      payload.CommentID,
	})

	actionType := model.ActionTypeReplyPost
	if payload.RootID > 0 {
		actionType = model.ActionTypeReplyComment
	}

	notif := &model.UserNotification{
		ID:          snowflake.GenID(),
		RecipientID: payload.ReplyToUserID,
		ActorID:     payload.AuthorID,
		ActionType:  actionType,
		EntityType:  model.EntityTypeComment,
		EntityID:    payload.CommentID,
		ExtraInfo:   string(extra),
	}

	if err := c.notifDao.CreateNotification(ctx, notif); err != nil {
		zap.L().Error("consumer: create notification failed", zap.Error(err))
		return err
	}

	// 增加 Redis 红点计数
	_ = c.notifCache.IncrUnread(ctx, payload.ReplyToUserID, "reply")
	return nil
}

func (c *WorkersContainer) handleUserFollowed(ctx context.Context, raw *event.RawEvent) error {
	var payload event.UserFollowedEvent
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return err
	}

	if payload.Action != "follow" || payload.FollowerID == payload.FollowingID {
		return nil
	}

	actor, _ := c.userDao.CheckUserExistsByID(ctx, payload.FollowerID)
	actorName := "某用户"
	if actor != nil {
		actorName = actor.UserName
	}

	extra, _ := json.Marshal(map[string]interface{}{
		"actor_name": actorName,
	})

	notif := &model.UserNotification{
		ID:          snowflake.GenID(),
		RecipientID: payload.FollowingID,
		ActorID:     payload.FollowerID,
		ActionType:  model.ActionTypeFollow,
		EntityType:  model.EntityTypeUser,
		EntityID:    payload.FollowerID,
		ExtraInfo:   string(extra),
	}

	if err := c.notifDao.CreateNotification(ctx, notif); err != nil {
		zap.L().Error("consumer: create follow notification failed", zap.Error(err))
		return err
	}

	_ = c.notifCache.IncrUnread(ctx, payload.FollowingID, "follow")
	return nil
}

func (c *WorkersContainer) handlePostPublished(ctx context.Context, raw *event.RawEvent) error {
	var payload event.PostPublishedEvent
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return err
	}

	// 获取作者所有粉丝
	followerIDs, _, err := c.relationDao.GetFollowerList(ctx, payload.AuthorID, 1, 1000)
	if err != nil || len(followerIDs) == 0 {
		return nil
	}

	actor, _ := c.userDao.CheckUserExistsByID(ctx, payload.AuthorID)
	actorName := "关注的人"
	if actor != nil {
		actorName = actor.UserName
	}

	extra, _ := json.Marshal(map[string]interface{}{
		"actor_name": actorName,
		"post_title": payload.Title,
		"post_id":    payload.PostID,
	})

	// 为每个在线或活跃粉丝写入一条通知
	for _, fid := range followerIDs {
		notif := &model.UserNotification{
			ID:          snowflake.GenID(),
			RecipientID: fid,
			ActorID:     payload.AuthorID,
			ActionType:  model.ActionTypeReplyPost,
			EntityType:  model.EntityTypePost,
			EntityID:    payload.PostID,
			ExtraInfo:   string(extra),
		}
		_ = c.notifDao.CreateNotification(ctx, notif)
		_ = c.notifCache.IncrUnread(ctx, fid, "system")
	}

	return nil
}

func (c *WorkersContainer) Close() {
	// Worker 清理工作
}
