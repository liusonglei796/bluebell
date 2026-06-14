package port

import "context"

// ========== 事件 DTO（应用层定义的事件数据结构） ==========
//
// 这些结构体属于应用层，描述了"发生了什么业务事件"。
// 具体的序列化、传输协议（RabbitMQ / Kafka / gRPC）由基础设施层负责。

// ActivityEvent 用户动态事件
// 当用户执行发帖、投票、关注等操作时产生。
type ActivityEvent struct {
	Type       string // "post_created", "vote_up", "follow", etc.
	TargetID   string
	TargetName string
	UserID     int64
	Timestamp  int64
}

// VoteEvent 投票异步计数事件
// 通过消息队列异步消费后更新 Redis 中的投票计数。
type VoteEvent struct {
	MsgID  string
	PostID string
	UserID string
	Action int // 1=upvote, -1=downvote
}

// ========== 出站端口接口 ==========

// EventPublisher 事件发布端口
//
// 应用层通过此接口发布领域事件，不直接依赖 RabbitMQ / Kafka 等具体实现。
// 基础设施层提供适配器，将事件 DTO 序列化并发送到具体的消息中间件。
type EventPublisher interface {
	// PublishActivity 发布用户动态事件
	PublishActivity(ctx context.Context, event *ActivityEvent) error
	// PublishVote 发布投票事件
	PublishVote(ctx context.Context, event *VoteEvent) error
}
