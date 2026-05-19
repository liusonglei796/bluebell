package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
)

// ActivityConsumer 用户动态消息消费者
type ActivityConsumer struct {
	ch         *amqp.Channel
	socialRepo domain.SocialRepository
}

// NewActivityConsumer 创建一个新的用户动态消费者
func NewActivityConsumer(ch *amqp.Channel, socialRepo domain.SocialRepository) *ActivityConsumer {
	return &ActivityConsumer{ch: ch, socialRepo: socialRepo}
}

// Start 启动监听
func (c *ActivityConsumer) Start(ctx context.Context) error {
	if err := c.ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("设置 QoS 失败: %w", err)
	}

	msgs, err := c.ch.Consume(QueueActivity, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("消费队列 %s 失败: %w", QueueActivity, err)
	}

	log.Printf("用户动态消费者已启动: %s", QueueActivity)

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("消费者通道已关闭")
			}
			if err := c.handleDelivery(ctx, d); err != nil {
				log.Printf("处理用户动态消息失败: %v", err)
				_ = d.Nack(false, false)
			} else {
				_ = d.Ack(false)
			}
		}
	}
}

func (c *ActivityConsumer) handleDelivery(ctx context.Context, d amqp.Delivery) error {
	// 从 Headers 提取 Trace 上下文
	ctx = otel.GetTextMapPropagator().Extract(ctx, AmqpHeadersCarrier(d.Headers))

	var msg ActivityMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		return fmt.Errorf("activity_consumer: 反序列化消息失败: %w", err)
	}

	// 转换为领域实体
	activity := &entity.Activity{
		UserID:     msg.UserID,
		Type:       msg.Type,
		TargetID:   msg.TargetID,
		TargetName: msg.TargetName,
		CreatedAt:  time.Unix(msg.Timestamp, 0),
	}

	// 保存到数据库
	if err := c.socialRepo.CreateActivity(ctx, activity); err != nil {
		return fmt.Errorf("activity_consumer: 保存用户动态失败: %w", err)
	}

	return nil
}
