package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"bluebell/internal/config"
)

// Init 初始化 Logger
func Init(cfg *config.Config, mode string) (err error) {
	writeSyncer := getWriteSyncer(cfg.Log.FileName, cfg.Log.MaxSize, cfg.Log.MaxAge, cfg.Log.MaxBackups)
	var core zapcore.Core
	if mode == "dev" {
		filecore := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()), writeSyncer, zapcore.DebugLevel)
		consolecore := zapcore.NewCore(zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()), os.Stdout, zapcore.DebugLevel)
		core = zapcore.NewTee(filecore, consolecore)
	} else {
		var level zapcore.Level
		level.UnmarshalText([]byte(cfg.Log.Level))
		core = zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), writeSyncer, level)
	}
	logger := zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(logger)
	return nil
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
