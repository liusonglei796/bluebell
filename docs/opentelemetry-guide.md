# OpenTelemetry 使用指南

## 概述

本项目使用 **OpenTelemetry (OTel)** 标准方案实现分布式追踪。通过 W3C Trace Context 协议管理 TraceID 的生成、传递和上报，支持 HTTP 请求追踪、SQL 操作追踪和自定义业务追踪。

---

## 技术架构

### 组件关系

```
外部请求
    │
    ▼
┌─────────────────────────────────────────┐
│ otelgin.Middleware (HTTP 入口追踪)      │
│ - 自动提取/生成 W3C Trace Context       │
│ - 创建根 span                           │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ GinLogger (日志集成)                    │
│ - 从 context 提取 traceID 注入日志      │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ Handler / Service 层 (自定义 Span)     │
│ - StartSpan / StartSpanFromContext     │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ Repository 层 (GORM SQL 追踪)           │
│ - otelgorm 插件自动追踪                 │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ OTLP Exporter (数据上报)                │
│ - 默认发送到 localhost:4318             │
└─────────────────────────────────────────┘
```

---

## 依赖配置

项目 `go.mod` 中包含以下 OTel 相关依赖：

```go
require (
    go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.67.0
    go.opentelemetry.io/otel v1.43.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.43.0
    go.opentelemetry.io/otel/sdk v1.43.0
    go.opentelemetry.io/otel/trace v1.43.0
    gorm.io/plugin/opentelemetry v0.1.16
)
```

---

## 配置说明

### 配置文件 (config.yaml)

```yaml
otel:
  enabled: true                    # 是否启用 OTel
  endpoint: "http://localhost:4318"  # OTLP Collector 地址 (gRPC)
  service_name: "bluebell"         # 服务名称
```

### 配置结构体 (internal/config/config.go)

```go
type otelConfig struct {
    Enabled     bool   `mapstructure:"enabled"`
    Endpoint    string `mapstructure:"endpoint"`
    ServiceName string `mapstructure:"service_name"`
}
```

---

## 核心实现

### 1. TracerProvider 初始化

**文件**: `internal/infrastructure/otel/tracer.go`

```go
func InitTracerProvider(cfg *config.Config) func() {
    // 1. 检查是否启用
    if cfg.Otel == nil || !cfg.Otel.Enabled {
        return func() {}
    }

    // 2. 配置默认值
    endpoint := cfg.Otel.Endpoint
    if endpoint == "" {
        endpoint = "http://localhost:4318"
    }

    serviceName := cfg.Otel.ServiceName
    if serviceName == "" {
        serviceName = "bluebell"
    }

    // 3. 创建 OTLP gRPC Exporter
    exporter, err := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(endpoint),
        otlptracegrpc.WithInsecure(),
    )

    // 4. 创建 Resource (服务信息)
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceName(serviceName),
            semconv.ServiceVersion(version),
        ),
    )

    // 5. 创建 TracerProvider (批量导出)
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
    )

    // 6. 设置全局 TracerProvider
    otel.SetTracerProvider(tp)

    // 7. 返回 shutdown 函数
    return func() {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()
        tp.Shutdown(ctx)
    }
}
```

**入口调用** (cmd/bluebell/main.go):

```go
otelShutdown := otel.InitTracerProvider(cfg)
defer otelShutdown()
```

### 2. HTTP 追踪中间件

**文件**: `internal/middleware/tracing.go`

```go
// TracingMiddleware 返回用于 Gin 的 OTel 追踪中间件
// 自动为每个 HTTP 请求创建 trace span
func TracingMiddleware() gin.HandlerFunc {
    return otelgin.Middleware("")  // 使用全局配置的 service.name
}
```

**路由注册** (internal/router/router.go):

```go
r.Use(
    middleware.TracingMiddleware(),  // ← 必须最先注册
    middleware.GinLogger(),
    middleware.GinRecovery(true),
    // ... 其他中间件
)
```

> **注意**: 中间件注册顺序很重要。Gin 中间件执行顺序是反向的（先注册的最外层），所以 `TracingMiddleware` 必须在最前面，这样才能确保 trace 最先建立。

### 3. 自定义 Span

#### 在 Handler 中创建子 Span

```go
func (h *PostHandler) CreatePost(c *gin.Context) {
    // 从 gin.Context 创建子 span
    ctx, span := middleware.StartSpan(c, "CreatePost")
    defer span.End()

    // 业务逻辑...
    if err := h.postSvc.CreatePost(ctx, &req); err != nil {
        middleware.RecordError(span, err)  // 记录错误到 span
        return
    }
}
```

#### 在 Service 中创建子 Span

```go
func (s *PostService) CreatePost(ctx context.Context, req *CreatePostReq) error {
    // 从 context.Context 创建子 span
    ctx, span := middleware.StartSpanFromContext(ctx, "PostService.CreatePost",
        attribute.String("user.id", userID),
    )
    defer span.End()

    // 业务逻辑...
}
```

### 4. 辅助函数 (internal/middleware/tracing.go)

#### StartSpan - 从 gin.Context 创建子 span

```go
// StartSpan 从 gin.Context 创建子 span
// 用于 Handler 层，直接传入 *gin.Context
func StartSpan(c *gin.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
    // 从 gin.Context 获取底层 context（otelgin 中间件已将 trace 注入其中）
    ctx := c.Request.Context()

    // 创建子 span
    ctx, span := otel.TracerProvider().Tracer("bluebell").Start(ctx, spanName)

    // 添加自定义属性
    if len(attrs) > 0 {
        span.SetAttributes(attrs...)
    }

    // 将更新后的 context 写回 gin.Context，以便后续代码能获取到 span 信息
    c.Request = c.Request.WithContext(ctx)

    return ctx, span
}
```

**使用场景**: Handler 层，有 `*gin.Context` 时

```go
func (h *PostHandler) CreatePost(c *gin.Context) {
    ctx, span := middleware.StartSpan(c, "CreatePost",
        attribute.String("post.type", req.Type),
    )
    defer span.End()
    
    // 后续调用会自动传递 ctx
    if err := h.postSvc.CreatePost(ctx, &req); err != nil {
        middleware.RecordError(span, err)
        return
    }
}
```

#### StartSpanFromContext - 从 context.Context 创建子 span

```go
// StartSpanFromContext 从 context.Context 创建子 span
// 用于 Service/Repository 层，只依赖标准 context.Context
func StartSpanFromContext(ctx context.Context, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
    // 创建子 span（自动继承 ctx 中的父 span）
    ctx, span := otel.TracerProvider().Tracer("bluebell").Start(ctx, spanName)

    // 添加自定义属性
    if len(attrs) > 0 {
        span.SetAttributes(attrs...)
    }

    return ctx, span
}
```

**使用场景**: Service/Repository 层，只有 `context.Context` 时

```go
func (s *PostService) CreatePost(ctx context.Context, req *CreatePostReq) error {
    ctx, span := middleware.StartSpanFromContext(ctx, "PostService.CreatePost",
        attribute.String("user.id", userID),
        attribute.String("post.title", req.Title),
    )
    defer span.End()
    
    // 业务逻辑...
    post, err := s.postRepo.Create(ctx, req)
    if err != nil {
        span.SetStatus(codes.Error, "create post failed")
        return err
    }
    
    return nil
}
```

#### 对比总结

| 函数 | 参数类型 | 适用层 | 是否需要 gin.Context |
|------|----------|--------|---------------------|
| `StartSpan` | `*gin.Context` | Handler | ✅ 是 |
| `StartSpanFromContext` | `context.Context` | Service/Repository | ❌ 否 |

**核心区别**:  
- `StartSpan` 会自动将新 context 更新回 `gin.Context`，确保 Handler 层后续代码能获取最新 context
- `StartSpanFromContext` 只操作标准 context，适合不依赖 Gin 的业务层

| 函数 | 用途 |
|------|------|
| `TracingMiddleware()` | HTTP 入口追踪中间件 |
| `StartSpan(c *gin.Context, name)` | 从 gin.Context 创建子 span |
| `StartSpanFromContext(ctx, name, attrs...)` | 从 context.Context 创建子 span |
| `RecordError(span, err)` | 在 span 上记录错误 |

### 5. GORM SQL 追踪

**文件**: `internal/dao/database/init.go`

```go
import "gorm.io/plugin/opentelemetry/tracing"

// 初始化 GORM 后注册插件
db, err := gorm.Open(mysql.Open(dsn), gormConfig)

// 注册 OTel 追踪插件
if err := db.Use(tracing.Provider); err != nil {
    zap.L().Error("register otelgorm plugin failed", zap.Error(err))
}
```

自动追踪效果：
- 每个 `db.Query()` / `db.Exec()` 创建子 span
- 记录 SQL 语句、耗时、错误信息
- 事务 (Begin/Commit/Rollback) 创建独立 span

### 6. 日志集成

**文件**: `internal/middleware/gin.go`

GinLogger 从 context 提取 traceID 并注入日志：

```go
spanCtx := trace.SpanContextFromContext(c.Request.Context())
if spanCtx.HasTraceID() {
    fields = append(fields, zap.String("trace_id", spanCtx.TraceID().String()))
}
```

日志输出示例：

```json
{
  "level": "info",
  "msg": "http request",
  "trace_id": "abc123def456789",
  "status": 200,
  "method": "GET",
  "path": "/api/v1/post/1",
  "ip": "127.0.0.1",
  "cost": "15.3ms"
}
```

---

## 完整追踪流程

```
外部请求 (带/不带 traceparent header)
    │
    ▼
┌──────────────────────────────────────────┐
│ 1. TracingMiddleware (otelgin)           │
│    - 提取 W3C Trace Context              │
│    - 无上游则自动生成新 TraceID           │
│    - 创建根 Span: "HTTP {method} {path}" │
│    - 注入 c.Request.Context()            │
└──────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────┐
│ 2. GinLogger                             │
│    - 提取 traceID 注入日志               │
└──────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────┐
│ 3. 其他中间件 (Recovery/CORS/Timeout)    │
│    - 传递 context                        │
└──────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────┐
│ 4. Handler                               │
│    - StartSpan 创建子 span               │
│    - 调用 Service                        │
└──────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────┐
│ 5. Service                               │
│    - StartSpanFromContext 创建子 span    │
│    - 调用 Repository                     │
└──────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────┐
│ 6. Repository                            │
│    - GORM 操作自动被 otelgorm 追踪       │
└──────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────┐
│ 7. OTLP Exporter                         │
│    - 批量上报到 Collector                │
│    - 默认 localhost:4318 (gRPC)          │
└──────────────────────────────────────────┘
```

---

## 关键文件清单

| 文件路径 | 作用 |
|---------|------|
| `internal/infrastructure/otel/tracer.go` | TracerProvider 初始化，配置 OTLP gRPC exporter |
| `internal/middleware/tracing.go` | 封装 otelgin.Middleware、StartSpan、RecordError |
| `internal/middleware/gin.go` | GinLogger 集成 traceID 到日志 |
| `internal/router/router.go` | 中间件注册顺序 |
| `cmd/bluebell/main.go` | 启动时初始化 TracerProvider |
| `internal/dao/database/init.go` | GORM 初始化 + 注册 otelgorm 插件 |
| `internal/config/config.go` | otelConfig 配置结构 |
| `config.yaml` | OTel 配置参数 |

---

## OTLP Collector 部署

项目默认将追踪数据发送到 `localhost:4318` (gRPC)。需要部署 OTel Collector 接收数据：

### docker-compose.yml 示例

```yaml
services:
  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    command: ["--config", "/etc/otel-collector-config.yaml"]
    volumes:
      - ./otel-collector-config.yaml:/etc/otel-collector-config.yaml
    ports:
      - "4317:4317"   # gRPC receiver
      - "4318:4318"   # HTTP receiver
      - "8889:8889"   # Prometheus metrics

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686" # UI
```

### otel-collector-config.yaml

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: "0.0.0.0:4317"
      http:
        endpoint: "0.0.0.0:4318"

processors:
  batch:
    timeout: 10s

exporters:
  jaeger:
    endpoint: "jaeger:14250"
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [jaeger]
```

---

## 当前功能状态

| 功能 | 状态 |
|------|------|
| TraceID 生成 (otelgin) | ✅ 已启用 |
| Context 注入 | ✅ 已启用 |
| 跨层 Context 传递 | ✅ 已启用 |
| 日志集成 traceID | ✅ 已启用 |
| GORM SQL 追踪 | ✅ 已启用 |
| 自定义 Span (Handler) | ✅ 已启用 |
| 自定义 Span (Service) | ✅ 已启用 |
| HTTP 下游传递 | ❌ 未实现 (无下游 HTTP 调用) |
| 响应头返回 traceid | ❌ 未实现 |
| Redis 追踪 | ❌ 未实现 |

---

## 扩展指南

### 1. HTTP 下游传递

如果项目需要调用下游 HTTP 服务，使用 `otelhttp` 包装客户端：

```go
import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

client := &http.Client{
    Transport: otelhttp.NewTransport(http.DefaultTransport),
}
```

trace 会通过 `traceparent` header 自动传播。

### 2. 响应头返回 traceid

在中间件中将 TraceID 写入响应头：

```go
func TracingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        spanCtx := trace.SpanContextFromContext(c.Request.Context())
        if spanCtx.HasTraceID() {
            c.Header("X-Trace-Id", spanCtx.TraceID().String())
        }
    }
}
```

### 3. Redis 追踪

使用 `go.opentelemetry.io/contrib/instrumentation/github.com/go-redis/redis/v9/otelredis` 包装 Redis 客户端。

---

## 常见问题

### Q: OTel 数据没有发送到 Collector？

1. 检查配置 `config.yaml` 中 `otel.enabled` 是否为 `true`
2. 检查 `otel.endpoint` 是否正确（默认 `http://localhost:4318`）
3. 检查 Collector 是否启动并监听对应端口

### Q: 如何验证追踪是否工作？

1. 访问项目接口
2. 打开 Jaeger UI (默认 `http://localhost:16686`)
3. 选择服务 `bluebell`，搜索最近的服务调用

### Q: 如何关闭 OTel？

将 `config.yaml` 中的 `otel.enabled` 设为 `false`，或完全删除 `otel` 配置节点。