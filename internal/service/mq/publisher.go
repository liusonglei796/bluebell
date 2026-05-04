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
	confirm <-chan amqp.Confirmation // 生产者确认通道
	returns <-chan amqp.Return       // 无法路由消息退回通道
}

func newPublisher(conn *MQConnection) *MQPublisher {
	ch := conn.Channel()

	// 注册 Return 监听通道：
	// 当消息到达 Exchange 但无法路由到任何 Queue 时，RabbitMQ 会将消息退回
	returns := ch.NotifyReturn(make(chan amqp.Return, 1))

	return &MQPublisher{
		channel: ch,
		confirm: ch.NotifyPublish(make(chan amqp.Confirmation, 1)),
		returns: returns,
	}
}

// StartReturnHandler 启动后台协程处理被退回的消息
//
// 触发场景：
//   - 消息到达了 Exchange，但没有匹配的 Binding Key（找不到对应的 Queue）
//   - 此时 RabbitMQ 会将消息退回给生产者
//
// 处理策略：
//   - 记录详细错误日志（包含消息体、Exchange、RoutingKey、错误码）
//   - 可在此处接入报警系统或死信队列
//
// 调用位置：
//
//	cmd/bluebell/main.go 或 InitMQ 中（必须在发送消息前启动）
func (p *MQPublisher) StartReturnHandler(ctx context.Context) {
	go func() {
		for ret := range p.returns {
			zap.L().Error("message returned from exchange",
				zap.String("exchange", ret.Exchange),
				zap.String("routing_key", ret.RoutingKey),
				zap.String("body", string(ret.Body)),
				zap.Uint16("reply_code", ret.ReplyCode),
				zap.String("reply_text", ret.ReplyText),
			)
			// TODO: 可在此处加入报警逻辑或死信队列处理
		}
	}()
}

func (p *MQPublisher) publish(ctx context.Context, exchange, routingKey string, msg any, logName string, deliveryMode uint8) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "marshal %s message failed", logName)
	}

	publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = p.channel.PublishWithContext(
		publishCtx, exchange, routingKey,
		true,  // mandatory: true
		false, // immediate: false
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: deliveryMode, // 动态设置投递模式（瞬态/持久化）
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

	// 等待生产者确认（Publisher Confirm）：
	// 阻塞等待 RabbitMQ 返回 Ack 或 Nack，确保消息已成功到达 Exchange
	select {
	case confirm := <-p.confirm:
		if !confirm.Ack {
			// 消息未到达 Exchange（如 Exchange 不存在或路由失败）
			zap.L().Error("message not acked by exchange",
				zap.String("exchange", exchange),
				zap.Uint64("delivery_tag", confirm.DeliveryTag),
			)
			return errorx.Newf(errorx.CodeInfraError, "message not acked by exchange: %s", exchange)
		}
	case <-publishCtx.Done():
		// 等待确认超时
		return errorx.Newf(errorx.CodeInfraError, "waiting for publisher confirm timeout: %s", exchange)
	}

	zap.L().Debug("publish confirmed by exchange", zap.String("exchange", exchange), zap.Int("size", len(body)))
	return nil
}

func (p *MQPublisher) PublishVote(ctx context.Context, msg *VoteMessage) error {
	return p.publish(ctx, ExchangeVote, RoutingKeyVote, msg, "vote", amqp.Transient)
}

func (p *MQPublisher) PublishSearch(ctx context.Context, msg any) error {
	return p.publish(ctx, ExchangeSearch, RoutingKeySearch, msg, "search", amqp.Transient)
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
