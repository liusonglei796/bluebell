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
		// 1. 获取 Authorization Header
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			handler.ResponseError(c, errorx.ErrNeedLogin)
			c.Abort()
			return
		}

		// 2. 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			handler.ResponseError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		// 3. 解析并校验 aToken
		userID, err := jwt.ParseToken(jwtCfg, parts[1])
		if err != nil {
			handler.ResponseError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		// 4. 将用户信息存入上下文
		c.Set(handler.CtxUserIDKey, userID)
		c.Next()
	}
}
