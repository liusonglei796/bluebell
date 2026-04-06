package mq

import (
	"bluebell/pkg/errorx"
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// ==================== MQPublisher ====================

type MQPublisher struct {
	channel *amqp.Channel
}

func newPublisher(conn *MQConnection) *MQPublisher {
	return &MQPublisher{channel: conn.Channel()}
}

func (p *MQPublisher) publish(ctx context.Context, exchange, routingKey string, msg any, logName string) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "marshal %s message failed", logName)
	}

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(
		publishCtx, exchange, routingKey, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		zap.L().Error("publish message failed",
			zap.String("exchange", exchange),
			zap.String("routing_key", routingKey),
			zap.Error(err),
		)
		return errorx.Wrapf(err, errorx.CodeInfraError, "publish %s to %s failed", logName, exchange)
	}

	zap.L().Debug("publish success", zap.String("exchange", exchange), zap.Int("size", len(body)))
	return nil
}

func (p *MQPublisher) PublishVote(ctx context.Context, msg *VoteMessage) error {
	return p.publish(ctx, ExchangeVote, RoutingKeyVote, msg, "vote")
}

func (p *MQPublisher) PublishAudit(ctx context.Context, msg *AuditMessage) error {
	routingKey := RoutingKeyAuditPost
	if msg.Type == "remark" {
		routingKey = RoutingKeyAuditRemark
	}
	return p.publish(ctx, ExchangeAudit, routingKey, msg, "audit")
}

func (p *MQPublisher) PublishSearch(ctx context.Context, msg any) error {
	return p.publish(ctx, ExchangeSearch, RoutingKeySearch, msg, "search")
}

func (p *MQPublisher) Close() error {
	if p.channel == nil {
		return nil
	}
	if err := p.channel.Close(); err != nil {
		return errorx.Wrap(err, errorx.CodeInfraError, "close publisher channel failed")
	}
	zap.L().Info("publisher channel closed")
	return nil
}

func (p *MQPublisher) String() string {
	return fmt.Sprintf("MQPublisher{channel: %p}", p.channel)
}
