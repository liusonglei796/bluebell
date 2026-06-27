# Bluebell

> Go 语言实战项目 —— 社区 Web 框架（DDD 架构）

Bluebell 是一个基于 **Go 语言** 构建的现代化社区 Web 应用，采用 **领域驱动设计（DDD）** 分层架构，提供了完整的用户系统、内容发布、社交互动、全文搜索等社区核心功能。项目包含 Go 后端 API 服务与 Vue 3 前端 SPA，配备完整的可观测性基础设施。

---

## 技术栈

### 后端

| 类别 | 技术 | 用途 |
|------|------|------|
| **语言** | Go 1.25 | 高性能编译型语言 |
| **框架** | Gin | HTTP Web 框架 |
| **架构** | DDD（领域驱动设计） | 分层解耦：Interface → Application → Domain → Infrastructure |
| **数据库** | MySQL 8.0 + GORM | 关系型数据持久化 |
| **缓存** | Redis | 帖子排名、Token 缓存、投票计数 |
| **消息队列** | RabbitMQ | 异步投票处理、ES 索引同步、用户动态推送 |
| **搜索引擎** | Elasticsearch | 帖子全文搜索（含高亮） |
| **认证鉴权** | JWT + Redis SSO | Access/Refresh Token 双令牌 + 单设备登录校验 |
| **ID 生成** | Snowflake | 分布式唯一 ID |
| **配置管理** | Viper | 配置文件读取 + 热更新 |
| **日志** | Zap + Lumberjack | 结构化日志 + 日志轮转 |
| **参数校验** | go-playground/validator | 请求参数校验 + 中文翻译 |
| **可观测性** | OpenTelemetry + Pyroscope + Prometheus | 链路追踪（Tempo）、指标（Mimir/Metrics）、日志（Loki）、持续性能剖析 |
| **限流** | 令牌桶 | 接口级限流保护 |
| **文档** | Swagger | API 接口文档自动生成 |

### 前端

| 类别 | 技术 |
|------|------|
| **框架** | Vue 3（Composition API） |
| **语言** | TypeScript |
| **构建** | Vite |
| **样式** | Tailwind CSS |
| **状态管理** | Pinia |
| **路由** | Vue Router |
| **HTTP** | Axios |
| **图标** | Lucide |

---

## 项目架构

```
bluebell/
├── cmd/server/main.go               # 应用入口（HTTP + 内嵌消费者）
├── internal/
│   ├── config/                        # 配置定义 + Viper 热加载
│   ├── domain/                        # 领域层（核心业务规则）
│   │   ├── entity/                    #   领域实体（User, Post, Vote, Remark...）
│   │   ├── repository.go             #   仓储接口定义
│   │   ├── search.go                 #   搜索引擎仓储接口
│   │   └── token.go                  #   令牌服务接口
│   ├── application/                   # 应用层（用例编排）
│   │   ├── post/                     #   帖子服务
│   │   ├── user/                     #   用户服务
│   │   ├── community/                #   社区服务
│   │   ├── social/                   #   社交服务
│   │   ├── bookmark/                 #   收藏服务
│   │   ├── dto/                      #   请求/响应 DTO
│   │   └── interfaces.go            #   应用服务接口定义
│   ├── interfaces/http/               # 接口层（HTTP 协议适配）
│   │   ├── handler/                  #   API 处理器
│   │   ├── router/                   #   路由注册
│   │   └── render/                   #   响应渲染
│   ├── middleware/                    # Gin 中间件
│   │   ├── auth.go                   #   JWT 认证 + SSO
│   │   ├── cors.go                   #   跨域
│   │   ├── gin.go                    #   日志 + Recovery
│   │   ├── ratelimit_token.go        #   令牌桶限流
│   │   └── timeout.go                #   请求超时
│   ├── infrastructure/                # 基础设施层
│   │   ├── persistence/mysql/        #   MySQL 仓储实现
│   │   ├── persistence/redis/        #   Redis 仓储实现
│   │   ├── es/                       #   Elasticsearch 客户端
│   │   ├── mq/                       #   RabbitMQ 生产者/消费者
│   │   ├── jwt/                      #   JWT 令牌实现
│   │   ├── snowflake/                #   雪花 ID 生成
│   │   ├── logger/                   #   Zap 日志初始化
│   │   ├── metrics/                  #   自定义业务指标
│   │   ├── otel/                     #   OpenTelemetry SDK
│   │   ├── profiler/                 #   Pyroscope 持续剖析
│   │   └── translate/                #   Validator 中文翻译
│   └── di/                           # 依赖注入（手动 DI）
├── frontend/                          # Vue 3 前端 SPA
│   ├── src/
│   │   ├── pages/                    #   页面组件
│   │   ├── components/               #   通用组件
│   │   ├── api/                      #   API 调用封装
│   │   ├── store/                    #   Pinia 状态管理
│   │   └── router/                   #   前端路由
│   └── ...                           #   Vite + Tailwind 配置
├── sql/migrations/                    # 数据库迁移脚本
├── pkg/enum/                          # 公共枚举
├── docker-compose.yml                 # 完整服务栈编排（含可观测性）
└── config.yaml                        # 应用配置文件
```

---

## 核心功能

### 👤 用户系统
- **注册/登录** — 用户名密码注册，bcrypt 加密存储，支持登录后 JWT 双令牌（Access + Refresh）
- **GitHub OAuth** — 第三方社交登录，自动创建本地账号
- **SSO 单点登录** — Redis 实时校验，同一账号仅允许一个设备在线（后登录踢掉前者）
- **Token 自动续期** — Refresh Token 无感刷新，减少重复登录
- **头像上传** — 用户可上传自定义头像

### 📝 帖子系统
- **发布帖子** — 支持选择社区分类，Snowflake 分布式 ID
- **帖子列表** — 支持按时间、热度排序，分页查询
- **帖子详情** — 包含作者信息、社区归属、评论列表
- **软删除** — 仅作者可删除，标记状态而非物理删除
- **全文搜索** — 基于 Elasticsearch 的帖子标题+内容搜索，支持搜索结果高亮显示

### 👍 投票与热度
- **点赞/取消** — 抖音风格（仅点赞，无反对），每票影响帖子热度分数
- **Redis ZSet 排序** — 帖子按时间戳和分数分别维护两个有序集合
- **异步落库** — 投票请求先写入 Redis，通过 RabbitMQ 缓冲聚合后批量落 MySQL
- **热度分数定时刷新** — Gravity 算法定期计算帖子热度

### 💬 评论系统
- **发布评论** — 对帖子发表评论，支持回复评论（嵌套回复）
- **评论列表** — 按时间排序，关联作者信息
- **删除评论** — 支持按 ID 或按帖子批量删除

### 🏘️ 社区系统
- **社区列表/详情** — 公开浏览所有社区
- **创建社区** — 仅管理员可创建
- **社区帖子分类** — 每个帖子归属一个社区，支持按社区筛选帖子列表

### 🤝 社交功能
- **关注/取消关注** — 用户间关注关系
- **用户资料** — 个人简介、GitHub 绑定、头像展示
- **用户动态** — 记录用户行为（发帖、点赞、关注、评论），形成时间线

### 🔖 收藏功能
- **收藏/取消收藏帖子** — 用户收藏喜欢的帖子
- **收藏列表** — 分页查看个人收藏

### 🛡️ 安全与防护
- **JWT 认证中间件** — 保护需要登录的接口
- **令牌桶限流** — 可配置的接口级访问限流
- **请求超时控制** — 防止慢请求占用资源
- **CORS 跨域** — 支持前后端分离部署
- **密码 bcrypt 加密** — 10 轮加密成本

---

## 可观测性（Observability）

项目集成了完整的 **Grafana 可观测性栈**，通过 Docker Compose 一键部署：

| 组件 | 用途 |
|------|------|
| **OpenTelemetry** | 分布式链路追踪（Trace）、指标采集（Metric）、日志关联（Log） |
| **Tempo** | 链路追踪后端存储 |
| **Loki** | 日志聚合，支持 LogQL 查询 |
| **Mimir** | 指标数据后端，兼容 Prometheus |
| **Pyroscope** | 持续性能剖析（Continuous Profiling），可视化 CPU/内存热点 |
| **Prometheus** | Go 运行时指标（goroutine、GC、内存） |
| **Grafana** | 统一可视化仪表盘，预配置数据源和看板 |

---

## 快速开始

### 环境要求

- Go 1.25+
- Node.js 20+
- Docker & Docker Compose（可选，用于基础设施）

### 本地开发

```bash
# 1. 启动基础设施（MySQL、Redis、RabbitMQ、ES）
docker compose up -d mysql redis rabbitmq elasticsearch

# 2. 启动后端
go run ./cmd/server --conf ./config.yaml

# 3. 启动前端
cd frontend && npm install && npm run dev

# 4. 访问
# 前端: http://localhost:5173
# API:  http://localhost:8082/api/v1
# Swagger: http://localhost:8082/swagger/index.html
```

### Docker 完整部署

```bash
# 启动全栈（含可观测性）
docker compose up -d

# 访问
# 应用: http://localhost:8899
# Grafana: http://localhost:3000
# Grafana 默认匿名登录，预配置了所有数据源和仪表盘
```

---

## API 概览

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| POST | `/api/v1/signup` | 用户注册 | 公开 |
| POST | `/api/v1/login` | 用户登录 | 公开 |
| POST | `/api/v1/refresh_token` | 刷新 Token | 公开 |
| POST | `/api/v1/logout` | 登出 | JWT |
| GET | `/api/v1/community` | 社区列表 | 公开 |
| GET | `/api/v1/community/:id` | 社区详情 | JWT |
| POST | `/api/v1/community` | 创建社区（管理员） | JWT |
| GET | `/api/v1/posts` | 帖子列表（支持排序） | 公开 |
| GET | `/api/v1/post/:id` | 帖子详情 | 公开 |
| POST | `/api/v1/post` | 创建帖子 | JWT |
| DELETE | `/api/v1/post/:id` | 删除帖子 | JWT |
| POST | `/api/v1/vote` | 点赞/取消点赞 | JWT |
| POST | `/api/v1/remark` | 发表评论 | JWT |
| GET | `/api/v1/post/:id/remarks` | 评论列表 | 公开 |
| GET | `/api/v1/search` | 全文搜索 | 公开 |
| POST | `/api/v1/follow/:id` | 关注用户 | JWT |
| DELETE | `/api/v1/follow/:id` | 取消关注 | JWT |
| GET | `/api/v1/user/:id` | 用户资料 | 公开 |
| POST | `/api/v1/bookmark` | 收藏帖子 | JWT |
| DELETE | `/api/v1/bookmark/:id` | 取消收藏 | JWT |

---

## 设计原则

### DDD 分层架构

```
┌─────────────────────────────────┐
│     Interface Layer (HTTP)      │  ← Gin Router + Handlers
├─────────────────────────────────┤
│   Application Layer (Service)   │  ← Use Case Orchestration
├─────────────────────────────────┤
│       Domain Layer (Entity)     │  ← Core Business Rules
├─────────────────────────────────┤
│  Infrastructure Layer (Repo)    │  ← MySQL, Redis, ES, MQ
└─────────────────────────────────┘
```

- **领域层**（Domain）— 不依赖任何外部框架，定义核心业务实体和规则（密码加密、帖子校验、投票规则）
- **应用层**（Application）— 编排业务流程，协调领域层和基础设施层，无业务规则
- **接口层**（Interface）— 适配具体通信协议（HTTP），将外部请求转化为应用层调用
- **基础设施层**（Infrastructure）— 实现仓储接口，封装数据持久化、缓存、消息队列等技术细节

### 并发安全

本项目面向 HTTP 高并发场景，所有涉及共享资源读写的操作均做了并发安全优化：

- **Redis 缓存操作** — 使用 `sync.Map` 或互斥锁保护并发写入
- **投票缓冲聚合** — 使用带缓冲 channel + 定时 flush 机制，减少数据库写入压力
- **配置热更新** — 使用 `atomic.Value` 保证配置的无锁安全读取

---

## License

本项目仅供学习参考，未经许可不得用于商业用途。
