package mq

import (
	"bluebell/internal/config"
	"bluebell/pkg/errorx"
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// ==================== 常量 ====================

const (
	ExchangeVote   = "vote.exchange"
	QueueVote      = "vote.queue"
	RoutingKeyVote = "vote.count"

	ExchangeAudit         = "audit.exchange"
	QueueAudit            = "audit.queue"
	RoutingKeyAuditPost   = "audit.post"
	RoutingKeyAuditRemark = "audit.remark"

	ExchangeSearch   = "search.exchange"
	QueueSearch      = "search.queue"
	RoutingKeySearch = "search.sync"
)

// ==================== MQConnection ====================

type MQConnection struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func newMQConnection(ctx context.Context, cfg *config.Config) (*MQConnection, error) {
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

	mqConn := &MQConnection{conn: conn, channel: ch}

	if err := mqConn.ping(); err != nil {
		mqConn.Close()
		return nil, errorx.Wrap(err, errorx.CodeInfraError, "rabbitmq ping failed")
	}

	zap.L().Info("init rabbitmq connection success", zap.String("url", cfg.RabbitMQ.URL))
	return mqConn, nil
}

func (m *MQConnection) ping() error {
	q, err := m.channel.QueueDeclare("", false, true, true, false, nil)
	if err != nil {
		return err
	}
	_, err = m.channel.QueueDelete(q.Name, false, false, false)
	return err
}

func (m *MQConnection) DeclareExchanges() error {
	if err := m.channel.ExchangeDeclare(ExchangeVote, "direct", true, false, false, false, nil); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "declare exchange %s failed", ExchangeVote)
	}
	if err := m.channel.ExchangeDeclare(ExchangeAudit, "direct", true, false, false, false, nil); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "declare exchange %s failed", ExchangeAudit)
	}
	if err := m.channel.ExchangeDeclare(ExchangeSearch, "direct", true, false, false, false, nil); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "declare exchange %s failed", ExchangeSearch)
	}
	zap.L().Info("declare all exchanges success")
	return nil
}

func (m *MQConnection) DeclareQueues() error {
	// vote
	if _, err := m.channel.QueueDeclare(QueueVote, true, false, false, false, nil); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "declare queue %s failed", QueueVote)
	}
	if err := m.channel.QueueBind(QueueVote, RoutingKeyVote, ExchangeVote, false, nil); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "bind queue %s failed", QueueVote)
	}

	// audit
	if _, err := m.channel.QueueDeclare(QueueAudit, true, false, false, false, nil); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "declare queue %s failed", QueueAudit)
	}
	if err := m.channel.QueueBind(QueueAudit, RoutingKeyAuditPost, ExchangeAudit, false, nil); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "bind queue %s failed", QueueAudit)
	}
	if err := m.channel.QueueBind(QueueAudit, RoutingKeyAuditRemark, ExchangeAudit, false, nil); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "bind queue %s failed", QueueAudit)
	}

	// search
	if _, err := m.channel.QueueDeclare(QueueSearch, true, false, false, false, nil); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "declare queue %s failed", QueueSearch)
	}
	if err := m.channel.QueueBind(QueueSearch, RoutingKeySearch, ExchangeSearch, false, nil); err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "bind queue %s failed", QueueSearch)
	}

	zap.L().Info("declare all queues and bindings success")
	return nil
}

func (m *MQConnection) Channel() *amqp.Channel {
	return m.channel
}

func (m *MQConnection) Close() {
	if m.channel != nil {
		_ = m.channel.Close()
	}
	if m.conn != nil {
		_ = m.conn.Close()
	}
	zap.L().Info("rabbitmq connection closed")
}
