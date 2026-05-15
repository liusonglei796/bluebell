package mq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"bluebell/internal/infrastructure/es"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
)

// SyncConsumer ES 同步消费者
type SyncConsumer struct {
	ch     *amqp.Channel
	client *es.Client
}

// NewSyncConsumer 创建一个新的搜索同步消费者
func NewSyncConsumer(ch *amqp.Channel, esClient *es.Client) *SyncConsumer {
	return &SyncConsumer{ch: ch, client: esClient}
}

// Start 启动监听
func (c *SyncConsumer) Start(ctx context.Context) error {
	if err := c.ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("设置 QoS 失败: %w", err)
	}

	msgs, err := c.ch.Consume(QueueSearch, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("消费队列 %s 失败: %w", QueueSearch, err)
	}

	log.Printf("搜索同步消费者已启动: %s", QueueSearch)

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("消费者通道已关闭")
			}
			if err := c.handleDelivery(ctx, d); err != nil {
				log.Printf("处理搜索消息失败: %v", err)
				_ = d.Nack(false, false)
			} else {
				_ = d.Ack(false)
			}
		}
	}
}

func (c *SyncConsumer) handleDelivery(ctx context.Context, d amqp.Delivery) error {
	// 从 Headers 提取 Trace 上下文
	ctx = otel.GetTextMapPropagator().Extract(ctx, AmqpHeadersCarrier(d.Headers))

	var msg SyncMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		log.Printf("解析搜索消息失败: %v", err)
		_ = d.Nack(false, false)
		return nil
	}

	switch msg.Action {
	case "delete":
		if err := c.client.DeleteDocument(ctx, es.IndexPost, msg.PostID); err != nil {
			return err
		}
	default:
		doc := map[string]interface{}{
			"post_id":      msg.PostID,
			"author_id":    msg.AuthorID,
			"community_id": msg.CommunityID,
			"post_title":   msg.PostTitle,
			"content":      msg.Content,
			"status":       msg.Status,
			"created_at":   msg.CreatedAt,
		}
		body, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("序列化文档失败: %w", err)
		}
		if err := c.client.IndexDocument(ctx, es.IndexPost, msg.PostID, bytes.NewReader(body)); err != nil {
			return err
		}
	}

	_ = d.Ack(false)
	return nil
}
