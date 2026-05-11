package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"bluebell/internal/domain/dbdomain"
	"bluebell/pkg/errorx"

	amqp091 "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// VoteConsumer 投票消息消费者
type VoteConsumer struct {
	conn     *MQConnection           // RabbitMQ 连接
	voteRepo dbdomain.VoteRepository // MySQL 仓储
	rdb      *redis.Client           // Redis 客户端，用于防重
}

// NewVoteConsumer 创建投票消费者实例
func NewVoteConsumer(conn *MQConnection, voteRepo dbdomain.VoteRepository, rdb *redis.Client) *VoteConsumer {
	return &VoteConsumer{
		conn:     conn,
		voteRepo: voteRepo,
		rdb:      rdb,
	}
}

// Start 启动投票消费者，阻塞监听消息队列
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
		false,     // auto-ack: false 表示手动 ACK
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "consume vote queue failed")
	}

	zap.L().Info("vote consumer started", zap.String("queue", QueueVote))

	for {
		select {
		case <-ctx.Done():
			zap.L().Info("vote consumer stopping")
			return nil
		case d, ok := <-msgs:
			if !ok {
				zap.L().Warn("vote consumer channel closed")
				return errorx.New(errorx.CodeInfraError, "vote consumer channel closed")
			}
			if err := c.HandleDelivery(d); err != nil {
				zap.L().Error("vote consumer handle delivery failed", zap.Error(err))
				_ = d.Nack(false, false) // 失败不入队
			} else {
				_ = d.Ack(false) // 成功 ACK
			}
		}
	}
}

// HandleDelivery 处理单条投票消息（含三重幂等检查）
func (c *VoteConsumer) HandleDelivery(d amqp091.Delivery) error {
	var msg VoteMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		return errorx.Wrapf(err, errorx.CodeInvalidParam, "unmarshal vote message failed")
	}

	// 1. [第一层幂等] 基于 Redis SETNX 的分布式去重
	dedupKey := fmt.Sprintf("bluebell:mq:dedup:vote:%s", msg.MsgID)
	ok, err := c.rdb.SetNX(context.Background(), dedupKey, "1", 24*time.Hour).Result()
	if err != nil {
		return errorx.Wrap(err, errorx.CodeCacheError, "redis dedup check failed")
	}
	if !ok {
		zap.L().Warn("message already processed, skipping", zap.String("msg_id", msg.MsgID))
		return nil
	}

	// 2. 业务校验
	if msg.Action != 1 && msg.Action != -1 && msg.Action != 0 {
		return errorx.Newf(errorx.CodeInvalidParam, "invalid vote action: %d", msg.Action)
	}

	userID, _ := strconv.ParseInt(msg.UserID, 10, 64)
	postID, _ := strconv.ParseInt(msg.PostID, 10, 64)

	// 3. [第二层幂等] MySQL 联合唯一索引 (SaveVote 内部执行 INSERT ... ON DUPLICATE KEY UPDATE)
	ctx := context.Background()
	if err := c.voteRepo.SaveVote(ctx, userID, postID, int8(msg.Action)); err != nil {
		// 写入失败则删除去重 Key，允许重试
		c.rdb.Del(ctx, dedupKey)
		return errorx.Wrap(err, errorx.CodeDBError, "save vote to mysql failed")
	}

	zap.L().Debug("vote message persisted to mysql", zap.String("msg_id", msg.MsgID))
	return nil
}
