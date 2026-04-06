package mq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"bluebell/internal/infrastructure/es"
	"bluebell/pkg/errorx"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// SyncConsumer ES 同步消费者
// 从 RabbitMQ 消费帖子数据并索引到 Elasticsearch
type SyncConsumer struct {
	conn   *MQConnection // RabbitMQ 连接，用于创建 Channel 消费消息
	client *es.Client    // ES 客户端，用于执行索引/删除操作
}

// NewSyncConsumer 创建 ES 同步消费者实例
// Called by: cmd/bluebell/main.go (mq.NewSyncConsumer(conn, esClient))
func NewSyncConsumer(conn *MQConnection, esClient *es.Client) *SyncConsumer {
	return &SyncConsumer{
		conn:   conn,
		client: esClient,
	}
}

// Start 启动 ES 同步消费者，阻塞监听消息队列
// Called by: cmd/bluebell/main.go (esConsumer.Start(ctx))
func (c *SyncConsumer) Start(ctx context.Context) error {
	// 从连接获取 Channel（AMQP 操作管道）
	ch := c.conn.Channel()

	// 设置 QoS：prefetch_count=1，确保每次只预取一条消息，实现负载均衡
	if err := ch.Qos(
		1,     // prefetch count：每次只取一条
		0,     // prefetch size：0 表示不限制字节数
		false, // global：false 表示仅作用于本 consumer
	); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "set qos for sync consumer failed")
	}

	// 注册消费者，开始从 search.queue 消费消息
	msgs, err := ch.Consume(
		QueueSearch, // queue name：search.queue
		"",          // consumer tag：空表示由服务器自动生成
		false,       // auto-ack：false 表示手动确认（处理成功后才 Ack）
		false,       // exclusive：false 表示允许多个 consumer 共享
		false,       // no-local：false 表示可以接收自己发布的消息
		false,       // no-wait：false 表示等待服务器响应
		nil,         // args：无额外参数
	)
	if err != nil {
		return errorx.Wrap(err, errorx.CodeInfraError, "Failed to register search consumer")
	}

	zap.L().Info("ES sync consumer started", zap.String("queue", QueueSearch))

	// 阻塞循环，持续监听消息
	for {
		select {
		case <-ctx.Done():
			// 收到取消信号，优雅退出
			zap.L().Info("ES sync consumer stopped")
			return nil
		case d, ok := <-msgs:
			// 收到一条消息
			if !ok {
				// Channel 已关闭，退出
				zap.L().Error("Search consumer channel closed")
				return nil
			}
			// 处理消息，失败则 Nack（不重新入队）
			if err := c.handleDelivery(d); err != nil {
				zap.L().Error("Search delivery handling failed", zap.Error(err))
				_ = d.Nack(false, false) // requeue=false：丢弃消息，避免死循环
			}
		}
	}
}

// handleDelivery 处理单条同步消息
// Called by: Start (line 71: c.handleDelivery(d))
func (c *SyncConsumer) handleDelivery(d amqp.Delivery) error {
	// 解析 JSON 消息体
	var msg SyncMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		zap.L().Error("Failed to parse sync message", zap.Error(err))
		_ = d.Nack(false, false) // 解析失败，丢弃消息
		return nil               // 返回 nil 表示不再重试
	}

	zap.L().Info("Processing sync request",
		zap.String("post_id", msg.PostID),
		zap.String("action", msg.Action))

	// 根据 action 执行不同操作
	switch msg.Action {
	case "delete":
		// 删除 ES 文档
		if err := c.deleteDocument(msg.PostID); err != nil {
			zap.L().Error("Failed to delete document from ES",
				zap.String("post_id", msg.PostID), zap.Error(err))
			_ = d.Nack(false, false)
			return nil
		}
	case "index":
		fallthrough // 继续执行 default
	default:
		// 索引 ES 文档（默认行为）
		if err := c.indexDocument(&msg); err != nil {
			zap.L().Error("Failed to index document to ES",
				zap.String("post_id", msg.PostID), zap.Error(err))
			_ = d.Nack(false, false)
			return nil
		}
	}

	// 处理成功，确认消息（RabbitMQ 将其从队列移除）
	_ = d.Ack(false)
	zap.L().Debug("Sync operation completed", zap.String("post_id", msg.PostID))
	return nil
}

// indexDocument 将帖子文档索引到 ES
// Called by: handleDelivery (line 104: c.indexDocument(&msg))
func (c *SyncConsumer) indexDocument(msg *SyncMessage) error {
	// 构建 ES 文档（字段需与 mapping.go 中的 PostMapping 一致）
	doc := map[string]interface{}{
		"post_id":      msg.PostID,
		"author_id":    msg.AuthorID,
		"community_id": msg.CommunityID,
		"post_title":   msg.PostTitle,
		"content":      msg.Content,
		"status":       msg.Status,
		"created_at":   msg.CreatedAt,
	}

	// 序列化为 JSON
	body, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal document failed: %w", err)
	}

	// 发送 Index 请求到 ES，以 post_id 作为文档 ID（幂等更新）
	req := bytes.NewReader(body)
	res, err := c.client.ES().Index(
		es.IndexPost, // index name：post
		req,          // request body
		c.client.ES().Index.WithDocumentID(msg.PostID), // 指定文档 ID，实现 upsert
	)
	if err != nil {
		return fmt.Errorf("ES index failed: %w", err)
	}
	defer res.Body.Close() // 释放 HTTP 连接

	// 检查 ES 响应
	if res.IsError() {
		respBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES index error: %s", string(respBody))
	}

	return nil
}

// deleteDocument 从 ES 删除帖子文档
// Called by: handleDelivery (line 95: c.deleteDocument(msg.PostID))
func (c *SyncConsumer) deleteDocument(postID string) error {
	// 发送 Delete 请求到 ES
	res, err := c.client.ES().Delete(es.IndexPost, postID)
	if err != nil {
		return fmt.Errorf("ES delete failed: %w", err)
	}
	defer res.Body.Close() // 释放 HTTP 连接

	// 404 表示文档已不存在，视为成功；其他错误才返回
	if res.IsError() && res.StatusCode != 404 {
		respBody, _ := io.ReadAll(res.Body)
		return fmt.Errorf("ES delete error: %s", string(respBody))
	}

	return nil
}
