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
//
// 职责：
//
//	消费 RabbitMQ vote.queue 中的投票消息，异步将投票记录持久化到 MySQL。
//
// 工作流程：
//  1. 用户投票时，Service 层先同步更新 Redis（计算 Gravity 分数并更新排名）
//  2. 然后发送 VoteMessage 到 MQ，立即返回成功给前端
//  3. VoteConsumer 在后台异步消费消息，写入 MySQL 作为持久化备份
//  4. 实现最终一致性：Redis 是"真相源"，MySQL 是"持久化归档"
//
// 设计优势：
//   - 高性能：用户无需等待 MySQL 写入，响应时间大幅降低
//   - 高可用：即使 MySQL 短暂不可用，投票功能仍可正常工作
//   - 幂等性：SaveVote 使用 INSERT ... ON DUPLICATE KEY UPDATE，重复消费无副作用
//
// 调用链路：
//
//	前端请求 → POST /api/v1/vote → handler → service.VoteForPost()
//	→ Redis.VoteForPost() [同步，含分数计算]
//	→ MQ.PublishVote()    [异步，不阻塞]
//	→ 返回响应给前端
//	→ VoteConsumer.Start() [后台 goroutine]
//	→ VoteConsumer.HandleDelivery()
//	→ VoteRepo.SaveVote()  [MySQL 异步落盘]
type VoteConsumer struct {
	conn     *MQConnection           // RabbitMQ 连接（共享）
	voteRepo dbdomain.VoteRepository // 投票数据访问层（MySQL）
}

// NewVoteConsumer 创建投票消费者实例
//
// 参数说明：
//
//	conn:     RabbitMQ 连接对象，用于获取 Channel 消费消息
//	voteRepo: 投票仓储接口，负责将投票记录写入 MySQL
//
// 返回值：
//
//	*VoteConsumer: 配置好的消费者实例
//
// 调用位置：
//
//	cmd/bluebell/main.go:145 (mq.NewVoteConsumer(conn, repositoriesUOW.Vote))
func NewVoteConsumer(conn *MQConnection, voteRepo dbdomain.VoteRepository) *VoteConsumer {
	return &VoteConsumer{
		conn:     conn,
		voteRepo: voteRepo,
	}
}

// Start 启动投票消费者，阻塞监听消息队列
//
// 工作流程：
//  1. 设置 QoS（预取计数=1）：确保每次只处理一条消息，实现多实例负载均衡
//  2. 订阅 vote.queue：开始接收投票消息
//  3. 进入阻塞循环：持续处理消息，直到 ctx 取消或通道关闭
//
// QoS 设计说明：
//
//	prefetch count = 1 表示 RabbitMQ 每次只分发给该消费者 1 条未确认的消息。
//	这样当有多个消费者实例时，消息会均匀分配，而不是堆积在某个慢消费者中。
//
// 消息确认机制：
//   - auto-ack = false：手动确认模式，处理成功后才 ACK
//   - 处理成功：隐式确认（不 Nack 即视为成功）
//   - 处理失败：d.Nack(false, false) - 不重新入队（避免毒消息无限循环）
//
// 参数说明：
//
//	ctx: 上下文，用于控制消费者生命周期。调用 ctx.Done() 可优雅停止消费者
//
// 返回值：
//
//	error: 仅在通道意外关闭或 QoS/Consume 初始化失败时返回错误
//
// 调用位置：
//
//	cmd/bluebell/main.go:147 (go func() { voteConsumer.Start(ctx) }())
//
// 注意：
//
//	此方法是阻塞式的，会在独立 goroutine 中运行，直到程序退出
func (c *VoteConsumer) Start(ctx context.Context) error {
	channel := c.conn.Channel()

	// 设置 QoS：每次只预取一条消息，确保负载均衡
	if err := channel.Qos(
		1,     // prefetch count：预取计数，限制未确认消息数量
		0,     // prefetch size：预取大小（0 表示不限制字节数）
		false, // global：false=仅应用于当前消费者，true=应用于整个 Channel
	); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "set qos for vote consumer failed")
	}

	// 消费 vote.queue：从队列中接收消息
	msgs, err := channel.Consume(
		QueueVote, // queue：队列名称
		"",        // consumer：消费者名称（空表示自动生成）
		false,     // auto-ack：手动确认模式，处理成功后才 ACK
		false,     // exclusive：非独占模式，允许多个消费者订阅
		false,     // no-local：允许接收本连接发布的消息
		false,     // no-wait：同步等待服务器响应
		nil,       // args：额外参数
	)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "consume vote queue failed")
	}

	zap.L().Info("vote consumer started",
		zap.String("queue", QueueVote),
	)

	// 阻塞处理消息：持续监听消息通道
	for {
		select {
		case <-ctx.Done():
			// 上下文被取消（程序退出），优雅停止消费者
			zap.L().Info("vote consumer stopping",
				zap.String("reason", ctx.Err().Error()),
			)
			return nil
		case d, ok := <-msgs:
			if !ok {
				// 消息通道被关闭（RabbitMQ 连接断开），视为致命错误
				zap.L().Warn("vote consumer channel closed")
				return errorx.New(errorx.CodeInfraError, "vote consumer channel closed unexpectedly")
			}
			if err := c.HandleDelivery(d); err != nil {
				// 消息处理失败，记录错误日志
				zap.L().Error("vote consumer handle delivery failed",
					zap.String("body", string(d.Body)),
					zap.Error(err),
				)
				// Nack 消息：false=不批量处理，false=不重新入队
				// 不 requeue 的原因：
				//   1. 防止"毒消息"（格式错误/参数非法）无限循环
				//   2. MySQL 短暂不可用时，可通过监控告警人工介入
				//   3. Redis 已保证投票功能正常，MySQL 最终一致性即可
				_ = d.Nack(false, false)
			}
			// 处理成功：消息自动确认（RabbitMQ 从队列中删除）
		}
	}
}

// HandleDelivery 处理单条投票消息
//
// 处理流程：
//  1. 反序列化 JSON 消息体 → VoteMessage 结构体
//  2. 校验 Action 字段合法性（必须是 1 或 -1）
//  3. 解析 UserID 和 PostID 字符串为 int64
//  4. 调用仓储层 SaveVote 写入 MySQL（upsert 操作，幂等）
//  5. 记录调试日志，便于问题排查
//
// 消息格式：
//
//	{
//	  "post_id": "123456",  // 帖子 ID（字符串格式）
//	  "user_id": "789",     // 用户 ID（字符串格式）
//	  "action": 1           // 投票动作：1=赞成，-1=反对
//	}
//
// 幂等性保证：
//
//	SaveVote 使用 SQL：
//	  INSERT INTO vote_record (user_id, post_id, direction)
//	  VALUES (?, ?, ?)
//	  ON DUPLICATE KEY UPDATE direction = VALUES(direction)
//
//	即使消息重复消费，结果也是一样的，不会产生重复数据。
//
// 参数说明：
//
//	d: RabbitMQ 消息投递对象，包含消息体、确认句柄等
//
// 返回值：
//
//	error: 消息解析失败、参数校验失败或数据库写入错误时返回
func (c *VoteConsumer) HandleDelivery(d amqp091.Delivery) error {
	// 解析消息体：JSON → VoteMessage 结构体
	var msg VoteMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		return errorx.Wrapf(err, errorx.CodeInvalidParam, "unmarshal vote message failed")
	}

	// 校验 action：只允许 1（赞成）或 -1（反对）
	if msg.Action != 1 && msg.Action != -1 {
		return errorx.Newf(errorx.CodeInvalidParam, "invalid vote action: %d", msg.Action)
	}

	// 解析 userID 和 postID：从字符串转为 int64（数据库字段类型）
	userID, err := strconv.ParseInt(msg.UserID, 10, 64)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInvalidParam, "parse user_id failed: %s", msg.UserID)
	}
	postID, err := strconv.ParseInt(msg.PostID, 10, 64)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInvalidParam, "parse post_id failed: %s", msg.PostID)
	}

	// 异步落盘到 MySQL（upsert，天然幂等）
	// 注意：这里使用 context.Background() 而非传入的 ctx，因为：
	//   1. 消息处理是独立的，不应该被外部 ctx 取消
	//   2. 即使程序正在关闭，也应该完成当前消息的处理
	ctx := context.Background()
	if err := c.voteRepo.SaveVote(ctx, userID, postID, int8(msg.Action)); err != nil {
		return errorx.Wrapf(err, errorx.CodeDBError, "save vote to mysql failed (user_id: %s, post_id: %s)", msg.UserID, msg.PostID)
	}

	// 记录调试日志：便于追踪投票落盘情况
	zap.L().Debug("vote message persisted to mysql",
		zap.String("post_id", msg.PostID),
		zap.String("user_id", msg.UserID),
		zap.Int("action", msg.Action),
	)

	return nil
}
