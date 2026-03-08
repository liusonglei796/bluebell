package middleware

import (
	"bluebell/internal/backfront"
	"bluebell/internal/config"
	"bluebell/internal/domain/cachedomain"
	"bluebell/internal/infrastructure/jwt"
	"bluebell/pkg/errorx"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware 基于JWT的认证中间件，包含 SSO 校验
func JWTAuthMiddleware(cfg *config.Config, tokenRepo cachedomain.UserTokenCacheRepository) gin.HandlerFunc {
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
		tokenStr := parts[1]

		// 3. 解析并校验 aToken
		userID, err := jwt.ParseToken(cfg, tokenStr)
		if err != nil {
			backfront.HandleError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		// 4. SSO 校验 (Redis 实时校验)
		activeToken, err := tokenRepo.GetUserAccessToken(c.Request.Context(), userID)
		if err != nil {
			// Redis 异常时降级处理：仅依赖 JWT 自身校验结果
			// 此处如果不满足业务强限制，也可以选择直接 Abort
		} else {
			if activeToken != tokenStr {
				backfront.HandleError(c, errorx.New(errorx.CodeInvalidToken, "账号已在其他地方登录"))
				c.Abort()
				return
			}
		}

		// 5. 将用户信息存入上下文
		c.Set("UserIDKey", userID)
		c.Next()
	}
}
