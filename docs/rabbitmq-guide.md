# Bluebell 项目 RabbitMQ 使用教学文档

## 目录

1. [架构概览](#1-架构概览)
2. [依赖与配置](#2-依赖与配置)
3. [核心 API 详解](#3-核心-api-详解)
4. [完整使用流程](#4-完整使用流程)
5. [实战案例](#5-实战案例)
6. [最佳实践](#6-最佳实践)
7. [消息结构参考](#7-消息结构参考)

---

## 1. 架构概览

### 1.1 文件结构

```
internal/infrastructure/mq/
├── connection.go      # 连接管理、Exchange/Queue 声明
├── publisher.go       # 消息发布器
├── consumer.go        # 投票消费者（VoteConsumer）
├── audit_consumer.go  # AI 审核消费者（AuditConsumer）
├── notify_consumer.go # 通知推送消费者（NotifyConsumer）
├── message.go         # 消息结构定义（VoteMessage, AuditMessage, NotifyMessage）
└── init.go            # 统一初始化 + 消费者启动（隔离模式）

internal/infrastructure/es/
└── sync_consumer.go   # ES 同步消费者（也使用 RabbitMQ）
```

### 1.2 Exchange / Queue 拓扑

```
┌─────────────────┐     vote.count     ┌──────────────┐
│ vote.exchange   │ ──────────────────→ │ vote.queue   │
│ (direct)        │                     └──────────────┘
└─────────────────┘                          │
                                              ▼
                                     VoteConsumer → Redis

┌─────────────────┐   audit.post     ┌──────────────┐
│ audit.exchange  │ ─────────────────→│ audit.queue  │
│ (direct)        │   audit.remark    │              │
└─────────────────┘                   └──────────────┘
                                              │
                                              ▼
                                     AuditConsumer → AI Auditor

┌─────────────────┐   search.sync    ┌──────────────┐
│ search.exchange │ ─────────────────→│ search.queue │
│ (direct)        │                   └──────────────┘
└─────────────────┘                          │
                                              ▼
                                     SyncConsumer → ES

┌─────────────────┐   (fanout, 无 key) ┌──────────────┐
│ notify.exchange │ ──────────────────→ │ notify.queue │
│ (fanout)        │                     └──────────────┘
└─────────────────┘                          │
                                              ▼
                                     NotifyConsumer → Redis RPUSH
```

### 1.3 四种 Exchange 类型对比

| Exchange | 类型 | Routing Key | 用途 |
|----------|------|-------------|------|
| `vote.exchange` | direct | `vote.count` | 投票异步计数 |
| `audit.exchange` | direct | `audit.post` / `audit.remark` | AI 内容审核 |
| `search.exchange` | direct | `search.sync` | ES 数据同步 |
| `notify.exchange` | fanout | （空） | 通知广播推送 |

---

## 2. 依赖与配置

### 2.1 Go 依赖

```go
// go.mod
github.com/rabbitmq/amqp091-go v1.10.0
```

这是 RabbitMQ 官方维护的 Go 客户端库，包名 `amqp091` 对应 AMQP 0-9-1 协议。

### 2.2 配置结构

```go
// internal/config/config.go
type rabbitmqConfig struct {
    URL string `mapstructure:"url"`
}

type Config struct {
    RabbitMQ *rabbitmqConfig `mapstructure:"rabbitmq"`
    // ...
}
```

### 2.3 配置文件示例 (YAML)

```yaml
rabbitmq:
  url: "amqp://guest:guest@localhost:5672/"
```

URL 格式：`amqp://用户名:密码@主机:端口/`

---

## 3. 核心 API 详解

### 3.1 连接管理 — `MQConnection`

**文件**: `internal/infrastructure/mq/connection.go`

#### 3.1.1 建立连接

```go
func NewMQConnection(ctx context.Context, cfg *config.Config) (*MQConnection, error)
```

**底层 API 调用链：**

```go
// 1. amqp.Dial — 建立 TCP 连接 + AMQP 握手
conn, err := amqp.Dial(cfg.RabbitMQ.URL)

// 2. conn.Channel() — 创建 Channel（轻量级虚拟连接）
ch, err := conn.Channel()

// 3. QueueDeclarePassive — 被动声明验证连接可用性（ping）
_, err := ch.QueueDeclarePassive("", true, false, false, false, nil)
```

**关键概念：**
- **Connection** = TCP 连接，重量级，一个进程通常只需要一个
- **Channel** = 虚拟连接，轻量级，在 Connection 上复用，所有操作都通过 Channel 进行

#### 3.1.2 声明 Exchange

```go
func (m *MQConnection) DeclareExchanges() error
```

**底层 API：**

```go
ch.ExchangeDeclare(
    "vote.exchange",  // name: Exchange 名称
    "direct",         // type: direct / fanout / topic / headers
    true,             // durable: 重启后是否保留
    false,            // auto-deleted: 最后一个绑定队列删除后是否自动删除
    false,            // internal: 是否仅用于内部交换
    false,            // no-wait: 是否等待服务器响应
    nil,              // arguments: 额外参数
)
```

**项目中的 4 个 Exchange：**

| 代码 | 类型 | Durable | 说明 |
|------|------|---------|------|
| `ExchangeVote` | direct | ✅ | 投票计数 |
| `ExchangeAudit` | direct | ✅ | AI 审核 |
| `ExchangeSearch` | direct | ✅ | ES 同步 |
| `ExchangeNotify` | fanout | ✅ | 通知广播 |

#### 3.1.3 声明 Queue 并绑定

```go
func (m *MQConnection) DeclareQueues() error
```

**底层 API — 声明队列：**

```go
ch.QueueDeclare(
    "vote.queue",     // name: 队列名称
    true,             // durable: 持久化（重启后保留）
    false,            // delete when unused: 不自动删除
    false,            // exclusive: 非独占（允许多个消费者）
    false,            // no-wait: 等待服务器响应
    nil,              // arguments: 额外参数
)
```

**底层 API — 绑定队列到 Exchange：**

```go
ch.QueueBind(
    "vote.queue",     // queue name
    "vote.count",     // routing key
    "vote.exchange",  // exchange name
    false,            // no-wait
    nil,              // arguments
)
```

**fanout Exchange 不需要 routing key：**

```go
// notify.exchange 是 fanout 类型，绑定时无需 routing key
ch.QueueBind("notify.queue", "", "notify.exchange", false, nil)
```

#### 3.1.4 关闭连接

```go
func (m *MQConnection) Close() {
    if m.channel != nil {
        _ = m.channel.Close()
    }
    if m.conn != nil {
        _ = m.conn.Close()
    }
}
```

**注意：** 关闭顺序必须是先 Channel 后 Connection。

---

### 3.2 消息发布 — `MQPublisher`

**文件**: `internal/infrastructure/mq/publisher.go`

#### 3.2.1 创建 Publisher

```go
publisher := mq.NewPublisher(conn)
```

内部持有 `conn.Channel()` 的引用。

#### 3.2.2 通用发布方法

```go
func (p *MQPublisher) publish(ctx context.Context, exchange, routingKey string, msg interface{}, logName string) error
```

**底层 API 调用：**

```go
// 1. JSON 序列化
body, err := json.Marshal(msg)

// 2. 设置超时
publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

// 3. 发布消息
err = p.channel.PublishWithContext(
    publishCtx,
    exchange,       // exchange 名称
    routingKey,     // routing key
    false,          // mandatory: 消息无法路由时是否返回
    false,          // immediate: 是否立即投递（已废弃）
    amqp.Publishing{
        ContentType:  "application/json",
        DeliveryMode: amqp.Persistent,  // 持久化投递 (mode=2)
        Body:         body,
        Timestamp:    time.Now(),
    },
)
```

**关键参数说明：**

| 参数 | 值 | 说明 |
|------|-----|------|
| `DeliveryMode` | `amqp.Persistent` (2) | 消息持久化到磁盘，Broker 重启不丢失 |
| `mandatory` | `false` | 设为 `true` 时，无法路由的消息会通过 Return 回调返回 |
| `ContentType` | `"application/json"` | 标明消息格式 |

#### 3.2.3 业务封装方法

```go
// 发布投票消息
publisher.PublishVote(ctx, &mq.VoteMessage{
    PostID: "123456",
    UserID: "789",
    Action: 1,  // 1=upvote, -1=downvote
})

// 发布审核消息（自动根据 Type 字段选择 routing key）
publisher.PublishAudit(ctx, &mq.AuditMessage{
    PostID:   "123",
    Title:    "帖子标题",
    Content:  "帖子内容",
    Type:     "post",   // "post" → audit.post, "remark" → audit.remark
    AuthorID: 456,
})

// 发布搜索同步消息
publisher.PublishSearch(ctx, syncMsg)

// 发布通知消息（fanout，routing key 为空）
publisher.PublishNotify(ctx, notifyMsg)
```

---

### 3.3 消息消费 — Consumer

#### 3.3.1 VoteConsumer（投票消费者）

**文件**: `internal/infrastructure/mq/consumer.go`

**启动流程：**

```go
func (c *VoteConsumer) Start(ctx context.Context) error {
    channel := c.conn.Channel()

    // 步骤 1: 设置 QoS（预取数量）
    channel.Qos(
        1,     // prefetch count: 每次只预取 1 条消息
        0,     // prefetch size: 不限制字节数
        false, // global: 仅应用于当前 channel
    )

    // 步骤 2: 注册消费者
    msgs, err := channel.Consume(
        QueueVote,  // queue name
        "",         // consumer tag: 空表示自动生成
        false,      // auto-ack: 关闭自动确认（手动 ack）
        false,      // exclusive: 非独占
        false,      // no-local: 允许接收本连接发布的消息
        false,      // no-wait: 等待服务器响应
        nil,        // args
    )

    // 步骤 3: 阻塞循环处理消息
    for {
        select {
        case <-ctx.Done():
            return nil  // 优雅退出
        case d, ok := <-msgs:
            if !ok {
                return errors.New("channel closed")
            }
            c.HandleDelivery(d)
        }
    }
}
```

**消息处理与确认：**

```go
func (c *VoteConsumer) HandleDelivery(d amqp091.Delivery) error {
    // 1. 解析消息
    var msg VoteMessage
    json.Unmarshal(d.Body, &msg)

    // 2. 业务处理
    // ...

    // 3. 成功 → Ack
    d.Ack(false)  // false = 不批量确认

    // 失败 → Nack
    d.Nack(false, false)  // (multiple=false, requeue=false)
}
```

**Ack/Nack 参数说明：**

| 方法 | 参数 | 说明 |
|------|------|------|
| `Ack(multiple)` | `false` | 仅确认当前消息 |
| `Ack(multiple)` | `true` | 确认所有未确认的消息 |
| `Nack(multiple, requeue)` | `false, false` | 不重新入队（丢弃/进死信队列） |
| `Nack(multiple, requeue)` | `false, true` | 重新入队（可能无限循环） |

#### 3.3.2 AuditConsumer（AI 审核消费者）

**文件**: `internal/infrastructure/mq/audit_consumer.go`

消费 `audit.queue` 队列，调用 AI Auditor 对帖子/评论内容进行审核。

```go
// AuditMessage 审核消息体
type AuditMessage struct {
    PostID   string `json:"post_id"`
    Title    string `json:"title"`
    Content  string `json:"content"`
    Type     string `json:"type"`     // "post" 或 "remark"
    AuthorID int64  `json:"author_id"`
}

type AuditConsumer struct {
    conn    *MQConnection
    auditor *ai.Auditor
}
```

**处理流程：**
1. 解析 `AuditMessage`
2. 若 auditor 未启用 → 直接 Ack 跳过
3. 调用 `auditor.Audit(ctx, title, content)` 进行 AI 审核
4. 审核不通过时记录违规信息（score, violations, reason）
5. Ack 确认消息

**注意：** `PublishAudit` 方法会根据 `AuditMessage.Type` 自动选择 routing key：
- `Type == "post"` → `audit.post`
- `Type == "remark"` → `audit.remark`

#### 3.3.3 NotifyConsumer（通知推送消费者）

**文件**: `internal/infrastructure/mq/notify_consumer.go`

消费 `notify.queue` 队列，将通知消息存储到 Redis 列表中供后续推送。

```go
// NotifyMessage 通知推送消息
type NotifyMessage struct {
    UserID  int64  `json:"user_id"`
    Type    string `json:"type"`    // "vote", "comment", "follow", "system"
    Title   string `json:"title"`
    Content string `json:"content"`
    PostID  string `json:"post_id,omitempty"`
}

type NotifyConsumer struct {
    conn        *MQConnection
    redisClient *redis.Client
}
```

**处理流程：**
1. 解析 `NotifyMessage`
2. 通过 Redis `RPUSH` 将通知推入用户专属列表 `notify:user:{userID}`
3. 可选设置列表最大长度限制（`LTRIM`）
4. Ack 确认消息

#### 3.3.4 SyncConsumer（ES 同步消费者）

**文件**: `internal/infrastructure/es/sync_consumer.go`

与 VoteConsumer 结构完全相同，区别在于：
- 消费 `QueueSearch` 队列
- 处理 `SyncMessage`（帖子数据同步到 ES）
- 根据 `Action` 字段执行 `index` 或 `delete` 操作

---

### 3.4 统一初始化与启动

**文件**: `internal/infrastructure/mq/init.go`

#### 3.4.1 InitMQ — 初始化基础设施

```go
func InitMQ(ctx context.Context, cfg *config.Config) (*MQConnection, *MQPublisher, error)
```

**内部流程：**
1. `NewMQConnection()` — 建立连接
2. `conn.DeclareExchanges()` — 声明所有 Exchange
3. `conn.DeclareQueues()` — 声明所有 Queue 并绑定
4. `NewPublisher(conn)` — 创建 Publisher

#### 3.4.2 StartConsumers — 启动所有消费者

```go
func StartConsumers(ctx context.Context, conn *MQConnection, redisClient *redis.Client, auditor *ai.Auditor, consumers ...Consumer)
```

**内部使用 `sync.WaitGroup` 隔离启动，消费者彼此独立：**

```go
var wg sync.WaitGroup

// VoteConsumer
wg.Add(1)
go func() {
    defer wg.Done()
    voteConsumer := NewVoteConsumer(conn, redisClient)
    if err := voteConsumer.Start(ctx); err != nil {
        zap.L().Error("consumer exited with error", ...)
    }
}()

// AuditConsumer（仅当 auditor 不为 nil 时启动）
if auditor != nil {
    wg.Add(1)
    go func() {
        defer wg.Done()
        auditConsumer := NewAuditConsumer(conn, auditor)
        if err := auditConsumer.Start(ctx); err != nil {
            zap.L().Error("consumer exited with error", ...)
        }
    }()
}

// NotifyConsumer
wg.Add(1)
go func() {
    defer wg.Done()
    notifyConsumer := NewNotifyConsumer(conn, redisClient)
    if err := notifyConsumer.Start(ctx); err != nil {
        zap.L().Error("consumer exited with error", ...)
    }
}()

// 传入的额外消费者（如 SyncConsumer）
for i, c := range consumers {
    consumer := c
    wg.Add(1)
    go func() {
        defer wg.Done()
        if err := consumer.Start(ctx); err != nil {
            zap.L().Error("consumer exited with error", ...)
        }
    }()
}

// 阻塞等待所有消费者退出
wg.Wait()
```

**特点：** 消费者之间完全隔离 —— 任意一个消费者出错不会影响其他消费者继续运行。这与 `errgroup` 的行为不同（`errgroup` 中一个错误会取消所有消费者）。

---

## 4. 完整使用流程

### 4.1 启动时初始化（main.go）

```go
// 1. 初始化 MQ 基础设施
conn, publisher, err := mq.InitMQ(ctx, cfg)
if err != nil {
    zap.L().Error("init MQ failed", zap.Error(err))
    conn = nil
    publisher = nil
}

// 2. 创建并启动消费者（非阻塞 goroutine）
if conn != nil && esClient != nil {
    syncConsumer := es.NewSyncConsumer(conn, esClient)
    go func() {
        mq.StartConsumers(ctx, conn, rdb, auditor, syncConsumer)
    }()
}

// 3. 优雅关机时清理（http_server.Run 返回后执行）
if conn != nil {
    conn.Close()
}
```

### 4.2 发布消息（Handler/Service 中）

```go
// 在 handler 或 service 中注入 publisher
type PostHandler struct {
    publisher *mq.MQPublisher
    // ...
}

// 发布投票消息
func (h *PostHandler) VotePost(ctx *gin.Context) {
    // 同步处理核心逻辑...

    // 异步发送 MQ 消息更新计数
    h.publisher.PublishVote(ctx.Request.Context(), &mq.VoteMessage{
        PostID: postID,
        UserID: userID,
        Action: 1,
    })

    // 立即返回响应，不等消费者处理
}
```

### 4.3 消费消息（Consumer 中）

消费者独立运行，不需要在 handler 中调用。启动后自动阻塞监听队列。

---

## 5. 实战案例

### 5.1 新增一个消息队列场景

假设要新增一个 "邮件通知" 队列：

**Step 1: 在 `connection.go` 中定义常量**

```go
// Email Exchange
const (
    ExchangeEmail   = "email.exchange"
    QueueEmail      = "email.queue"
    RoutingKeyEmail = "email.send"
)
```

**Step 2: 在 `DeclareExchanges()` 中添加 Exchange 声明**

```go
if err := m.channel.ExchangeDeclare(
    ExchangeEmail,
    "direct",
    true,
    false,
    false,
    false,
    nil,
); err != nil {
    return errorx.Wrapf(err, errorx.CodeInfraError, "declare exchange %s failed", ExchangeEmail)
}
```

**Step 3: 在 `DeclareQueues()` 中添加 Queue 声明和绑定**

```go
if err := m.declareAndBind(QueueEmail, ExchangeEmail, RoutingKeyEmail, "direct"); err != nil {
    return err
}
```

**Step 4: 在 `publisher.go` 中添加发布方法**

```go
func (p *MQPublisher) PublishEmail(ctx context.Context, msg interface{}) error {
    return p.publish(ctx, ExchangeEmail, RoutingKeyEmail, msg, "email")
}
```

**Step 5: 创建新的 Consumer**

```go
// email_consumer.go
type EmailConsumer struct {
    conn *mq.MQConnection
    // mailer ...
}

func (c *EmailConsumer) Start(ctx context.Context) error {
    ch := c.conn.Channel()
    ch.Qos(1, 0, false)

    msgs, err := ch.Consume(mq.QueueEmail, "", false, false, false, false, nil)
    if err != nil {
        return err
    }

    for {
        select {
        case <-ctx.Done():
            return nil
        case d, ok := <-msgs:
            if !ok {
                return errors.New("email consumer channel closed")
            }
            // 处理邮件发送...
            d.Ack(false)
        }
    }
}
```

**Step 6: 在 `init.go` 中启动**

```go
func StartConsumers(...) {
    var wg sync.WaitGroup

    // ... 现有消费者

    // EmailConsumer
    wg.Add(1)
    go func() {
        defer wg.Done()
        emailConsumer := NewEmailConsumer(conn, mailer)
        if err := emailConsumer.Start(ctx); err != nil {
            zap.L().Error("consumer exited with error", ...)
        }
    }()

    wg.Wait()
}
```

---

## 6. 最佳实践

### 6.1 连接与 Channel 管理

- **一个进程一个 Connection**，多个 Channel 复用
- Channel 不是线程安全的，不要在 goroutine 间共享
- 关闭时先 Channel 后 Connection

### 6.2 消息持久化

```go
// 发送端：DeliveryMode = Persistent
amqp.Publishing{
    DeliveryMode: amqp.Persistent,  // 消息写盘
}

// 队列端：durable = true
ch.QueueDeclare("my.queue", true, ...)

// Exchange 端：durable = true
ch.ExchangeDeclare("my.exchange", "direct", true, ...)
```

三者缺一不可，否则 Broker 重启消息丢失。

### 6.3 QoS 与负载均衡

```go
ch.Qos(1, 0, false)  // prefetch count = 1
```

- 多个消费者订阅同一队列时，`prefetch count = 1` 确保公平分发
- 不设置 QoS 会导致 Broker 一次性推送大量消息给某个消费者

### 6.4 手动 Ack/Nack

```go
// 处理成功
d.Ack(false)

// 处理失败，不重新入队（避免无限重试）
d.Nack(false, false)

// 处理失败，重新入队（谨慎使用，可能无限循环）
d.Nack(false, true)
```

**项目中采用的策略：** 失败不 requeue。生产环境建议配合死信队列（DLQ）使用。

### 6.5 超时控制

```go
publishCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
p.channel.PublishWithContext(publishCtx, ...)
```

发布消息必须设置超时，避免网络异常时永久阻塞。

### 6.6 错误处理

项目中 MQ 初始化失败不会阻止服务启动（非关键依赖）：

```go
conn, publisher, err := mq.InitMQ(ctx, cfg)
if err != nil {
    zap.L().Error("init MQ failed", zap.Error(err))
    conn = nil
    publisher = nil
}
```

业务代码在使用前应检查 `publisher != nil`。

### 6.7 Exchange 类型选择

| 场景 | 类型 | 理由 |
|------|------|------|
| 点对点精确路由 | direct | 通过 routing key 精确匹配 |
| 广播通知 | fanout | 所有绑定队列都收到消息 |
| 多条件匹配 | topic | routing key 支持通配符（如 `audit.*`） |

项目中 `notify.exchange` 使用 fanout 是因为通知需要推送给所有订阅方。

---

## 7. 消息结构参考

### 7.1 VoteMessage

```go
type VoteMessage struct {
    PostID string `json:"post_id"`
    UserID string `json:"user_id"`
    Action int    `json:"action"` // 1=upvote, -1=downvote
}
```

### 7.2 AuditMessage

```go
type AuditMessage struct {
    PostID   string `json:"post_id"`
    Title    string `json:"title"`
    Content  string `json:"content"`
    Type     string `json:"type"`     // "post" 或 "remark"
    AuthorID int64  `json:"author_id"`
}
```

`PublishAudit` 方法根据 `Type` 字段自动选择 routing key：
- `"post"` → `audit.post`
- `"remark"` → `audit.remark`

### 7.3 NotifyMessage

```go
type NotifyMessage struct {
    UserID  int64  `json:"user_id"`
    Type    string `json:"type"`    // "vote", "comment", "follow", "system"
    Title   string `json:"title"`
    Content string `json:"content"`
    PostID  string `json:"post_id,omitempty"`
}
```

### 7.4 SyncMessage（定义在 es/sync_consumer.go）

```go
type SyncMessage struct {
    PostID      string   `json:"post_id"`
    AuthorID    string   `json:"author_id"`
    CommunityID string   `json:"community_id"`
    PostTitle   string   `json:"post_title"`
    Content     string   `json:"content"`
    CreatedAt   string   `json:"created_at"`
    Action      string   `json:"action"` // "create", "update", "delete"
}
```
