package logger

import (
	"bluebell/pkg/errorx"
	"errors"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"bluebell/internal/config"
)

// Init 初始化 Logger
func Init(cfg *config.LogConfig, mode string) (err error) {
	if cfg == nil {
		return errorx.New(errorx.CodeConfigError, "logger.Init 收到空配置")
	}

	writeSyncer := getLogWriter(
		cfg.FileName,
		cfg.MaxSize,
		cfg.MaxBackups,
		cfg.MaxAge,
	)
	encoder := getEncoder()

	var level zapcore.Level
	if err = level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return
	}
	var core zapcore.Core
	if mode == "dev" || mode == gin.DebugMode {
		consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
		fileCore := zapcore.NewCore(encoder, writeSyncer, level)
		consoleCore := zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), zapcore.DebugLevel)
		core = zapcore.NewTee(fileCore, consoleCore)
	} else {
		core = zapcore.NewCore(encoder, writeSyncer, level)
	}
	lg := zap.New(core, zap.AddCaller())
	zap.ReplaceGlobals(lg)
	return
}

func getLogWriter(filename string, maxSize int, maxBackups int, maxAge int) zapcore.WriteSyncer {
	lumberjackLogger := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
	}
	return zapcore.AddSync(lumberjackLogger)
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = "time"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

// GinLogger Gin 日志中间件
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		cost := time.Since(start)

		zap.L().Info("http request",
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("CLientIP", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("cost", cost),
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
		)
	}
}

// GinRecovery panic 恢复中间件
func GinRecovery(stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				var brokenPipe bool
				if err, ok := rec.(error); ok {
					brokenPipe = isBrokenPipeError(err)
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)
				requestStr := string(httpRequest)

				fields := []zap.Field{
					zap.Any("error", rec),
					zap.String("request", requestStr),
				}

				if brokenPipe {
					zap.L().Error("broken pipe",
						append(fields, zap.String("path", c.Request.URL.Path))...,
					)
					c.Error(rec.(error))
					c.Abort()
					return
				}

				if stack {
					fields = append(fields, zap.String("stack", string(debug.Stack())))
				}
				zap.L().Error("[Recovery from panic]", fields...)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

func isBrokenPipeError(err error) bool {
	if err == nil {
		return false
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		var syscallErr *os.SyscallError
		if errors.As(opErr.Err, &syscallErr) {
			msg := strings.ToLower(syscallErr.Error())
			return strings.Contains(msg, "broken pipe") ||
				strings.Contains(msg, "connection reset by peer")
		}
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset by peer")
}
