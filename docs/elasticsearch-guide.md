# Bluebell Elasticsearch 集成文档

## 概述

Bluebell 项目集成了 Elasticsearch 8.x 用于帖子的全文搜索功能。系统采用异步架构，通过 RabbitMQ 消息队列实现数据同步，确保搜索服务与主业务解耦。

### 核心特性

- ✅ 中文全文检索（IK 分词器）
- ✅ 多字段搜索（标题 + 内容）
- ✅ 搜索结果高亮
- ✅ 异步数据同步（RabbitMQ）
- ✅ 自动容错（ES 故障不影响主服务）
- ✅ 幂等操作（基于 post_id）

---

## 架构设计

```
用户创建/删除帖子
        │
        ▼
┌──────────────────────┐
│  Post Handler        │  发布 SyncMessage
└────────┬─────────────┘
         │ PublishSearch()
         ▼
┌──────────────────────┐
│  RabbitMQ            │  search.exchange → search.queue
└────────┬─────────────┘
         │ 消费
         ▼
┌──────────────────────┐
│  SyncConsumer        │  indexDocument / deleteDocument
└────────┬─────────────┘
         │ ES API
         ▼
┌──────────────────────┐
│  Elasticsearch       │  索引: post
└──────────────────────┘
```

---

## 配置说明

### 1. 配置文件

在 `config.yaml` 中配置 Elasticsearch 连接信息：

```yaml
es:
  addresses:
    - "http://localhost:9200"
  username: ""
  password: ""
```

### 2. Docker 部署配置

`docker-compose.yml` 中定义了 Elasticsearch 服务：

```yaml
elasticsearch:
  image: elasticsearch:8.12.0
  container_name: bluebell_elasticsearch
  ports:
    - "9200:9200"
  environment:
    - discovery.type=single-node          # 单节点模式
    - ES_JAVA_OPTS=-Xms512m -Xmx512m     # JVM 堆内存
    - xpack.security.enabled=false        # 禁用安全认证
  volumes:
    - ./data/elasticsearch:/usr/share/elasticsearch/data  # 数据持久化
  networks:
    - bluebell_net
  restart: always
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:9200/_cluster/health"]
    interval: 10s
    timeout: 10s
    retries: 5
```

### 3. Go 依赖

```go
// go.mod
github.com/elastic/go-elasticsearch/v8 v8.19.3
github.com/elastic/elastic-transport-go/v8 v8.8.0
```

---

## 索引结构

### 索引名称

```
post
```

### 映射定义

```json
{
  "mappings": {
    "properties": {
      "post_id": {
        "type": "keyword"
      },
      "author_id": {
        "type": "long"
      },
      "community_id": {
        "type": "long"
      },
      "post_title": {
        "type": "text",
        "analyzer": "ik_max_word",
        "search_analyzer": "ik_smart"
      },
      "content": {
        "type": "text",
        "analyzer": "ik_max_word",
        "search_analyzer": "ik_smart"
      },
      "status": {
        "type": "integer"
      },
      "created_at": {
        "type": "date"
      }
    }
  },
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  }
}
```

### 字段说明

| 字段名 | 类型 | 说明 | 示例 |
|--------|------|------|------|
| `post_id` | keyword | 帖子唯一 ID（用作文档 ID） | `"1234567890"` |
| `author_id` | long | 作者用户 ID | `1001` |
| `community_id` | long | 所属社区 ID | `100` |
| `post_title` | text | 帖子标题（IK 分词） | `"Go 语言并发编程指南"` |
| `content` | text | 帖子内容（IK 分词） | `"本文介绍 Go 的 goroutine..."` |
| `status` | integer | 帖子状态（1=已发布） | `1` |
| `created_at` | date | 创建时间（RFC3339） | `"2024-01-01T12:00:00Z"` |

### 分词器说明

项目使用 **IK 中文分词器**，支持中文文本的精确分词：

- **`ik_max_word`**（写入时）：最细粒度分词，提高召回率
  - 示例：`"Go语言并发编程"` → `["Go", "语言", "并发", "编程", "Go语言", "并发编程"]`

- **`ik_smart`**（搜索时）：智能分词，减少误匹配
  - 示例：`"Go语言并发编程"` → `["Go语言", "并发编程"]`

> ⚠️ **注意**：需要确保 Elasticsearch 已安装 IK 分词器插件。

---

## 客户端初始化

### 代码位置

`internal/infrastructure/es/client.go`

### 初始化流程

```go
// 1. 创建 ES 客户端
esClient, err := es.NewClient(cfg)
if err != nil {
    zap.L().Error("init ES client failed", zap.Error(err))
    // ES 是非关键依赖，失败也会继续启动
    esClient = nil
} else {
    // 2. 确保索引存在
    if err := esClient.CreatePostIndex(ctx); err != nil {
        zap.L().Error("create ES post index failed", zap.Error(err))
    }
}
```

### 主要方法

```go
type Client struct {
    es *elasticsearch.Client
}

// NewClient 创建并验证 ES 客户端
func NewClient(cfg *config.Config) (*Client, error)

// CreatePostIndex 创建 post 索引（如果不存在）
func (c *Client) CreatePostIndex(ctx context.Context) error

// ES 返回底层 ES 客户端（用于直接调用 API）
func (c *Client) ES() *elasticsearch.Client
```

---

## 数据同步机制

### 1. 消息结构

```go
type SyncMessage struct {
    PostID      string `json:"post_id"`
    AuthorID    int64  `json:"author_id"`
    CommunityID int64  `json:"community_id"`
    PostTitle   string `json:"post_title"`
    Content     string `json:"content"`
    Status      int8   `json:"status"`
    CreatedAt   string `json:"created_at"`
    Action      string `json:"action"` // "index" 或 "delete"
}
```

### 2. RabbitMQ 配置

```go
ExchangeSearch   = "search.exchange"
QueueSearch      = "search.queue"
RoutingKeySearch = "search.sync"
```

### 3. 发布消息

#### 创建帖子时（索引）

```go
// post_handler/handler.go
syncMsg := &mq.SyncMessage{
    PostID:      postID,
    AuthorID:    userID.(int64),
    CommunityID: p.CommunityID,
    PostTitle:   p.Title,
    Content:     p.Content,
    Status:      model.PostStatusPublished,
    CreatedAt:   time.Now().Format(time.RFC3339),
    Action:      "index",
}
h.publisher.PublishSearch(ctx, syncMsg)
```

#### 删除帖子时（删除）

```go
syncMsg := map[string]interface{}{
    "post_id": strconv.FormatInt(postID, 10),
    "action":  "delete",
}
h.publisher.PublishSearch(ctx, syncMsg)
```

### 4. 消费者处理

```go
// es_consumer.go
func (c *SyncConsumer) handleDelivery(delivery amqp.Delivery) {
    var msg SyncMessage
    json.Unmarshal(delivery.Body, &msg)
    
    switch msg.Action {
    case "delete":
        c.deleteDocument(msg.PostID)
    default: // "index"
        c.indexDocument(&msg)
    }
    
    delivery.Ack(false) // 手动确认
}
```

### 5. ES 文档操作

#### 索引文档

```go
func (c *SyncConsumer) indexDocument(msg *SyncMessage) error {
    doc := map[string]interface{}{
        "post_id":      msg.PostID,
        "author_id":    msg.AuthorID,
        "community_id": msg.CommunityID,
        "post_title":   msg.PostTitle,
        "content":      msg.Content,
        "status":       msg.Status,
        "created_at":   msg.CreatedAt,
    }
    
    body, _ := json.Marshal(doc)
    res, err := c.client.ES().Index(
        es.IndexPost,
        bytes.NewReader(body),
        c.client.ES().Index.WithDocumentID(msg.PostID), // 幂等
    )
}
```

#### 删除文档

```go
func (c *SyncConsumer) deleteDocument(postID string) error {
    res, err := c.client.ES().Delete(es.IndexPost, postID)
    // 404 视为成功（文档已不存在）
}
```

---

## 搜索功能

### 1. API 接口

```go
type SearchRequest struct {
    Keyword  string `json:"keyword" binding:"required"`
    Page     int    `json:"page"`
    PageSize int    `json:"page_size"`
}

type SearchResponse struct {
    Total    int64           `json:"total"`
    Page     int             `json:"page"`
    PageSize int             `json:"page_size"`
    Posts    []SearchPostDoc `json:"posts"`
}
```

### 2. 搜索方法

```go
func (c *Client) Search(ctx context.Context, req *SearchRequest) (*SearchResponse, error)
```

### 3. 查询 DSL

```json
{
  "from": 0,
  "size": 20,
  "query": {
    "multi_match": {
      "query": "搜索关键词",
      "fields": ["post_title^2", "content"],
      "type": "best_fields"
    }
  },
  "highlight": {
    "fields": {
      "post_title": {
        "type": "unified",
        "pre_tags": ["<em class='highlight'>"],
        "post_tags": ["</em>"]
      },
      "content": {
        "fragment_size": 150,
        "number_of_fragments": 3,
        "pre_tags": ["<em class='highlight'>"],
        "post_tags": ["</em>"]
      }
    }
  },
  "_source": {
    "includes": ["post_id", "author_id", "community_id", "post_title", "content", "status", "created_at"]
  },
  "sort": [
    { "created_at": { "order": "desc" } }
  ]
}
```

### 4. 搜索特性

| 特性 | 说明 |
|------|------|
| **多字段匹配** | 同时搜索 `post_title` 和 `content` |
| **权重设置** | `post_title^2` 表示标题权重是内容的 2 倍 |
| **最佳字段** | `best_fields` 取最佳匹配字段分数 |
| **结果高亮** | 标题全文高亮，内容返回最多 3 个片段（每段 150 字符） |
| **排序** | 按 `created_at` 降序 |
| **分页** | 默认 page=1, page_size=20，最大 50 |

### 5. 返回文档结构

```go
type SearchPostDoc struct {
    PostID           string   `json:"post_id"`
    AuthorID         int64    `json:"author_id"`
    CommunityID      int64    `json:"community_id"`
    PostTitle        string   `json:"post_title"`
    Content          string   `json:"content"`
    Status           int8     `json:"status"`
    CreatedAt        string   `json:"created_at"`
    HighlightTitle   []string `json:"highlight_title,omitempty"`
    HighlightContent []string `json:"highlight_content,omitempty"`
}
```

---

## 使用示例

### 1. 启动完整服务

```bash
# 使用 Docker Compose 启动所有服务（包括 ES）
docker-compose up -d

# 查看 ES 状态
curl http://localhost:9200/_cluster/health
```

### 2. 创建帖子（自动同步到 ES）

```bash
POST /api/v1/posts
{
  "community_id": 100,
  "title": "Go 语言并发编程实战",
  "content": "介绍 goroutine 和 channel 的使用方法..."
}
```

### 3. 搜索帖子

```bash
GET /api/v1/search?keyword=并发&page=1&page_size=10
```

响应示例：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "total": 25,
    "page": 1,
    "page_size": 10,
    "posts": [
      {
        "post_id": "1234567890",
        "author_id": 1001,
        "community_id": 100,
        "post_title": "Go 语言<em class='highlight'>并发</em>编程实战",
        "content": "本文介绍 goroutine 和 channel 的<em class='highlight'>并发</em>使用方法...",
        "status": 1,
        "created_at": "2024-01-01T12:00:00Z",
        "highlight_title": ["Go 语言<em class='highlight'>并发</em>编程实战"],
        "highlight_content": ["本文介绍 goroutine 和 channel 的<em class='highlight'>并发</em>使用方法..."]
      }
    ]
  }
}
```

### 4. 删除帖子（自动从 ES 移除）

```bash
DELETE /api/v1/posts/1234567890
```

---

## 高级操作

### 手动创建索引

```bash
curl -X PUT "localhost:9200/post" -H 'Content-Type: application/json' -d'
{
  "mappings": {
    "properties": {
      "post_id": { "type": "keyword" },
      "author_id": { "type": "long" },
      "community_id": { "type": "long" },
      "post_title": {
        "type": "text",
        "analyzer": "ik_max_word",
        "search_analyzer": "ik_smart"
      },
      "content": {
        "type": "text",
        "analyzer": "ik_max_word",
        "search_analyzer": "ik_smart"
      },
      "status": { "type": "integer" },
      "created_at": { "type": "date" }
    }
  },
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  }
}'
```

### 查看索引信息

```bash
# 索引统计
curl "localhost:9200/_cat/indices/post?v"

# 索引设置
curl "localhost:9200/post/_settings"

# 索引映射
curl "localhost:9200/post/_mapping?pretty"
```

### 测试分词效果

```bash
# 测试 ik_max_word
curl -X POST "localhost:9200/_analyze" -H 'Content-Type: application/json' -d'
{
  "analyzer": "ik_max_word",
  "text": "Go语言并发编程实战"
}'

# 测试 ik_smart
curl -X POST "localhost:9200/_analyze" -H 'Content-Type: application/json' -d'
{
  "analyzer": "ik_smart",
  "text": "Go语言并发编程实战"
}'
```

### 直接搜索 ES

```bash
curl -X POST "localhost:9200/post/_search" -H 'Content-Type: application/json' -d'
{
  "query": {
    "multi_match": {
      "query": "并发",
      "fields": ["post_title^2", "content"]
    }
  },
  "highlight": {
    "fields": {
      "post_title": {},
      "content": {
        "fragment_size": 150,
        "number_of_fragments": 3
      }
    }
  },
  "sort": [{ "created_at": "desc" }]
}'
```

### 重新索引数据

如果需要重建索引（如修改分词器后），可以：

```bash
# 1. 删除旧索引
curl -X DELETE "localhost:9200/post"

# 2. 创建新索引
curl -X PUT "localhost:9200/post" -H 'Content-Type: application/json' -d'
{ ... 映射定义 ... }'

# 3. 重新同步数据（通过业务逻辑或脚本）
```

---

## 故障排查

### 1. ES 连接失败

**症状**：日志显示 `ES ping failed`

**排查步骤**：

```bash
# 检查 ES 服务是否运行
curl "localhost:9200/_cluster/health"

# 检查 Docker 容器状态
docker ps | grep elasticsearch

# 查看 ES 日志
docker logs bluebell_elasticsearch
```

**解决方案**：
- 确保 Docker 容器已启动：`docker-compose up -d elasticsearch`
- 检查端口占用：`netstat -tunlp | grep 9200`
- 验证配置文件中的 `addresses` 是否正确

### 2. IK 分词器未安装

**症状**：创建索引时报错 `unknown analyzer [ik_max_word]`

**解决方案**：

```bash
# 进入 ES 容器
docker exec -it bluebell_elasticsearch bash

# 安装 IK 分词器
elasticsearch-plugin install https://github.com/medcl/elasticsearch-analysis-ik/releases/download/v8.12.0/elasticsearch-analysis-ik-8.12.0.zip

# 重启 ES 服务
docker restart bluebell_elasticsearch
```

### 3. 搜索结果为空

**可能原因**：
- 数据未同步到 ES
- 分词效果不符合预期
- 搜索关键词拼写错误

**排查步骤**：

```bash
# 检查索引中的文档数量
curl "localhost:9200/_cat/count/post?v"

# 查看具体文档
curl "localhost:9200/post/_doc/1234567890"

# 测试分词
curl -X POST "localhost:9200/_analyze" -H 'Content-Type: application/json' -d'
{
  "analyzer": "ik_smart",
  "text": "你的搜索关键词"
}'
```

### 4. 消息队列积压

**症状**：帖子创建后搜索不到

**排查步骤**：

```bash
# 检查 RabbitMQ 队列
docker exec -it bluebell_rabbitmq rabbitmqctl list_queues

# 检查消费者日志
docker logs <bluebell_app_container>
```

**解决方案**：
- 确保 `SyncConsumer` 已启动
- 检查 RabbitMQ 连接是否正常
- 重启应用服务

### 5. 索引性能优化

如果搜索性能下降，可以考虑：

```bash
# 1. 增加分片数（需在创建索引时设置）
"settings": {
  "number_of_shards": 3,
  "number_of_replicas": 1
}

# 2. 优化 JVM 堆内存（docker-compose.yml）
ES_JAVA_OPTS=-Xms2g -Xmx2g

# 3. 添加更多 ES 节点（集群模式）
```

---

## 最佳实践

### 1. 容错设计

- ES 是**非关键依赖**，初始化失败不会阻止应用启动
- 消息消费失败时 `Nack` 不重新入队（`requeue=false`），避免死循环
- 删除不存在的文档（404）视为成功

### 2. 幂等性保证

- 使用 `post_id` 作为文档 ID，重复索引同一文档会覆盖而非新增
- 消费者手动确认消息，确保处理成功

### 3. 性能优化

- 异步消息解耦，不阻塞主业务流程
- 限制搜索结果分页大小（最大 50）
- 内容高亮限制片段数量和大小（3 段 × 150 字符）

### 4. 监控建议

```bash
# 监控 ES 集群状态
curl "localhost:9200/_cluster/health"

# 监控索引性能
curl "localhost:9200/_cat/indices/post?v&h=index,docs.count,store.size"

# 监控慢查询
curl -X PUT "localhost:9200/_cluster/settings" -H 'Content-Type: application/json' -d'
{
  "persistent": {
    "search.slowlog.threshold.query.warn": "10s",
    "search.slowlog.threshold.fetch.warn": "1s"
  }
}'
```

---

## 文件结构

```
internal/
├── infrastructure/es/
│   ├── client.go      # ES 客户端初始化和连接管理
│   ├── mapping.go     # 索引映射定义和创建逻辑
│   └── search.go      # 搜索功能实现（DSL 构建、解析）
├── service/mq/
│   ├── es_consumer.go # ES 同步消费者（索引/删除文档）
│   ├── message.go     # 同步消息结构体定义
│   ├── publisher.go   # 消息发布器
│   └── connection.go  # MQ 连接、Exchange 和 Queue 声明
├── handler/post_handler/
│   └── handler.go     # 发布同步消息的业务逻辑
├── config/
│   └── config.go      # ES 配置结构定义
cmd/bluebell/
└── main.go            # 应用入口，ES 初始化和消费者启动
```

---

## 相关文档

- [RabbitMQ 集成文档](./rabbitmq-guide.md)
- [Elasticsearch 官方文档](https://www.elastic.co/guide/en/elasticsearch/reference/current/index.html)
- [IK 分词器文档](https://github.com/medcl/elasticsearch-analysis-ik)
- [Go ES 客户端文档](https://pkg.go.dev/github.com/elastic/go-elasticsearch/v8)
