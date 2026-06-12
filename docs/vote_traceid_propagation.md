# Bluebell 投票接口 TraceID 传递链路详解

> 适用版本：Bluebell（DDD 架构 / Gin / GORM / Redis / RabbitMQ / OpenTelemetry / Grafana Tempo）
> 目标读者：后端开发、SRE、可观测性平台维护者
> 阅读时长：约 25 分钟

---

## 0. 引言：为什么要关注 TraceID？

在分布式系统中，一个"点赞"操作可能横跨：

- 1 次 HTTP 请求
- 2 次 Redis 调用（Lua 脚本原子投票 + 社区归属查询）
- 1 次 RabbitMQ 消息发布（异步落库）
- 1 次 GORM/MySQL 写入（消费端落库）
- 数次日志输出

当线上出现"用户点赞没生效"或"投票数对不上"的问题时，如果只有孤立的日志行，几乎无法还原全貌。**TraceID 是一条把这次请求所有相关行为串起来的"红线"**——同一请求在不同进程、不同中间件中产生的所有 span、日志、指标，都通过 TraceID 关联起来。

本文以 `POST /api/v1/vote` 这一个接口为线索，**逐行讲解 Bluebell 项目里 TraceID 是如何诞生、传递、跨进程、最终落到日志里的**。

---

## 1. 端到端流程总览

下图给出了一次投票请求的完整 TraceID 传播路径：

```
┌──────────────────────────────────────────────────────────────────────────┐
│  浏览器 / 客户端                                                         │
│  POST /api/v1/vote  (Body: {"post_id":"123","direction":1})             │
│  可选 header: traceparent: 00-xxxxxxxx-xxxx-xxxx-01                     │
└─────────────────────────────┬────────────────────────────────────────────┘
                              │ ①
                              ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  Gin 引擎：otelgin.Middleware("bluebell")                               │
│  ────────────────────                                                   │
│  ① 从 HTTP 头提取 traceparent（如果客户端传了）                          │
│  ② 没传 → 用 SDK 自动生成新的 TraceID                                   │
│  ③ 创建 Root Span，存到 c.Request.Context()                             │
│  ★ 此刻 TraceID 诞生 ★                                                 │
└─────────────────────────────┬────────────────────────────────────────────┘
                              │ ctx (携带 root span)
                              ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  Handler: PostHandler.PostVoteHandler                                   │
│  ────────────────────                                                   │
│  tracer.Start(c.Request.Context(), "PostHandler.PostVoteHandler")       │
│  → 产生子 Span #1，TraceID 不变、SpanID 新增                            │
│  → c.Request = c.Request.WithContext(ctx)  // 关键：替换请求上下文       │
└─────────────────────────────┬────────────────────────────────────────────┘
                              │ ctx
                              ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  Service: PostService.VoteForPost                                       │
│  ────────────────────                                                   │
│  tracerPost.Start(ctx, "PostService.VoteForPost")                       │
│  → 产生子 Span #2                                                       │
│  → 顺便把 user.id、post.id 写到 span 属性里                              │
└─────┬──────────────────┬──────────────────┬──────────────────┐
      │                  │                  │                  │
      ▼                  ▼                  ▼                  ▼
┌──────────┐      ┌──────────┐       ┌──────────┐      ┌──────────┐
│ Redis 1  │      │ Redis 2  │       │  MQ Pub  │      │  Logger  │
│ Lua 脚本 │      │ 查社区ID │       │ 异步落库 │      │ 输出日志 │
│ Span#3a  │      │ Span#3b  │       │ 注入到   │      │ 自动加   │
│          │      │          │       │ AMQP头   │      │ trace_id │
└──────────┘      └──────────┘       └────┬─────┘      └──────────┘
                                          │ ④
                                          ▼ HTTP 立即返回 200
                                ┌──────────────────────┐
                                │ RabbitMQ             │
                                │  message.headers     │
                                │  traceparent: ...    │
                                └──────────┬───────────┘
                                           │ ⑤
                                           ▼
┌──────────────────────────────────────────────────────────────────────────┐
│  Consumer: VoteConsumer.handleDelivery                                  │
│  ────────────────────                                                   │
│  ⑤ Extract from AMQP headers → 重建 ctx（TraceID 与发布端一致）         │
│  ⑥ tracerConsumer.Start(ctx, "VoteConsumer.handleDelivery")            │
│     → 产生新 Span（异步边界的"连接点"）                                 │
│  ┌──────────────────┐   ┌──────────────────┐                             │
│  │ Redis 幂等检查   │   │ MySQL SaveVote   │                             │
│  │ Span#6a          │   │ Span#6b +        │                             │
│  │                  │   │ otelgorm SQL#6c  │                             │
│  └──────────────────┘   └──────────────────┘                             │
└──────────────────────────────────────────────────────────────────────────┘
```

> **核心要点**：
> 1. TraceID **只在 HTTP 入口（otelgin）** 诞生一次。
> 2. 进程内通过 Go `context.Context` 隐式传递。
> 3. 跨进程（HTTP→AMQP→Consumer）通过 **AMQP 消息头中的 `traceparent`** 显式传递。
> 4. 每个 span 有自己的 SpanID，但共享同一个 TraceID。

---

## 2. 阶段 1：OTel SDK 初始化 —— TraceID 体系的"地基"

**文件**：`internal/infrastructure/otel/otel.go`

`InitOTEL` 在服务启动时被调用，它完成三件关键事：

1. 创建全局 `TracerProvider`，并指向 OTLP gRPC 上报端点（Alloy）。
2. **注册 W3C TraceContext 传播器**（这就是为什么叫 `traceparent`）。
3. 后续的 `redisotel`、`otelgorm` 都使用这个全局 Provider。

```go
// internal/infrastructure/otel/otel.go:35-73
func InitOTEL(ctx context.Context, cfg *config.OtelConfig) (func(context.Context) error, error) {
    if cfg == nil || !cfg.Enabled {
        return func(context.Context) error { return nil }, nil
    }

    res, err := resource.New(ctx,
        resource.WithAttributes(semconv.ServiceNameKey.String(cfg.ServiceName)),
    )

    // 1. 创建 OTLP gRPC 连接
    conn, err := grpc.NewClient(cfg.Endpoint, /* ... */)

    // 2. 创建并注册 TraceProvider ★
    traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithBatcher(traceExporter),
        sdktrace.WithResource(res),
    )
    otel.SetTracerProvider(tp)

    // 3. 注册 W3C TraceContext 传播器 ★★★
    //   这决定了：HTTP 头里 traceparent 的格式、AMQP 头里字段名、
    //   以及 Inject/Extract 怎么序列化/反序列化
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{}, // W3C 标准
        propagation.Baggage{},       // 自定义 KV 透传
    ))

    // ... meter / logger / shutdown 略
    return shutdown, nil
}
```

**调用时机**：`cmd/bluebell/main.go:63` 和 `cmd/server/main.go:53`，早于 router 启动。

---

## 3. 阶段 2：HTTP 入口 —— TraceID 诞生

**文件**：`internal/interfaces/http/router/router.go:53-60`

```go
r.Use(
    otelgin.Middleware("bluebell"),   // ← 必须是中间件链的第一位！
    middleware.GinLogger(),
    middleware.GinRecovery(true),
    middleware.Cors(),
    middleware.RateLimitMiddleware(fillInterval, cfg.RateLimit.Capacity),
    middleware.TimeoutMiddleware(timeout),
)
```

> **关键设计**：`otelgin.Middleware` 必须在最前面，否则下游中间件读到的 `c.Request.Context()` 还没有 span。

`otelgin` 内部做了三件事（来自 contrib 库源码逻辑）：

```go
// 伪代码（go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin）
func Middleware(service string, opts ...Option) gin.HandlerFunc {
    tracer := otel.Tracer(service)
    return func(c *gin.Context) {
        // 1. 从 HTTP header 提取 traceparent（如果有）
        ctx := otel.GetTextMapPropagator().Extract(
            c.Request.Context(),
            propagation.HeaderCarrier(c.Request.Header),
        )

        // 2. 启动 Root Span
        spanName := c.FullPath()
        if spanName == "" { spanName = c.Request.URL.Path }
        ctx, span := tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindServer))

        defer span.End()

        // 3. 把带 span 的 ctx 写回 request
        c.Request = c.Request.WithContext(ctx)
        c.Next()
    }
}
```

**TraceID 诞生**：

- 客户端带了 `traceparent` 头 → 用客户端的 TraceID（实现跨服务串联）
- 客户端没带 → SDK 生成 32 字符的十六进制 TraceID（形如 `4bf92f3577b34da6a3ce929d0e0e4736`）

**验证方法**：

```bash
# 不带 traceparent → 服务端会生成新的
curl -X POST http://localhost:8082/api/v1/vote \
  -H "Content-Type: application/json" \
  -d '{"post_id":"123","direction":1}'

# 带 traceparent → 服务的 root span 与你给的 TraceID 相同
curl -X POST http://localhost:8082/api/v1/vote \
  -H "Content-Type: application/json" \
  -H "traceparent: 00-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-bbbbbbbbbbbbbbbb-01" \
  -d '{"post_id":"123","direction":1}'
```

---

## 4. 阶段 3：Handler 层 —— 创建第一个业务 Span

**文件**：`internal/interfaces/http/handler/post_handler/handler.go:162-196`

```go
// 文件 22 行：定义模块级 Tracer
var tracer = infratrace.TracerForModule("handler/post")

// 文件 162-196 行：PostVoteHandler
func (h *Handler) PostVoteHandler(c *gin.Context) {
    // ① 从 otelgin 写入的 ctx 启动子 span ★
    ctx, span := tracer.Start(c.Request.Context(), "PostHandler.PostVoteHandler")
    defer span.End()

    // ② ★★★ 关键：把带新 span 的 ctx 替换回 c.Request
    //    不替换的话，下游 c.ShouldBindJSON 取不到带 span 的 ctx
    c.Request = c.Request.WithContext(ctx)

    p := &postreq.VoteRequest{}
    if err := c.ShouldBindJSON(p); err != nil { /* ... */ }

    userID, exist := c.Get("UserIDKey")
    if !exist { /* ... */ }

    // ③ 把 ctx 继续往 service 传
    if err := h.postService.VoteForPost(ctx, userID.(int64), p); err != nil {
        if errors.Is(err, entity.ErrVoteRepeated) {
            render.HandleSuccess(c, nil)
            return
        }
        render.HandleError(c, err)
        return
    }

    metrics.RecordSuccess(ctx, metrics.Votes)
    render.HandleSuccess(c, nil)
}
```

`TracerForModule` 是项目封装的小工具（`internal/infrastructure/trace/trace.go:10-12`）：

```go
func TracerForModule(module string) trace.Tracer {
    return otel.Tracer(module)  // 实际调用全局 OTel Tracer
}
```

它的作用只是为每个模块起一个独立的"名字空间"，在 Tempo 里看时方便筛选（比如筛 `service.post.handler`）。

> **新手常见错误**：忘记 `c.Request = c.Request.WithContext(ctx)`。这种情况下：
> - `c.ShouldBindJSON` 拿到的还是旧 ctx（没 span）
> - 后续的 service 调用也没法记录到当前 handler 之下
> - 在 Tempo 里 handler span 就会和下游 span 断开

---

## 5. 阶段 4：Service 层 —— 业务编排 + Span 属性标注

**文件**：`internal/application/post_service.go:353-400`

```go
var tracerPost = trace.TracerForModule("service/post")  // 第 20 行

func (s *PostService) VoteForPost(ctx context.Context, userID int64, p *postreq.VoteRequest) error {
    // ① 在 handler 的 ctx 之上再起一层 span
    ctx, span := tracerPost.Start(ctx, "PostService.VoteForPost")
    defer span.End()
    trace.WithUserID(ctx, userID)  // 写入 user.id 属性

    // ② 业务校验
    vote := &entity.Vote{
        PostID:    p.PostID,
        UserID:    userID,
        Direction: p.Direction,
    }
    if err := vote.Validate(); err != nil {
        return err
    }

    postIDStr := strconv.FormatInt(p.PostID, 10)
    userIDStr := strconv.FormatInt(userID, 10)

    // ③ Redis：查社区 ID —— ctx 顺带传过去
    communityID, err := s.postCache.GetPostCommunityID(ctx, p.PostID)
    if err != nil {
        return fmt.Errorf("get community id for vote failed: %w", err)
    }

    // ④ Redis：原子投票（Lua 脚本）—— ctx 顺带传过去
    if err := s.postCache.VoteForPost(ctx, userIDStr, postIDStr,
        strconv.FormatInt(communityID, 10), float64(p.Direction)); err != nil {
        return err
    }

    // ⑤ MQ：异步落库 —— ctx 顺带传过去（关键，见阶段 6）
    if s.publisher != nil {
        _ = s.publisher.PublishVote(ctx, &mq.VoteMessage{
            MsgID:  strconv.FormatInt(snowflake.GenID(), 10),
            PostID: postIDStr,
            UserID: userIDStr,
            Action: int(p.Direction),
        })
        _ = s.publisher.PublishActivity(ctx, &mq.ActivityMessage{ /* ... */ })
    }

    trace.SetSpanSuccess(ctx)
    return nil
}
```

`trace.WithUserID` / `trace.WithPostID` 是项目封装（`internal/infrastructure/trace/trace.go`）：

```go
// 第 14-22 行
func WithUserID(ctx context.Context, userID int64) {
    span := trace.SpanFromContext(ctx)  // 从 ctx 取出当前 active span
    span.SetAttributes(attribute.Int64("user.id", userID))
}

func WithPostID(ctx context.Context, postID int64) {
    span := trace.SpanFromContext(ctx)
    span.SetAttributes(attribute.Int64("post.id", postID))
}
```

它们给当前 span 增加**业务属性**，在 Tempo 里可以基于这些属性做搜索/过滤。

---

## 6. 阶段 5：Redis 层 —— Lua 脚本 + 自动 Span

**文件**：`internal/infrastructure/persistence/redis/post/post.go:144-252`

```go
var tracer = infratrace.TracerForModule("dao/redis/post")  // 第 17 行

func (c *cacheStruct) VoteForPost(ctx context.Context, userID, postID, communityID string, value float64) error {
    // ① 显式创建一个 span
    ctx, span := tracer.Start(ctx, "RedisPostDAO.VoteForPost")
    defer span.End()

    // ② ZSet 检查帖子是否过期（1 周内可投票）
    postTime := c.rdb.ZScore(ctx, redisKey(keyPostTimeZSet), postID).Val()
    if float64(timeNow())-postTime > oneWeekInSeconds {
        return entity.ErrVoteTimeExpire
    }

    // ③ Lua 脚本（关键：保证"读旧值 + 计算 delta + 写新值"原子性）
    const voteLua = `
        local voteKey = KEYS[1]
        local metaKey = KEYS[2]
        local userID  = ARGV[1]
        local newValue = tonumber(ARGV[2])

        local oldValue = redis.call('ZSCORE', voteKey, userID)
        if not oldValue then oldValue = 0
        else oldValue = tonumber(oldValue) end

        if newValue == oldValue then
            return 'err_repeated'  -- 重复投票
        end

        -- 计算 delta ...
        if newValue == 0 then
            redis.call('ZREM', voteKey, userID)
        else
            redis.call('ZADD', voteKey, newValue, userID)
        end

        if voteUpDelta ~= 0 then
            redis.call('HINCRBY', metaKey, 'vote_up', voteUpDelta)
        end
        if voteDownDelta ~= 0 then
            redis.call('HINCRBY', metaKey, 'vote_down', voteDownDelta)
        end

        local result = redis.call('HMGET', metaKey, 'vote_up', 'vote_down', 'create_time')
        return result[1] .. ',' .. result[2] .. ',' .. result[3]
    `

    // ④ 执行 Lua 脚本 —— ctx 顺带传过去
    result, err := c.rdb.Eval(ctx, voteLua, []string{
        redisKey(keyPostVotedZSetPrefix + postID),
        redisKey(keyPostMetaPrefix + postID),
    }, userID, value).Text()

    // ... 错误处理略

    // ⑤ 更新全局/社区的 gravity 分数
    score := CalculateGravityScore(voteUp, voteDown, createTime)
    c.rdb.ZAdd(ctx, redisKey(keyPostScoreZSet), redis.Z{Score: score, Member: postID})
    c.rdb.ZAdd(ctx, redisKey(keyCommunityPostScorePrefix+communityID),
        redis.Z{Score: score, Member: postID})

    return nil
}
```

**为什么会有两个 Span 来源？**

1. 显式：`ctx, span := tracer.Start(ctx, "RedisPostDAO.VoteForPost")` —— 自定义业务名。
2. **自动**：`redisotel` 在 `rdb.Eval(ctx, ...)` 内部还会再创建子 Span（如 `redis:eval`）。

**`redisotel` 自动埋点初始化**（`internal/infrastructure/persistence/redis/init.go:30-36`）：

```go
rdb := redis.NewClient(&redis.Options{ /* ... */ })

// 关键：把 Redis 客户端包装成"会自动产生 span 的版本"
if err := redisotel.InstrumentTracing(rdb); err != nil {
    zap.L().Error("redisotel.InstrumentTracing failed", zap.Error(err))
}
if err := redisotel.InstrumentMetrics(rdb); err != nil {
    zap.L().Error("redisotel.InstrumentMetrics failed", zap.Error(err))
}
```

> **设计哲学**：业务 Span 表达"业务概念"（RedisPostDAO.VoteForPost），自动 Span 表达"网络/命令细节"（redis:eval、db.system=redis）。前者便于看业务，后者便于排查性能。

---

## 7. 阶段 6：MQ 边界 —— **跨进程 TraceID 传递（最关键）**

这是整个链路里**最容易出错也最容易被忽略**的一步：HTTP 请求已经返回，但异步落库还**没完成**。TraceID 必须跨越 RabbitMQ 这个进程边界，否则消费端的 span 在 Tempo 里就是孤儿。

### 7.1 发布端：把 TraceID 写进 AMQP 头

**文件**：`internal/infrastructure/mq/publisher.go:25-48`

```go
func (p *Publisher) Send(ctx context.Context, exchange, routingKey string, msg interface{}) error {
    body, err := json.Marshal(msg)
    if err != nil {
        return fmt.Errorf("序列化失败: %w", err)
    }

    // ① 创建 AMQP 头表
    headers := make(amqp.Table)

    // ② ★★★ 关键：把 ctx 里的 trace 信息注入到 headers
    //   本质是写一个 traceparent: 00-{TraceID}-{SpanID}-{Flags} 字段
    otel.GetTextMapPropagator().Inject(ctx, AmqpHeadersCarrier(headers))

    return p.ch.PublishWithContext(ctx,
        exchange,
        routingKey,
        false, false,
        amqp.Publishing{
            Headers:      headers,        // ★ 带 trace 的头
            DeliveryMode: amqp.Transient,
            ContentType:  "application/json",
            Body:         body,
            Timestamp:    time.Now(),
        },
    )
}
```

`AmqpHeadersCarrier` 是项目自定义的 OTel 传播适配器（`internal/infrastructure/mq/propagation.go`）：

```go
// internal/infrastructure/mq/propagation.go
type AmqpHeadersCarrier amqp.Table

// 实现 OTel 的 TextMapCarrier 接口
func (c AmqpHeadersCarrier) Get(key string) string {
    if v, ok := c[key]; ok {
        if s, ok := v.(string); ok { return s }
    }
    return ""
}

func (c AmqpHeadersCarrier) Set(key string, value string) {
    c[key] = value
}

func (c AmqpHeadersCarrier) Keys() []string {
    keys := make([]string, 0, len(c))
    for k := range c { keys = append(keys, k) }
    return keys
}

var _ propagation.TextMapCarrier = AmqpHeadersCarrier{}  // 编译期断言
```

> **为什么需要自己写 Carrier？** OTel 默认的 `HeaderCarrier` 是给 HTTP Header 用的，AMQP 的 header 是 `amqp.Table`（map），字段类型不一定是 string，所以需要薄薄包一层。

**`Inject` 实际写入了什么？**

在 RabbitMQ Management 界面（`http://localhost:15672`）选一条消息看 Properties → headers：

```
headers = {
    traceparent: "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
    tracestate:  ""  // 一般为空，除非用了 baggage
}
```

这就是 W3C TraceContext 标准的格式：`version-traceid-spanid-flags`。

### 7.2 消费端：从 AMQP 头重建 TraceID

**文件**：`internal/infrastructure/mq/vote_consumer.go:66-76`

```go
func (c *VoteConsumer) handleDelivery(ctx context.Context, d amqp.Delivery) error {
    // ① ★★★ 关键：从消息头里 Extract 出 traceparent
    //   这一刻 TraceID 跨进程重建成功，与发布端完全一致
    ctx = otel.GetTextMapPropagator().Extract(ctx, AmqpHeadersCarrier(d.Headers))

    // ② 在重建出的 ctx 上启动消费端 Span
    //   它会成为发布端 span 的"孩子"（即使是不同进程、不同 goroutine）
    ctx, span := tracerConsumer.Start(ctx, "VoteConsumer.handleDelivery")
    defer span.End()

    // ③ 反序列化消息
    var msg VoteMessage
    if err := json.Unmarshal(d.Body, &msg); err != nil { /* ... */ }

    // ④ Redis 幂等检查 —— ctx 自动带 span
    dedupKey := fmt.Sprintf("bluebell:mq:dedup:vote:%s", msg.MsgID)
    ok, err := c.rdb.SetNX(ctx, dedupKey, "1", 24*time.Hour).Result()
    if err != nil { /* ... */ }
    if !ok { return nil }  // 重复消息

    // ⑤ MySQL 落库 —— ctx 自动带 span（见阶段 8）
    if msg.Action != 0 {
        vote := &entity.Vote{ /* ... */ }
        if err := c.voteRepo.SaveVote(ctx, userID, postID, int8(msg.Action)); err != nil {
            c.rdb.Del(ctx, dedupKey)
            return fmt.Errorf("vote_consumer: 保存投票失败: %w", err)
        }
    } else {
        if err := c.voteRepo.DeleteVote(ctx, userID, postID); err != nil {
            c.rdb.Del(ctx, dedupKey)
            return fmt.Errorf("vote_consumer: 删除投票失败: %w", err)
        }
    }

    return nil
}
```

> **重要前提**：消费端进程（`cmd/consumer/vote/main.go:43`）**也必须调用 `InitOTEL`**，否则 `otel.GetTextMapPropagator()` 拿到的就是空 propagator，`Extract` 没有任何效果。Bluebell 的两个 cmd 都执行了相同的初始化。

### 7.3 进程拓扑图

```
                ┌─────────── 进程 1：bluebell server ───────────┐
                │                                                │
HTTP POST /vote │  otelgin → Handler → Service → Redis (Lua)    │
       │        │                  │                             │
       │        │                  │ PublishVote                 │
       │        │                  ▼                             │
       │        │  ┌──────── RabbitMQ (broker) ────────┐         │
       │        │  │  message.headers.traceparent     │         │
       │        │  │  body: {"msg_id":..,"post_id":..}│         │
       │        │  └──────────────────┬───────────────┘         │
       │        └─────────────────────┼─────────────────────────┘
       │                              │ 异步
       │        ┌────────── 进程 2：vote consumer ──────────────┐
       │        │                                                │
       ▼        │  Extract traceparent → Start span → SaveVote  │
   200 OK       │                                                │
                └────────────────────────────────────────────────┘
```

---

## 8. 阶段 7：MySQL 落库 —— GORM + otelgorm

**文件**：`internal/infrastructure/persistence/mysql/votedb/vote.go:14-43`

```go
var tracer = infratrace.TracerForModule("dao/mysql/vote")  // 第 14 行

func (r *voteRepoStruct) SaveVote(ctx context.Context, userID, postID int64, direction int8) error {
    // ① 显式创建 span
    ctx, span := tracer.Start(ctx, "VoteDAO.SaveVote")
    defer span.End()

    vote := &model.Vote{
        UserID:    userID,
        PostID:    postID,
        Direction: direction,
    }

    // ② ★ 关键：WithContext(ctx) 把 ctx 传给 GORM
    //   otelgorm 插件会在 SQL 执行前后自动创建子 span
    err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
        Columns:   []clause.Column{{Name: "user_id"}, {Name: "post_id"}},
        DoUpdates: clause.AssignmentColumns([]string{"direction", "updated_at"}),
    }).Create(vote).Error

    if err != nil {
        return fmt.Errorf("保存投票数据失败: %w", err)
    }
    return nil
}
```

**otelgorm 插件注册**（`internal/infrastructure/persistence/mysql/init.go:71-73`）：

```go
db, err := gorm.Open(mysql.Open(dsn), gormConfig)

// 关键：注册 otelgorm 插件
if err := db.Use(otelgorm.NewPlugin()); err != nil {
    zap.L().Error("register otelgorm plugin failed", zap.Error(err))
}
```

**生成的 Span 树**（在 Tempo 里看到的样子）：

```
VoteConsumer.handleDelivery              (consumer 端入口)
└── VoteDAO.SaveVote                     (votedb/vote.go:25)
    └── gorm:vote.create                 (otelgorm 自动)
        ├── db.system: mysql
        ├── db.statement: INSERT INTO vote ...
        └── db.user: bluebell
```

---

## 9. 日志中追踪 TraceID —— 把它"打出来"才能搜得到

光有 Span 没用，**日志里能看到 TraceID 才是真本事**——线上 90% 的排查是从 grep 日志开始的。

### 9.1 核心：`logger.WithContext`

**文件**：`internal/infrastructure/logger/logger.go:67-77`

```go
// WithContext 是关键函数：它从 ctx 里取出当前 span，
// 并把 TraceID/SpanID 作为 zap 字段附加到 logger 上
func WithContext(ctx context.Context) *zap.Logger {
    l := zap.L()
    span := trace.SpanFromContext(ctx)  // OTel API，从 ctx 拿 span
    if span.SpanContext().IsValid() {
        return l.With(
            zap.String("trace_id", span.SpanContext().TraceID().String()),
            zap.String("span_id",  span.SpanContext().SpanID().String()),
        )
    }
    return l
}
```

### 9.2 业务代码中怎么用

**Service 层**（`internal/application/post_service.go:74-86` 范例）：

```go
err = s.postRepo.CreatePost(ctx, post)
if err != nil {
    // ★ 用 WithContext 包装，trace_id 自动进日志
    logger.WithContext(ctx).Error("postRepo.CreatePost failed",
        zap.Int64("post_id", postIDInt),
        zap.Error(err))
    return "", entity.Wrap(entity.ErrServerBusy, err)
}
```

**HTTP 访问日志**（`internal/middleware/gin.go:38-48`）：

```go
func GinLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        cost := time.Since(start)

        // ... 业务字段 ...

        ctx := c.Request.Context()
        status := c.Writer.Status()

        // ★ 关键：用 WithContext 包装
        log := logger.WithContext(ctx)

        if status >= 500 {
            log.Error("server error", fields...)
        } else if status >= 400 {
            log.Warn("client error", fields...)
        } else {
            log.Info("http request", fields...)
        }
    }
}
```

### 9.3 实际日志输出示例

```json
{
    "level": "info",
    "ts": "2026-06-02T10:23:45.123Z",
    "caller": "middleware/gin.go:47",
    "msg": "http request",
    "status": 200,
    "method": "POST",
    "path": "/api/v1/vote",
    "ip": "127.0.0.1",
    "user_agent": "curl/7.81.0",
    "cost": 0.0234,
    "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
    "span_id":  "00f067aa0ba902b7"
}
```

### 9.4 进阶：zap → OTel Logs（双桥接）

`logger.go:48-54` 还把 zap 日志桥接到了 OTel 的日志通道：

```go
lp := global.GetLoggerProvider()
if lp != nil {
    // otelzap 是一个 zap Core，
    // 每次写日志时会同时调用 OTel LoggerProvider 转发到 Loki
    otelCore := otelzap.NewCore("bluebell", otelzap.WithLoggerProvider(lp))
    cores = append(cores, otelCore)
}
```

这意味着：

- **结构化字段**（trace_id、span_id、msg）→ 通过 zap 写到本地文件
- **同样的日志** → 也会通过 otelzap 发到 OTel Collector → Loki

在 Grafana 里，就能用 `{trace_id="4bf92f3577b34da6a3ce929d0e0e4736"}` 搜出该请求的所有日志行。

---

## 10. 关键设计总结

| 机制 | 位置 | 作用 |
|------|------|------|
| `otel.SetTextMapPropagator(TraceContext{})` | `otel/otel.go:73` | 定义 TraceID 的"格式"和"传播规则" |
| `otelgin.Middleware` 放在最前 | `router/router.go:54` | 确保下游所有中间件都能读到带 span 的 ctx |
| `c.Request = c.Request.WithContext(ctx)` | `post_handler/handler.go:165` | 把新 span 写回 Gin 内部用的 ctx |
| `rdb.WithContext(ctx)` | `votedb/vote.go:34` | 让 otelgorm 能拿到 span |
| `db.Use(otelgorm.NewPlugin())` | `mysql/init.go:71` | 自动埋点所有 SQL |
| `redisotel.InstrumentTracing(rdb)` | `redis/init.go:30` | 自动埋点所有 Redis 命令 |
| `Inject(ctx, AmqpHeadersCarrier)` | `mq/publisher.go:33` | 跨进程：把 TraceID 写入 AMQP 头 |
| `Extract(ctx, AmqpHeadersCarrier)` | `mq/vote_consumer.go:68` | 跨进程：从 AMQP 头重建 TraceID |
| `logger.WithContext(ctx)` | `logger/logger.go:67` | 把 TraceID 写入每条日志 |

**Span 命名规范**：`"<Package>.<FunctionName>"`，例如：

- `PostHandler.PostVoteHandler`（接口层）
- `PostService.VoteForPost`（应用层）
- `RedisPostDAO.VoteForPost`（基础设施 - Redis）
- `VoteConsumer.handleDelivery`（基础设施 - MQ Consumer）
- `VoteDAO.SaveVote`（基础设施 - MySQL）

**Span Kind**：
- otelgin 起的根 span：SERVER
- HTTP 客户端调用：C LIENT
- MQ publish / consume：PRODUCER / CONSUMER
- 数据库：CLIENT

---

## 11. 实战调试技巧

### 11.1 在浏览器/客户端注入 TraceID

```bash
# 终端 1：发起一次带自定义 traceparent 的请求
curl -X POST http://localhost:8082/api/v1/vote \
  -H "Content-Type: application/json" \
  -H "traceparent: 00-deadbeefdeadbeefdeadbeefdeadbeef-1111111111111111-01" \
  -d '{"post_id":"123","direction":1}'
```

### 11.2 用 TraceID 在 Grafana 串联一切

**Tempo（链路）**：
```logql
{ span.trace_id = "deadbeefdeadbeefdeadbeefdeadbeef" }
```

**Loki（日志）**：
```logql
{ app="bluebell" } | json | trace_id="deadbeefdeadbeefdeadbeefdeadbeef"
```

**Mimir（指标）**：
- HTTP 延迟直方图 `bluebell_http_request_duration_seconds` 的 exemplar 字段会带 trace_id（如果 client 端打开 exemplars）

### 11.3 验证 MQ 跨进程是否真的串起来

1. 给 `POST /vote` 打个特定 traceparent
2. 立刻查 RabbitMQ Management → 看消息头里 traceparent 是否一致
3. 等待消费者处理完，查 Tempo 搜索这个 TraceID
4. 应该能看到：root span（otelgin）→ handler span → service span → Redis span → RabbitMQ produce span → 异步消费 span → MySQL span

### 11.4 常见故障

| 现象 | 原因 | 解决 |
|------|------|------|
| 所有 span 都没父 | otelgin 没在中间件链首位 | 把它放第一 |
| MQ 消费端 span 是孤儿 | 消费端没 `InitOTEL` 或用了错的 propagator | 检查 `cmd/consumer/*/main.go` |
| 日志没有 trace_id | 用了 `zap.L()` 而不是 `logger.WithContext(ctx)` | 全文 grep 替换 |
| GORM SQL 没产生 span | 忘了 `db.Use(otelgorm.NewPlugin())` | 重新检查 init |
| Redis Lua 没产生 span | 忘了 `redisotel.InstrumentTracing(rdb)` | 重新检查 init |
| 跨服务 trace 不通 | 上游没发 `traceparent` 或格式错 | 用 W3C 标准格式 `00-{32hex}-{16hex}-{2hex}` |

---

## 12. 一句话总结

> **Bluebell 的 TraceID 传播 = `otelgin` 诞生 + Go `context.Context` 进程内传递 + AMQP 头 `traceparent` 跨进程传递 + `logger.WithContext` 写入日志。**

掌握了这个公式，看任何 OTel 化的 Go 项目都能快速定位 TraceID 在哪一环断了。

---

## 附录 A：相关文件速查

| 关注点 | 文件 | 关键行 |
|--------|------|--------|
| OTel SDK 初始化 | `internal/infrastructure/otel/otel.go` | 35-73 |
| Tracer 工厂 | `internal/infrastructure/trace/trace.go` | 10-27 |
| Logger 桥接 | `internal/infrastructure/logger/logger.go` | 67-77 |
| Gin 路由 + otelgin | `internal/interfaces/http/router/router.go` | 53-60 |
| Gin Logger | `internal/middleware/gin.go` | 38-48 |
| Vote Handler | `internal/interfaces/http/handler/post_handler/handler.go` | 162-196 |
| Vote Service | `internal/application/post_service.go` | 353-400 |
| Redis Post DAO | `internal/infrastructure/persistence/redis/post/post.go` | 144-252 |
| Redis Init（埋点） | `internal/infrastructure/persistence/redis/init.go` | 30-36 |
| MQ Publisher | `internal/infrastructure/mq/publisher.go` | 25-48 |
| MQ Carrier | `internal/infrastructure/mq/propagation.go` | 10-37 |
| MQ Consumer | `internal/infrastructure/mq/vote_consumer.go` | 66-76 |
| MySQL Vote Repo | `internal/infrastructure/persistence/mysql/votedb/vote.go` | 24-43 |
| MySQL Init（埋点） | `internal/infrastructure/persistence/mysql/init.go` | 71-73 |
| Server 入口 | `cmd/bluebell/main.go` | 45-272 |
| Consumer 入口 | `cmd/consumer/vote/main.go` | 24-137 |

## 附录 B：TraceID 格式速查（W3C TraceContext）

```
traceparent: <version>-<trace-id>-<parent-id>-<trace-flags>
             00         4bf92f3577b34da6a3ce929d0e0e4736  00f067aa0ba902b7  01

- version: 2 字符，目前固定 00
- trace-id: 32 字符十六进制（128 bit）
- parent-id: 16 字符十六进制（64 bit），即 SpanID
- trace-flags: 2 字符，01 表示 sampled
```
