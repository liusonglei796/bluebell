package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
)

// VoteConsumer 投票消息消费者
type VoteConsumer struct {
	ch       *amqp.Channel
	voteRepo domain.VoteRepository
	rdb      *redis.Client
}

// NewVoteConsumer 创建一个新的投票消费者
func NewVoteConsumer(ch *amqp.Channel, voteRepo domain.VoteRepository, rdb *redis.Client) *VoteConsumer {
	return &VoteConsumer{ch: ch, voteRepo: voteRepo, rdb: rdb}
}

// Start 启动监听
func (c *VoteConsumer) Start(ctx context.Context) error {
	if err := c.ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("设置 QoS 失败: %w", err)
	}

	msgs, err := c.ch.Consume(QueueVote, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("消费队列 %s 失败: %w", QueueVote, err)
	}

	log.Printf("投票消费者已启动: %s", QueueVote)

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("消费者通道已关闭")
			}
			if err := c.handleDelivery(ctx, d); err != nil {
				log.Printf("处理投票消息失败: %v", err)
				_ = d.Nack(false, false)
			} else {
				_ = d.Ack(false)
			}
		}
	}
}

func (c *VoteConsumer) handleDelivery(ctx context.Context, d amqp.Delivery) error {
	// 从 Headers 提取 Trace 上下文
	ctx = otel.GetTextMapPropagator().Extract(ctx, AmqpHeadersCarrier(d.Headers))

	var msg VoteMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		return fmt.Errorf("vote_consumer: 反序列化投票消息失败: %w", err)
	}

	// 幂等检查
	dedupKey := fmt.Sprintf("bluebell:mq:dedup:vote:%s", msg.MsgID)
	ok, err := c.rdb.SetNX(ctx, dedupKey, "1", 24*time.Hour).Result()
	if err != nil {
		return fmt.Errorf("vote_consumer: 幂等检查失败 (msg_id: %s): %w", msg.MsgID, err)
	}
	if !ok {
		log.Printf("重复投票消息，跳过: %s", msg.MsgID)
		return nil
	}

	userID, err := strconv.ParseInt(msg.UserID, 10, 64)
	if err != nil {
		return fmt.Errorf("vote_consumer: 无效的 UserID %q: %w", msg.UserID, err)
	}
	postID, err := strconv.ParseInt(msg.PostID, 10, 64)
	if err != nil {
		return fmt.Errorf("vote_consumer: 无效的 PostID %q: %w", msg.PostID, err)
	}

	// 使用领域模型校验
	vote := &entity.Vote{
		UserID:    userID,
		PostID:    postID,
		Direction: int8(msg.Action),
	}
	if err := vote.Validate(); err != nil {
		return fmt.Errorf("领域校验失败: %w", err)
	}

	if err := c.voteRepo.SaveVote(ctx, userID, postID, int8(msg.Action)); err != nil {
		c.rdb.Del(ctx, dedupKey) // 失败则删除去重 key，允许重试
		return fmt.Errorf("vote_consumer: 保存投票数据失败: %w", err)
	}

	return nil
}
