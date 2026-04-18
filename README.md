# Bluebell 社区 Web 应用

[![Go Version](https://img.shields.io/badge/go-1.25.0-blue.svg)](https://golang.org)
[![Gin](https://img.shields.io/badge/gin-1.12.0-green.svg)](https://github.com/gin-gonic/gin)
[![Vue](https://img.shields.io/badge/vue-3.x-brightgreen.svg)](https://vuejs.org)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Bluebell（蓝铃花）是一个基于 Go + Vue 3 构建的现代化社区 Web 应用，采用 DDD（领域驱动设计）架构，支持完整的用户认证、帖子发布、评论互动、内容审核等功能。

## 🌟 特性

- **后端技术栈**
  - Go 1.25+ 高性能后端服务
  - Gin Web 框架
  - GORM ORM + MySQL 8.0
  - Redis 缓存与热度分数计算
  - RabbitMQ 消息队列异步处理
  - Elasticsearch 全文搜索
  - OpenTelemetry 链路追踪 + Jaeger
  - JWT 身份认证
  - Swagger API 文档
  - AI 内容审核（支持 OpenAI/ModelScope）

- **前端技术栈**
  - Vue 3 + TypeScript
  - Vite 构建工具
  - Tailwind CSS 样式

- **架构特性**
  - DDD 领域驱动设计
  - 依赖注入（DI）
  - 分层架构（Handler → Service → Repository）
  - 工作单元模式（UoW）
  - 优雅关机
  - Docker 容器化部署

## 📦 项目结构

```
bluebell/
├── cmd/                    # 应用程序入口
│   └── bluebell/          # 主程序
├── internal/              # 内部业务代码（不可外部引用）
│   ├── config/            # 配置管理
│   ├── dao/               # 数据访问层（MySQL/Redis）
│   ├── domain/            # 领域模型
│   ├── dto/               # 数据传输对象
│   ├── handler/           # HTTP 处理器
│   ├── http_server/       # HTTP 服务器配置
│   ├── infrastructure/    # 基础设施（ES/AI/MQ/日志等）
│   ├── middleware/        # Gin 中间件
│   ├── model/             # 数据库模型
│   ├── router/            # 路由配置
│   └── service/           # 业务逻辑层
├── frontend/              # Vue 3 前端项目
├── docs/                  # 项目文档
├── nginx/                 # Nginx 配置
├── prometheus/            # Prometheus 监控配置
├── sql/                   # 数据库初始化脚本
├── docker-compose.yml     # Docker Compose 编排
├── Dockerfile             # 后端 Docker 镜像
├── Makefile               # Make 构建脚本
├── config.yaml            # 应用配置文件
└── go.mod                 # Go 模块定义
```

## 🚀 快速开始

### 环境要求

- Go 1.25+
- Node.js 18+
- Docker & Docker Compose
- MySQL 8.0+
- Redis 6+

### 方式一：Docker Compose 部署（推荐）

一键启动所有服务（MySQL、Redis、RabbitMQ、Elasticsearch、Jaeger、应用服务等）：

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看应用日志
docker-compose logs -f bluebell

# 停止所有服务
docker-compose down
```

服务访问地址：
- **前端页面**: http://localhost:80
- **API 接口**: http://localhost:80/api/v1
- **Swagger 文档**: http://localhost:80/swagger/index.html
- **RabbitMQ 管理**: http://localhost:15672 (账号：bluebell / 密码：bluebell123)
- **Jaeger 追踪**: http://localhost:16686
- **Elasticsearch**: http://localhost:9200
- **MySQL**: localhost:3307 (root / 15939087780Ll@)
- **Redis**: localhost:6380

### 方式二：本地开发

#### 1. 启动依赖服务

```bash
docker-compose up -d mysql redis rabbitmq elasticsearch
```

#### 2. 初始化数据库

```bash
# 导入 SQL 脚本到 MySQL
mysql -h 127.0.0.1 -P 3307 -u root -p15939087780Ll@ bluebell < sql/init.sql
```

#### 3. 配置应用

编辑 `config.yaml` 文件，确保数据库、Redis 等连接信息正确。

#### 4. 安装依赖并运行

```bash
# 后端
go mod download
make run

# 前端
cd frontend
npm install
npm run dev
```

## 🔧 开发命令

```bash
# 格式化代码并编译
make all

# 仅编译（生成 Linux amd64 二进制）
make build

# 本地运行（使用 config.yaml）
make run

# 代码格式化和检查
make gotool

# 清理二进制文件
make clean

# 查看帮助
make help
```

## 📚 API 文档

启动应用后访问 Swagger UI：

```
http://localhost:8080/swagger/index.html
```

或查看 [docs/docs.go](docs/docs.go) 了解 API 详情。

## 🏗️ 架构说明

### 分层架构

```
┌─────────────────────────────────────────┐
│            Handler Layer                │  ← HTTP 请求处理
├─────────────────────────────────────────┤
│            Service Layer                │  ← 业务逻辑
├─────────────────────────────────────────┤
│         Repository Layer (UoW)          │  ← 数据访问
├─────────────────────────────────────────┤
│      Infrastructure (MySQL/Redis/ES)    │  ← 基础设施
└─────────────────────────────────────────┘
```

### 核心模块

| 模块 | 说明 |
|------|------|
| **用户认证** | JWT Token 认证、刷新令牌、权限控制 |
| **帖子管理** | 发布、编辑、删除、投票、热度排序 |
| **评论系统** | 嵌套评论、点赞、审核 |
| **社区板块** | 多板块分类、版主管理 |
| **全文搜索** | Elasticsearch 帖子/评论搜索 |
| **内容审核** | AI 自动审核（OpenAI/ModelScope） |
| **异步任务** | RabbitMQ 投票计数、审核、ES 同步 |
| **链路追踪** | OpenTelemetry + Jaeger 全链路监控 |

## ⚙️ 配置说明

主要配置项在 `config.yaml`：

```yaml
app:
  name: "bluebell"
  port: 8080
  mode: "debug"  # debug/release

mysql:
  host: "localhost"
  port: 3307
  db_name: "bluebell"
  user: "root"
  passwd: "your_password"

redis:
  host: "localhost"
  port: 6380
  db_name: 1

rabbitmq:
  url: "amqp://user:pass@localhost:5672/"

es:
  addresses:
    - "http://localhost:9200"

ai_audit:
  enabled: true
  base_url: "https://api.openai.com/v1"
  api_key: "your-api-key"
  model: "gpt-4o-mini"
```

## 📊 监控与可观测性

- **链路追踪**: OpenTelemetry Collector → Jaeger
- **指标监控**: Prometheus + Grafana（可选）
- **日志收集**: Zap 结构化日志 + Lumberjack 日志轮转

## 🧪 测试

```bash
# 运行单元测试
go test ./...

# 带覆盖率
go test -cover ./...
```

## 📖 详细文档

- [DDD 架构设计](docs/ddd_architecture.md)
- [Elasticsearch 使用指南](docs/elasticsearch-guide.md)
- [RabbitMQ 使用指南](docs/rabbitmq-guide.md)
- [OpenTelemetry 链路追踪](docs/opentelemetry-guide.md)

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

MIT License

## 👥 作者

Bluebell Team

---

**注意**: 生产环境部署时，请务必修改默认密码和密钥！
