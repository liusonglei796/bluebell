package otel

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"bluebell/internal/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

// prometheusHandler 保存 Prometheus exporter 的 HTTP handler，供 /metrics 端点使用
var prometheusHandler http.Handler

// InitOTEL 初始化 OpenTelemetry SDK（Traces + Metrics + Logs）。
// 它会设置全局的 TracerProvider, MeterProvider 和 LoggerProvider。
func InitOTEL(ctx context.Context, cfg *config.OtelConfig) (func(context.Context) error, error) {
	if cfg == nil || !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	// 1. 创建资源描述
	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceNameKey.String(cfg.ServiceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 2. 共享 gRPC 连接
	conn, err := grpc.NewClient(
		cfg.Endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(10 * 1024 * 1024),
		),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  100 * time.Millisecond,
				MaxDelay:   1 * time.Second,
			},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	// 3. TracerProvider
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExporter), sdktrace.WithResource(res))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	// 4. MeterProvider
	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Prometheus exporter（用于 /metrics scrape）
	promRegistry := prometheus.NewRegistry()
	promExporter, err := otelprom.New(otelprom.WithRegisterer(promRegistry))
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}
	prometheusHandler = promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{})

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(10*time.Second))),
		sdkmetric.WithReader(promExporter),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)

	// 5. LoggerProvider
	logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}
	lp := sdklog.NewLoggerProvider(sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)), sdklog.WithResource(res))
	global.SetLoggerProvider(lp)

	// 6. Shutdown
	shutdown := func(ctx context.Context) error {
		var errs []error
		if err := tp.Shutdown(ctx); err != nil { errs = append(errs, err) }
		if err := mp.Shutdown(ctx); err != nil { errs = append(errs, err) }
		if err := lp.Shutdown(ctx); err != nil { errs = append(errs, err) }
		if err := conn.Close(); err != nil { errs = append(errs, err) }
		if len(errs) > 0 { return fmt.Errorf("shutdown errors: %v", errs) }
		return nil
	}

	return shutdown, nil
}

// GetPrometheusHandler 返回 Prometheus metrics HTTP handler（用于 /metrics 端点）。
// 如果 OTel 未启用（InitOTEL 未调用或返回 no-op），则返回 nil。
func GetPrometheusHandler() http.Handler {
	return prometheusHandler
}
