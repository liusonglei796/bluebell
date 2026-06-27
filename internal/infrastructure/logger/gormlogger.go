package logger

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// GormZapLogger 实现 gormlogger.Interface，将 GORM 日志桥接到 Zap。
// 慢查询和 SQL 错误通过 OTel 链路发送到 Loki，可在 Grafana 中检索并关联 trace_id。
type GormZapLogger struct {
	SlowThreshold             time.Duration
	LogLevel                  gormlogger.LogLevel
	IgnoreRecordNotFoundError bool
}

// NewGormLogger 创建 GORM → Zap 的日志适配器。
//   - mode="dev":  Info 级别，记录所有 SQL（开发调试用），慢查询阈值 10ms
//   - mode!="dev": Warn 级别，仅记录慢查询和错误（生产环境），慢查询阈值 200ms
func NewGormLogger(mode string) gormlogger.Interface {
	if mode == "dev" {
		return &GormZapLogger{
			SlowThreshold:             10 * time.Millisecond,
			LogLevel:                  gormlogger.Info,
			IgnoreRecordNotFoundError: true,
		}
	}
	return &GormZapLogger{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  gormlogger.Warn,
		IgnoreRecordNotFoundError: true,
	}
}

// LogMode 返回指定日志级别的新 logger 实例。
func (l *GormZapLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info 转发 GORM Info 级别日志到 Zap（连接信息、迁移提示等）。
func (l *GormZapLogger) Info(_ context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Info {
		zap.L().Info(msg, zap.Any("args", args))
	}
}

// Warn 转发 GORM Warn 级别日志到 Zap。
func (l *GormZapLogger) Warn(_ context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Warn {
		zap.L().Warn(msg, zap.Any("args", args))
	}
}

// Error 转发 GORM Error 级别日志到 Zap（SQL 执行错误）。
func (l *GormZapLogger) Error(_ context.Context, msg string, args ...interface{}) {
	if l.LogLevel >= gormlogger.Error {
		zap.L().Error(msg, zap.Any("args", args))
	}
}

// Trace 拦截 GORM 的 SQL 执行追踪，实现慢查询和错误日志的结构化输出。
// 慢查询日志自动携带 trace_id（通过 WithContext），可在 Grafana 中
// 从 Loki 日志直接跳转到 Tempo 链路。
func (l *GormZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.String("sql", sql),
		zap.Int64("rows", rows),
		zap.Duration("elapsed", elapsed),
	}

	zl := WithContext(ctx)

	// SQL 执行错误
	if err != nil {
		if l.IgnoreRecordNotFoundError && errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		if l.LogLevel >= gormlogger.Error {
			zl.Error("gorm sql error", append(fields, zap.Error(err))...)
		}
		return
	}

	// 慢查询
	if l.SlowThreshold > 0 && elapsed > l.SlowThreshold {
		if l.LogLevel >= gormlogger.Warn {
			zl.Warn("gorm slow query", fields...)
		}
		return
	}

	// 正常 SQL（仅 dev 模式 Info 级别）
	if l.LogLevel >= gormlogger.Info {
		zl.Debug("gorm sql", fields...)
	}
}
