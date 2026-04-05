# TraceID 完整流程文档

## 技术选型

项目使用 **OpenTelemetry** 标准方案（非自定义 traceid），通过 W3C Trace Context 协议管理 TraceID 的生成、传递和上报。

**追踪组件：**
- **HTTP 入口**：`otelgin.Middleware` 自动创建根 span
- **SQL 操作**：`otelgorm` 插件自动为每个 GORM 操作创建子 span
- **自定义 Span**：`StartSpan` / `StartSpanFromContext` 手动创建业务级子 span

---

## 完整流程

```
外部请求 (可选携带 traceparent header)
    │
    ▼
┌──────────────────────────────────────────┐
│ 1. TracingMiddleware (入口)              │
│    - 从 Header 提取 W3C Trace Context    │
│    - 若无上游 trace，自动生成新 TraceID   │
│    - 创建根 Span: "HTTP {method} {path}" │
│    - 注入到 c.Request.Context()          │
└──────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────┐
│ 2. GinLogger (日志中间件)                │
│    - 从 context 提取 traceID 注入日志    │
│    - 输出: method/path/ip/cost/trace_id  │
└──────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────┐
│ 3. 其他中间件 (Recovery/CORS/Timeout)    │
│    - 传递 context                         │
└──────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────┐
│ 4. JWTAuthMiddleware                     │
│    - 使用 c.Request.Context()（已有trace）│
└──────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────┐
│ 5. Handler 层                            │
│    - c.Request.Context() 获取 context    │
│    - 可用 StartSpan 创建自定义子 span    │
└──────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────┐
│ 6. Service 层                            │
│    - ctx context.Context 传递            │
│    - 可用 StartSpanFromContext 创建子 span│
└──────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────┐
│ 7. Repository 层 (MySQL/Redis)           │
│    - GORM 操作自动被 otelgorm 追踪       │
│    - 每个 Query/Exec/Transaction 创建 span│
│    - context 继续传递                    │
└──────────────────────────────────────────┘
    │
    ▼
┌──────────────────────────────────────────┐
│ 8. OTLP gRPC Exporter                    │
│    - 批量上报到 Collector                │
│    - 默认端点: localhost:4317 (gRPC)     │
└──────────────────────────────────────────┘
```

---

## 关键文件

| 文件路径 | 作用 |
|---------|------|
| `internal/infrastructure/otel/tracer.go` | TracerProvider 初始化，配置 OTLP gRPC exporter |
| `internal/middleware/tracing.go` | 封装 `otelgin.Middleware`、`StartSpan`、`StartSpanFromContext`、`RecordError` |
| `internal/middleware/gin.go` | GinLogger 中间件，从 context 提取 traceID 注入日志 |
| `internal/router/router.go` | 路由注册，中间件挂载（TracingMiddleware 必须在 GinLogger 之前） |
| `cmd/bluebell/main.go` | 入口文件，启动时初始化 TracerProvider |
| `internal/dao/database/init.go` | GORM 初始化 + 注册 otelgorm 插件 |
| `internal/config/config.go` | otelConfig 配置结构（enabled/endpoint/service_name） |

---

## 详细流程说明

### 1. 初始化阶段（启动时）

**文件**: `internal/infrastructure/otel/tracer.go`

```go
func InitTracerProvider(cfg *config.Config) func() {
    // 创建 OTLP gRPC exporter（默认端点 localhost:4317）
    exporter, _ := otlptracegrpc.New(ctx,
        otlptracegrpc.WithEndpoint(endpoint),
        otlptracegrpc.WithInsecure(),
    )
    // 创建 TracerProvider，设置 service.name 和 service.version
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(exporter),
        sdktrace.WithResource(res),
    )
    otel.SetTracerProvider(tp)  // 设为全局 TracerProvider
}
```

**入口**: `cmd/bluebell/main.go`
```go
otelShutdown := otel.InitTracerProvider(cfg)
defer otelShutdown()
```

### 2. GORM 追踪初始化

**文件**: `internal/dao/database/init.go`

在 GORM 初始化后注册 otelgorm 插件：

```go
import "gorm.io/plugin/opentelemetry"

db.Use(opentelemetry.New())
```

自动追踪：
- 每个 `db.Query()` / `db.Exec()` 创建子 span
- 事务（Begin/Commit/Rollback）创建独立 span
- 记录 SQL 语句、耗时、错误信息

### 3. Trace 生成入口（HTTP 请求到达时）

**文件**: `internal/middleware/tracing.go`

```go
func TracingMiddleware() gin.HandlerFunc {
    return otelgin.Middleware(defaultServiceName)  // "bluebell"
}
```

`otelgin.Middleware` 自动完成：
- 从请求 Header 中提取 W3C Trace Context（`traceparent` / `tracestate`）
- 如果没有上游 trace，则**自动生成新的 TraceID**
- 将 trace 信息注入到 `context.Context` 中
- 创建根 span（`HTTP {method} {path}`）

### 4. 中间件注册顺序

**文件**: `internal/router/router.go`

```go
r.Use(
    middleware.TracingMiddleware(),    // ← 必须最先注册（最外层），确保 trace 先建立
    middleware.GinLogger(),            // ← 能从 context 提取 traceID
    middleware.GinRecovery(true),
    middleware.Cors(),
    middleware.TimeoutMiddleware(timeout),
    middleware.PrometheusMiddleware(),
)
```

> **注意**：Gin 中间件是反向执行的（先注册的最外层）。`TracingMiddleware` 必须在 `GinLogger` 之前注册，这样 GinLogger 执行时 trace 已经存在。

### 5. Context 传递链路

Trace 通过 Go 标准 `context.Context` 逐层传递：

```
HTTP Request
    │
    ▼
TracingMiddleware (otelgin)  ← 提取/生成 trace，注入 c.Request.Context()
    │
    ▼
GinLogger  ← 从 context 提取 traceID 写入日志
    │
    ▼
GinRecovery / CORS / Timeout / Prometheus
    │
    ▼
JWTAuthMiddleware  ← 使用 c.Request.Context()（已有 trace）
    │
    ▼
Handler (c.Request.Context())  ← 传递给 Service
    │
    ▼
Service (ctx context.Context)  ← 传递给 Repository/Cache
    │
    ▼
Repository (ctx context.Context)  ← GORM 操作被 otelgorm 自动追踪
```

**关键传递方式**: 所有 handler 都通过 `c.Request.Context()` 获取 context，service 层接收 `ctx context.Context` 参数。

### 6. 自定义 Span 创建（已全面启用）

**Handler 层** — 所有 handler 方法均已启用子 span：
```go
// post_handler/handler.go, user_handler/handler.go, community_handler/handler.go
ctx, span := middleware.StartSpan(c, "MethodName")
defer span.End()
if err := doSomething(ctx); err != nil {
    middleware.RecordError(span, err)
}
```

**Service 层** — 所有 service 方法均已启用子 span：
```go
// postsvc/post_service.go, usersvc/user_service.go, communitysvc/community_service.go
ctx, span := middleware.StartSpanFromContext(ctx, "MethodName",
    attribute.String("key", value),
)
defer span.End()
if err != nil {
    middleware.RecordError(span, err)
}
```

### 7. 辅助函数

**文件**: `internal/middleware/tracing.go`

| 函数 | 用途 |
|------|------|
/

### 8. Logger 与 TraceID 的集成

**已集成** ✅

**文件**: `internal/middleware/gin.go`

GinLogger 从 `c.Request.Context()` 提取 traceID 并注入日志：

```go
spanCtx := trace.SpanContextFromContext(c.Request.Context())
if spanCtx.HasTraceID() {
    fields = append(fields, zap.String("trace_id", spanCtx.TraceID().String()))
}
```

日志输出示例（JSON 格式）：
```json
{
  "level": "info",
  "msg": "http request",
  "trace_id": "abc123def456...",
  "status": 200,
  "method": "GET",
  "path": "/api/v1/post/1",
  "ip": "127.0.0.1",
  "cost": "15.3ms"
}
```

### 9. HTTP 下游传递

- 项目当前**没有** HTTP 客户端调用下游服务的代码
- 如果需要使用 `net/http` 或 `http.Client` 调用下游，需使用 `otelhttp` 包装客户端，trace 会通过 `traceparent` header 自动传播

### 10. 配置

**otelConfig** (`internal/config/config.go`):
```go
type otelConfig struct {
    Enabled     bool   `mapstructure:"enabled"`
    Endpoint    string `mapstructure:"endpoint"`     // 默认 localhost:4317 (gRPC)
    ServiceName string `mapstructure:"service_name"` // 默认 "bluebell"
}
```

---

## 当前状态总结

| 维度 | 状态 |
|------|------|
| TraceID 生成 | ✅ `otelgin.Middleware` 自动生成（W3C Trace Context 标准） |
| Context 注入 | ✅ 自动注入到 `c.Request.Context()` |
| 跨层传递 | ✅ 通过 `context.Context` 逐层传递 |
| Logger 集成 | ✅ 已集成 — GinLogger 从 context 提取 traceID 写入日志 |
| SQL 追踪 | ✅ `otelgorm` 插件自动追踪 GORM 操作 |
| 自定义 Span | ✅ 已全面启用 — handler 和 service 层所有关键方法均创建子 span |
| HTTP 下游传递 | ❌ 未实现 — 无下游 HTTP 调用代码 |
| 响应头返回 | ❌ 未将 traceid 写回响应头 |
| 上报方式 | OTLP gRPC → Collector（默认 `localhost:4317`） |

---

## 可能的增强方向

1. **启用自定义 Span** — 取消 handler/service 中注释的 StartSpan 代码，增加业务维度的追踪粒度
2. **HTTP 客户端传播** — 使用 `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` 包装 http.Client
3. **响应头返回 traceid** — 在中间件中将 TraceID 写入 `X-Trace-Id` 响应头，方便前端排查问题
4. **Redis 追踪** — 使用 `otelredis` 插件自动追踪 Redis 操作
