# 第08章:JWT认证与登录功能实现

> **本章导读**
>
> 在上一章中,我们完成了用户的注册和密码加密存储。现在的核心任务是:**如何证明"我是我"?**
>
> 传统的 Web 开发常用 Cookie-Session 模式,但在前后端分离和微服务架构中,**JWT (JSON Web Token)** 已成为事实上的标准。本章将带领你实现基于 JWT 的登录认证系统。

---

## 📚 本章目标

学完本章,你将掌握:

1. 理解 JWT 的核心原理 (Header.Payload.Signature)
2. 使用 `golang-jwt/jwt/v5` 库生成和解析 Token
3. 实现 Access Token (短效) + Refresh Token (长效) 双令牌机制
4. 编写 Gin 中间件拦截未登录请求
5. 掌握 JWT 的安全最佳实践

---

## 1. JWT 工具包实现

我们使用 `github.com/golang-jwt/jwt/v5` 库。

### 1.1 实现代码

#### 文件: `internal/infrastructure/jwt/jwt.go`

```go
package jwt

import (
	"bluebell/internal/config"
	"bluebell/pkg/errorx"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenToken 生成 Access Token 和 Refresh Token
func GenToken(cfg *config.JWTConfig, userID int64) (aToken, rToken string, err error) {
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(mustParseDuration(cfg.AccessExpiry))),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "bluebell",
	}
	aToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", "", errorx.Wrap(err, errorx.CodeInfraError, "生成 AccessToken 失败")
	}

	rToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", userID),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(mustParseDuration(cfg.RefreshExpiry))),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "bluebell",
	}).SignedString([]byte(cfg.Secret))
	
	return aToken, rToken, nil
}

// ParseToken 解析并验证 Token，返回 userID
func ParseToken(cfg *config.JWTConfig, tokenString string) (userID int64, err error) {
	claims := new(jwt.RegisteredClaims)
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.Secret), nil
	})
	if err != nil || !token.Valid {
		return 0, errorx.ErrInvalidToken
	}

	userID, _ = strconv.ParseInt(claims.Subject, 10, 64)
	return userID, nil
}
```

---

## 2. JWT 认证中间件

这是保护 API 接口的关键屏障。

#### 文件: `internal/middleware/auth.go`

```go
package middleware

import (
	"bluebell/internal/config"
	"bluebell/internal/handler"
	"bluebell/internal/infrastructure/jwt"
	"bluebell/pkg/errorx"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware 基于JWT的认证中间件
func JWTAuthMiddleware(jwtCfg *config.JWTConfig) func(c *gin.Context) {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			handler.ResponseError(c, errorx.ErrNeedLogin)
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			handler.ResponseError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		userID, err := jwt.ParseToken(jwtCfg, parts[1])
		if err != nil {
			handler.ResponseError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		c.Set(handler.CtxUserIDKey, userID)
		c.Next()
	}
}
```

---

## 3. Service 层调用

#### 文件: `internal/service/user/user_service.go`

```go
func (s *UserService) Login(ctx context.Context, p *request.LoginRequest) (string, string, error) {
	// ... 验证密码成功后 ...
	
	aToken, rToken, err := jwt.GenToken(s.jwtCfg, user.UserID)
	if err != nil {
		return "", "", errorx.ErrServerBusy
	}

	return aToken, rToken, nil
}
```

---

**下一章:** [第09章:Refresh Token 最佳实践](./09-Refresh_Token_最佳实践.md)

**返回目录:** [README.md](./README.md)
