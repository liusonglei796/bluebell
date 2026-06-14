package port

import "context"

// nopLogger 空操作日志实现，用于测试或无需日志的场景
type nopLogger struct{}

// NopLogger 返回一个不做任何日志输出的 Logger 实例
func NopLogger() Logger {
	return &nopLogger{}
}

func (l *nopLogger) Error(_ context.Context, _ string, _ ...Field) {}
func (l *nopLogger) Warn(_ context.Context, _ string, _ ...Field)  {}
func (l *nopLogger) Info(_ context.Context, _ string, _ ...Field)  {}
