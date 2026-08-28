# Bluebell 论坛系统后端

基于 **Go (Gin + GORM + Go-Redis + RabbitMQ)** 开发的高性能、高可用分布式论坛社区后端服务。

---

## 🌟 核心特性与架构设计

### 1. 架构分层设计
- **Controller 层**：基于 Gin 框架，处理 HTTP 请求绑定、参数校验与标准响应格式输出。
- **Service 层**：承载核心业务逻辑，负责多级缓存水合、发帖投票算法聚合、事件驱动发布。
- **DAO & Cache 层**：
  - **MySQL (GORM)**：高可用关系型数据库存储，支持反范式单表极速直查优化与事务一致性。
  - **Redis (Go-Redis v9)**：集成 **RESP3 原生客户端缓存 (Client-Side Caching)** 降低网络 RTT；维护 ZSet 热度排行榜、点赞状态与分布式去重防重锁。
- **MQ & Consumer 层**：
  - 基于 **RabbitMQ** 构建可靠的异步事件驱动体系（Topic Exchange + 专属死信队列 DLQ）。
  - 支持发帖、评论、关注等多业务消息分发，内置 Redis `SetNX` + MySQL `processed_events` 双重幂等保障。

### 2. 高性能热度算法 (Reddit / HackerNews 动态 Gravity)
- 帖子排行榜支持 `score`（动态重力衰减热度算法）与 `time`（时间发布序）多维排序。
- 内置纯函数置顶贴优先重排序机制（`IsPinned` 优先）。

### 3. 多级缓存体系
- **L1 进程内客户端缓存 (RESP3 CSC)**：基于 Redis 6.0+ `CLIENT TRACKING` 原生协议，0 网络 RTT，数据失效时自动精准推送剔除。
- **L2 Redis 分布式实体缓存**：缓存帖子、用户信息、未读红点计数。
- **L3 MySQL 兜底存储**：反范式冗余字段极速查询。

---

## 📁 目录结构

```text
bluebell/
├── cmd/
│   ├── api/             # API 服务主入口 (生产网关模式)
│   ├── bluebell/        # Monolith 单体服务主入口
│   └── consumer/        # 后台异步事件消费独立 Worker 入口
├── internal/
│   ├── config/          # Viper 配置文件解析与初始化
│   ├── consumer/        # 异步 Worker 业务容器 (Notification, Feed, Counter)
│   ├── controller/      # Gin 控制层 Handler
│   ├── dao/             # 数据访问层 (MySQL & Redis 缓存)
│   ├── dto/             # 请求 (Request) 与响应 (Response) DTO
│   ├── http_server/     # HTTP 优雅启停与中间件挂载
│   ├── jwt/             # JWT 鉴权 Token 生成与解析
│   ├── logger/          # Zap 日志集成与日志归档
│   ├── middleware/      # Gin 中间件 (Auth, RateLimit, CORS, Trace)
│   ├── model/           # GORM 数据模型与错误码常量
│   ├── mq/              # RabbitMQ 拓扑声明与 EventBus 事件总线
│   ├── router/          # API 路由注册
│   ├── service/         # 核心业务逻辑层
│   └── snowflake/       # 64 位雪花算法分布式 ID 生成器
├── pkg/
│   ├── enum/            # 领域状态枚举定义
│   └── event/           # 泛型事件信封与业务载荷定义
├── config.yaml          # 本地配置文件模板
├── docker-compose.yml   # 容器化编排 (API + Consumer + MySQL + Redis + RabbitMQ)
├── Dockerfile           # 多阶段容器构建文件
├── go.mod               # Go 模块依赖管理
└── README.md            # 项目文档说明
```

---

## 🚀 快速开始

### 1. 环境依赖
- Go 1.22+
- MySQL 8.0+
- Redis 6.0+ (建议开启 RESP3 支持)
- RabbitMQ 3.8+

### 2. 本地配置
复制或修改 `config.yaml`：
```yaml
app:
  name: "bluebell"
  mode: "dev"
  port: 8080

mysql:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "your_password"
  dbname: "bluebell"

redis:
  host: "127.0.0.1"
  port: 6379
  password: ""
  db: 0

rabbitmq:
  url: "amqp://guest:guest@127.0.0.1:5672/"
```

### 3. 本地编译与运行

#### 运行 API Web 服务
```bash
go run ./cmd/api/main.go -conf=./config.yaml
```

#### 运行后台异步消费服务 (Worker)
```bash
go run ./cmd/consumer/main.go -conf=./config.yaml
```

### 4. 运行测试
```bash
go test -v ./...
```

---

## 🐳 Docker 容器化部署

一键启动整套微服务群与基础设施：
```bash
docker-compose up -d --build
```
服务包含：
- `bluebell_api`: Web HTTP 网关服务 (端口 8080)
- `bluebell_consumer`: 后台消息消费 Worker
- `bluebell_mysql`: MySQL 8.0
- `bluebell_redis`: Redis 7.0
- `bluebell_rabbitmq`: RabbitMQ (包含管理面板 15672)

---

## 📄 License
MIT License
