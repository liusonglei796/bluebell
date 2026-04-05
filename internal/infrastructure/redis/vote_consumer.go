package redis

import (
	"context"
	"encoding/json"

	"bluebell/internal/infrastructure/mq"
	"bluebell/pkg/errorx"

	amqp091 "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Redis Key 前缀
const (
	voteKeyPrefixUp   = "bluebell:votes:up:"
	voteKeyPrefixDown = "bluebell:votes:down:"
)

// VoteConsumer 投票消息消费者
// 消费 RabbitMQ 中的投票消息，异步更新 Redis 投票计数
type VoteConsumer struct {
	conn        *mq.MQConnection
	redisClient *redis.Client
}

// NewVoteConsumer 创建投票消费者实例
// Called by: cmd/bluebell/main.go (redis.NewVoteConsumer(conn, rdb))
func NewVoteConsumer(conn *mq.MQConnection, redisClient *redis.Client) *VoteConsumer {
	return &VoteConsumer{
		conn:        conn,
		redisClient: redisClient,
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
		mq.QueueVote, // queue
		"",           // consumer
		false,        // auto-ack
		false,        // exclusive
		false,        // no-local
		false,        // no-wait
		nil,          // args
	)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "consume vote queue failed")
	}

	zap.L().Info("vote consumer started",
		zap.String("queue", mq.QueueVote),
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
	var msg mq.VoteMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		return errorx.Wrapf(err, errorx.CodeInvalidParam, "unmarshal vote message failed")
	}

	// 根据 action 更新 Redis 计数
	ctx := context.Background()
	switch msg.Action {
	case 1:
		// upvote: INCR bluebell:votes:up:{postID}
		if err := c.redisClient.Incr(ctx, voteKeyPrefixUp+msg.PostID).Err(); err != nil {
			return errorx.Wrapf(err, errorx.CodeCacheError, "incr vote up failed (post_id: %s)", msg.PostID)
		}
	case -1:
		// downvote: INCR bluebell:votes:down:{postID}
		if err := c.redisClient.Incr(ctx, voteKeyPrefixDown+msg.PostID).Err(); err != nil {
			return errorx.Wrapf(err, errorx.CodeCacheError, "incr vote down failed (post_id: %s)", msg.PostID)
		}
	default:
		return errorx.Newf(errorx.CodeInvalidParam, "invalid vote action: %d", msg.Action)
	}

	// 处理成功，ack 消息
	if err := d.Ack(false); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "ack vote message failed (post_id: %s)", msg.PostID)
	}

	zap.L().Debug("vote message processed",
		zap.String("post_id", msg.PostID),
		zap.String("user_id", msg.UserID),
		zap.Int("action", msg.Action),
	)

	return nil
}
