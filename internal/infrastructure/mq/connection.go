package mq

import (
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

// ==================== 资源常量 ====================

const (
	ExchangeVote   = "vote.exchange"
	QueueVote      = "vote.queue"
	RoutingKeyVote = "vote.count"

	ExchangeSearch   = "search.exchange"
	QueueSearch      = "search.queue"
	RoutingKeySearch = "search.sync"

	ExchangeActivity   = "activity.exchange"
	QueueActivity      = "activity.queue"
	RoutingKeyActivity = "activity.event"
)

// ==================== 手动挡工厂方法 ====================

// Dial 建立纯粹的 TCP 连接 (物理管道)
func Dial(url string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("RabbitMQ 连接失败: %w", err)
	}
	log.Printf("RabbitMQ 连接已建立: %s", url)
	return conn, nil
}

// SetupResources 在指定的信道上声明交换机和队列
// 一次性装修工作，装修完信道即可关闭
func SetupResources(ch *amqp.Channel) error {
	exchanges := []struct{ Name, Kind string }{
		{ExchangeVote, "direct"},
		{ExchangeSearch, "direct"},
		{ExchangeActivity, "direct"},
	}
	for _, ex := range exchanges {
		if err := ch.ExchangeDeclare(ex.Name, ex.Kind, true, false, false, false, nil); err != nil {
			return fmt.Errorf("声明交换机 %s 失败: %w", ex.Name, err)
		}
	}

	queues := []struct {
		Name     string
		Exchange string
		RK       string
	}{
		{QueueVote, ExchangeVote, RoutingKeyVote},
		{QueueSearch, ExchangeSearch, RoutingKeySearch},
		{QueueActivity, ExchangeActivity, RoutingKeyActivity},
	}
	for _, q := range queues {
		if _, err := ch.QueueDeclare(q.Name, true, false, false, false, nil); err != nil {
			return fmt.Errorf("声明队列 %s 失败: %w", q.Name, err)
		}
		if err := ch.QueueBind(q.Name, q.RK, q.Exchange, false, nil); err != nil {
			return fmt.Errorf("绑定队列 %s 失败: %w", q.Name, err)
		}
	}

	log.Println("RabbitMQ 交换机、队列、绑定关系初始化完毕")
	return nil
}
