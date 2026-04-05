package mq

import (
	"bluebell/pkg/errorx"
	"context"
	"encoding/json"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"time"
)

// MQPublisher 消息发布器
// 封装 RabbitMQ Channel，提供简单易用的 Publish 接口
type MQPublisher struct {
	channel *amqp.Channel
}

// NewPublisher 创建消息发布器实例
// Called by: mq/init.go (InitMQ 中 NewPublisher(conn))
func NewPublisher(conn *MQConnection) *MQPublisher {
	return &MQPublisher{
		channel: conn.Channel(),
	}
}

// publish 内部通用发布方法
func (p *MQPublisher) publish(ctx context.Context, exchange, routingKey string, msg interface{}, logName string) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "marshal %s message failed", logName)
	}

	// 设置 context 超时
	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(
		publishCtx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent, // delivery mode = 2 (persistent)
			Body:         body,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		zap.L().Error("publish message failed",
			zap.String("exchange", exchange),
			zap.String("routing_key", routingKey),
			zap.String("log_name", logName),
			zap.Error(err),
		)
		return errorx.Wrapf(err, errorx.CodeInfraError, "publish %s message to exchange %s failed", logName, exchange)
	}

	zap.L().Debug("publish message success",
		zap.String("exchange", exchange),
		zap.String("routing_key", routingKey),
		zap.String("log_name", logName),
		zap.Int("body_size", len(body)),
	)

	return nil
}

// PublishVote 发布投票消息到 vote.exchange
// routing key: vote.count
// Called by: handler/post_handler/handler.go (PostVoteHandler 中 h.publisher.PublishVote(ctx, voteMsg))
func (p *MQPublisher) PublishVote(ctx context.Context, msg *VoteMessage) error {
	return p.publish(ctx, ExchangeVote, RoutingKeyVote, msg, "vote")
}

// PublishAudit 发布审核消息到 audit.exchange
// 根据 msg.Type 自动选择 routing key: audit.post 或 audit.remark
// Called by: handler/post_handler/handler.go (CreatePostHandler 和 PostRemarkHandler 中 h.publisher.PublishAudit)
func (p *MQPublisher) PublishAudit(ctx context.Context, msg *AuditMessage) error {
	routingKey := RoutingKeyAuditPost
	if msg.Type == "remark" {
		routingKey = RoutingKeyAuditRemark
	}
	return p.publish(ctx, ExchangeAudit, routingKey, msg, "audit")
}

// PublishSearch 发布搜索同步消息到 search.exchange
// routing key: search.sync
// Called by: handler/post_handler/handler.go (CreatePostHandler 和 DeletePostHandler 中 h.publisher.PublishSearch)
func (p *MQPublisher) PublishSearch(ctx context.Context, msg interface{}) error {
	return p.publish(ctx, ExchangeSearch, RoutingKeySearch, msg, "search")
}

// PublishNotify 发布通知消息到 notify.exchange
// fanout exchange, routing key 为空
// Called by: 暂无调用方（预留接口）
func (p *MQPublisher) PublishNotify(ctx context.Context, msg interface{}) error {
	return p.publish(ctx, ExchangeNotify, "", msg, "notify")
}

// Close 关闭 publisher（关闭底层 Channel）
// Called by: 暂无调用方（预留清理接口）
func (p *MQPublisher) Close() error {
	if p.channel == nil {
		return nil
	}
	err := p.channel.Close()
	if err != nil {
		zap.L().Error("close publisher channel failed", zap.Error(err))
		return errorx.Wrap(err, errorx.CodeInfraError, "close publisher channel failed")
	}
	zap.L().Info("publisher channel closed")
	return nil
}

// String 实现 fmt.Stringer 接口
func (p *MQPublisher) String() string {
	return fmt.Sprintf("MQPublisher{channel: %p}", p.channel)
}
