package mq

import (
	"bluebell/internal/application/port"
	"context"
)

// eventPublisherAdapter 是 port.EventPublisher 的 RabbitMQ 适配器
// 它将应用层定义的事件 DTO 转换为基础设施层的 MQ 消息并发送。
type eventPublisherAdapter struct {
	publisher *Publisher
}

// NewEventPublisher 创建 port.EventPublisher 的适配器实例
func NewEventPublisher(p *Publisher) port.EventPublisher {
	return &eventPublisherAdapter{publisher: p}
}

func (a *eventPublisherAdapter) PublishActivity(ctx context.Context, event *port.ActivityEvent) error {
	msg := &ActivityMessage{
		Type:       event.Type,
		TargetID:   event.TargetID,
		TargetName: event.TargetName,
		UserID:     event.UserID,
		Timestamp:  event.Timestamp,
	}
	return a.publisher.PublishActivity(ctx, msg)
}

func (a *eventPublisherAdapter) PublishVote(ctx context.Context, event *port.VoteEvent) error {
	msg := &VoteMessage{
		MsgID:  event.MsgID,
		PostID: event.PostID,
		UserID: event.UserID,
		Action: event.Action,
	}
	return a.publisher.PublishVote(ctx, msg)
}
