# DI 架构完整指南

## 1. 分层架构

```
┌─────────────────────────────────────────────────────┐
│                  路由层 (Router)                      │
│            - 处理 HTTP 请求路由配置                  │
│            - 中间件装配                              │
└──────────────────┬──────────────────────────────────┘
                   │ depends on
┌──────────────────▼──────────────────────────────────┐
│            表现层 (Handler)                          │
│  ┌─────────────────────────────────────────────┐   │
│  │      HandlerProvider (DI 容器)              │   │
│  │  ┌──────────────────────────────────────┐  │   │
│  │  │ UserHandler   (依赖 UserService)     │  │   │
│  │  │ PostHandler   (依赖 PostService)     │  │   │
│  │  │ CommunityHandler (依赖 Community)    │  │   │
│  │  │ VoteHandler   (依赖 VoteService)     │  │   │
│  │  └──────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────┘   │
└──────────────────┬──────────────────────────────────┘
                   │ depends on (接口)
┌──────────────────▼──────────────────────────────────┐
│         业务逻辑层 (Service 接口)                    │
│  ┌─────────────────────────────────────────────┐   │
│  │ UserService   (定义在 domain/service)      │   │
│  │ PostService   (定义在 domain/service)      │   │
│  │ CommunityService (定义在 domain/service)   │   │
│  │ VoteService   (定义在 domain/service)      │   │
│  └─────────────────────────────────────────────┘   │
└──────────────────┬──────────────────────────────────┘
                   │ implements
┌──────────────────▼──────────────────────────────────┐
│         业务逻辑层 (Service 实现)                    │
│  ┌─────────────────────────────────────────────┐   │
│  │ service/user/UserService                    │   │
│  │ service/post/PostService                    │   │
│  │ service/community/CommunityService          │   │
│  │ service/vote/VoteService                    │   │
│  └─────────────────────────────────────────────┘   │
└──────────────────┬──────────────────────────────────┘
                   │ depends on (接口)
┌──────────────────▼──────────────────────────────────┐
│       数据访问层 (Repository 接口)                   │
│  ┌─────────────────────────────────────────────┐   │
│  │ UserRepository  (定义在 domain/repository)  │   │
│  │ PostRepository  (定义在 domain/repository)  │   │
│  │ CommunityRepository (定义在 domain/...)     │   │
│  │ VoteRepository  (定义在 domain/repository)  │   │
│  └─────────────────────────────────────────────┘   │
└──────────────────┬──────────────────────────────────┘
                   │ implements
┌──────────────────▼──────────────────────────────────┐
│       数据访问层 (Repository 实现)                   │
│  ┌─────────────────────────────────────────────┐   │
│  │ dao/mysql/UserRepository                    │   │
│  │ dao/mysql/PostRepository                    │   │
│  │ dao/redis/VoteCache                         │   │
│  │ dao/redis/PostCache                         │   │
│  └─────────────────────────────────────────────┘   │
└──────────────────┬──────────────────────────────────┘
                   │
┌──────────────────▼──────────────────────────────────┐
│    基础设施层 (MySQL、Redis 等)                      │
└─────────────────────────────────────────────────────┘
```

## 2. DI 流程详解

### Step 1: 创建底层依赖（Repository 层）

```go
// 基础设施：数据库连接
gormDB, err := mysql.Init(cfg)
redisClient := redis.Init(cfg)

// 创建 Repository 实例
repositoriesUOW := mysql.NewRepositories(gormDB)  // 包含 UserRepo、PostRepo 等
voteCache := redis.NewVoteCache()
tokenCache := redis.NewUserTokenCache()
```

### Step 2: 创建 Service 层

```go
// Service 层依赖 Repository 接口
services := service.NewServices(
    repositoriesUOW,     // UnitOfWork（包含所有 Repository）
    voteCache,           // VoteCacheRepository
    voteCache,           // PostCacheRepository
    tokenCache,          // UserTokenCacheRepository
    cfg,                 // 配置
)

// 结果：services.User、services.Post 等都是 Service 接口的实现
```

### Step 3: 创建 Handler 层（DI 注入 Service 接口）

```go
// 选项 A：自动装配（推荐简单场景）
handlerProvider := handler.NewHandlers(services)

// 选项 B：手动装配（推荐可控场景）
handlerProvider := handler.NewHandlerProvider(
    services.User,      // UserService 接口实现
    services.Post,      // PostService 接口实现
    services.Community, // CommunityService 接口实现
    services.Vote,      // VoteService 接口实现
)
```

### Step 4: 注册路由

```go
r, err := router.NewRouter(cfg.App.Mode, handlerProvider, cfg)
```

### Step 5: 启动服务

```go
http_server.Run(r, cfg.App.Port)
```

## 3. 依赖关系图

### 表现层（Handler）依赖

```
UserHandler
    ↓
    └─→ UserService 接口
            ↓
            └─→ UserRepository 接口
                    ↓
                    └─→ MySQL

PostHandler
    ↓
    └─→ PostService 接口
            ↓
            ├─→ PostRepository 接口
            │       └─→ MySQL
            ├─→ PostCacheRepository 接口
            │       └─→ Redis
            └─→ VoteCacheRepository 接口
                    └─→ Redis

CommunityHandler
    ↓
    └─→ CommunityService 接口
            ↓
            └─→ CommunityRepository 接口
                    └─→ MySQL

VoteHandler
    ↓
    └─→ VoteService 接口
            ↓
            ├─→ PostRepository 接口
            │       └─→ MySQL
            └─→ VoteCacheRepository 接口
                    └─→ Redis
```

## 4. 关键设计决策

### 为什么要分离 Handler 结构体？

**问题**：原来 Handlers 结构体持有整个 Services，容易导致：
- Handler 中无法准确知道自己依赖了哪些 Service
- 不必要的依赖暴露
- 难以进行单元测试

**解决**：
```go
// 原来：不清楚依赖关系
type Handlers struct {
    Services *service.Services
}

// 改后：清晰的依赖关系
type UserHandler struct {
    userService domain.service.UserService
}
```

### 为什么要使用 HandlerProvider？

**问题**：如果没有容器，Handler 分散在各个包，难以管理

**解决**：
```go
type HandlerProvider struct {
    UserHandler      *UserHandler
    PostHandler      *PostHandler
    CommunityHandler *CommunityHandler
    VoteHandler      *VoteHandler
}
```

### 为什么要依赖接口而非实现？

**优势**：
1. **解耦**：Handler 不知道 Service 的具体实现
2. **可测试**：可以注入 Mock 接口进行单元测试
3. **可扩展**：可以轻松切换 Service 实现（如从 MySQL 切到 MongoDB）
4. **符合 SOLID**：遵循依赖倒置原则（DIP）

```go
// 错误的方式（依赖实现）
type UserHandler struct {
    userService *service.UserService  // 具体实现
}

// 正确的方式（依赖接口）
type UserHandler struct {
    userService domain.service.UserService  // 接口
}
```

## 5. 测试示例

### 单元测试

```go
package handler

import (
    "testing"
    "bluebell/internal/dto/request"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/assert"
)

// Mock Service
type MockUserService struct {
    mock.Mock
}

func (m *MockUserService) SignUp(ctx context.Context, p *request.SignUpRequest) error {
    args := m.Called(ctx, p)
    return args.Error(0)
}

// 测试
func TestUserHandler_SignUp_Success(t *testing.T) {
    // Arrange: 创建 Mock
    mockService := new(MockUserService)
    mockService.On("SignUp", mock.Anything, mock.Anything).Return(nil)
    
    // 通过构造函数注入 Mock
    handler := NewUserHandler(mockService)
    
    // Act & Assert
    mockService.AssertCalled(t, "SignUp", mock.Anything, mock.Anything)
}

func TestUserHandler_SignUp_UserExists(t *testing.T) {
    // Arrange
    mockService := new(MockUserService)
    mockService.On("SignUp", mock.Anything, mock.Anything).
        Return(errorx.ErrUserExist)
    
    handler := NewUserHandler(mockService)
    
    // Act & Assert
    mockService.AssertCalled(t, "SignUp", mock.Anything, mock.Anything)
}
```

## 6. 配置文件示例

在实际项目中，可以考虑使用配置文件管理 DI（更高级）：

```yaml
# di.yaml
di:
  handlers:
    - name: user_handler
      service: user_service
    - name: post_handler
      service: post_service
    - name: community_handler
      service: community_service
    - name: vote_handler
      service: vote_service
```

然后用代码读取配置自动装配：

```go
// 高级用法（选择性实现）
handlerProvider, err := handler.NewHandlerProviderFromConfig(
    services,
    "di.yaml",
)
```

## 7. 快速参考

### 常用命令

```bash
# 编译项目
go build -o bin/bluebell.exe ./cmd/bluebell

# 运行项目
./bin/bluebell.exe -conf config.yaml

# 运行单元测试
go test ./... -v

# 生成覆盖率报告
go test ./... -cover
```

### 关键文件位置

```
bluebell/
├── cmd/bluebell/main.go          ← 完整的 DI 流程示例
├── internal/
│   ├── handler/
│   │   ├── handler.go            ← HandlerProvider（DI 容器）
│   │   ├── factory.go            ← 工厂方法
│   │   ├── user_handler.go       ← UserHandler 实现
│   │   ├── post_handler.go       ← PostHandler 实现
│   │   ├── community_handler.go  ← CommunityHandler 实现
│   │   └── vote_handler.go       ← VoteHandler 实现
│   ├── domain/service/           ← Service 接口定义
│   ├── service/                  ← Service 实现
│   ├── domain/repository/        ← Repository 接口定义
│   ├── dao/                      ← Repository 实现
│   └── router/router.go          ← 路由层
```

---

**这是一个完整的 DI 架构指南。通过这种设计，代码更易测试、更易维护、更易扩展。**
