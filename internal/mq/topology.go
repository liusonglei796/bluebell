package mq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeTopic = "bluebell.topic"
	ExchangeDLX   = "bluebell.dlx"

	// 业务正常工作队列
	QueueNotification = "q.notification.worker"
	QueueCounter      = "q.counter.worker"
	QueueFeed         = "q.feed.worker"

	// 业务专属死信队列 (Per-Business Dead Letter Queues)
	QueueNotificationDLQ = "q.notification.dlq"
	QueueCounterDLQ      = "q.counter.dlq"
	QueueFeedDLQ         = "q.feed.dlq"

	// 死信路由键
	DLXRoutingNotification = "dlx.notification"
	DLXRoutingCounter      = "dlx.counter"
	DLXRoutingFeed         = "dlx.feed"
)

// SetupTopology 声明交换机、队列、业务专属死信队列及路由绑定关系
func SetupTopology(ch *amqp.Channel) error {
	if ch == nil {
		return nil
	}

	// 1. 声明 Topic Exchange (正常业务分发)
	if err := ch.ExchangeDeclare(
		ExchangeTopic,
		"topic",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		return fmt.Errorf("declare exchange %s failed: %w", ExchangeTopic, err)
	}

	// 2. 声明 DLX Exchange (死信交换机)
	if err := ch.ExchangeDeclare(
		ExchangeDLX,
		"topic",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		return fmt.Errorf("declare exchange %s failed: %w", ExchangeDLX, err)
	}

	// 3. 按业务声明各自专属的死信队列 (DLQ)
	dlqQueues := []struct {
		Name       string
		RoutingKey string
	}{
		{Name: QueueNotificationDLQ, RoutingKey: DLXRoutingNotification},
		{Name: QueueCounterDLQ, RoutingKey: DLXRoutingCounter},
		{Name: QueueFeedDLQ, RoutingKey: DLXRoutingFeed},
	}

	for _, dlq := range dlqQueues {
		if _, err := ch.QueueDeclare(
			dlq.Name,
			true,  // durable
			false, // auto-delete
			false, // exclusive
			false, // no-wait
			nil,
		); err != nil {
			return fmt.Errorf("declare DLQ %s failed: %w", dlq.Name, err)
		}
		if err := ch.QueueBind(dlq.Name, dlq.RoutingKey, ExchangeDLX, false, nil); err != nil {
			return fmt.Errorf("bind DLQ %s to %s with rk %s failed: %w", dlq.Name, ExchangeDLX, dlq.RoutingKey, err)
		}
	}

	// 4. 声明各业务 Worker 队列
	queues := []struct {
		Name          string
		DLXRoutingKey string
		Bindings      []string
	}{
		{
			Name:          QueueNotification,
			DLXRoutingKey: DLXRoutingNotification,
			Bindings: []string{
				"event.comment.created",
				"event.user.followed",
				"event.post.published",
			},
		},
		{
			Name:          QueueCounter,
			DLXRoutingKey: DLXRoutingCounter,
			Bindings: []string{
				"event.vote.cast",
				"event.comment.created",
				"event.user.followed",
			},
		},
		{
			Name:          QueueFeed,
			DLXRoutingKey: DLXRoutingFeed,
			Bindings: []string{
				"event.post.published",
				"event.user.followed",
			},
		},
	}

	for _, q := range queues {
		args := amqp.Table{
			"x-dead-letter-exchange":    ExchangeDLX,
			"x-dead-letter-routing-key": q.DLXRoutingKey,
		}

		if _, err := ch.QueueDeclare(
			q.Name,
			true,  // durable
			false, // auto-delete
			false, // exclusive
			false, // no-wait
			args,
		); err != nil {
			return fmt.Errorf("declare queue %s failed: %w", q.Name, err)
		}

		for _, rk := range q.Bindings {
			if err := ch.QueueBind(q.Name, rk, ExchangeTopic, false, nil); err != nil {
				return fmt.Errorf("bind queue %s to %s with rk %s failed: %w", q.Name, ExchangeTopic, rk, err)
			}
		}
	}

	return nil
}
