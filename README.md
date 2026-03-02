# Bluebell 社区论坛

<div align="center">

![Go Version](https://img.shields.io/badge/Go-1.25%2B-blue?logo=go)
![License](https://img.shields.io/badge/license-MIT-green)
![Build Status](https://img.shields.io/badge/build-passing-brightgreen)
![Version](https://img.shields.io/badge/version-2.0.0-orange)

一个基于 Go 语言的高性能社区论坛后端服务，采用 **DDD (领域驱动设计) / Clean Architecture** 架构，支持用户认证、社区管理、帖子发布和投票系统。

[特性](#特性) • [快速开始](#快速开始) • [项目架构](#项目架构) • [API 文档](#api-文档) • [开发指南](#开发指南) • [教学文档](#教学文档)

</div>

---

## 📖 项目简介

Bluebell 是一个类似 Reddit 的社区论坛系统,用户可以在不同主题的社区下发布帖子、参与讨论并投票。项目采用现代化的 Go Web 开发技术栈,适合作为学习 Go 语言 Web 开发的实战项目。

### 核心功能

- 🔐 **用户系统**: 注册、登录、JWT 认证、Token 刷新、单点登录
- 🏘️ **社区管理**: 社区列表、社区详情、社区分类
- 📝 **帖子系统**: 发布帖子、查看详情、分页列表、社区筛选、软删除
- 👍 **投票机制**: Reddit 算法、赞成/反对票、防重复投票、7天投票期限
- 🔥 **排序策略**: 按时间排序、按热度排序
- 📊 **高性能**: Redis 缓存、Pipeline 批量查询、N+1 问题优化

---

## ✨ 特性

### 技术架构

- **Web 框架**: [Gin](https://github.com/gin-gonic/gin) - 高性能 HTTP Web 框架
- **数据库**: MySQL + [GORM](https://gorm.io) - 关系型数据存储与 ORM
- **缓存**: [Redis](https://redis.io) + [go-redis](https://github.com/redis/go-redis) - 高性能缓存和排序
- **日志**: [Zap](https://github.com/uber-go/zap) + [Lumberjack](https://github.com/natefinch/lumberjack) - 结构化日志和日志轮转
- **配置**: [Viper](https://github.com/spf13/viper) - 配置管理
- **认证**: JWT (golang-jwt/jwt) - 无状态认证
- **ID 生成**: [Snowflake](https://github.com/bwmarrin/snowflake) - 分布式唯一 ID
- **参数校验**: [validator](https://github.com/go-playground/validator) - 请求参数校验
- **文档**: [Swagger](https://swagger.io) - 自动生成 API 文档

### 工程特性

- ✅ **DDD / Clean Architecture** - 领域驱动设计，依赖倒置，接口抽象
- ✅ **UnitOfWork 模式** - 事务支持 + Repository 聚合访问
- ✅ **依赖注入** - Services 聚合器 + Handlers 聚合器，完整 DI 链路
- ✅ **Go Standard Layout** - `cmd/` + `internal/` + `pkg/` 标准目录结构
- ✅ **RESTful API** - 符合 REST 规范的接口设计
- ✅ **优雅关机** - 支持平滑关闭，不丢失请求
- ✅ **结构化日志** - Zap 高性能日志，日志分级和轮转
- ✅ **统一错误处理** - ErrorX 错误链 + Wrap/Unwrap + 业务错误码
- ✅ **中间件支持** - JWT 认证、日志、限流、超时控制（均通过接口注入）
- ✅ **完整文档** - Swagger API 文档 + 18 章教学文档

---

## 🚀 快速开始

### 环境要求

确保你的开发环境满足以下要求:

| 依赖 | 版本要求 | 说明 |
|------|---------|------|
| **Go** | 1.19+ | [官方下载](https://golang.org/dl/) |
| **MySQL** | 5.7+ / 8.0+ | 关系型数据库 |
| **Redis** | 5.0+ | 缓存和排序 |
| **Git** | 任意版本 | 版本控制 |

### 安装步骤

#### 1. 克隆项目

```bash
git clone <repository-url>
cd bluebell
```

#### 2. 安装依赖

```bash
go mod download
```

#### 3. 配置数据库

**启动 MySQL:**

```bash
# macOS (Homebrew)
brew services start mysql

# Linux (systemd)
sudo systemctl start mysql

# Windows
# 通过服务管理器启动 MySQL 服务
```

**创建数据库:**

```sql
CREATE DATABASE bluebell DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci;
```

**导入表结构:**

> 💡 **提示**: 项目使用 GORM 自动迁移，启动时会自动创建/更新表结构。你也可以手动创建数据库表。

主要表结构包括:
- `user` - 用户表
- `community` - 社区表
- `post` - 帖子表

#### 4. 配置 Redis

**启动 Redis:**

```bash
# macOS (Homebrew)
brew services start redis

# Linux (systemd)
sudo systemctl start redis

# Docker
docker run -d -p 6379:6379 redis:latest
```

#### 5. 修改配置文件

编辑 `config.yaml`:

```yaml
app:
  name: "bluebell"
  port: 8080
  mode: "dev"  # dev / release
  version: "0.1.0"

mysql:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "your_password"  # ⚠️ 修改为你的 MySQL 密码
  db_name: "bluebell"
  max_open_conns: 100
  max_idle_conns: 10

redis:
  host: "127.0.0.1"
  port: 6379
  password: ""  # 如果设置了密码,请填写
  db_name: 0
  pool_size: 100

log:
  level: "debug"  # debug / info / warn / error
  file_name: "bluebell.log"
  max_size: 100  # MB
  max_backups: 7
  max_age: 30  # days

snowflake:
  start_time: "2024-01-01"
  machine_id: 1
```

#### 6. 运行项目

**方式 1: 直接运行**

```bash
go run ./cmd/bluebell/ -conf ./config.yaml
```

**方式 2: 编译后运行**

```bash
# 编译
go build -o bluebell ./cmd/bluebell/

# 运行
./bluebell -conf ./config.yaml
```

#### 7. 验证运行

**访问 Swagger 文档:**

打开浏览器访问: [http://127.0.0.1:8080/swagger/index.html](http://127.0.0.1:8080/swagger/index.html)

**测试 API:**

```bash
# 健康检查 (如果有)
curl http://127.0.0.1:8080/ping

# 注册用户
curl -X POST http://127.0.0.1:8080/api/v1/signup \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "123456",
    "re_password": "123456"
  }'
```

---

## 📁 项目结构

```
bluebell/
├── cmd/
│   └── bluebell/
│       └── main.go                    # 程序入口 (DDD 风格依赖注入组装)
├── internal/
│   ├── config/
│   │   └── config.go                  # 配置加载 (Viper)
│   ├── domain/
│   │   └── repository/                # 领域层 — 接口定义 (依赖倒置核心)
│   │       ├── mysql.go               # PostRepo / CommunityRepo / UserRepo 接口
│   │       ├── cache.go               # VoteCache / PostCache / UserTokenCache 接口
│   │       ├── unit_of_work.go        # UnitOfWork 接口 + Transaction
│   │       └── errors.go              # 领域级哨兵错误
│   ├── model/                         # 实体模型 (GORM)
│   │   ├── user.go / post.go / community.go
│   ├── dto/
│   │   ├── request/                   # 请求 DTO
│   │   └── response/                  # 响应 DTO
│   ├── dao/                           # 数据访问层 — 接口实现
│   │   ├── mysql/                     # UnitOfWork 实现 + 各 Repo struct
│   │   │   ├── mysql.go               # Init + Repositories (UnitOfWork)
│   │   │   ├── user.go / post.go / community.go
│   │   └── redis/                     # Cache Repository 实现
│   │       ├── redis.go / keys.go / vote.go / user.go
│   ├── service/                       # 业务逻辑层
│   │   ├── services.go                # Services 聚合器
│   │   ├── post/ / community/ / user/ / vote/
│   ├── handler/                       # HTTP 处理层
│   │   ├── handler.go                 # Handlers 聚合 + 通用响应
│   │   ├── validator.go               # 参数校验
│   │   ├── user_handler.go / post_handler.go / ...
│   ├── router/
│   │   └── router.go                  # 路由注册
│   └── infrastructure/
│       ├── logger/logger.go           # Zap 日志
│       └── middleware/                # JWT 认证 / 限流 / 超时
├── pkg/                               # 公共工具包 (不依赖 internal)
│   ├── errorx/                        # 错误链 (Wrap/Unwrap/GetCode)
│   ├── jwt/                           # JWT 工具 (Init 配置模式)
│   ├── snowflake/                     # 雪花 ID 生成
│   ├── constants/                     # 业务常量 + Redis Key 前缀
│   └── enum/                          # 枚举常量 (按领域嵌套)
│       ├── post/post_status/ / post/post_order/
│       ├── vote/vote_direction/
│       └── user/user_status/
├── docs/                              # Swagger 文档
├── config.yaml                        # 配置文件
├── go.mod / go.sum
└── 教学文档/                           # 18 章教学文档
```

---

## 🏗️ 项目架构

### DDD / Clean Architecture

Bluebell 采用 **DDD (领域驱动设计) / Clean Architecture**，依赖方向严格由外向内：

```
┌─────────────────────────────────────────────────┐
│                  HTTP Client                     │
└────────────────────┬────────────────────────────┘
                     │ HTTP Request
                     ↓
┌─────────────────────────────────────────────────┐
│            Handler Layer (HTTP 处理层)            │
│  • 参数解析与校验 (ShouldBindJSON)                │
│  • 调用 Service 层                               │
│  • Handlers 聚合 struct (持有 Services 引用)      │
└────────────────────┬────────────────────────────┘
                     │
                     ↓
┌─────────────────────────────────────────────────┐
│            Service Layer (业务逻辑层)             │
│  • 按领域拆分 (post/community/user/vote)         │
│  • 依赖注入: 接收 Repository 接口                 │
│  • Services 聚合器统一管理                        │
└────────────────────┬────────────────────────────┘
                     │ 依赖接口 (repository.Interface)
                     ↓
┌─────────────────────────────────────────────────┐
│         Domain Layer (领域层 — 核心)              │
│  • Repository 接口定义 (mysql.go / cache.go)      │
│  • UnitOfWork 接口 (事务支持)                     │
│  • 领域级错误 (errors.go)                         │
└─────────────────────────────────────────────────┘
                     ↑ 实现接口
┌─────────────────────────────────────────────────┐
│           DAO Layer (数据访问层 — 实现)            │
│  • Repositories struct (实现 UnitOfWork)          │
│  • MySQL Repo structs (GORM)                     │
│  • Redis Cache structs (go-redis)                │
└────────────────────┬────────────────────────────┘
                     │
          ┌──────────┴──────────┐
          ↓                     ↓
    ┌──────────┐          ┌──────────┐
    │  MySQL   │          │  Redis   │
    └──────────┘          └──────────┘
```

### 依赖注入链路 (main.go)

```
config.Init() → mysql.Init() / redis.Init()
  → mysql.NewRepositories(db)           // UnitOfWork
  → redis.NewVoteCache() / NewUserTokenCache()  // Cache Repos
  → service.NewServices(uow, caches...)  // Services 聚合
  → handler.NewHandlers(services)        // Handlers 聚合
  → router.SetupRouter(handlers, tokenCache)
```

### 核心设计原则

- **依赖倒置原则 (DIP)**: Service 层依赖 `repository.Interface`，不依赖具体 DAO 实现
- **单一职责原则 (SRP)**: 每个 Service、Handler、Repo 只关注一个领域
- **开闭原则 (OCP)**: 切换存储层只需实现新的 Repository 接口

**示例流程 — 创建帖子:**

1. **Handler**: 解析 JSON 参数，从 JWT Context 获取 `userID`
2. **PostService**: 生成 Snowflake ID，调用 `PostRepository.CreatePost()` + `PostCacheRepository.CreatePost()`
3. **PostRepo (MySQL)**: 执行 GORM `Create()`
4. **VoteCache (Redis)**: 将帖子 ID 加入时间/分数 ZSet

---

## 📡 API 文档

### Swagger 文档

启动项目后访问: **[http://127.0.0.1:8080/swagger/index.html](http://127.0.0.1:8080/swagger/index.html)**

### 更新 Swagger 文档

修改代码后,需要重新生成 Swagger 文档:

```bash
# 安装 swag
go install github.com/swaggo/swag/cmd/swag@latest

# 生成文档
swag init

# 重启服务查看更新
```

### 主要 API 接口

#### 🔐 用户认证

| 接口 | 方法 | 路径 | 说明 | 需要认证 |
|------|------|------|------|---------|
| 用户注册 | POST | `/api/v1/signup` | 创建新用户 | ❌ |
| 用户登录 | POST | `/api/v1/login` | 获取 Access Token | ❌ |
| 刷新 Token | POST | `/api/v1/refresh_token` | 刷新 Access Token | ❌ |

#### 🏘️ 社区管理

| 接口 | 方法 | 路径 | 说明 | 需要认证 |
|------|------|------|------|---------|
| 获取社区列表 | GET | `/api/v1/community` | 获取所有社区 | ✅ |
| 获取社区详情 | GET | `/api/v1/community/:id` | 获取指定社区详情 | ✅ |

#### 📝 帖子管理

| 接口 | 方法 | 路径 | 说明 | 需要认证 |
|------|------|------|------|---------|
| 创建帖子 | POST | `/api/v1/post` | 发布新帖子 | ✅ |
| 获取帖子详情 | GET | `/api/v1/post/:id` | 获取指定帖子详情 | ✅ |
| 获取帖子列表 | GET | `/api/v1/posts2` | 分页获取帖子列表(支持排序和社区筛选) | ✅ |
| 删除帖子 | DELETE | `/api/v1/post/:id` | 软删除帖子(仅作者可删除) | ✅ |

#### 👍 投票功能

| 接口 | 方法 | 路径 | 说明 | 需要认证 |
|------|------|------|------|---------|
| 帖子投票 | POST | `/api/v1/vote` | 对帖子投赞成票/反对票/取消投票 | ✅ |

### 统一响应格式

所有接口都返回统一的 JSON 格式:

**成功响应:**
```json
{
  "code": 1000,
  "msg": "success",
  "data": { /* 实际数据 */ }
}
```

**错误响应:**
```json
{
  "code": 1007,
  "msg": "请求参数错误",
  "data": null
}
```

**常见错误码:**

| 错误码 | 说明 |
|-------|------|
| 1000 | 成功 |
| 1001 | 请求参数错误 |
| 1002 | 用户名已存在 |
| 1003 | 用户名不存在 |
| 1004 | 用户名或密码错误 |
| 1005 | 服务繁忙 |
| 1006 | 需要登录 |
| 1007 | 无效的Token |
| 1008 | 资源不存在 |
| 1009 | 投票时间已过 |
| 1010 | 不允许重复投票 |
| 1011 | 请求过于频繁 |
| 1012 | 无权限操作 |

---

## 🔧 开发指南

### 常用命令

```bash
# 开发环境运行 (热重载)
air

# 格式化代码
make gotool

# 编译项目 (Linux 静态编译)
make build

# 本地运行
make run

# 清理编译产物
make clean

# 生成 Swagger 文档
swag init

# 查看帮助
make help
```

### 开发流程

1. **创建新功能分支**
```bash
git checkout -b feature/your-feature
```

2. **编写代码** (遵循 DDD / Clean Architecture)
   - `internal/domain/repository/` — 定义 Repository 接口
   - `internal/dao/` — 实现 Repository 接口
   - `internal/service/` — 实现业务逻辑，注入 Repository 接口
   - `internal/handler/` — HTTP 处理，调用 Service
   - `internal/router/` — 注册路由

3. **更新 Swagger 文档**
```bash
swag init -g cmd/bluebell/main.go
```

4. **提交代码**
```bash
git add .
git commit -m "feat: your feature description"
git push origin feature/your-feature
```

### 添加新接口示例

**示例: 添加"收藏帖子"功能**

1. **定义 Repository 接口** (`internal/domain/repository/cache.go`)
```go
type FavoriteCacheRepository interface {
    AddFavorite(ctx context.Context, userID, postID int64) error
    RemoveFavorite(ctx context.Context, userID, postID int64) error
}
```

2. **实现 Repository** (`internal/dao/redis/favorite.go`)
```go
type FavoriteCache struct{}
func NewFavoriteCache() *FavoriteCache { return &FavoriteCache{} }
func (c *FavoriteCache) AddFavorite(ctx context.Context, userID, postID int64) error { ... }
```

3. **实现 Service** (`internal/service/favorite/favorite_service.go`)
```go
type FavoriteService struct { favoriteCache repository.FavoriteCacheRepository }
func NewFavoriteService(cache repository.FavoriteCacheRepository) *FavoriteService { ... }
```

4. **实现 Handler** (`internal/handler/favorite_handler.go`)
```go
func (h *Handlers) AddFavoriteHandler(c *gin.Context) { ... }
```

5. **注册路由** (`internal/router/router.go`)
```go
authGroup.POST("/favorite", h.AddFavoriteHandler)
```

---

## 📚 教学文档

项目包含完整的 **18 章教学文档**,涵盖从零开始构建整个项目的全过程。

### 文档目录

#### 📘 第一部分:基础篇 - 用户认证与JWT (1-12章)

| 章节 | 标题 | 核心知识点 |
|------|------|-----------|
| 01 | 用户表设计与数据建模 | MySQL表设计、字段类型、索引优化 |
| 02 | Snowflake算法生成分布式ID | 分布式ID、雪花算法原理 |
| 03 | 用户注册业务流程设计 | 三层架构(CLD)、业务流程 |
| 04 | 请求参数绑定与校验 | ShouldBindJSON、validator标签 |
| 05 | 优雅的参数校验与错误翻译 | 自定义校验规则、中文化 |
| 06 | 用户密码加密与数据持久化 | bcrypt、sqlx数据插入 |
| 07 | Zap日志系统集成与环境分离 | 结构化日志、Lumberjack轮转 |
| 08 | JWT认证与登录功能实现 | JWT原理、Token生成 |
| 09 | Refresh_Token_最佳实践 | Token刷新、安全性提升 |
| 10 | 单点登录与互踢模式 | 单设备登录、Redis存储Token |
| 11 | 集成Swagger自动生成文档 | Swaggo集成、注解编写 |
| 12 | 高效开发工具Makefile与Air | Makefile编译、Air热重载 |

#### 📗 第二部分:进阶篇 - 社区与帖子功能 (13-18章)

| 章节 | 标题 | 核心知识点 |
|------|------|-----------|
| 13 | 社区管理功能实现 | RESTful API设计、社区列表/详情 |
| 14 | 帖子发布功能实现 | 创建帖子、多表关联查询 |
| 15 | 帖子列表分页实现 | 分页原理、LIMIT OFFSET、N+1优化 |
| 16 | 帖子投票功能深度解析 | Reddit算法、投票规则、状态机 |
| 17 | Redis在帖子排序中的应用 | ZSet、Pipeline原子性、批量查询 |
| 18 | 按社区筛选帖子实现 | 统一接口设计、调度器模式 |

### 如何使用教学文档

1. **按顺序学习**: 从第 1 章开始,循序渐进
2. **动手实践**: 每章都要亲自编写代码
3. **理解原理**: 不要只记代码,要理解设计决策
4. **完成练习**: 每章末尾的练习题帮助巩固知识

**文档位置**: `教学文档/README.md`

---

## 🎯 核心技术亮点

### 1. JWT 认证机制

- **Access Token**: 短期有效 (2小时),用于 API 认证
- **Refresh Token**: 长期有效 (30天),用于刷新 Access Token
- **单点登录**: Redis 存储 Token,实现设备互踢

### 2. Reddit 投票算法

```
分数 = 帖子创建时间戳 + (赞成票数 - 反对票数) × 432
```

- **时间因素**: 新帖子天然获得更高初始分数
- **投票因素**: 每个赞成票增加 432 分 (约 1 票 = 12 小时时间优势)
- **防作弊**: 7天投票期限,超时无法投票

### 3. Redis 高性能优化

- **ZSet 排序**: 使用 `bluebell:post:score` 和 `bluebell:post:time` 两个有序集合
- **Pipeline 批量查询**: 减少网络往返次数
- **投票记录**: `bluebell:post:voted:{postID}` 存储每个用户的投票方向

### 4. N+1 问题优化

**问题**: 查询 100 个帖子,需要 1 + 100 + 100 = 201 次数据库查询

**优化**: 使用 GORM 预加载和批量查询,减少到 3 次

```go
// ❌ 低效: N+1 问题
for _, post := range posts {
    user := GetUserByID(post.AuthorID)     // N 次
    community := GetCommunityByID(post.CommunityID)  // N 次
}

// ✅ 优化: 批量查询 (GORM)
userIDs := collectUserIDs(posts)
var users []User
db.Where("user_id IN ?", userIDs).Find(&users)  // 1 次
```

---

## 🐛 常见问题

### 1. 数据库连接失败

**错误信息**: `dial tcp 127.0.0.1:3306: connect: connection refused`

**解决方案**:
```bash
# 检查 MySQL 是否启动
mysql -u root -p

# macOS
brew services start mysql

# Linux
sudo systemctl start mysql
```

### 2. Redis 连接失败

**错误信息**: `dial tcp 127.0.0.1:6379: connect: connection refused`

**解决方案**:
```bash
# 检查 Redis 是否启动
redis-cli ping

# macOS
brew services start redis

# Linux
sudo systemctl start redis
```

### 3. Swagger 文档不显示

**解决方案**:
```bash
# 重新生成文档
swag init

# 确保导入了 docs 包
# main.go 第 7 行: _ "bluebell/docs"

# 重启服务
```

### 4. JWT Token 过期

**解决方案**:
```bash
# 使用 refresh_token 刷新
curl -X POST http://127.0.0.1:8080/api/v1/refresh_token \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "your_refresh_token"}'
```

### 5. Air 热重载不生效

**解决方案**:
```bash
# 检查 .air.conf 配置
# 确保监控了正确的文件扩展名
include_ext = ["go", "yaml"]

# 重启 Air
pkill air
air
```

---

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request!

### 提交 Issue

- **Bug 报告**: 详细描述问题、复现步骤、环境信息
- **功能建议**: 说明需求场景和期望效果

### 提交 Pull Request

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交变更 (`git commit -m 'feat: Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 创建 Pull Request

### Commit Message 规范

遵循 [Conventional Commits](https://www.conventionalcommits.org/) 规范:

- `feat`: 新功能
- `fix`: 修复 Bug
- `docs`: 文档更新
- `style`: 代码格式调整
- `refactor`: 重构代码
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建/工具链更新

**示例:**
```
feat: 实现帖子删除功能
fix: 修复投票重复计数问题
docs: 更新 API 文档
```

---

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

---

## 🙏 致谢

- [Gin Web Framework](https://github.com/gin-gonic/gin) - 高性能 HTTP 框架
- [GORM](https://gorm.io) - 强大的 Go ORM 库
- [Zap](https://github.com/uber-go/zap) - 高性能日志库
- [Viper](https://github.com/spf13/viper) - 配置管理
- [Swagger](https://swagger.io) - API 文档工具

---

## 📮 联系方式

- **项目仓库**: [GitHub](https://github.com/yourusername/bluebell)
- **问题反馈**: [Issues](https://github.com/yourusername/bluebell/issues)
- **教学文档**: 见 `教学文档/` 目录

---

<div align="center">

**⭐ 如果这个项目对你有帮助,请给一个 Star ⭐**

**Made with ❤️ by Go Developers**

[⬆ 回到顶部](#bluebell-社区论坛)

</div>
