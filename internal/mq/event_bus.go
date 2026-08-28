package mq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"bluebell/pkg/event"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)


// EventPublisher 事件发布者接口
type EventPublisher interface {
	Publish(ctx context.Context, eventType string, eventID string, actorID int64, payload interface{}) error
	Close() error
}

// EventHandler 强类型消息消费处理器签名
type EventHandler func(ctx context.Context, raw *event.RawEvent) error

// EventBus 纯 RabbitMQ 事件总线（无内存通道，所有事件严格经由 RabbitMQ 传输）
type EventBus struct {
	conn   *amqp.Connection
	ch     *amqp.Channel
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewEventBus 创建 RabbitMQ 事件总线实例
func NewEventBus(amqpURL string) (*EventBus, error) {
	if amqpURL == "" {
		return nil, errors.New("amqpURL is empty, RabbitMQ connection required")
	}

	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ failed: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open RabbitMQ channel failed: %w", err)
	}

	// 声明完整的交换机、业务工作队列、业务专属死信队列拓扑
	if err := SetupTopology(ch); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("setup RabbitMQ topology failed: %w", err)
	}

	// 设置 QoS 预取（公平分发：单消费者每次预取最多 50 条消息）
	if err := ch.Qos(50, 0, false); err != nil {
		zap.L().Warn("set rabbitmq qos failed", zap.Error(err))
	}

	bus := &EventBus{
		conn:   conn,
		ch:     ch,
		stopCh: make(chan struct{}),
	}
	zap.L().Info("Pure RabbitMQ EventBus initialized successfully")
	return bus, nil
}

// Publish 发布领域事件到 RabbitMQ Topic Exchange
func (b *EventBus) Publish(ctx context.Context, eventType string, eventID string, actorID int64, payload interface{}) error {
	if b == nil || b.ch == nil {
		return errors.New("event bus is not connected to RabbitMQ")
	}

	envelope := event.NewEnvelope(eventID, eventType, actorID, "bluebell-app", payload)
	body, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope failed: %w", err)
	}

	err = b.ch.PublishWithContext(
		ctx,
		ExchangeTopic,
		eventType, // Routing Key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // 消息持久化落盘
			MessageId:    eventID,
			Type:         eventType,
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	)
	if err != nil {
		zap.L().Error("publish event to rabbitmq failed",
			zap.String("event_type", eventType),
			zap.String("event_id", eventID),
			zap.Error(err))
		return fmt.Errorf("publish to rabbitmq failed: %w", err)
	}

	return nil
}

// StartQueueConsumer 启动对指定 RabbitMQ 队列的监听消费（支持手动 ACK 与专属死信路由）
func (b *EventBus) StartQueueConsumer(ctx context.Context, queueName string, handler EventHandler) error {
	if b == nil || b.ch == nil {
		return errors.New("event bus is not connected to RabbitMQ")
	}

	msgs, err := b.ch.ConsumeWithContext(
		ctx,
		queueName,
		"",    // consumer tag (自动生成)
		false, // auto-ack = false (手动 ACK 确保可靠性)
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("start consume queue %s failed: %w", queueName, err)
	}

	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		zap.L().Info("RabbitMQ queue consumer started", zap.String("queue", queueName))

		for {
			select {
			case <-b.stopCh:
				return
			case <-ctx.Done():
				return
			case d, ok := <-msgs:
				if !ok {
					zap.L().Warn("RabbitMQ consumer delivery channel closed", zap.String("queue", queueName))
					return
				}

				var envelope event.RawEvent
				if err := json.Unmarshal(d.Body, &envelope); err != nil {
					zap.L().Error("unmarshal queue message to RawEvent failed, nack to DLQ",
						zap.String("queue", queueName),
						zap.Error(err))
					_ = d.Nack(false, false) // 无法解析直接打入死信队列
					continue
				}

				if err := handler(ctx, &envelope); err != nil {
					zap.L().Error("handler processing message failed, nack to DLQ",
						zap.String("queue", queueName),
						zap.String("event_type", envelope.EventType),
						zap.String("event_id", envelope.EventID),
						zap.Error(err))
					_ = d.Nack(false, false) // 失败投递到业务专属死信队列
				} else {
					_ = d.Ack(false) // 消费成功手动确认
				}
			}
		}
	}()

	return nil
}

// Close 优雅关闭 RabbitMQ 连接与消费者协程
func (b *EventBus) Close() error {
	if b == nil {
		return nil
	}
	close(b.stopCh)
	if b.ch != nil {
		_ = b.ch.Close()
	}
	if b.conn != nil {
		_ = b.conn.Close()
	}
	b.wg.Wait()
	zap.L().Info("RabbitMQ EventBus closed gracefully")
	return nil
}
