package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bluebell/internal/config"
	"bluebell/internal/dao/redis"
	"bluebell/internal/jwt"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTAuthMiddleware_JTIBlacklist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rdb := goredis.NewClient(&goredis.Options{
		Addr: "127.0.0.1:6379",
		DB:   15,
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("跳过测试：无法连接 Redis")
	}
	defer rdb.Close()
	_ = rdb.FlushDB(ctx).Err()

	tokenCache := redis.NewUserTokenCache(rdb)

	cfg := &config.Config{
		JWT: &config.JWTConfig{
			Secret:        "test-secret-middleware-123456",
			AccessExpiry:  "2h",
			RefreshExpiry: "168h",
		},
	}

	r := gin.New()
	r.Use(JWTAuthMiddleware(cfg, tokenCache))
	r.GET("/protected", func(c *gin.Context) {
		uid, _ := c.Get("UserIDKey")
		c.JSON(http.StatusOK, gin.H{"user_id": uid})
	})

	userID := int64(8888)
	aToken, _, err := jwt.GenToken(cfg, userID)
	require.NoError(t, err)

	claims, err := jwt.ParseTokenClaims(cfg, aToken, jwt.AccessTokenType)
	require.NoError(t, err)

	// Case 1: 正常请求（不在黑名单） -> 200 OK
	req1, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req1.Header.Set("Authorization", "Bearer "+aToken)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Case 2: 将 JTI 加入黑名单 -> 401 Unauthorized
	err = tokenCache.AddBlacklist(ctx, claims.ID, 1*time.Hour)
	require.NoError(t, err)

	req2, _ := http.NewRequest(http.MethodGet, "/protected", nil)
	req2.Header.Set("Authorization", "Bearer "+aToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusUnauthorized, w2.Code)
}

func TestJWTOptionalAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		JWT: &config.JWTConfig{
			Secret:        "test-secret-optional-123456",
			AccessExpiry:  "2h",
			RefreshExpiry: "168h",
		},
	}

	r := gin.New()
	r.Use(JWTOptionalAuthMiddleware(cfg, nil))
	r.GET("/public", func(c *gin.Context) {
		uid, exist := c.Get("UserIDKey")
		if exist {
			c.JSON(http.StatusOK, gin.H{"user_id": uid.(int64), "logged_in": true})
		} else {
			c.JSON(http.StatusOK, gin.H{"user_id": int64(0), "logged_in": false})
		}
	})

	// 1. 无 Token 请求 -> 200 OK, logged_in = false
	req1, _ := http.NewRequest(http.MethodGet, "/public", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), `"logged_in":false`)

	// 2. 有效 Token 请求 -> 200 OK, logged_in = true, user_id = 9999
	aToken, _, err := jwt.GenToken(cfg, 9999)
	require.NoError(t, err)

	req2, _ := http.NewRequest(http.MethodGet, "/public", nil)
	req2.Header.Set("Authorization", "Bearer "+aToken)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
	assert.Contains(t, w2.Body.String(), `"logged_in":true`)
	assert.Contains(t, w2.Body.String(), `"user_id":9999`)

	// 3. 非法 Token 请求 -> 200 OK, logged_in = false（不阻断访问）
	req3, _ := http.NewRequest(http.MethodGet, "/public", nil)
	req3.Header.Set("Authorization", "Bearer invalid-malformed-token")
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Contains(t, w3.Body.String(), `"logged_in":false`)
}
