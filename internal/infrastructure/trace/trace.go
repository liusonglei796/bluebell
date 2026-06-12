package trace

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TracerForModule(module string) trace.Tracer {
	return otel.Tracer(module)
}

func WithUserID(ctx context.Context, userID int64) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Int64("user.id", userID))
}

func WithPostID(ctx context.Context, postID int64) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Int64("post.id", postID))
}

func SetSpanSuccess(ctx context.Context) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Bool("success", true))
}

func TraceIDString(ctx context.Context) string {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if !sc.IsValid() {
		return "<no-trace>"
	}
	return sc.TraceID().String()
}
