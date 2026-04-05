package middleware

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	// userIDKey 与 auth 中间件中存储用户 ID 的 key 保持一致
	userIDKey = "UserIDKey"
)

// TracingMiddleware 返回用于 Gin 的 OpenTelemetry 追踪中间件。
// 该中间件会自动为每个 HTTP 请求创建 trace span，并记录 http.method、http.url 等属性。
// 使用空字符串 tracer name 以继承全局 TracerProvider 配置的 service.name。
func TracingMiddleware() gin.HandlerFunc {
	return otelgin.Middleware("")
}

// StartSpan 从 gin.Context 获取当前 trace 并创建一个子 span。
// tracerName 用于标识 span 来源模块，如 "bluebell/handler"。
// 适用于在 handler 或 service 层手动添加自定义 span 的场景。
// 返回的 context 应传递给后续调用，span 需由调用者负责 End()。
//
// 使用示例:
//
//	ctx, span := middleware.StartSpan(c, "bluebell/handler", "CreatePost")
//	defer span.End()
func StartSpan(c *gin.Context, tracerName, spanName string) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)

	// 从 gin.Context 获取底层 context（otelgin 中间件已将 trace 注入其中）
	ctx := c.Request.Context()

	// 添加通用属性
	attrs := []attribute.KeyValue{
		attribute.String("http.method", c.Request.Method),
		attribute.String("http.url", c.Request.URL.Path),
	}

	// 尝试从 gin.Context 中获取用户 ID
	if userID, exists := c.Get(userIDKey); exists {
		attrs = append(attrs, attribute.String("user.id", fmt.Sprintf("%v", userID)))
	}

	ctx, span := tracer.Start(ctx, spanName, trace.WithAttributes(attrs...))

	// 将更新后的 context 写回 gin.Context，以便后续代码能获取到 span 信息
	c.Request = c.Request.WithContext(ctx)

	return ctx, span
}

// StartSpanFromContext 从标准 context.Context 创建子 span。
// tracerName 用于标识 span 来源模块，如 "bluebell/service"。
// 适用于 service 层等无法访问 gin.Context 的场景。
// 要求上游已通过 otelgin 中间件或手动将 trace 注入到 context 中。
//
// 使用示例 (service 层):
//
//	ctx, span := middleware.StartSpanFromContext(ctx, "bluebell/service", "VoteForPost",
//	    attribute.Int64("user.id", userID),
//	    attribute.String("post.id", postIDStr),
//	)
//	defer span.End()
func StartSpanFromContext(ctx context.Context, tracerName, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	return tracer.Start(ctx, spanName, trace.WithAttributes(attrs...))
}

// RecordError 在 span 上记录错误信息，并将 span 状态设置为 Error。
// 适用于在业务逻辑中捕获错误后记录到当前 trace 的场景。
//
// 使用示例:
//
//	if err := someOperation(); err != nil {
//	    middleware.RecordError(span, err)
//	    return err
//	}
func RecordError(span trace.Span, err error) {
	if err == nil || span == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
