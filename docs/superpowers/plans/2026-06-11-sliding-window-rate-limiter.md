# Redis-Based Sliding Window Rate Limiter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a distributed IP-based sliding window rate limiter in Go using Redis Sorted Sets and integrate it into the Bluebell project router.

**Architecture:** A new Gin middleware `SlidingRateLimitMiddleware` executes an atomic Redis Lua script to clear expired timestamps, count active requests for a client IP, and either accept or block the request.

**Tech Stack:** Go, Gin framework, Redis (go-redis/v9), Lua.

---

### Task 1: Add Configurations for Sliding Window Rate Limiter

**Files:**
- Modify: [internal/config/config.go](file:///D:/download/project/bluebell/internal/config/config.go)
- Modify: [config.yaml](file:///D:/download/project/bluebell/config.yaml)
- Modify: [config.toml](file:///D:/download/project/bluebell/config.toml)
- Modify: [config.docker.toml](file:///D:/download/project/bluebell/config.docker.toml)

- [ ] **Step 1: Update configuration struct**
  Add `slidingRateLimitConfig` and update `Config` struct.
  ```go
  type slidingRateLimitConfig struct {
  	Enabled bool   `mapstructure:"enabled"`
  	Window  string `mapstructure:"window"`
  	Limit   int64  `mapstructure:"limit"`
  }

  type Config struct {
      // ...
  	RateLimit        *rateLimitConfig        `mapstructure:"ratelimit"`
  	SlidingRateLimit *slidingRateLimitConfig `mapstructure:"sliding_ratelimit"`
      // ...
  }
  ```

- [ ] **Step 2: Add configuration properties to config files**
  Add the following snippet under the configuration files:
  ```yaml
  sliding_ratelimit:
    enabled: true
    window: "60s"
    limit: 100
  ```

- [ ] **Step 3: Verify configuration compilation**
  Run: `go build ./cmd/server`
  Expected: Successful compilation.

- [ ] **Step 4: Commit**
  ```bash
  git add internal/config/config.go config.yaml config.toml config.docker.toml
  git commit -m "feat: add sliding rate limit configuration fields"
  ```

---

### Task 2: Install miniredis dependency for testing

**Files:**
- Modify: [go.mod](file:///D:/download/project/bluebell/go.mod)
- Modify: [go.sum](file:///D:/download/project/bluebell/go.sum)

- [ ] **Step 1: Install miniredis package**
  Run: `go get github.com/alicebob/miniredis/v2@v2.34.0`
  Expected: Successful download and update of go.mod / go.sum.

- [ ] **Step 2: Verify package installation**
  Run: `go test ./internal/domain/entity/...`
  Expected: Existing tests still pass.

- [ ] **Step 3: Commit**
  ```bash
  git add go.mod go.sum
  git commit -m "chore: add miniredis test dependency"
  ```

---

### Task 4: Implement and Test Sliding Window Middleware

**Files:**
- Create: [internal/middleware/ratelimit_sliding.go](file:///D:/download/project/bluebell/internal/middleware/ratelimit_sliding.go)
- Create: [internal/middleware/ratelimit_sliding_test.go](file:///D:/download/project/bluebell/internal/middleware/ratelimit_sliding_test.go)

- [ ] **Step 1: Write the failing test (`ratelimit_sliding_test.go`)**
  Create `internal/middleware/ratelimit_sliding_test.go` with:
  ```go
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

  	mr.FastForward(300 * time.Millisecond)

  	req4, _ := http.NewRequest("GET", "/test", nil)
  	w4 := httptest.NewRecorder()
  	r.ServeHTTP(w4, req4)
  	assert.Equal(t, http.StatusOK, w4.Code)
  }
  ```

- [ ] **Step 2: Run test to verify compilation/test failure**
  Run: `go test -v ./internal/middleware`
  Expected: FAIL with "SlidingRateLimitMiddleware undefined".

- [ ] **Step 3: Implement SlidingRateLimitMiddleware**
  Create `internal/middleware/ratelimit_sliding.go` with:
  ```go
  package middleware

  import (
  	"net/http"
  	"time"

  	"bluebell/internal/domain/entity"

  	"github.com/gin-gonic/gin"
  	"github.com/redis/go-redis/v9"
  )

  var slidingLimitScript = redis.NewScript(`
  	local key = KEYS[1]
  	local now = tonumber(ARGV[1])
  	local window = tonumber(ARGV[2])
  	local limit = tonumber(ARGV[3])
  	local clear_before = now - window

  	redis.call('ZREMRANGEBYSCORE', key, 0, clear_before)
  	local current_requests = redis.call('ZCARD', key)

  	if current_requests < limit then
  		redis.call('ZADD', key, now, now)
  		redis.call('EXPIRE', key, math.ceil(window / 1000) + 10)
  		return 1
  	else
  		return 0
  	end
  `)

  func SlidingRateLimitMiddleware(rdb *redis.Client, window time.Duration, limit int64) gin.HandlerFunc {
  	return func(c *gin.Context) {
  		ip := c.ClientIP()
  		key := "bluebell:ratelimit:sliding:" + ip

  		now := time.Now().UnixNano() / int64(time.Millisecond)
  		windowMs := window.Milliseconds()

  		ctx := c.Request.Context()
  		allowed, err := slidingLimitScript.Run(ctx, rdb, []string{key}, now, windowMs, limit).Int()
  		if err != nil {
  			// Fallback: fail-open if Redis fails
  			c.Next()
  			return
  		}

  		if allowed == 0 {
  			c.Header("Retry-After", "1")
  			c.Header("X-RateLimit-Limit", "rate-limited")
  			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
  				"error": entity.ErrRateLimitExceeded.Error(),
  			})
  			return
  		}

  		c.Next()
  	}
  }
  ```

- [ ] **Step 4: Run tests to verify they pass**
  Run: `go test -v ./internal/middleware`
  Expected: PASS

- [ ] **Step 5: Commit**
  ```bash
  git add internal/middleware/ratelimit_sliding.go internal/middleware/ratelimit_sliding_test.go
  git commit -m "feat: implement sliding window rate limiter middleware and test"
  ```

---

### Task 5: Integrate Middleware into Router and Main Entrypoint

**Files:**
- Modify: [internal/interfaces/http/router/router.go](file:///D:/download/project/bluebell/internal/interfaces/http/router/router.go)
- Modify: [cmd/server/main.go](file:///D:/download/project/bluebell/cmd/server/main.go)

- [ ] **Step 1: Update router signature and register middleware**
  In [router.go](file:///D:/download/project/bluebell/internal/interfaces/http/router/router.go):
  - Add `rdb *redis.Client` to `NewRouter` parameters.
  - Parse the sliding rate limit duration if enabled.
  - Apply the new `middleware.SlidingRateLimitMiddleware` conditionally.
  
- [ ] **Step 2: Update router call in main.go**
  In [main.go](file:///D:/download/project/bluebell/cmd/server/main.go):
  - Pass the initialized `rdb` as the final argument to `router.NewRouter`.

- [ ] **Step 3: Run full app build**
  Run: `go build ./cmd/server`
  Expected: Successful compilation.

- [ ] **Step 4: Commit**
  ```bash
  git add internal/interfaces/http/router/router.go cmd/server/main.go
  git commit -m "feat: integrate sliding rate limit middleware into router and server main"
  ```
