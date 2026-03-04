package middleware

import (
	"bluebell/internal/domain/repository"
	"bluebell/internal/handler"
	"bluebell/internal/infrastructure/jwt"
	"bluebell/pkg/errorx"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// tokenCache 本地 Token 缓存，减少 Redis 查询压力
type tokenCache struct {
	sync.RWMutex
	cache map[int64]*cacheEntry
}

type cacheEntry struct {
	token      string
	expireTime time.Time
}

var (
	localCache = &tokenCache{
		cache: make(map[int64]*cacheEntry),
	}
	cacheExpireDuration = 5 * time.Minute
	enableStrictSSO     = false
)

func init() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cleanExpiredCache()
		}
	}()
}

func cleanExpiredCache() {
	localCache.Lock()
	defer localCache.Unlock()
	now := time.Now()
	for userID, entry := range localCache.cache {
		if now.After(entry.expireTime) {
			delete(localCache.cache, userID)
		}
	}
}

func getTokenFromCache(userID int64) (string, bool) {
	localCache.RLock()
	defer localCache.RUnlock()
	entry, exists := localCache.cache[userID]
	if !exists {
		return "", false
	}
	if time.Now().After(entry.expireTime) {
		return "", false
	}
	return entry.token, true
}

func setTokenToCache(userID int64, token string) {
	localCache.Lock()
	defer localCache.Unlock()
	localCache.cache[userID] = &cacheEntry{
		token:      token,
		expireTime: time.Now().Add(cacheExpireDuration),
	}
}

// JWTAuthMiddleware 基于JWT的认证中间件
// tokenCache: 用户Token缓存仓储，通过依赖注入传入
func JWTAuthMiddleware(tokenCache repository.UserTokenCacheRepository) func(c *gin.Context) {
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

		mc, err := jwt.ParseToken(parts[1])
		if err != nil {
			handler.ResponseError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		if enableStrictSSO {
			if !validateTokenWithRedis(c, tokenCache, mc.UserID, parts[1]) {
				return
			}
		} else {
			validateTokenWithFallback(c, tokenCache, mc.UserID, parts[1])
		}

		c.Set(handler.CtxUserIDKey, mc.UserID)
		c.Next()
	}
}

func validateTokenWithRedis(c *gin.Context, tokenRepo repository.UserTokenCacheRepository, userID int64, token string) bool {
	cachedToken, exists := getTokenFromCache(userID)
	if exists && cachedToken == token {
		return true
	}

	redisToken, err := tokenRepo.GetUserAccessToken(c.Request.Context(), userID)
	if err != nil {
		handler.ResponseError(c, errorx.ErrNeedLogin)
		c.Abort()
		return false
	}

	if token != redisToken {
		handler.ResponseErrorWithMsg(c, errorx.CodeInvalidToken, "账号已在其他设备登录")
		c.Abort()
		return false
	}

	setTokenToCache(userID, token)
	return true
}

func validateTokenWithFallback(c *gin.Context, tokenRepo repository.UserTokenCacheRepository, userID int64, token string) {
	cachedToken, exists := getTokenFromCache(userID)
	if exists && cachedToken == token {
		return
	}

	redisToken, err := tokenRepo.GetUserAccessToken(c.Request.Context(), userID)
	if err != nil {
		zap.L().Warn("Redis Token 校验失败，启用降级模式",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return
	}

	if token != redisToken {
		handler.ResponseErrorWithMsg(c, errorx.CodeInvalidToken, "账号已在其他设备登录")
		c.Abort()
		return
	}

	setTokenToCache(userID, token)
}
