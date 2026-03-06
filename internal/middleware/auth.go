package middleware

import (
	"bluebell/internal/config"
	"bluebell/internal/handler"
	"bluebell/internal/infrastructure/jwt"
	"bluebell/internal/backfront"
	"bluebell/pkg/errorx"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware 基于JWT的认证中间件
func JWTAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取 Authorization Header
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			backfront.HandleError(c, errorx.ErrNeedLogin)
			c.Abort()
			return
		}

		// 2. 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			backfront.HandleError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		// 3. 解析并校验 aToken
		userID, err := jwt.ParseToken(cfg, parts[1])
		if err != nil {
			backfront.HandleError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		// 4. 将用户信息存入上下文可以自定义atoken的Claims填充这些字段，用自定义的claims生成atoken后再解析atoken（这个过程会填充一个空的自定义的claims）用这个空的自定义的claims返回字段
		c.Set(handler.CtxUserIDKey, userID)
		c.Next()
	}
}



