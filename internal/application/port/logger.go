// Package port 定义应用层的端口接口（Ports）
//
// 按照整洁架构的依赖规则，应用层（Application / Use Case）不应直接依赖基础设施层。
// 这些接口是应用层对外部能力的"需求描述"，由基础设施层提供具体实现（适配器）。
// 依赖方向：infrastructure → application/port（适配器实现端口），而非 application → infrastructure。
package port

import "context"

// Logger 结构化日志端口
//
// 应用层通过此接口记录业务日志，不直接依赖 zap / logrus 等具体实现。
// 具体实现由基础设施层提供（如 infrastructure/logger 中的 zap 适配器）。
type Logger interface {
	// Error 记录错误级别日志
	Error(ctx context.Context, msg string, fields ...Field)
	// Warn 记录警告级别日志
	Warn(ctx context.Context, msg string, fields ...Field)
	// Info 记录信息级别日志
	Info(ctx context.Context, msg string, fields ...Field)
}

// Field 日志字段（键值对）
// 应用层使用此类型传递结构化字段，与具体的日志框架解耦。
type Field struct {
	Key   string
	Value interface{}
}

// 便捷构造函数，模拟 zap 的 API 风格，降低迁移成本。

func Int64(key string, val int64) Field {
	return Field{Key: key, Value: val}
}

func String(key string, val string) Field {
	return Field{Key: key, Value: val}
}

func Err(err error) Field {
	return Field{Key: "error", Value: err}
}
