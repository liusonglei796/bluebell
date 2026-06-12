package logger

import (
	"context"
	"os"

	"go.opentelemetry.io/contrib/bridges/otelzap"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"bluebell/internal/config"
)

// Init 初始化 Logger，集成 OTel 桥接以将日志发送至 OTLP。
// 会自动从 OTel 全局注册中心获取 LoggerProvider。
func Init(cfg *config.Config, mode string) error {
	writeSyncer := getWriteSyncer(cfg.Log.FileName, cfg.Log.MaxSize, cfg.Log.MaxAge, cfg.Log.MaxBackups)

	var cores []zapcore.Core

	// 1. 基础配置
	encoderConfig := zap.NewProductionEncoderConfig()
	if mode == "dev" {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	// 2. 文件核心 (JSON)
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writeSyncer,
		zap.DebugLevel,
	)
	cores = append(cores, fileCore)

	// 3. 控制台核心 (Console)
	if mode == "dev" {
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			os.Stdout,
			zap.DebugLevel,
		)
		cores = append(cores, consoleCore)
	}

	// 4. OTel 桥接核心 (发送到 OTLP/Loki)
	// 从全局注册中心获取 LoggerProvider (在 InitOTEL 中设置)
	lp := global.GetLoggerProvider()
	if lp != nil {
		otelCore := otelzap.NewCore("bluebell", otelzap.WithLoggerProvider(lp))
		cores = append(cores, otelCore)
	}

	// 5. 创建底层 Zap Logger
	zapLogger := zap.New(zapcore.NewTee(cores...), zap.AddCaller())

	// 6. 替换全局 Logger (兼容某些直接用 zap.L() 的第三方库)
	zap.ReplaceGlobals(zapLogger)

	return nil
}

// WithContext 从 context 中提取 trace_id / span_id 并附加到日志字段。
// 在业务代码中使用 logger.WithContext(ctx).Info("xxx") 即可自动关联链路。
func WithContext(ctx context.Context) *zap.Logger {
	l := zap.L()
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return l.With(
			zap.String("trace_id", span.SpanContext().TraceID().String()),
			zap.String("span_id", span.SpanContext().SpanID().String()),
		)
	}
	return l
}

func getWriteSyncer(filename string, maxsize int, maxage int, maxbackups int) zapcore.WriteSyncer {
	lumberjackLogger := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxsize,
		MaxAge:     maxage,
		MaxBackups: maxbackups,
	}
	return zapcore.AddSync(lumberjackLogger)
}
