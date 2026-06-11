package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bluebell/internal/domain/entity"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestSlidingRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	// Mock the time source for testing and defer restoring it
	mockTime := time.Now()
	origTimeNow := timeNow
	timeNow = func() time.Time {
		return mockTime
	}
	defer func() {
		timeNow = origTimeNow
	}()

	r := gin.New()
	r.Use(SlidingRateLimitMiddleware(rdb, 200*time.Millisecond, 2))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req1, _ := http.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	req2, _ := http.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	req3, _ := http.NewRequest("GET", "/test", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
	assert.Equal(t, "rate-limited", w3.Header().Get("X-RateLimit-Limit"))
	assert.Contains(t, w3.Body.String(), entity.ErrRateLimitExceeded.Error())

	// Fast forward both miniredis and the mock time
	mockTime = mockTime.Add(300 * time.Millisecond)
	mr.FastForward(300 * time.Millisecond)

	req4, _ := http.NewRequest("GET", "/test", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)
}
