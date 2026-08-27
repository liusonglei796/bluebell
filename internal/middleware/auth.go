package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"bluebell/internal/config"
	"bluebell/internal/dao/redis"
	"bluebell/internal/jwt"
	"bluebell/internal/model"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// JWTAuthMiddleware 基于 JWT 的认证中间件（支持 JTI 黑名单快速拦截）
func JWTAuthMiddleware(cfg *config.Config, tokenRepo *redis.UserTokenCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取 Authorization Header
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":  1006,
				"msg":   model.ErrNeedLogin.Error(),
				"error": model.ErrNeedLogin.Error(),
			})
			c.Abort()
			return
		}

		// 2. 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":  1007,
				"msg":   model.ErrInvalidToken.Error(),
				"error": model.ErrInvalidToken.Error(),
			})
			c.Abort()
			return
		}
		tokenStr := parts[1]

		// 3. 解析并校验 aToken Claims
		claims, err := jwt.ParseTokenClaims(cfg, tokenStr, jwt.AccessTokenType)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":  1007,
				"msg":   model.ErrInvalidToken.Error(),
				"error": model.ErrInvalidToken.Error(),
			})
			c.Abort()
			return
		}

		// 4. JTI 黑名单拦截校验（无状态 AccessToken + Redis 仅存储注销/失效的 JTI）
		if claims.ID != "" && tokenRepo != nil {
			isBlacklisted, err := tokenRepo.IsBlacklisted(c.Request.Context(), claims.ID)
			if err != nil {
				zap.L().Warn("check jwt blacklist error, fallback", zap.String("jti", claims.ID), zap.Error(err))
			} else if isBlacklisted {
				c.JSON(http.StatusUnauthorized, gin.H{
					"code":  1007,
					"msg":   "Token已注销或已失效",
					"error": "Token已注销或已失效",
				})
				c.Abort()
				return
			}
		}

		userID, err := strconv.ParseInt(claims.Subject, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":  1007,
				"msg":   model.ErrInvalidToken.Error(),
				"error": model.ErrInvalidToken.Error(),
			})
			c.Abort()
			return
		}
		// 5. 将用户上下文及 Token 元数据写入 Gin Context
		c.Set("UserIDKey", userID)
		c.Set("JTIKey", claims.ID)
		if claims.ExpiresAt != nil {
			c.Set("TokenExpKey", claims.ExpiresAt.Time)
		}
		c.Next()
	}
}

// JWTOptionalAuthMiddleware 可选 JWT 认证中间件（有有效 Token 则解析注入 Context，无 Token 或 Token 无效则静默放行）
func JWTOptionalAuthMiddleware(cfg *config.Config, tokenRepo *redis.UserTokenCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.Next()
			return
		}

		claims, err := jwt.ParseTokenClaims(cfg, parts[1], jwt.AccessTokenType)
		if err != nil || claims == nil {
			c.Next()
			return
		}

		if claims.ID != "" && tokenRepo != nil {
			isBlacklisted, err := tokenRepo.IsBlacklisted(c.Request.Context(), claims.ID)
			if err == nil && isBlacklisted {
				c.Next()
				return
			}
		}

		if userID, err := strconv.ParseInt(claims.Subject, 10, 64); err == nil {
			c.Set("UserIDKey", userID)
			c.Set("JTIKey", claims.ID)
			if claims.ExpiresAt != nil {
				c.Set("TokenExpKey", claims.ExpiresAt.Time)
			}
		}

		c.Next()
	}
}
