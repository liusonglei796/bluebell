package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// 全局仪表
var meter metric.Meter

// ====== 业务 Counter ======

// PostsCreated 帖子创建总数
var PostsCreated metric.Int64Counter

// Votes 投票总数
var Votes metric.Int64Counter

// UsersRegistered 用户注册总数
var UsersRegistered metric.Int64Counter

// UsersLoggedIn 用户登录总数
var UsersLoggedIn metric.Int64Counter

// Comments 评论总数
var Comments metric.Int64Counter

// TokenRefreshes token 刷新总数
var TokenRefreshes metric.Int64Counter

// Errors 按类型区分的错误总数
var Errors metric.Int64Counter

// ====== HTTP Request Duration Histogram ======

// HTTPRequestDuration HTTP 请求延迟直方图（秒）
var HTTPRequestDuration metric.Float64Histogram

// Init 初始化自定义业务指标。
// 必须在 InitOTEL 之后调用，以确保全局 MeterProvider 已设置。
func Init(serviceName string) error {
	meter = otel.Meter(serviceName)

	var err error

	PostsCreated, err = meter.Int64Counter("bluebell.posts.created.total",
		metric.WithDescription("帖子创建总数"),
	)
	if err != nil {
		return fmt.Errorf("create posts_created counter: %w", err)
	}

	Votes, err = meter.Int64Counter("bluebell.votes.total",
		metric.WithDescription("投票总数"),
	)
	if err != nil {
		return fmt.Errorf("create votes counter: %w", err)
	}

	UsersRegistered, err = meter.Int64Counter("bluebell.users.registered.total",
		metric.WithDescription("用户注册总数"),
	)
	if err != nil {
		return fmt.Errorf("create users_registered counter: %w", err)
	}

	UsersLoggedIn, err = meter.Int64Counter("bluebell.users.logged_in.total",
		metric.WithDescription("用户登录总数"),
	)
	if err != nil {
		return fmt.Errorf("create users_logged_in counter: %w", err)
	}

	Comments, err = meter.Int64Counter("bluebell.comments.total",
		metric.WithDescription("评论总数"),
	)
	if err != nil {
		return fmt.Errorf("create comments counter: %w", err)
	}

	TokenRefreshes, err = meter.Int64Counter("bluebell.token_refreshes.total",
		metric.WithDescription("Token 刷新总数"),
	)
	if err != nil {
		return fmt.Errorf("create token_refreshes counter: %w", err)
	}

	Errors, err = meter.Int64Counter("bluebell.errors.total",
		metric.WithDescription("按错误类型分类的错误总数"),
	)
	if err != nil {
		return fmt.Errorf("create errors counter: %w", err)
	}

	HTTPRequestDuration, err = meter.Float64Histogram("bluebell.http.request.duration.seconds",
		metric.WithDescription("HTTP 请求延迟（秒）"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	if err != nil {
		return fmt.Errorf("create http_request_duration histogram: %w", err)
	}

	return nil
}

// RecordError 便捷函数：记录带错误类型的 error counter
func RecordError(ctx context.Context, errorType string) {
	Errors.Add(ctx, 1, metric.WithAttributes(
		AttributeErrorType.String(errorType),
	))
}

// RecordSuccess 便捷函数：记录不受限的业务 counter
func RecordSuccess(ctx context.Context, counter metric.Int64Counter) {
	counter.Add(ctx, 1)
}

// RecordHTTPDuration 便捷函数：记录 HTTP 请求延迟到 Histogram
func RecordHTTPDuration(ctx context.Context, duration float64, method string, statusCode int) {
	HTTPRequestDuration.Record(ctx, duration, metric.WithAttributes(
		AttributeHTTPMethod.String(method),
		AttributeHTTPStatus.Int(statusCode),
	))
}
