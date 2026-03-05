package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GinLogger 日志中间件
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// 1. 执行后续中间件和业务逻辑
		c.Next()
		// 2. 只有在业务处理完后才计算耗时
		cost := time.Since(start)

		// 3. 性能优化：构造字段切片
		// 预分配 8-10 个容量，减少 append 导致的扩容
		fields := make([]zap.Field, 0, 10)
		fields = append(fields,
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query",c.Request.URL.RawQuery),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Duration("cost", cost),
		)

		// 4. 只有存在错误时才添加 errors 字段，避免空字符串占用空间
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()))
		}

		// 5. 根据状态码自动切换日志级别
		if c.Writer.Status() >= 500 {
			zap.L().Error("server error", fields...)
		} else if c.Writer.Status() >= 400 {
			zap.L().Warn("client error", fields...)
		} else {
			zap.L().Info("http request", fields...)
		}
	}
}

// GinRecovery panic 恢复中间件
func GinRecovery(stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				// 1. 检查是否为 broken pipe
				err, ok := rec.(error)
				if ok && isBrokenPipeError(err) {
					zap.L().Error("broken pipe", zap.Error(err), zap.String("path", c.Request.URL.Path))
					c.Abort()
					return
				}

				// 2. 构造日志字段
				fields := []zap.Field{
					zap.Any("panic", rec),
					zap.String("path", c.Request.URL.Path),
				}

				// 如果开启 stack，利用 zap 的内置 Stack 提高性能和可读性
				if stack {
					fields = append(fields, zap.Stack("stacktrace"))
				}

				zap.L().Error("[Recovery from panic]", fields...)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

func isBrokenPipeError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "broken pipe") || strings.Contains(msg, "connection reset by peer")
}
