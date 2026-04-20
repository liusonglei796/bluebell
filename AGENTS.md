# Bluebell - Go 社区论坛项目

## 项目概览

- **语言**: Go 1.25
- **框架**: Gin + GORM
- **架构**: 分层架构 + DDD 模式 (internal/)
- **入口**: `cmd/bluebell/main.go`

## 核心命令

```bash
# 本地运行 (需要 config.yaml)
go run ./main.go ./config.yaml

# Docker 启动全部服务
docker-compose up -d

# 停止全部服务
docker-compose down
```

## 服务端口 (config.yaml)

| 服务 | 端口 |
|------|------|
| App | 8080 |
| MySQL | 3307 |
| Redis | 6380 |
| RabbitMQ | 5672 (管理: 15672) |
| Elasticsearch | 9200 |
| Jaeger | 16686 |

## 目录结构

```
cmd/bluebell/main.go       # 入口文件
internal/
  config/                  # 配置加载
  dao/cache/               # Redis 访问
  dao/database/            # MySQL 访问
  domain/                  # 领域模型
  dto/                     # 数据传输对象
  handler/                 # HTTP 处理层
  middleware/              # 中间件 (JWT, 限流)
  model/                   # 数据库模型
  router/                  # 路由注册
  service/                 # 业务逻辑层
  infrastructure/          # 基础设施 (logger, otel, es, ai, mq)
```

## 关键架构约束

1. **MQ 消费者启动顺序**: RabbitMQ 初始化必须在 services 之前 (main.go line 105)
2. **Redis 热度刷新**: 启动时自动启动 `HotScoreRefresher` 后台 goroutine (main.go line 98)
3. **Gin 模式**: 强制 ReleaseMode (main.go line 52)
4. **Config 加载**: 通过 `-conf` flag 指定配置文件，默认 `./config.yaml`

## 常见陷阱

- **Redis 连接失败**: 服务无法启动，会 fatal exit
- **MySQL 连接失败**: 服务无法启动，会 fatal exit
- **RabbitMQ 连接失败**: 服务继续启动，但 MQ 功能不可用 (vote/audit/sync consumers 不会启动)
- **ES 客户端失败**: 服务继续启动，仅搜索功能不可用
- **AI Auditor 失败**: 服务继续启动，仅内容审核功能不可用

## 数据库初始化

SQL 文件位于 `sql/` 目录，docker-compose 会自动执行初始化脚本:
```yaml
volumes:
  - ./sql:/docker-entrypoint-initdb.d
```

## 测试

项目目前没有单元测试目录，所有验证通过手动 API 测试或集成测试。

## Swagger 文档

访问 `http://127.0.0.1:8080/swagger/index.html` (服务启动后)

## NON-STANDARD PATTERNS

- **No GitHub workflows**: 无 .github/workflows 目录，无 CI/CD
- **No linter config**: 无 .golangci.yml
- **No test files**: 项目无 *_test.go 文件，手动 API 测试

## WHERE TO LOOK

| Task | Location |
|------|----------|
| API 路由 | internal/router/router.go |
| 业务逻辑 | internal/service/{usersvc,postsvc,communitysvc}/ |
| Handler 层 | internal/handler/{user_handler,post_handler,community_handler}/ |
| 中间件 | internal/middleware/ |

## 参考文档

- `docs/ddd_architecture.md` - 完整的 DDD 架构设计方案
- `docs/rabbitmq-guide.md` - RabbitMQ 消费者实现
- `docs/elasticsearch-guide.md` - ES 数据同步
- `docs/opentelemetry-guide.md` - 链路追踪配置
