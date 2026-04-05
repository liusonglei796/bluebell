package mq

import (
	"bluebell/internal/config"
	"bluebell/pkg/errorx"
	"context"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// ==================== Exchange & Queue 常量定义 ====================

// Vote Exchange
const (
	ExchangeVote   = "vote.exchange"
	QueueVote      = "vote.queue"
	RoutingKeyVote = "vote.count"
)

// Audit Exchange
const (
	ExchangeAudit         = "audit.exchange"
	QueueAudit            = "audit.queue"
	RoutingKeyAuditPost   = "audit.post"
	RoutingKeyAuditRemark = "audit.remark"
)

// Search Exchange
const (
	ExchangeSearch   = "search.exchange"
	QueueSearch      = "search.queue"
	RoutingKeySearch = "search.sync"
)

// Notify Exchange
const (
	ExchangeNotify = "notify.exchange"
	QueueNotify    = "notify.queue"
)

// MQConnection RabbitMQ 连接封装
// 持有 Connection 和 Channel，负责 Exchange/Queue 的声明
type MQConnection struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewMQConnection 建立 RabbitMQ 连接并创建 Channel
func NewMQConnection(ctx context.Context, cfg *config.Config) (*MQConnection, error) {
	_ = ctx // 预留 context 用于未来超时控制
	if cfg == nil || cfg.RabbitMQ == nil {
		return nil, errorx.New(errorx.CodeConfigError, "rabbitmq config is nil")
	}

	conn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		return nil, errorx.Wrap(err, errorx.CodeInfraError, "connect to rabbitmq failed")
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, errorx.Wrap(err, errorx.CodeInfraError, "create rabbitmq channel failed")
	}

	mqConn := &MQConnection{
		conn:    conn,
		channel: ch,
	}

	// Ping 验证连接可用性
	if err := mqConn.ping(); err != nil {
		mqConn.Close()
		return nil, errorx.Wrap(err, errorx.CodeInfraError, "rabbitmq ping failed")
	}

	zap.L().Info("init rabbitmq connection success",
		zap.String("url", cfg.RabbitMQ.URL),
	)

	return mqConn, nil
}

// ping 通过声明一个临时队列来验证连接可用性
func (m *MQConnection) ping() error {
	_, err := m.channel.QueueDeclarePassive("", true, false, false, false, nil)
	return err
}

// DeclareExchanges 声明所有 Exchange
func (m *MQConnection) DeclareExchanges() error {
	// vote.exchange (direct): 投票异步计数
	if err := m.channel.ExchangeDeclare(
		ExchangeVote, // name
		"direct",     // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments
	); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "declare exchange %s failed", ExchangeVote)
	}

	// audit.exchange (direct): AI 内容审核
	if err := m.channel.ExchangeDeclare(
		ExchangeAudit,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "declare exchange %s failed", ExchangeAudit)
	}

	// search.exchange (direct): ES 数据同步
	if err := m.channel.ExchangeDeclare(
		ExchangeSearch,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "declare exchange %s failed", ExchangeSearch)
	}

	// notify.exchange (fanout): 通知推送
	if err := m.channel.ExchangeDeclare(
		ExchangeNotify,
		"fanout",
		true,
		false,
		false,
		false,
		nil,
	); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "declare exchange %s failed", ExchangeNotify)
	}

	zap.L().Info("declare all exchanges success",
		zap.String("exchanges", fmt.Sprintf("%s, %s, %s, %s",
			ExchangeVote, ExchangeAudit, ExchangeSearch, ExchangeNotify)),
	)

	return nil
}

// DeclareQueues 声明所有 Queue 并绑定到对应 Exchange
func (m *MQConnection) DeclareQueues() error {
	// vote.queue -> vote.exchange (routing key: vote.count)
	if err := m.declareAndBind(QueueVote, ExchangeVote, RoutingKeyVote, "direct"); err != nil {
		return err
	}

	// audit.queue -> audit.exchange (routing keys: audit.post, audit.remark)
	if err := m.declareAndBind(QueueAudit, ExchangeAudit, RoutingKeyAuditPost, "direct"); err != nil {
		return err
	}
	if err := m.declareAndBind(QueueAudit, ExchangeAudit, RoutingKeyAuditRemark, "direct"); err != nil {
		return err
	}

	// search.queue -> search.exchange (routing key: search.sync)
	if err := m.declareAndBind(QueueSearch, ExchangeSearch, RoutingKeySearch, "direct"); err != nil {
		return err
	}

	// notify.queue -> notify.exchange (fanout, no routing key needed)
	if err := m.declareAndBind(QueueNotify, ExchangeNotify, "", "fanout"); err != nil {
		return err
	}

	zap.L().Info("declare all queues and bindings success",
		zap.String("queues", fmt.Sprintf("%s, %s, %s, %s",
			QueueVote, QueueAudit, QueueSearch, QueueNotify)),
	)

	return nil
}

// declareAndBind 声明队列并绑定到 Exchange
func (m *MQConnection) declareAndBind(queueName, exchangeName, routingKey, exchangeType string) error {
	_, err := m.channel.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "declare queue %s failed", queueName)
	}

	// fanout exchange 不需要 routing key
	if exchangeType != "fanout" {
		if err := m.channel.QueueBind(
			queueName,    // queue name
			routingKey,   // routing key
			exchangeName, // exchange
			false,        // no-wait
			nil,          // arguments
		); err != nil {
			return errorx.Wrapf(err, errorx.CodeInfraError,
				"bind queue %s to exchange %s with key %s failed",
				queueName, exchangeName, routingKey)
		}
	}
	return nil
}

// Channel 获取底层 AMQP Channel
func (m *MQConnection) Channel() *amqp.Channel {
	return m.channel
}

// Close 关闭 RabbitMQ 连接
func (m *MQConnection) Close() {
	if m.channel != nil {
		_ = m.channel.Close()
	}
	if m.conn != nil {
		_ = m.conn.Close()
	}
	zap.L().Info("rabbitmq connection closed")
}
