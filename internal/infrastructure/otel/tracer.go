package otel

import (
	"context"
	"fmt"
	"time"

	"bluebell/internal/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracerProvider 初始化全局 TracerProvider
// 返回一个 shutdown 函数，应在应用退出时调用
func InitTracerProvider(cfg *config.Config) func() {
	// 如果未启用 OTel，返回空 shutdown 函数
	if cfg.Otel == nil || !cfg.Otel.Enabled {
		return func() {}
	}

	endpoint := cfg.Otel.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:4318"
	}

	serviceName := cfg.Otel.ServiceName
	if serviceName == "" {
		serviceName = "bluebell"
	}

	version := "dev"
	if cfg.App != nil && cfg.App.Version != "" {
		version = cfg.App.Version
	}

	ctx := context.Background()

	// 创建 OTLP gRPC exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		fmt.Printf("[OTel] Failed to create exporter: %v\n", err)
		return func() {}
	}

	// 创建 resource，包含 service.name 和 service.version 属性
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
		),
	)
	if err != nil {
		fmt.Printf("[OTel] Failed to create resource: %v\n", err)
		return func() {}
	}

	// 创建 TracerProvider，使用 batcher 批量导出
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// 设置为全局 TracerProvider
	otel.SetTracerProvider(tp)

	fmt.Printf("[OTel] TracerProvider initialized (endpoint=%s, service=%s)\n", endpoint, serviceName)

	// 返回 shutdown 函数
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			fmt.Printf("[OTel] Failed to shutdown TracerProvider: %v\n", err)
		}
	}
}
