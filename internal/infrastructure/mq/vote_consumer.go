package mq

import (
	"context"
	"encoding/json"
	"strconv"

	"bluebell/internal/domain/dbdomain"
	"bluebell/pkg/errorx"

	amqp091 "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// VoteConsumer 投票消息消费者
// 消费 RabbitMQ 中的投票消息，异步落盘到 MySQL
type VoteConsumer struct {
	conn     *MQConnection
	voteRepo dbdomain.VoteRepository
}

// NewVoteConsumer 创建投票消费者实例
// Called by: cmd/bluebell/main.go (mq.NewVoteConsumer(conn, voteRepo))
func NewVoteConsumer(conn *MQConnection, voteRepo dbdomain.VoteRepository) *VoteConsumer {
	return &VoteConsumer{
		conn:     conn,
		voteRepo: voteRepo,
	}
}

// Start 启动投票消费者，阻塞监听消息队列
// Called by: cmd/bluebell/main.go (voteConsumer.Start(ctx))
func (c *VoteConsumer) Start(ctx context.Context) error {
	channel := c.conn.Channel()

	// 设置 QoS：每次只预取一条消息，确保负载均衡
	if err := channel.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "set qos for vote consumer failed")
	}

	// 消费 vote.queue
	msgs, err := channel.Consume(
		QueueVote, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "consume vote queue failed")
	}

	zap.L().Info("vote consumer started",
		zap.String("queue", QueueVote),
	)

	// 阻塞处理消息
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("vote consumer stopping",
				zap.String("reason", ctx.Err().Error()),
			)
			return nil
		case d, ok := <-msgs:
			if !ok {
				zap.L().Warn("vote consumer channel closed")
				return errorx.New(errorx.CodeInfraError, "vote consumer channel closed unexpectedly")
			}
			if err := c.HandleDelivery(d); err != nil {
				zap.L().Error("vote consumer handle delivery failed",
					zap.String("body", string(d.Body)),
					zap.Error(err),
				)
				// nack 消息，不 requeue
				_ = d.Nack(false, false)
			}
		}
	}
}

// HandleDelivery 处理单条投票消息
func (c *VoteConsumer) HandleDelivery(d amqp091.Delivery) error {
	// 解析消息体
	var msg VoteMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		return errorx.Wrapf(err, errorx.CodeInvalidParam, "unmarshal vote message failed")
	}

	// 校验 action
	if msg.Action != 1 && msg.Action != -1 {
		return errorx.Newf(errorx.CodeInvalidParam, "invalid vote action: %d", msg.Action)
	}

	// 解析 userID 和 postID
	userID, err := strconv.ParseInt(msg.UserID, 10, 64)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInvalidParam, "parse user_id failed: %s", msg.UserID)
	}
	postID, err := strconv.ParseInt(msg.PostID, 10, 64)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInvalidParam, "parse post_id failed: %s", msg.PostID)
	}

	// 异步落盘到 MySQL（upsert，天然幂等）
	ctx := context.Background()
	if err := c.voteRepo.SaveVote(ctx, userID, postID, int8(msg.Action)); err != nil {
		return errorx.Wrapf(err, errorx.CodeDBError, "save vote to mysql failed (user_id: %s, post_id: %s)", msg.UserID, msg.PostID)
	}

	zap.L().Debug("vote message persisted to mysql",
		zap.String("post_id", msg.PostID),
		zap.String("user_id", msg.UserID),
		zap.Int("action", msg.Action),
	)

	return nil
}
