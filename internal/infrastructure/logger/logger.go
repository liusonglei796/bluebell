package logger

import (
	"context"
	"log"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"bluebell/internal/config"
)

// Init 初始化 Logger
func Init(cfg *config.Config, mode string) error {
	writeSyncer := getWriteSyncer(cfg.Log.FileName, cfg.Log.MaxSize, cfg.Log.MaxAge, cfg.Log.MaxBackups)

	// Redirect std log (log.Printf/Println) to file instead of stderr
	log.SetOutput(writeSyncer)

	var cores []zapcore.Core

	encoderConfig := zap.NewProductionEncoderConfig()
	if mode == "dev" {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}

	// 文件核心 (JSON)
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writeSyncer,
		zap.DebugLevel,
	)
	cores = append(cores, fileCore)

	// 控制台核心 (Console)
	if mode == "dev" {
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			os.Stdout,
			zap.DebugLevel,
		)
		cores = append(cores, consoleCore)
	}

	zapLogger := zap.New(zapcore.NewTee(cores...), zap.AddCaller())
	zap.ReplaceGlobals(zapLogger)

	return nil
}

// WithContext 兼容接口：直接返回全局 Logger
func WithContext(_ context.Context) *zap.Logger {
	return zap.L()
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
