package middlewares

import (
	"bluebell/controller"
	"bluebell/dao/redis"
	"bluebell/pkg/errorx"
	"bluebell/pkg/jwt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// tokenCache 本地 Token 缓存，减少 Redis 查询压力
type tokenCache struct {
	sync.RWMutex
	cache map[int64]*cacheEntry // userID -> cacheEntry
}

type cacheEntry struct {
	token      string
	expireTime time.Time
}

var (
	localCache = &tokenCache{
		cache: make(map[int64]*cacheEntry),
	}
	// 本地缓存有效期（5分钟）
	cacheExpireDuration = 5 * time.Minute
	// 是否启用单点登录强制校验（可配置）
	enableStrictSSO = false // 设置为 false 启用降级模式
)

// 定期清理过期缓存
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

// 从本地缓存获取 Token
func getTokenFromCache(userID int64) (string, bool) {
	localCache.RLock()
	defer localCache.RUnlock()

	entry, exists := localCache.cache[userID]
	if !exists {
		return "", false
	}

	// 检查是否过期
	if time.Now().After(entry.expireTime) {
		return "", false
	}

	return entry.token, true
}

// 设置本地缓存
func setTokenToCache(userID int64, token string) {
	localCache.Lock()
	defer localCache.Unlock()

	localCache.cache[userID] = &cacheEntry{
		token:      token,
		expireTime: time.Now().Add(cacheExpireDuration),
	}
}

// JWTAuthMiddleware 基于JWT的认证中间件（优化版）
func JWTAuthMiddleware() func(c *gin.Context) {
	return func(c *gin.Context) {
		// 1. 获取 Authorization header
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			controller.ResponseError(c, errorx.ErrNeedLogin)
			c.Abort()
			return
		}

		// 2. 按空格分割
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			controller.ResponseError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		// 3. 解析 Token
		mc, err := jwt.ParseToken(parts[1])
		if err != nil {
			controller.ResponseError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		// 4. 单点登录校验（优化版：本地缓存 + 降级策略）
		if enableStrictSSO {
			// 严格模式：必须校验 Redis Token
			if !validateTokenWithRedis(c, mc.UserID, parts[1]) {
				return
			}
		} else {
			// 宽松模式（降级策略）：优先使用缓存，Redis 失败时允许通过
			validateTokenWithFallback(c, mc.UserID, parts[1])
		}

		// 5. 将当前请求的 userID 信息保存到请求的上下文
		c.Set(controller.CtxUserIDKey, mc.UserID)
		c.Next()
	}
}

// validateTokenWithRedis 严格模式：必须校验 Redis
func validateTokenWithRedis(c *gin.Context, userID int64, token string) bool {
	// 先查本地缓存
	cachedToken, exists := getTokenFromCache(userID)
	if exists && cachedToken == token {
		return true // 缓存命中，直接通过
	}

	// 缓存未命中，查询 Redis
	redisToken, err := redis.GetUserAccessToken(userID)
	if err != nil {
		controller.ResponseError(c, errorx.ErrNeedLogin)
		c.Abort()
		return false
	}

	// 校验 Token 是否一致
	if token != redisToken {
		controller.ResponseErrorWithMsg(c, errorx.CodeInvalidToken, "账号已在其他设备登录")
		c.Abort()
		return false
	}

	// 更新本地缓存
	setTokenToCache(userID, token)
	return true
}

// validateTokenWithFallback 宽松模式：Redis 失败时降级
func validateTokenWithFallback(c *gin.Context, userID int64, token string) {
	// 先查本地缓存
	cachedToken, exists := getTokenFromCache(userID)
	if exists && cachedToken == token {
		return // 缓存命中，直接通过
	}

	// 缓存未命中，查询 Redis
	redisToken, err := redis.GetUserAccessToken(userID)
	if err != nil {
		// Redis 查询失败，降级处理：仅依赖 JWT 本身的有效性
		zap.L().Warn("Redis Token 校验失败，启用降级模式",
			zap.Int64("user_id", userID),
			zap.Error(err))
		return // 允许通过（JWT 本身已验证通过）
	}

	// Redis 查询成功，校验 Token 是否一致
	if token != redisToken {
		controller.ResponseErrorWithMsg(c, errorx.CodeInvalidToken, "账号已在其他设备登录")
		c.Abort()
		return
	}

	// 更新本地缓存
	setTokenToCache(userID, token)
}
