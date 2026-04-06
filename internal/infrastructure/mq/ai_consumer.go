package mq

import (
	"context"
	"encoding/json"

	"bluebell/internal/infrastructure/ai"
	"bluebell/pkg/errorx"

	amqp091 "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// AuditFailedHandler 审核失败回调函数类型
// 用于在审核不通过时执行后续操作（如隐藏帖子、删除违规评论等）
type AuditFailedHandler func(ctx context.Context, msgType string, postID string, remarkID uint, violations []string, reason string)

// AuditConsumer AI 内容审核消费者
type AuditConsumer struct {
	conn     *MQConnection
	auditor  *ai.Auditor
	onFailed AuditFailedHandler // 审核失败回调
}

// NewAuditConsumer 创建审核消费者实例
// Called by: cmd/bluebell/main.go (mq.NewAuditConsumer(conn, auditor, onFailed))
func NewAuditConsumer(conn *MQConnection, auditor *ai.Auditor, onFailed AuditFailedHandler) *AuditConsumer {
	return &AuditConsumer{
		conn:     conn,
		auditor:  auditor,
		onFailed: onFailed,
	}
}

// Start 启动审核消费者，阻塞监听消息队列
// Called by: cmd/bluebell/main.go (auditConsumer.Start(ctx))
func (c *AuditConsumer) Start(ctx context.Context) error {
	channel := c.conn.Channel()

	// 设置 QoS：每次只预取一条消息
	if err := channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "set qos for audit consumer failed")
	}

	// 消费 audit.queue
	msgs, err := channel.Consume(
		QueueAudit, // queue
		"",         // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "consume audit queue failed")
	}

	zap.L().Info("audit consumer started",
		zap.String("queue", QueueAudit),
	)

	// 阻塞处理消息
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("audit consumer stopping",
				zap.String("reason", ctx.Err().Error()),
			)
			return nil
		case d, ok := <-msgs:
			if !ok {
				zap.L().Warn("audit consumer channel closed")
				return errorx.New(errorx.CodeInfraError, "audit consumer channel closed unexpectedly")
			}
			if err := c.HandleDelivery(d); err != nil {
				zap.L().Error("audit consumer handle delivery failed",
					zap.String("body", string(d.Body)),
					zap.Error(err),
				)
				// nack 消息，不 requeue
				_ = d.Nack(false, false)
			}
		}
	}
}

// HandleDelivery 处理单条审核消息
func (c *AuditConsumer) HandleDelivery(d amqp091.Delivery) error {
	// 解析消息体
	var msg AuditMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		return errorx.Wrapf(err, errorx.CodeInvalidParam, "unmarshal audit message failed")
	}

	// auditor 为 nil 或未启用时，直接 Ack
	if c.auditor == nil || !c.auditor.IsEnabled() {
		if err := d.Ack(false); err != nil {
			return errorx.Wrapf(err, errorx.CodeInfraError, "ack audit message failed (post_id: %s)", msg.PostID)
		}
		zap.L().Debug("audit skipped, auditor not enabled",
			zap.String("post_id", msg.PostID),
		)
		return nil
	}

	// 调用 AI 审核
	auditCtx := context.Background()
	result, err := c.auditor.Audit(auditCtx, msg.Title, msg.Content)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "audit content failed (post_id: %s)", msg.PostID)
	}

	// 审核不通过，执行回调 + 记录违规信息
	if !result.IsSafe {
		zap.L().Warn("content audit failed",
			zap.String("post_id", msg.PostID),
			zap.String("type", msg.Type),
			zap.Int64("author_id", msg.AuthorID),
			zap.Int("score", result.Score),
			zap.Strings("violations", result.Violations),
			zap.String("reason", result.Reason),
		)

		// 触发审核失败回调（如隐藏帖子、删除评论）
		if c.onFailed != nil {
			auditCtx := context.Background()
			c.onFailed(auditCtx, msg.Type, msg.PostID, msg.RemarkID, result.Violations, result.Reason)
		}
	}

	// 处理完成，Ack 消息
	if err := d.Ack(false); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "ack audit message failed (post_id: %s)", msg.PostID)
	}

	zap.L().Debug("audit message processed",
		zap.String("post_id", msg.PostID),
		zap.String("type", msg.Type),
		zap.Bool("is_safe", result.IsSafe),
	)

	return nil
}
