package logger

import (
	"bluebell/internal/application/port"
	"context"

	"go.uber.org/zap"
)

// zapLogger 是 port.Logger 的 zap 实现（适配器模式）
// 它将应用层的 port.Logger 接口调用，适配到基础设施层的 zap.Logger。
type zapLogger struct{}

// New 创建 port.Logger 的 zap 适配器实例
func New() port.Logger {
	return &zapLogger{}
}

func (l *zapLogger) Error(ctx context.Context, msg string, fields ...port.Field) {
	WithContext(ctx).Error(msg, toZapFields(fields)...)
}

func (l *zapLogger) Warn(ctx context.Context, msg string, fields ...port.Field) {
	WithContext(ctx).Warn(msg, toZapFields(fields)...)
}

func (l *zapLogger) Info(ctx context.Context, msg string, fields ...port.Field) {
	WithContext(ctx).Info(msg, toZapFields(fields)...)
}

// toZapFields 将应用层定义的 port.Field 转换为 zap.Field
func toZapFields(fields []port.Field) []zap.Field {
	zf := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		switch v := f.Value.(type) {
		case error:
			zf = append(zf, zap.Error(v))
		case int64:
			zf = append(zf, zap.Int64(f.Key, v))
		case string:
			zf = append(zf, zap.String(f.Key, v))
		default:
			zf = append(zf, zap.Any(f.Key, v))
		}
	}
	return zf
}
