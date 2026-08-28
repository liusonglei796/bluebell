package consumer

import (
	"context"
	"encoding/json"
	"time"

	"bluebell/internal/dao/mysql"
	"bluebell/internal/dao/redis"
	"bluebell/internal/model"
	"bluebell/internal/mq"
	"bluebell/internal/snowflake"
	"bluebell/pkg/event"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	ConsumerGroupNotification = "notification_worker"
	ConsumerGroupFeed         = "feed_worker"
	ConsumerGroupCounter      = "counter_worker"

	// 大 V 粉丝数阈值：超过 500 粉丝则视为大 V，避免全量同步写扩散
	BigVFollowerThreshold = 500
)

// WorkersContainer 管理所有后台异步消费 Worker
type WorkersContainer struct {
	db          *gorm.DB
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
	db *gorm.DB,
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
		db:          db,
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

	// 4. 监听专属死信队列 (DLQ) 并输出告警日志与监控追踪
	dlqList := []string{mq.QueueNotificationDLQ, mq.QueueFeedDLQ, mq.QueueCounterDLQ}
	for _, dlqName := range dlqList {
		queue := dlqName
		if err := c.eventBus.StartQueueConsumer(ctx, queue, func(ctx context.Context, raw *event.RawEvent) error {
			return c.handleDeadLetterQueue(ctx, queue, raw)
		}); err != nil {
			zap.L().Warn("start DLQ consumer warning", zap.String("queue", queue), zap.Error(err))
		}
	}

	return nil
}

// handleDeadLetterQueue 处理死信队列消息，记录严重告警日志供可观测性系统采集
func (c *WorkersContainer) handleDeadLetterQueue(ctx context.Context, queueName string, raw *event.RawEvent) error {
	zap.L().Error("CRITICAL: Dead Letter Queue (DLQ) message received, requires investigation or manual retry",
		zap.String("dlq_queue", queueName),
		zap.String("event_id", raw.EventID),
		zap.String("event_type", raw.EventType),
		zap.Int64("actor_id", raw.ActorID),
		zap.String("producer", raw.Producer),
		zap.Int64("timestamp", raw.Timestamp),
	)
	return nil
}

func (c *WorkersContainer) handleNotificationQueue(ctx context.Context, raw *event.RawEvent) error {
	// 1. Redis 前置快速防重拦截
	ok, err := c.dedupCache.AcquireEventLock(ctx, ConsumerGroupNotification, raw.EventID, 60*time.Second)
	if err != nil || !ok {
		return nil // 已有协程正在处理或已处理，直接跳过
	}
	defer func() {
		_ = c.dedupCache.MarkEventDone(ctx, ConsumerGroupNotification, raw.EventID, 24*time.Hour)
	}()

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
	// 1. Redis 前置快速防重拦截
	ok, err := c.dedupCache.AcquireEventLock(ctx, ConsumerGroupFeed, raw.EventID, 60*time.Second)
	if err != nil || !ok {
		return nil
	}
	defer func() {
		_ = c.dedupCache.MarkEventDone(ctx, ConsumerGroupFeed, raw.EventID, 24*time.Hour)
	}()

	switch raw.EventType {
	case event.EventTypePostPublished:
		return c.handleFeedPostPublished(ctx, raw)
	case event.EventTypeUserFollowed:
		return c.handleFeedUserFollowed(ctx, raw)
	}
	return nil
}

func (c *WorkersContainer) handleCounterQueue(ctx context.Context, raw *event.RawEvent) error {
	// 1. Redis 前置快速防重拦截
	ok, err := c.dedupCache.AcquireEventLock(ctx, ConsumerGroupCounter, raw.EventID, 60*time.Second)
	if err != nil || !ok {
		return nil
	}
	defer func() {
		_ = c.dedupCache.MarkEventDone(ctx, ConsumerGroupCounter, raw.EventID, 24*time.Hour)
	}()

	switch raw.EventType {
	case event.EventTypeVoteCast:
		return c.handleCounterVoteCast(ctx, raw)
	}
	return nil
}

// handleFeedPostPublished 当发布新帖时，采用推拉结合 + Pipeline 批量写入粉丝 Feed 流
func (c *WorkersContainer) handleFeedPostPublished(ctx context.Context, raw *event.RawEvent) error {
	var payload event.PostPublishedEvent
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return err
	}

	// 查询粉丝总数与第一批粉丝
	followerIDs, total, err := c.relationDao.GetFollowerList(ctx, payload.AuthorID, 1, 1000)
	if err != nil || len(followerIDs) == 0 {
		return nil
	}

	now := time.Now().Unix()

	// 1. 如果是大 V (粉丝量超出阈值)，限制单次最大推扩散人数（如前 500 名活跃粉丝），其余走读扩散拉取
	pushTargets := followerIDs
	if total > BigVFollowerThreshold && len(pushTargets) > BigVFollowerThreshold {
		pushTargets = pushTargets[:BigVFollowerThreshold]
		zap.L().Info("author is Big-V, apply hybrid push-pull feed strategy",
			zap.Int64("author_id", payload.AuthorID),
			zap.Int64("total_followers", total),
			zap.Int("pushed_count", len(pushTargets)),
		)
	}

	// 2. 利用 Pipeline 批量推入 Feed 缓存
	_ = c.feedCache.BatchPushFeed(ctx, pushTargets, payload.PostID, now, 10*time.Minute)

	return nil
}

// handleFeedUserFollowed 当关注新用户时，将新关注者近期的动态写入关注者的 Feed 流
func (c *WorkersContainer) handleFeedUserFollowed(ctx context.Context, raw *event.RawEvent) error {
	var payload event.UserFollowedEvent
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return err
	}

	if payload.Action != "follow" || payload.FollowerID == payload.FollowingID {
		return nil
	}

	// 异步拉取被关注者近期的帖子并写入关注者的 Feed 流
	posts, err := c.postDao.GetPostListByAuthorIDs(ctx, []int64{payload.FollowingID}, 20)
	if err != nil || len(posts) == 0 {
		return nil
	}

	postIDs := make([]int64, len(posts))
	timestamps := make([]int64, len(posts))
	for i, p := range posts {
		postIDs[i] = int64(p.ID)
		timestamps[i] = p.CreatedAt.Unix()
	}

	_ = c.feedCache.SetUserFeed(ctx, payload.FollowerID, postIDs, timestamps, 10*time.Minute)
	return nil
}

// handleCounterVoteCast 异步消费投票事件记录及同步统计（支持强幂等落库）
func (c *WorkersContainer) handleCounterVoteCast(ctx context.Context, raw *event.RawEvent) error {
	var payload event.VoteCastEvent
	if err := json.Unmarshal(raw.Payload, &payload); err != nil {
		return err
	}

	// 本地 MySQL 事务：记录强幂等 processed_events
	if c.db != nil {
		err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return c.eventLogDao.InsertProcessedEvent(ctx, tx, raw.EventID, ConsumerGroupCounter, raw.EventType)
		})
		if err != nil {
			zap.L().Warn("counter event already processed in DB transaction, skipping",
				zap.String("event_id", raw.EventID),
				zap.Error(err))
			return nil
		}
	}

	zap.L().Info("consumed vote event for counter aggregation",
		zap.Int64("post_id", payload.PostID),
		zap.Int64("user_id", payload.UserID),
		zap.Int8("direction", payload.Direction),
	)

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

	// 本地事务：原子落库 Notification 与 ProcessedEvent
	if c.db != nil {
		err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := c.notifDao.CreateNotificationInTx(ctx, tx, notif); err != nil {
				return err
			}
			return c.eventLogDao.InsertProcessedEvent(ctx, tx, raw.EventID, ConsumerGroupNotification, raw.EventType)
		})
		if err != nil {
			zap.L().Error("consumer: atomic create notification & event log failed", zap.Error(err))
			return err
		}
	} else {
		if err := c.notifDao.CreateNotification(ctx, notif); err != nil {
			zap.L().Error("consumer: create notification failed", zap.Error(err))
			return err
		}
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

	// 本地事务：原子落库 Notification 与 ProcessedEvent
	if c.db != nil {
		err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := c.notifDao.CreateNotificationInTx(ctx, tx, notif); err != nil {
				return err
			}
			return c.eventLogDao.InsertProcessedEvent(ctx, tx, raw.EventID, ConsumerGroupNotification, raw.EventType)
		})
		if err != nil {
			zap.L().Error("consumer: atomic create follow notification & event log failed", zap.Error(err))
			return err
		}
	} else {
		if err := c.notifDao.CreateNotification(ctx, notif); err != nil {
			zap.L().Error("consumer: create follow notification failed", zap.Error(err))
			return err
		}
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
