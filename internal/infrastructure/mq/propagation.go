package mq

import (
	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel/propagation"
)

// AmqpHeadersCarrier 为 RabbitMQ 的 Headers (amqp.Table) 实现 TextMapCarrier 接口。
// 这使得 OpenTelemetry 能够将 Trace 上下文注入到 MQ 消息头中，或从中提取。
type AmqpHeadersCarrier amqp.Table

// Get 从 Headers 中获取指定 key 的值
func (c AmqpHeadersCarrier) Get(key string) string {
	if v, ok := c[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// Set 向 Headers 中设置指定 key 的值
func (c AmqpHeadersCarrier) Set(key string, value string) {
	c[key] = value
}

// Keys 返回 Headers 中所有的 key
func (c AmqpHeadersCarrier) Keys() []string {
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// 确保 AmqpHeadersCarrier 实现了 propagation.TextMapCarrier 接口
var _ propagation.TextMapCarrier = AmqpHeadersCarrier{}
