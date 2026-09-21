package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimitMiddleware_ExceedCapacity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	// 容量为 3，每秒填充 1 个令牌（突发最多 3 个请求）
	r.Use(RateLimitMiddleware(1*time.Second, 3))
	r.GET("/test/post/:id", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"msg": "ok"})
	})

	// 连续发送 3 次请求 -> 应当全部成功 (200)
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test/post/1001", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "在令牌桶容量内的请求必须正常放行")
	}

	// 第 4 次请求 -> 令牌已耗尽，必须被限流拦截 (429)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test/post/1001", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "超出令牌桶容量的请求必须返回 429")
	assert.Equal(t, "1", w.Header().Get("Retry-After"))
}
