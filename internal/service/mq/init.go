package mq

import (
	"bluebell/internal/config"
	"bluebell/pkg/errorx"
	"context"
	"go.uber.org/zap"
)

// InitMQ 初始化 RabbitMQ 完整基础设施
// 流程: 建立连接 → 声明 Exchange → 声明 Queue & Binding → 创建 Publisher
// 返回: MQConnection, MQPublisher, error
func InitMQ(ctx context.Context, cfg *config.Config) (*MQConnection, *MQPublisher, error) {
	// 1. 建立连接
	conn, err := newMQConnection(ctx, cfg)
	if err != nil {
		return nil, nil, errorx.Wrap(err, errorx.CodeInfraError, "init rabbitmq connection failed")
	}

	// 2. 声明所有 Exchange
	if err := conn.DeclareExchanges(); err != nil {
		conn.Close()
		return nil, nil, errorx.Wrap(err, errorx.CodeInfraError, "declare rabbitmq exchanges failed")
	}

	// 3. 声明所有 Queue 并绑定
	if err := conn.DeclareQueues(); err != nil {
		conn.Close()
		return nil, nil, errorx.Wrap(err, errorx.CodeInfraError, "declare rabbitmq queues failed")
	}

	// 4. 创建 Publisher
	publisher := newPublisher(conn)

	zap.L().Info("init rabbitmq infrastructure success",
		zap.String("publisher", publisher.String()),
	)

	return conn, publisher, nil
}
