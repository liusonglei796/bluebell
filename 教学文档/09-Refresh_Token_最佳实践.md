# 第09章:Refresh Token 最佳实践

> **本章导读**
>
> 在上一章中,我们实现了基础的 JWT 认证。但在实际生产环境中,Access Token 的有效期通常很短(如 10 分钟)以降低安全风险。
>
> 带来的问题是:用户每 10 分钟就需要重新登录一次,体验极差。
>
> 本章将深入探讨 **Refresh Token(刷新令牌)** 机制,实现"无感续签",平衡安全性与用户体验,并解决 Refresh Token 面临的各种安全挑战。

---

## 📚 本章目标

学完本章,你将掌握:

1. 理解 Access Token 与 Refresh Token 的双令牌架构设计原理
2. 掌握 Refresh Token 的安全最佳实践(轮转、绑定、黑名单)
3. 实现完整的 Token 刷新业务逻辑
4. 编写安全的刷新接口并处理各种异常场景
5. 客户端如何配合实现"无感刷新"
6. 防御 Refresh Token 被盗用的攻击场景

---

## 1. 双令牌机制深度剖析

### 1.1 为什么需要两个 Token?

在回答这个问题前,我们先看看只用一个 Token 会遇到什么问题:

#### 方案一:长有效期 Access Token (❌ 不推荐)

```
Access Token 有效期: 30 天
```

**优点:**
- ✅ 用户体验好,30 天内无需重新登录

**致命缺点:**
- ❌ **安全风险极高**: Token 泄露后,攻击者可以在 30 天内冒充用户
- ❌ **无法撤销**: Token 一旦签发,在过期前无法失效(即使用户修改了密码)
- ❌ **合规性差**: 金融、医疗等行业的安全规范不允许如此长的有效期

#### 方案二:短有效期 Access Token (❌ 体验差)

```
Access Token 有效期: 5 分钟
```

**优点:**
- ✅ 安全性高,Token 泄露后影响时间短

**致命缺点:**
- ❌ **用户体验极差**: 用户每 5 分钟就要重新登录一次
- ❌ **服务器压力大**: 频繁登录导致认证接口负载激增

#### 方案三:双 Token 机制 (✅ 推荐)

```
Access Token 有效期: 10 分钟
Refresh Token 有效期: 30 天
```

**核心思想:**

- **Access Token 短效高频**: 用于日常接口认证,泄露后影响有限。
- **Refresh Token 长效低频**: 仅用于换取新的 Access Token,降低泄露概率。

| Token 类型 | 有效期 | 使用频率 | 作用 | 存储位置 | 泄露风险 |
|-----------|-------|---------|------|---------|---------|
| **Access Token** | 短 (10分钟) | 极高 (每个请求) | 访问业务接口 | 内存/Header | 低 (很快过期) |
| **Refresh Token** | 长 (30天) | 极低 (10分钟一次) | **仅**换取新 Access Token | 本地存储/Cookie | 中 (可撤销) |

**安全性分析:**

```
攻击者窃取 Access Token:
  ↓
影响时间: 最多 10 分钟
  ↓
10 分钟后 Token 自动失效
  ↓
攻击者需要 Refresh Token 才能继续攻击
  ↓
但 Refresh Token 使用频率低,被窃取概率更小
```

### 1.2 完整交互流程

**正常流程:**

```
┌─────────┐                 ┌─────────┐                 ┌─────────┐
│ 客户端  │                 │ 服务器  │                 │  Redis  │
└────┬────┘                 └────┬────┘                 └────┬────┘
     │                           │                           │
     │ 1. POST /login            │                           │
     ├──────────────────────────>│                           │
     │   {username, password}    │                           │
     │                           │ 2. 验证用户               │
     │                           │                           │
     │                           │ 3. 生成双 Token           │
     │                           ├──────────────────────────>│
     │                           │   存储 Token              │
     │<──────────────────────────┤                           │
     │ {access_token, refresh_token}                         │
     │                           │                           │
     │ 4. GET /api/posts         │                           │
     ├──────────────────────────>│                           │
     │   Header: Bearer {access_token}                       │
     │                           │ 5. 验证 Token             │
     │                           ├──────────────────────────>│
     │                           │   比对 Redis              │
     │<──────────────────────────┤                           │
     │   {posts: [...]}          │                           │
     │                           │                           │
     │ ... 8 分钟后 ...          │                           │
     │                           │                           │
     │ 6. GET /api/posts         │                           │
     ├──────────────────────────>│                           │
     │   Header: Bearer {access_token}                       │
     │                           │ 7. Token 过期!            │
     │<──────────────────────────┤                           │
     │   401 Token Expired       │                           │
     │                           │                           │
     │ 8. POST /refresh_token    │                           │
     ├──────────────────────────>│                           │
     │   {refresh_token}         │                           │
     │                           │ 9. 验证 Refresh Token     │
     │                           │                           │
     │                           │ 10. 查询用户状态          │
     │                           │    (是否被封禁/密码是否变更)
     │                           │                           │
     │                           │ 11. 生成新双 Token        │
     │                           ├──────────────────────────>│
     │                           │   更新 Redis              │
     │<──────────────────────────┤                           │
     │ {new_access_token, new_refresh_token}                 │
     │                           │                           │
     │ 12. 重试 GET /api/posts   │                           │
     ├──────────────────────────>│                           │
     │   Header: Bearer {new_access_token}                   │
     │<──────────────────────────┤                           │
     │   {posts: [...]}          │                           │
```

**关键点:**

1. **步骤 7**: 前端收到 `401 Token Expired` 错误。
2. **步骤 8**: 前端**自动**调用刷新接口,用户无感知。
3. **步骤 10**: **重点**! 服务器查询数据库,确保用户状态正常。
4. **步骤 11**: 生成**新的双 Token**,旧的全部作废(轮转机制)。
5. **步骤 12**: 前端使用新 Token **自动重试**原请求。

**对用户而言,这一过程是完全透明的。**

### 1.3 为什么 Refresh Token 也要轮转?

**场景: Refresh Token 被盗用**

假设攻击者窃取了用户的 Refresh Token:

#### ❌ 不轮转的情况 (危险)

```
用户 A 在第 10 分钟刷新 Token:
  ↓
服务器返回新 Access Token, Refresh Token 不变
  ↓
攻击者在第 20 分钟用同一个 Refresh Token 刷新:
  ↓
服务器依然返回新 Token (因为 Refresh Token 未变)
  ↓
攻击者可以持续使用偷来的 Refresh Token 直到 30 天后过期!
```

#### ✅ 轮转的情况 (安全)

```
用户 A 在第 10 分钟刷新 Token:
  ↓
服务器返回新 Access Token + 新 Refresh Token
  ↓
旧 Refresh Token 立即作废
  ↓
攻击者在第 20 分钟用旧 Refresh Token 刷新:
  ↓
服务器检测到 Refresh Token 已被替换 → 拒绝!
  ↓
同时触发安全警报,冻结该账号
```

**轮转的核心价值:**

- ✅ **限制攻击窗口**: 即使 Refresh Token 被盗,攻击者只能在下次合法刷新前使用。
- ✅ **检测异常行为**: 如果旧 Token 被使用,说明有人在非法访问,立即触发告警。
- ✅ **自动修复**: 合法用户下次刷新时,会拿到新 Token,自动"夺回"控制权。

---

## 2. 代码实现:JWT 工具包改造

我们已经在第08章中实现了双 Token 生成,本章重点是**验证 Refresh Token 并重新查询用户信息**。

### 2.1 为什么要重新查询数据库?

**错误做法 (❌ 不安全):**

```go
// ❌ 直接从 Refresh Token 的 Claims 中提取用户信息
func RefreshToken(rToken string) (newAToken, newRToken string, err error) {
    claims, _ := jwt.ParseToken(rToken)  // 直接解析
    userID := claims.UserID
    username := claims.Username

    // 直接用旧数据生成新 Token
    return jwt.GenToken(userID, username)
}
```

**问题:**

1. **用户修改了用户名**: 新 Token 中仍是旧用户名。
2. **用户被管理员封禁**: 新 Token 仍能正常使用。
3. **用户修改了密码**: 旧 Token 理应失效,但刷新后又能用。

**正确做法 (✅ 安全):**

```go
// ✅ 验证 Refresh Token 后,从数据库重新查询用户最新信息
func ValidateRefreshToken(rTokenString string) (*models.User, error) {
    // 1. 解析 Token,验证签名和过期时间
    claims := new(jwt.RegisteredClaims)
    token, err := jwt.ParseWithClaims(rTokenString, claims, func(t *jwt.Token) (interface{}, error) {
        return MySecret, nil
    })

    if err != nil || !token.Valid {
        return nil, errors.New("refresh token 无效")
    }

    // 2. 从 Claims 中提取 UserID
    userID, err := strconv.ParseInt(claims.Subject, 10, 64)
    if err != nil {
        return nil, errors.New("token 数据异常")
    }

    // 3. 【关键】查询数据库,获取用户最新信息
    // 这一步会确保:
    //   - 用户未被删除
    //   - 用户未被封禁 (如果有 status 字段)
    //   - 用户名等信息是最新的
    user, err := mysql.GetUserByID(userID)
    if err != nil {
        return nil, errors.New("用户不存在")
    }

    // 4. (可选) 检查用户状态
    // if user.Status == "banned" {
    //     return nil, errors.New("账号已被封禁")
    // }

    return user, nil
}
```

**对比:**

| 方案 | 用户改名后 | 用户被封禁后 | 用户改密后 | 安全性 |
|------|----------|------------|----------|--------|
| ❌ 直接解析 Token | 旧用户名生效 | 仍可刷新 | 仍可刷新 | 低 |
| ✅ 查询数据库 | 新用户名生效 | 拒绝刷新 | 拒绝刷新 (配合 Redis) | 高 |

### 2.2 完整的 ValidateRefreshToken 实现

在 `pkg/jwt/jwt.go` 中:

```go
package jwt

import (
	"bluebell/dao/mysql"
	"bluebell/models"
	"errors"
	"strconv"

	"github.com/golang-jwt/jwt/v5"
)

// ValidateRefreshToken 验证刷新令牌,并返回用户信息
// 为什么返回 *models.User: 刷新时需要最新的用户信息来生成新 Token
func ValidateRefreshToken(rTokenString string) (user *models.User, err error) {
	// 1. 解析 Token
	// 注意: Refresh Token 使用的是 jwt.RegisteredClaims,不包含自定义字段
	claims := new(jwt.RegisteredClaims)
	token, err := jwt.ParseWithClaims(rTokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return MySecret, nil
	})

	if err != nil || !token.Valid {
		return user, errors.New("refresh token 无效")
	}

	// 2. 从 Subject 字段提取 UserID
	// 为什么用 Subject: 这是 JWT 标准中存储"主体"的字段,我们在生成 Token 时将 UserID 放在这里
	userID := claims.Subject
	bizUserID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return user, errors.New("token数据异常")
	}

	// 3. 【核心】从数据库查询用户最新信息
	// 为什么: 确保用户状态正常,未被删除或封禁
	user, err = mysql.GetUserByID(bizUserID)
	if err != nil {
		return user, errors.New("用户不存在")
	}

	// 4. (可选) 额外的业务校验
	// if user.Status == "banned" {
	//     return nil, errors.New("账号已被封禁")
	// }
	// if user.ForceLogout {
	//     return nil, errors.New("账号已被强制下线")
	// }

	return user, nil
}
```

**设计要点:**

1. **Refresh Token 不包含业务数据**: 只有标准 Claims (Subject, ExpiresAt, Issuer)。
2. **每次刷新都查库**: 虽然增加了数据库查询,但刷新频率低(10分钟一次),可接受。
3. **返回完整 User 对象**: 包含最新的 UserID 和 Username,用于生成新 Token。

---

## 3. 业务逻辑层 (Logic)

在 `logic/user.go` 中新增刷新逻辑。

```go
package logic

import (
	"bluebell/dao/redis"
	"bluebell/pkg/jwt"
)

// RefreshToken 刷新 Token
// 参数:
//   aToken: 旧的 Access Token (可用于日志记录或级联撤销)
//   rToken: Refresh Token
// 返回:
//   newAToken: 新的 Access Token
//   newRToken: 新的 Refresh Token (轮转机制)
//   err: 错误信息
func RefreshToken(aToken, rToken string) (newAToken, newRToken string, err error) {
	// 1. 验证 Refresh Token 并获取用户信息
	// 为什么: ValidateRefreshToken 内部会:
	//   - 验证 Token 签名和过期时间
	//   - 查询数据库,确保用户状态正常
	//   - 返回最新的用户信息
	user, err := jwt.ValidateRefreshToken(rToken)
	if err != nil {
		return "", "", err
	}

	// 2. 【关键】从数据库重新查询用户最新信息
	// 为什么: 确保使用的是最新的用户名、权限等状态
	// 注意: ValidateRefreshToken 已经查询过一次,这里是再次确认
	// (在实际项目中,这一步可以省略,因为 ValidateRefreshToken 已经返回了最新信息)

	// 3. 使用最新用户信息生成新 Token
	// 为什么要生成新的 Refresh Token (轮转):
	//   - 旧 Refresh Token 立即作废,限制攻击窗口
	//   - 如果旧 Token 被盗,下次合法用户刷新时会生成新 Token,自动"夺回"控制权
	newAToken, newRToken, err = jwt.GenToken(user.UserID, user.Username)
	if err != nil {
		return "", "", err
	}

	// 4. 更新 Redis 中的 Token
	// 为什么: 实现单点登录 (SSO) 和互踢功能
	//   - 新 Token 覆盖 Redis 中的旧 Token
	//   - 旧 Token 即使未过期,也会在中间件中被拒绝 (与 Redis 不匹配)
	err = redis.SetUserToken(user.UserID, newAToken, newRToken, jwt.AccessTokenExpireDuration, jwt.RefreshTokenExpireDuration)
	if err != nil {
		return "", "", err
	}

	return newAToken, newRToken, nil
}
```

**逻辑流程图:**

```
客户端请求刷新
    ↓
Logic.RefreshToken(aToken, rToken)
    ├─→ 1. jwt.ValidateRefreshToken(rToken)
    │       ├─ 解析 Token,验证签名
    │       ├─ 检查是否过期
    │       ├─ 提取 UserID
    │       └─ mysql.GetUserByID() ── 查询用户最新信息
    │
    ├─→ 2. jwt.GenToken(user.UserID, user.Username)
    │       ├─ 生成新 Access Token (10分钟)
    │       └─ 生成新 Refresh Token (30天)
    │
    └─→ 3. redis.SetUserToken()
            ├─ 存储新 Access Token (覆盖旧的)
            └─ 存储新 Refresh Token (覆盖旧的)
    ↓
返回新双 Token
```

**为什么每次都生成新的 Refresh Token?**

这就是 **Refresh Token 轮转(Rotation)** 机制,核心优势:

1. **限制攻击窗口**: 即使 Refresh Token 被盗,攻击者只能在下次合法刷新前使用。
2. **异常检测**: 如果旧 Refresh Token 被使用,说明有人在非法访问,可以触发告警。
3. **自动修复**: 合法用户下次刷新时,会拿到新 Token,自动"夺回"控制权。

---

## 4. 接口层 (Controller)

在 `controller/user.go` 中新增 Handler。

```go
package controller

import (
	"bluebell/logic"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RefreshTokenHandler 刷新 Access Token
// @Summary 刷新令牌
// @Description 使用 Refresh Token 换取新的 Access Token 和 Refresh Token
// @Tags 用户相关
// @Accept application/json
// @Produce application/json
// @Param refresh_token query string true "Refresh Token"
// @Param Authorization header string true "Bearer {旧的 Access Token}"
// @Success 200 {object} ResponseData
// @Router /refresh_token [get]
func RefreshTokenHandler(c *gin.Context) {
	// 1. 获取 Refresh Token (从 Query 参数)
	// 为什么用 Query 而不是 Body:
	//   - GET 请求更符合 RESTful 规范 (获取新资源)
	//   - 方便在浏览器地址栏测试
	//   - 也可以改用 POST + Body,根据团队规范决定
	rt := c.Query("refresh_token")
	if rt == "" {
		ResponseErrorWithMsg(c, CodeInvalidToken, "缺少 refresh_token 参数")
		c.Abort()
		return
	}

	// 2. 获取旧的 Access Token (从 Authorization Header)
	// 为什么需要旧 Token:
	//   - 格式校验,确保请求合法
	//   - 可用于日志记录,追踪刷新行为
	//   - 未来可扩展为"级联撤销"(刷新时主动撤销旧 Token)
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		ResponseErrorWithMsg(c, CodeInvalidToken, "请求头缺少 Auth Token")
		c.Abort()
		return
	}

	// 3. 解析 Authorization Header
	// 期望格式: "Bearer <token>"
	parts := strings.SplitN(authHeader, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		ResponseErrorWithMsg(c, CodeInvalidToken, "Token 格式错误")
		c.Abort()
		return
	}
	aToken := parts[1]

	// 4. 调用业务逻辑
	// Logic 层会:
	//   - 验证 Refresh Token
	//   - 查询数据库,确保用户状态正常
	//   - 生成新双 Token
	//   - 更新 Redis
	newAToken, newRToken, err := logic.RefreshToken(aToken, rt)
	if err != nil {
		// 记录错误日志,方便排查
		zap.L().Error("logic.RefreshToken failed", zap.Error(err))

		// 返回统一错误码
		// 注意: 不要暴露详细错误信息给前端,避免信息泄露
		ResponseError(c, CodeInvalidToken)
		return
	}

	// 5. 返回新 Token
	// 前端收到后,应该:
	//   - 更新内存中的 Access Token
	//   - 更新本地存储的 Refresh Token
	//   - 重试刚才失败的请求
	ResponseSuccess(c, map[string]string{
		"access_token":  newAToken,
		"refresh_token": newRToken,
	})
}
```

**接口设计要点:**

1. **GET vs POST**:
   - 本项目用 GET (RESTful 风格,获取新资源)
   - 也可以用 POST (更符合"修改状态"的语义)

2. **Refresh Token 传递方式**:
   - Query 参数 (本项目)
   - Body 参数 (POST 请求)
   - HttpOnly Cookie (最安全,但跨域麻烦)

3. **错误处理**:
   - 不要暴露详细错误给前端 ("用户不存在" vs "Token 无效")
   - 记录详细日志到服务器,方便排查

---

## 5. 路由注册

在 `routers/routers.go` 中:

```go
package routers

import (
	"bluebell/controller"
	"bluebell/logger"
	"bluebell/middlewares"

	"github.com/gin-gonic/gin"
)

func SetupRouter(mode string) *gin.Engine {
	if mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(logger.GinLogger(), logger.GinRecovery(true))

	v1 := r.Group("/api/v1")

	// 公开接口 (不需要登录)
	v1.POST("/signup", controller.SignUpHandler)
	v1.POST("/login", controller.LoginHandler)

	// 【重点】刷新 Token 接口
	// 为什么不放在认证路由组:
	//   - Access Token 可能已经过期,无法通过 JWTAuthMiddleware
	//   - 刷新接口本身就是用来处理"Token 过期"的场景
	//   - 在 Handler 内部会手动验证 Refresh Token
	v1.GET("/refresh_token", controller.RefreshTokenHandler)

	// 认证接口 (需要登录)
	v1.Use(middlewares.JWTAuthMiddleware())
	{
		v1.GET("/ping", func(c *gin.Context) {
			c.String(200, "pong")
		})

		// 其他业务接口...
		v1.GET("/community", controller.CommunityHandler)
		v1.POST("/post", controller.CreatePostHandler)
		// ...
	}

	return r
}
```

**为什么刷新接口不经过 JWTAuthMiddleware?**

因为刷新接口被调用时,Access Token 通常已经过期,如果经过中间件会被直接拒绝。所以刷新接口放在公开路由组,但在 Handler 内部手动验证 Refresh Token。

---

## 6. 客户端实现无感刷新

前端需要配合实现"自动刷新并重试"机制。

### 6.1 Vue 3 + Axios 实现

```javascript
// utils/request.js
import axios from 'axios'
import { useUserStore } from '@/stores/user'
import router from '@/router'

const request = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
  timeout: 10000
})

// 请求拦截器: 自动添加 Token
request.interceptors.request.use(
  config => {
    const userStore = useUserStore()
    if (userStore.accessToken) {
      config.headers.Authorization = `Bearer ${userStore.accessToken}`
    }
    return config
  },
  error => Promise.reject(error)
)

// 响应拦截器: 自动刷新 Token
let isRefreshing = false  // 是否正在刷新
let failedQueue = []      // 失败的请求队列

// 处理队列中的请求
const processQueue = (error, token = null) => {
  failedQueue.forEach(prom => {
    if (error) {
      prom.reject(error)
    } else {
      prom.resolve(token)
    }
  })
  failedQueue = []
}

request.interceptors.response.use(
  response => response,
  async error => {
    const originalRequest = error.config

    // 1. 如果是 401 错误 (Token 过期)
    if (error.response?.status === 401 && !originalRequest._retry) {
      // 2. 如果正在刷新,将当前请求加入队列
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject })
        })
          .then(token => {
            originalRequest.headers.Authorization = `Bearer ${token}`
            return request(originalRequest)
          })
          .catch(err => Promise.reject(err))
      }

      originalRequest._retry = true
      isRefreshing = true

      const userStore = useUserStore()
      const refreshToken = userStore.refreshToken

      if (!refreshToken) {
        // 没有 Refresh Token,跳转登录页
        router.push('/login')
        return Promise.reject(error)
      }

      try {
        // 3. 调用刷新接口
        const { data } = await axios.get(
          `http://localhost:8080/api/v1/refresh_token?refresh_token=${refreshToken}`,
          {
            headers: {
              Authorization: `Bearer ${userStore.accessToken}`
            }
          }
        )

        const { access_token, refresh_token } = data.data

        // 4. 更新 Token
        userStore.setToken(access_token, refresh_token)

        // 5. 更新原请求的 Token
        originalRequest.headers.Authorization = `Bearer ${access_token}`

        // 6. 处理队列中的请求
        processQueue(null, access_token)

        // 7. 重试原请求
        return request(originalRequest)
      } catch (refreshError) {
        // 刷新失败,清空 Token 并跳转登录页
        processQueue(refreshError, null)
        userStore.clearToken()
        router.push('/login')
        return Promise.reject(refreshError)
      } finally {
        isRefreshing = false
      }
    }

    return Promise.reject(error)
  }
)

export default request
```

**核心机制:**

1. **请求拦截器**: 自动在 Header 中添加 Access Token。
2. **响应拦截器**: 捕获 401 错误,自动刷新 Token 并重试。
3. **请求队列**: 如果多个请求同时失败,只刷新一次 Token,其他请求等待。
4. **错误处理**: 刷新失败后,清空 Token 并跳转登录页。

### 6.2 Pinia Store (用户状态管理)

```javascript
// stores/user.js
import { defineStore } from 'pinia'

export const useUserStore = defineStore('user', {
  state: () => ({
    accessToken: localStorage.getItem('access_token') || null,
    refreshToken: localStorage.getItem('refresh_token') || null,
    userInfo: null
  }),

  actions: {
    // 设置 Token (登录和刷新时调用)
    setToken(accessToken, refreshToken) {
      this.accessToken = accessToken
      this.refreshToken = refreshToken

      // 存储到 localStorage (刷新页面不丢失)
      localStorage.setItem('access_token', accessToken)
      localStorage.setItem('refresh_token', refreshToken)
    },

    // 清空 Token (登出时调用)
    clearToken() {
      this.accessToken = null
      this.refreshToken = null
      this.userInfo = null

      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
    }
  }
})
```

**存储策略:**

| Token 类型 | 存储位置 | 原因 |
|-----------|---------|------|
| Access Token | Pinia State + LocalStorage | 频繁使用,需要快速访问 |
| Refresh Token | LocalStorage | 低频使用,刷新页面不丢失 |

**安全性说明:**

- ⚠️ LocalStorage 易受 XSS 攻击,生产环境建议用 HttpOnly Cookie。
- ✅ 配合 CSP (Content Security Policy) 可以降低 XSS 风险。

---

## 7. 安全最佳实践

### 7.1 Refresh Token 轮转 (Rotation)

**已实现:** 我们的代码每次刷新都生成新的 Refresh Token。

**进阶: 检测重放攻击**

如果旧的 Refresh Token 被使用,可能说明发生了攻击:

```go
// redis/user.go 中新增
func IsRefreshTokenRevoked(userID int64, rToken string) bool {
    // 从 Redis 获取当前有效的 Refresh Token
    validToken, _ := GetUserRefreshToken(userID)

    // 如果请求的 Token 与 Redis 中的不一致,说明是旧 Token
    if rToken != validToken {
        // 记录异常日志
        zap.L().Warn("检测到旧 Refresh Token 被使用",
            zap.Int64("user_id", userID),
            zap.String("token", rToken))

        // 可选: 冻结账号,发送告警邮件
        // mysql.FreezeUser(userID)
        // email.SendSecurityAlert(userID)

        return true
    }
    return false
}
```

### 7.2 Refresh Token 黑名单

**场景:** 用户修改密码后,所有旧 Token 应立即失效。

**实现:**

```go
// 修改密码后,撤销所有 Token
func ChangePassword(userID int64, oldPwd, newPwd string) error {
    // 1. 验证旧密码...

    // 2. 更新密码...

    // 3. 删除 Redis 中的 Token (强制重新登录)
    err := redis.DeleteUserToken(userID)
    if err != nil {
        zap.L().Error("delete user token failed", zap.Error(err))
    }

    // 4. (可选) 将当前 Refresh Token 加入黑名单
    // currentRefreshToken := ...
    // redis.AddToBlacklist(currentRefreshToken, jwt.RefreshTokenExpireDuration)

    return nil
}
```

### 7.3 设备绑定 (Device Binding)

**原理:** 将 Refresh Token 与设备指纹绑定,限制跨设备使用。

```go
type UserClaims struct {
    UserID     int64  `json:"user_id"`
    Username   string `json:"username"`
    DeviceID   string `json:"device_id"`  // 新增: 设备指纹
    jwt.RegisteredClaims
}

// 刷新时验证设备
func RefreshToken(aToken, rToken, deviceID string) (newAToken, newRToken string, err error) {
    user, err := jwt.ValidateRefreshToken(rToken)
    if err != nil {
        return "", "", err
    }

    // 验证设备ID
    claims, _ := jwt.ParseToken(aToken)
    if claims.DeviceID != deviceID {
        return "", "", errors.New("设备不匹配,拒绝刷新")
    }

    // ...
}
```

**设备指纹生成 (前端):**

```javascript
// 简单示例 (生产环境建议用 fingerprintjs2 等专业库)
const deviceID = navigator.userAgent + screen.width + screen.height
```

### 7.4 IP 白名单/地域限制

```go
func RefreshToken(c *gin.Context, aToken, rToken string) (newAToken, newRToken string, err error) {
    // 获取客户端 IP
    clientIP := c.ClientIP()

    // 从 Redis 获取上次登录的 IP
    lastIP, _ := redis.GetUserLastIP(userID)

    // 如果 IP 变化且跨地域 (需要 IP 地址库)
    if !isSameRegion(clientIP, lastIP) {
        // 发送验证码或二次认证
        return "", "", errors.New("检测到异地登录,需要验证")
    }

    // 更新最后登录 IP
    redis.SetUserLastIP(userID, clientIP)

    // ...
}
```

---

## 8. 常见问题 (FAQ)

### Q1: Refresh Token 存储在哪里最安全?

**A:** 安全性排序:

1. **HttpOnly Cookie (最安全)**: JS 无法访问,防 XSS。但跨域麻烦,需要配置 CORS。
2. **IndexedDB**: 比 LocalStorage 安全,容量更大。
3. **LocalStorage (便捷)**: 最常用,但易受 XSS 攻击。
4. **Memory (内存)**: 最安全,但刷新页面就丢失。

**推荐方案:**

- Access Token 存储在内存 (Pinia State)
- Refresh Token 存储在 HttpOnly Cookie

### Q2: Refresh Token 的有效期应该设置多长?

**A:** 根据业务场景决定:

| 场景 | Refresh Token 有效期 | 原因 |
|------|-------------------|------|
| **社交应用** | 90 天 | 用户粘性高,希望长期免登录 |
| **电商平台** | 30 天 | 平衡安全性和用户体验 |
| **金融应用** | 7 天 | 安全性优先,降低风险 |
| **企业内部系统** | 24 小时 | 高安全要求,每天重新登录 |

**动态调整策略:**

- 用户勾选"记住我": 90 天
- 未勾选: 7 天

### Q3: 如果 Refresh Token 也过期了怎么办?

**A:** 只能重新登录。

**用户体验优化:**

1. **提前提醒**: Refresh Token 过期前 7 天,提示用户。
2. **无感延期**: 用户有活跃行为时,自动延长 Refresh Token 有效期。
3. **快速登录**: 支持短信验证码、指纹、Face ID 等快捷登录方式。

### Q4: 多设备登录如何处理?

**A:** 三种策略:

1. **互踢模式**: 新设备登录,旧设备被踢下线 (我们已实现)。
2. **多设备共存**: 每个设备独立的 Token,都存储在 Redis (Key: `user:token:{uid}:{device_id}`)。
3. **限制数量**: 最多允许 3 个设备同时在线,超过后踢掉最早登录的。

**实现多设备共存:**

```go
// Redis Key: bluebell:user:token:{userID}:{deviceID}
func SetUserToken(userID int64, deviceID string, aToken, rToken string, aExp, rExp time.Duration) error {
    key := fmt.Sprintf("bluebell:user:token:%d:%s", userID, deviceID)
    // ...
}
```

### Q5: 如何实现"强制下线"功能?

**A:** 管理员端调用接口,删除用户的 Redis Token。

```go
// 管理员接口
func AdminForceLogout(c *gin.Context) {
    userID := c.Query("user_id")

    // 删除 Redis 中的 Token
    err := redis.DeleteUserToken(userID)
    if err != nil {
        ResponseError(c, CodeServerBusy)
        return
    }

    // (可选) 将当前 Refresh Token 加入黑名单
    // ...

    ResponseSuccess(c, nil)
}
```

**用户下次请求时:**

- Access Token 与 Redis 不匹配 → 中间件拒绝
- Refresh Token 刷新时 → Redis 无该用户的 Token → 拒绝

---

## 9. 课后练习

### 练习 1: 实现 Refresh Token 黑名单

**需求:** 用户修改密码后,将旧 Refresh Token 加入黑名单,禁止其继续使用。

**提示:**

```go
// Redis Key: bluebell:token:blacklist:{token_hash}
func AddToBlacklist(token string, ttl time.Duration) error {
    // 1. 对 Token 做 SHA256 哈希 (避免 Key 过长)
    hash := sha256.Sum256([]byte(token))
    hashStr := hex.EncodeToString(hash[:])

    // 2. 存入 Redis,TTL 为 Token 的剩余有效期
    key := fmt.Sprintf("bluebell:token:blacklist:%s", hashStr)
    return rdb.Set(ctx, key, "1", ttl).Err()
}

// 刷新时检查黑名单
func RefreshToken(aToken, rToken string) (newAToken, newRToken string, err error) {
    // 检查 Refresh Token 是否在黑名单中
    if IsInBlacklist(rToken) {
        return "", "", errors.New("Token 已被撤销")
    }
    // ...
}
```

### 练习 2: 实现"记住我"功能

**需求:** 登录时,如果用户勾选"记住我",Refresh Token 有效期延长到 90 天。

**提示:**

```go
// models/params.go
type ParamLogin struct {
    Username   string `json:"username" binding:"required"`
    Password   string `json:"password" binding:"required"`
    RememberMe bool   `json:"remember_me"`  // 新增
}

// logic/user.go
func Login(p *models.ParamLogin) (aToken, rToken string, err error) {
    // ...

    // 根据 RememberMe 设置不同的过期时间
    var rExp time.Duration
    if p.RememberMe {
        rExp = time.Hour * 24 * 90  // 90 天
    } else {
        rExp = jwt.RefreshTokenExpireDuration  // 30 天
    }

    err = redis.SetUserToken(user.UserID, aToken, rToken, jwt.AccessTokenExpireDuration, rExp)
    // ...
}
```

### 练习 3: 实现设备管理

**需求:** 用户可以查看所有登录设备,并踢掉指定设备。

**提示:**

```go
// 数据结构
type Device struct {
    DeviceID   string    `json:"device_id"`
    DeviceName string    `json:"device_name"`  // "iPhone 13"
    IP         string    `json:"ip"`
    Location   string    `json:"location"`     // "北京市"
    LastActive time.Time `json:"last_active"`
}

// 接口
func GetMyDevices(c *gin.Context) {
    userID, _ := GetCurrentUser(c)
    devices := redis.GetUserDevices(userID)
    ResponseSuccess(c, devices)
}

func KickDevice(c *gin.Context) {
    userID, _ := GetCurrentUser(c)
    deviceID := c.Query("device_id")

    // 删除该设备的 Token
    redis.DeleteUserTokenByDevice(userID, deviceID)

    ResponseSuccess(c, nil)
}
```

---

## 10. 本章总结

本章我们深入探讨了 Refresh Token 的最佳实践,核心要点回顾:

### 核心机制

1. ✅ **双 Token 架构**: Access Token (短效高频) + Refresh Token (长效低频)
2. ✅ **轮转机制**: 每次刷新都生成新的 Refresh Token,限制攻击窗口
3. ✅ **数据库校验**: 刷新时重新查询用户信息,确保状态正常
4. ✅ **Redis 状态管理**: 配合 Redis 实现单点登录和强制下线

### 安全实践

1. ✅ **HttpOnly Cookie**: 最安全的 Refresh Token 存储方式
2. ✅ **设备绑定**: 限制 Token 跨设备使用
3. ✅ **IP 白名单**: 检测异地登录,触发二次认证
4. ✅ **黑名单机制**: 修改密码后,撤销所有旧 Token

### 用户体验

1. ✅ **无感刷新**: 前端自动处理 401 错误,用户无感知
2. ✅ **请求队列**: 多个请求同时失败时,只刷新一次 Token
3. ✅ **降级策略**: 刷新失败后,友好提示并跳转登录页

### 架构设计

```
客户端
    ├─ Axios 拦截器 (自动刷新)
    └─ Pinia Store (Token 管理)
        ↓
服务器
    ├─ Controller (接口层)
    ├─ Logic (业务逻辑)
    │   ├─ JWT 验证
    │   ├─ 数据库查询
    │   └─ Redis 更新
    └─ Middleware (认证拦截)
```

### 下一章预告

在第10章中,我们将探讨 **单点登录与互踢模式**,包括:
- 如何实现多设备互踢
- 设备管理和在线状态
- WebSocket 实时推送下线通知
- 企业级 SSO (单点登录) 方案

---

## 11. 延伸阅读

- 📖 [RFC 6749: OAuth 2.0 Authorization Framework](https://tools.ietf.org/html/rfc6749)
- 📖 [OWASP Authentication Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html)
- 📖 [JWT Best Practices](https://tools.ietf.org/html/rfc8725)
- 📖 下一章: [第10章:单点登录与互踢模式](./10-单点登录与互踢模式.md)
