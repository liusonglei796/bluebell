# Restore Why-Focused Comments Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Re-apply descriptive comments explaining architectural rationale to 7 backend files across Domain, Application, Interface, and Infrastructure layers.

**Architecture:** Surgical edits via `replace` to add comments that follow Go documentation standards.

**Tech Stack:** Go.

---

### Task 1: Domain Layer Entities

**Files:**
- Modify: `internal/domain/entity/user.go`
- Modify: `internal/domain/entity/post.go`

- [ ] **Step 1: Add comments to `internal/domain/entity/user.go`**

```go
<<<<
package entity

import "golang.org/x/crypto/bcrypt"
====
// Package entity 定义领域实体
//
// 领域层是 DDD 的核心，包含了最稳定的业务规则。
// 这些规则（如加密成本、角色定义）不随技术栈或外部框架的改变而改变。
package entity

import "golang.org/x/crypto/bcrypt"
>>>>
<<<<
// IsAdmin 判断用户是否为管理员
func (u *User) IsAdmin() bool {
====
// IsAdmin 判断用户是否为管理员
// 这是一个领域规则：管理员权限的判定逻辑被封装在实体内部，与外部鉴权框架解耦。
func (u *User) IsAdmin() bool {
>>>>
<<<<
// HashPassword 对明文密码进行 bcrypt 加密，返回密文
// 这是领域层的核心业务逻辑：密码策略由领域层决定，不依赖 ORM 钩子
func HashPassword(raw string) (string, error) {
====
// HashPassword 对明文密码进行 bcrypt 加密，返回密文
// 这是领域层的核心业务逻辑：密码策略（加密算法、成本）由领域规则决定。
// 将其放在领域层可确保：无论用户是通过 API 注册还是后台导入，其密码安全策略都是强制一致的。
func HashPassword(raw string) (string, error) {
>>>>
```

- [ ] **Step 2: Add comments to `internal/domain/entity/post.go`**

```go
<<<<
package entity

import (
	"strings"
	"time"
)

// 帖子状态常量
const (
	PostStatusPublished = 1  // 已发布
	PostStatusDeleted   = 0  // 已删除（软删除）
)
====
package entity

import (
	"strings"
	"time"
)

// 帖子状态常量
// 状态定义在领域层，因为状态流转（如发布、删除）是核心业务逻辑，
// 它决定了帖子在系统生命周期中的行为，不应受数据库存储方式的影响。
const (
	PostStatusPublished = 1  // 已发布
	PostStatusDeleted   = 0  // 已删除（软删除）
)
>>>>
<<<<
// Validate 校验帖子内容是否合法
func (p *Post) Validate() error {
====
// Validate 校验帖子内容是否合法
// 核心业务约束：帖子标题和内容不能为空。
// 在领域层进行校验可确保这些“绝对真理”在任何业务场景下都被强制执行。
func (p *Post) Validate() error {
>>>>
<<<<
// CanBeDeletedBy 校验指定用户是否有权删除此帖子
// 核心业务规则：只有帖子的作者才能删除自己的帖子
func (p *Post) CanBeDeletedBy(userID int64) error {
====
// CanBeDeletedBy 校验指定用户是否有权删除此帖子
// 核心业务权限：只有帖子的作者才能删除自己的帖子。
// 将权限判断下沉到领域实体，可防止权限逻辑在多个 Service 中被重复实现或遗漏。
func (p *Post) CanBeDeletedBy(userID int64) error {
>>>>
```

- [ ] **Step 3: Commit Domain Entity changes**

```bash
git add internal/domain/entity/user.go internal/domain/entity/post.go
git commit -m "docs: add why-focused comments to domain entities"
```

### Task 2: Domain Layer Interfaces

**Files:**
- Modify: `internal/domain/repository.go`

- [ ] **Step 1: Add comments to `internal/domain/repository.go`**

```go
<<<<
// Package domain 提供领域层仓储接口定义
//
// DDD: 仓储接口属于领域层，实现属于基础设施层
// 通过依赖倒置原则，领域层定义接口，基础设施层提供实现
package domain
====
// Package domain 提供领域层仓储接口定义
//
// 依赖倒置原则 (DIP):
// 领域层定义了业务运行所需的“能力接口”（Repositories），而具体的实现（MySQL/Redis/ES）
// 放在基础设施层。这样领域层就不再依赖具体的技术细节，实现了业务逻辑的高度解耦。
package domain
>>>>
<<<<
// PostCacheRepository 帖子缓存仓储接口（Redis）
type PostCacheRepository interface {
====
// PostCacheRepository 帖子缓存仓储接口（Redis）
// 虽然缓存通常被视为技术细节，但在高性能社交系统中，
// “如何缓存及如何维护排序数据”本身就是一种关键的业务支撑需求。
type PostCacheRepository interface {
>>>>
<<<<
// UserTokenCacheRepository 用户 Token 缓存仓储接口（Redis）
type UserTokenCacheRepository interface {
====
// UserTokenCacheRepository 用户 Token 缓存仓储接口（Redis）
// 将 Token 存储抽象为领域需求，是因为 Token 的生命周期和安全性是用户管理的业务边界。
type UserTokenCacheRepository interface {
>>>>
```

- [ ] **Step 2: Commit Domain Repository changes**

```bash
git add internal/domain/repository.go
git commit -m "docs: add why-focused comments to domain repositories"
```

### Task 3: Application Layer Services

**Files:**
- Modify: `internal/application/user/user_service.go`
- Modify: `internal/application/post/post_service.go`

- [ ] **Step 1: Add comments to `internal/application/user/user_service.go`**

```go
<<<<
package usersvc
====
// Package usersvc 实现用户应用服务
//
// 应用层（Application Layer）充当“指挥官”角色：
// 它不包含核心业务规则，而是协调领域实体、仓储接口和外部基础设施来完成一个完整的用例。
package usersvc
>>>>
<<<<
// userServiceStruct 用户业务逻辑服务
type userServiceStruct struct {
====
// userServiceStruct 用户业务逻辑服务
// 它持有多个仓储接口，用于跨越不同数据源（MySQL、Redis）协调用户信息。
type userServiceStruct struct {
>>>>
<<<<
// SocialLogin 处理社交账号登录 (如 GitHub)
func (s *userServiceStruct) SocialLogin(ctx context.Context, githubID, username, email, avatarURL string) (string, string, error) {
====
// SocialLogin 处理社交账号登录 (如 GitHub)
// 此方法体现了应用层的“编排”作用：它首先查询社交 Profile，
// 如果不存在则协调 UserRepo 创建新账号，最后通过 JWT 基础设施生成 Token 并存入缓存。
func (s *userServiceStruct) SocialLogin(ctx context.Context, githubID, username, email, avatarURL string) (string, string, error) {
>>>>
<<<<
// SignUp 处理用户注册业务逻辑
func (s *userServiceStruct) SignUp(ctx context.Context, p *userreq.SignUpRequest) (err error) {
====
// SignUp 处理用户注册业务逻辑
// 应用层在这里执行了：校验重名 -> 调用领域规则加密密码 -> 通过 Repo 持久化。
func (s *userServiceStruct) SignUp(ctx context.Context, p *userreq.SignUpRequest) (err error) {
>>>>
```

- [ ] **Step 2: Add comments to `internal/application/post/post_service.go`**

```go
<<<<
package postsvc
====
// Package postsvc 实现帖子应用服务
//
// 帖子服务是系统中复杂度最高的应用服务之一。
// 它编排了数据库（MySQL）、缓存（Redis）、搜索引擎（ES）和消息队列（MQ）
// 来确保帖子的创建、查询、投票等高并发场景下的数据最终一致性。
package postsvc
>>>>
<<<<
// postServiceStruct 帖子业务逻辑服务
type postServiceStruct struct {
====
// postServiceStruct 帖子业务逻辑服务
// 持有多种基础设施客户端，以支持复杂用例（如投票时的多级缓存与异步持久化）。
type postServiceStruct struct {
>>>>
```

- [ ] **Step 3: Commit Application Service changes**

```bash
git add internal/application/user/user_service.go internal/application/post/post_service.go
git commit -m "docs: add why-focused comments to application services"
```

### Task 4: Interface and Infrastructure Layers

**Files:**
- Modify: `internal/interfaces/http/router/router.go`
- Modify: `internal/infrastructure/persistence/mysql/userdb/user.go`

- [ ] **Step 1: Add comments to `internal/interfaces/http/router/router.go`**

```go
<<<<
package router
====
// Package router 负责 HTTP 路由与中间件的编排
//
// 接口层（Interface Layer）负责处理外部通信协议。
// 通过在此处定义路由并注入 Handlers，我们确保了 HTTP/Gin 的具体实现
// 不会渗透到 Application 和 Domain 层。
package router
>>>>
<<<<
// NewRouter 初始化路由配置
// 接收 Provider（DI 容器）作为参数
func NewRouter(
====
// NewRouter 初始化路由配置
// 使用依赖注入（Dependency Injection）模式将 Handler Provider 注入路由。
// 这样做的好处是路由定义清晰，且易于在单元测试中 Mock 掉底层的业务逻辑。
func NewRouter(
>>>>
```

- [ ] **Step 2: Add comments to `internal/infrastructure/persistence/mysql/userdb/user.go`**

```go
<<<<
package userdb
====
// Package userdb 用户数据库访问的具体实现
//
// 基础设施层（Infrastructure Layer）负责与具体的技术细节（GORM、MySQL）打交道。
// 它实现了领域层定义的 UserRepository 接口，将底层的 CRUD 操作细节对上层隐藏。
package userdb
>>>>
<<<<
// fromModelUser 将数据库模型转换为领域实体
func fromModelUser(m *model.User) *entity.User {
====
// fromModelUser 将数据库模型转换为领域实体
// 转换的必要性：为了保持领域层的纯净，不让 GORM 相关的 tag 或结构体定义侵入业务逻辑。
// 即使数据库结构发生变化，只要转换逻辑更新，领域层代码就无需改动。
func fromModelUser(m *model.User) *entity.User {
>>>>
```

- [ ] **Step 3: Commit Interface and Infrastructure changes**

```bash
git add internal/interfaces/http/router/router.go internal/infrastructure/persistence/mysql/userdb/user.go
git commit -m "docs: add why-focused comments to interface and infrastructure layers"
```
