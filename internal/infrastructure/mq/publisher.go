package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
)

// Publisher 生产者：只认信道，不认连接
type Publisher struct {
	ch *amqp.Channel
}

// NewPublisher 创建一个新的生产者
func NewPublisher(ch *amqp.Channel) *Publisher {
	return &Publisher{ch: ch}
}

// Send 通用的发送方法
func (p *Publisher) Send(ctx context.Context, exchange, routingKey string, msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	// 注入 Trace 上下文到 Headers
	headers := make(amqp.Table)
	otel.GetTextMapPropagator().Inject(ctx, AmqpHeadersCarrier(headers))

	return p.ch.PublishWithContext(ctx,
		exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			Headers:      headers,
			DeliveryMode: amqp.Transient,
			ContentType:  "application/json",
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
}

// PublishVote 发布投票消息
func (p *Publisher) PublishVote(ctx context.Context, msg *VoteMessage) error {
	return p.Send(ctx, ExchangeVote, RoutingKeyVote, msg)
}

// PublishSearch 发布搜索同步消息
func (p *Publisher) PublishSearch(ctx context.Context, msg interface{}) error {
	return p.Send(ctx, ExchangeSearch, RoutingKeySearch, msg)
}
