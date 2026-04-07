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
// 参数说明：
//   - ctx: 上下文，用于取消或传递超时
//   - msgType: 消息类型，"post" 表示帖子，"remark" 表示评论
//   - postID: 帖子 ID（雪花 ID 字符串）
//   - remarkID: 评论 ID（仅在 msgType 为 "remark" 时有效）
//   - violations: 违规类型列表，如 ["政治敏感内容", "暴力恐怖内容"]
//   - reason: AI 返回的审核原因说明
type AuditFailedHandler func(ctx context.Context, msgType string, postID string, remarkID uint, violations []string, reason string)

// AuditConsumer AI 内容审核消费者
// 从 RabbitMQ 的 audit.queue 消费消息，调用 AI 审核内容，并根据审核结果触发回调
type AuditConsumer struct {
	conn     *MQConnection      // RabbitMQ 连接，用于创建 Channel 消费消息
	auditor  *ai.Auditor        // AI 审核服务，负责调用大模型进行内容审核
	onFailed AuditFailedHandler // 审核失败回调，当内容被判定为不安全时执行
}

// NewAuditConsumer 创建审核消费者实例
// 参数说明：
//   - conn: RabbitMQ 连接对象
//   - auditor: AI 审核服务实例
//   - onFailed: 审核失败时的回调函数
//
// 返回值：*AuditConsumer 消费者实例
// Called by: cmd/bluebell/main.go (mq.NewAuditConsumer(conn, auditor, onFailed))
func NewAuditConsumer(conn *MQConnection, auditor *ai.Auditor, onFailed AuditFailedHandler) *AuditConsumer {
	// 初始化消费者结构体，保存依赖项
	return &AuditConsumer{
		conn:     conn,     // 保存 MQ 连接
		auditor:  auditor,  // 保存 AI 审核服务
		onFailed: onFailed, // 保存失败回调
	}
}

// Start 启动审核消费者，阻塞监听消息队列
// 该方法会一直阻塞，直到 ctx 被取消或发生错误才返回
// 参数说明：
//   - ctx: 上下文，用于控制消费者的生命周期
//
// 返回值：error，正常退出返回 nil，异常退出返回错误
// Called by: cmd/bluebell/main.go (auditConsumer.Start(ctx))
func (c *AuditConsumer) Start(ctx context.Context) error {
	// 从连接获取 AMQP Channel，用于消费消息
	channel := c.conn.Channel()

	// 设置 QoS（服务质量）：控制消费者每次预取的消息数量
	// prefetch count = 1：每次只预取一条消息，确保多个消费者实例时消息均匀分配
	// prefetch size = 0：不限制预取消息的字节数
	// global = false：仅对当前 consumer 生效，不影响同一 channel 上的其他 consumer
	if err := channel.Qos(
		1,     // prefetch count：每次只取一条
		0,     // prefetch size：0 表示不限制字节数
		false, // global：false 表示仅作用于本 consumer
	); err != nil {
		// QoS 设置失败，包装错误并返回（errorx 统一错误处理）
		return errorx.Wrapf(err, errorx.CodeInfraError, "set qos for audit consumer failed")
	}

	// 注册消费者，开始从 audit.queue 消费消息
	// 返回值 msgs 是一个只读 channel，收到消息后会通过该 channel 投递
	msgs, err := channel.Consume(
		QueueAudit, // queue：队列名称，从常量 QueueAudit 读取
		"",         // consumer tag：空字符串表示由 RabbitMQ 服务器自动生成唯一标识
		false,      // auto-ack：false 表示手动确认（必须处理成功后手动调用 Ack）
		false,      // exclusive：false 表示允许多个 consumer 共享该队列
		false,      // no-local：false 表示可以接收自己发布的消息（通常无影响）
		false,      // no-wait：false 表示等待服务器响应 Consume 请求
		nil,        // args：无额外的消费参数
	)
	if err != nil {
		// 注册消费者失败，包装错误并返回
		return errorx.Wrapf(err, errorx.CodeInfraError, "consume audit queue failed")
	}

	// 记录消费者启动成功日志
	zap.L().Info("audit consumer started",
		zap.String("queue", QueueAudit), // 打印监听的队列名称
	)

	// 阻塞循环，持续监听消息队列并处理消息
	for {
		select {
		case <-ctx.Done():
			// 收到取消信号（如进程关闭），优雅退出
			zap.L().Info("audit consumer stopping",
				zap.String("reason", ctx.Err().Error()), // 记录取消原因
			)
			return nil // 正常退出，返回 nil
		case d, ok := <-msgs:
			// 从 msgs channel 收到一条消息
			if !ok {
				// Channel 已关闭（通常为 RabbitMQ 连接断开），视为异常退出
				zap.L().Warn("audit consumer channel closed")
				return errorx.New(errorx.CodeInfraError, "audit consumer channel closed unexpectedly")
			}
			// 调用 HandleDelivery 处理消息
			if err := c.HandleDelivery(d); err != nil {
				// 处理消息失败，记录错误日志
				zap.L().Error("audit consumer handle delivery failed",
					zap.String("body", string(d.Body)), // 打印消息内容（便于排查）
					zap.Error(err),                     // 打印错误堆栈
				)
				// Nack（不确认）消息：requeue=false 表示丢弃消息，避免处理失败的消息反复入队导致死循环
				_ = d.Nack(false, false)
			}
			// 处理成功则在 HandleDelivery 内部调用 Ack 确认
		}
	}
}

// HandleDelivery 处理单条审核消息
// 该方法负责：解析消息 → 调用 AI 审核 → 根据结果触发回调 → 确认消息
// 参数说明：
//   - d: RabbitMQ 投递的消息对象，包含消息体、确认方法等
//
// 返回值：error，处理失败返回错误（调用方会根据错误决定是否 Nack）
func (c *AuditConsumer) HandleDelivery(d amqp091.Delivery) error {
	// 解析消息体：将 JSON 反序列化为 AuditMessage 结构体
	var msg AuditMessage
	if err := json.Unmarshal(d.Body, &msg); err != nil {
		// JSON 解析失败，说明消息格式异常，包装错误并返回
		// 返回错误后，调用方会执行 Nack（丢弃消息）
		return errorx.Wrapf(err, errorx.CodeInvalidParam, "unmarshal audit message failed")
	}

	// 检查 AI 审核服务是否可用：auditor 为 nil 或未启用时，跳过审核
	if c.auditor == nil || !c.auditor.IsEnabled() {
		// 审核服务不可用，直接 Ack 消息（从队列中移除），避免堆积
		if err := d.Ack(false); err != nil {
			// Ack 失败，返回错误（调用方会执行 Nack）
			return errorx.Wrapf(err, errorx.CodeInfraError, "ack audit message failed (post_id: %s)", msg.PostID)
		}
		// 记录审核跳过的调试日志
		zap.L().Debug("audit skipped, auditor not enabled",
			zap.String("post_id", msg.PostID),
		)
		return nil // 返回 nil 表示处理完成
	}

	// 构造 AI 审核的上下文（独立上下文，不受外部取消影响）
	auditCtx := context.Background()
	// 构造审核输入：从消息中提取标题和内容
	input := ai.AuditInput{
		Title:   msg.Title,   // 帖子/评论标题
		Content: msg.Content, // 帖子/评论正文内容
	}
	// 调用 AI 审核服务，传入标题和内容，获取审核结果
	result, err := c.auditor.Audit(auditCtx, input)
	if err != nil {
		// AI 审核调用失败（如 LLM 接口异常、超时等），返回错误
		// 调用方会执行 Nack，消息不会重新入队
		return errorx.Wrapf(err, errorx.CodeInfraError, "audit content failed (post_id: %s)", msg.PostID)
	}

	// 检查审核结果：IsSafe == false 表示内容违规
	if !result.IsSafe {
		// 记录审核不通过的警告日志
		zap.L().Warn("content audit failed",
			zap.String("post_id", msg.PostID),            // 帖子 ID
			zap.String("type", msg.Type),                 // 消息类型：post 或 remark
			zap.Int64("author_id", msg.AuthorID),         // 作者 ID
			zap.Strings("violations", result.Violations), // 违规类型列表，如 ["政治敏感", "暴力"]
			zap.String("reason", result.Reason),          // AI 返回的审核原因
		)

		// 如果注册了审核失败回调，则执行回调函数
		// 回调通常负责：隐藏帖子、删除评论、通知管理员等
		if c.onFailed != nil {
			auditCtx := context.Background() // 创建独立上下文
			// 调用回调，传入违规信息，由业务层执行后续操作
			c.onFailed(auditCtx, msg.Type, msg.PostID, msg.RemarkID, result.Violations, result.Reason)
		}
	}
	// 注意：即使审核不通过，也继续往下走，因为 AI 已成功返回结果

	// 审核处理完成，手动 Ack 消息（从队列中移除）
	// multiple=false：仅确认当前这条消息
	if err := d.Ack(false); err != nil {
		// Ack 失败，返回错误
		return errorx.Wrapf(err, errorx.CodeInfraError, "ack audit message failed (post_id: %s)", msg.PostID)
	}

	// 记录消息处理完成的调试日志
	zap.L().Debug("audit message processed",
		zap.String("post_id", msg.PostID),  // 帖子 ID
		zap.String("type", msg.Type),       // 消息类型
		zap.Bool("is_safe", result.IsSafe), // 审核结果：true=安全，false=违规
	)

	return nil // 处理成功，返回 nil
}
