# Redis 滑动窗口限流实现文档

本文档整理了项目中基于 Redis `ZSET` 和 Lua 脚本实现的分布式滑动窗口限流的全部代码与配置。

---

## 1. 配置文件定义与结构体

### 1.1 配置文件

在项目的配置文件中添加了 `sliding_ratelimit` 的配置项：

**`config.yaml`**:
```yaml
sliding_ratelimit:
  enabled: true
  window: "60s"
  limit: 100
```

**`config.toml`** / **`config.docker.toml`**:
```toml
[sliding_ratelimit]
enabled = true
window = "60s"
limit = 100
```

### 1.2 Go 配置结构体
在 [internal/config/config.go](file:///D:/download/project/bluebell/internal/config/config.go) 中映射：

```go
type slidingRateLimitConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Window  string `mapstructure:"window"`
	Limit   int64  `mapstructure:"limit"`
}

type Config struct {
	// ... 其他配置项
	RateLimit        *rateLimitConfig        `mapstructure:"ratelimit"`
	SlidingRateLimit *slidingRateLimitConfig `mapstructure:"sliding_ratelimit"`
}
```

---

## 2. 滑动窗口限流中间件

实现文件为 [internal/middleware/ratelimit_sliding.go](file:///D:/download/project/bluebell/internal/middleware/ratelimit_sliding.go)：

```go
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"bluebell/internal/domain/entity"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	// processID 在启动时生成一次，确保多实例部署时 ZSET 成员的唯一性
	processID      = uuid.NewString()
	requestCounter uint64
	// timeNow 便于在单元测试中 Mock 时间
	// ⚠️ 注意：此全局变量在测试中会被篡改，不可用于并行测试 (t.Parallel())
	timeNow = time.Now
)

var slidingLimitScript = redis.NewScript(`
	local key = KEYS[1]
	local now = tonumber(ARGV[1])
	local window = tonumber(ARGV[2])
	local limit = tonumber(ARGV[3])
	local member = ARGV[4]
	local clear_before = now - window

	-- 1. 清理当前窗口之前的过期历史数据
	redis.call('ZREMRANGEBYSCORE', key, '-inf', clear_before)

	-- 2. 统计当前窗口内的请求总数
	local current_requests = redis.call('ZCARD', key)

	-- 3. 判断是否超出阈值限制
	if current_requests < limit then
		-- 未超限：添加当前请求成员
		redis.call('ZADD', key, now, member)
		-- 设置过期时间（窗口时长 + 10秒缓冲），防止冷 IP 数据长期滞留 Redis
		redis.call('EXPIRE', key, math.ceil(window / 1000) + 10)
		return 1 -- 允许访问
	else
		return 0 -- 限流拦截
	end
`)

// SlidingRateLimitMiddleware 构造基于 Redis ZSET 的 IP 滑动窗口限流中间件
func SlidingRateLimitMiddleware(rdb *redis.Client, window time.Duration, limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		key := "bluebell:ratelimit:sliding:" + ip

		// 毫秒时间戳与本地原子自增计数器结合，确保单实例高并发请求 member 唯一
		now := timeNow().UnixNano() / int64(time.Millisecond)
		windowMs := window.Milliseconds()
		count := atomic.AddUint64(&requestCounter, 1)
		member := strconv.FormatInt(now, 10) + "-" + processID + "-" + strconv.FormatUint(count, 10)

		ctx := c.Request.Context()
		allowed, err := slidingLimitScript.Run(ctx, rdb, []string{key}, now, windowMs, limit, member).Int()
		if err != nil {
			// Fail-open: Redis 故障时降级放行，但打印错误日志便于监控报警
			zap.L().Error("sliding rate limit check failed, failing open", zap.Error(err))
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

---

## 3. 中间件单元测试

基于 `miniredis` 实现的单元测试 [internal/middleware/ratelimit_sliding_test.go](file:///D:/download/project/bluebell/internal/middleware/ratelimit_sliding_test.go)：

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

	// 1. 初始化内存 Redis 实例
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer rdb.Close()

	// 2. Mock 单元测试的时钟源
	baseTime := time.Date(2026, 6, 11, 12, 0, 0, 0, time.UTC)
	timeNow = func() time.Time {
		return baseTime
	}
	defer func() {
		timeNow = time.Now // 测试完成后恢复默认时钟源
	}()

	// 3. 构建测试路由（限流规则：200ms 内限制最多访问 2 次）
	r := gin.New()
	r.Use(SlidingRateLimitMiddleware(rdb, 200*time.Millisecond, 2))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// 第 1 次请求：应当放行
	req1, _ := http.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// 第 2 次请求：应当放行
	req2, _ := http.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	// 第 3 次请求（超限）：应当返回 429
	req3, _ := http.NewRequest("GET", "/test", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	assert.Equal(t, http.StatusTooManyRequests, w3.Code)
	assert.Equal(t, "rate-limited", w3.Header().Get("X-RateLimit-Limit"))
	assert.Contains(t, w3.Body.String(), entity.ErrRateLimitExceeded.Error())

	// 4. 将时钟快进 300ms（超出限流时间窗口），验证是否能恢复放行
	baseTime = baseTime.Add(300 * time.Millisecond)
	mr.FastForward(300 * time.Millisecond)

	// 第 4 次请求：恢复放行
	req4, _ := http.NewRequest("GET", "/test", nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	assert.Equal(t, http.StatusOK, w4.Code)
}
```

---

## 4. 服务注册与集成

### 4.1 路由层声明
在 [internal/interfaces/http/router/router.go](file:///D:/download/project/bluebell/internal/interfaces/http/router/router.go) 中：

```go
func NewRouter(
	mode string,
	hp *handler.Provider,
	cfg *config.Config,
	tokenService domain.TokenService,
	tokenCache domain.UserTokenCacheRepository,
	rdb *redis.Client, // 注入 Redis 客户端
) (*gin.Engine, error) {

	r := gin.New()

	fillInterval, err := time.ParseDuration(cfg.RateLimit.FillInterval)
	if err != nil {
		return nil, fmt.Errorf("parse rate limit fill interval failed: %w", err)
	}

	timeout, err := time.ParseDuration(cfg.Timeout.Timeout)
	if err != nil {
		return nil, fmt.Errorf("parse request timeout failed: %w", err)
	}

	// 解析滑动窗口时间配置
	var slidingWindow time.Duration
	if cfg.SlidingRateLimit != nil && cfg.SlidingRateLimit.Enabled {
		if rdb == nil {
			return nil, fmt.Errorf("redis client (rdb) is required when sliding rate limit is enabled")
		}
		slidingWindow, err = time.ParseDuration(cfg.SlidingRateLimit.Window)
		if err != nil {
			return nil, fmt.Errorf("parse sliding rate limit window failed: %w", err)
		}
	}

	r.Use(
		otelgin.Middleware("bluebell"), 
		middleware.GinLogger(),
		middleware.GinRecovery(true),
		middleware.Cors(), 
		middleware.RateLimitMiddleware(fillInterval, cfg.RateLimit.Capacity), // 保留原令牌桶限流
	)

	// 若启用滑动窗口限流则应用中间件
	if cfg.SlidingRateLimit != nil && cfg.SlidingRateLimit.Enabled {
		r.Use(middleware.SlidingRateLimitMiddleware(rdb, slidingWindow, cfg.SlidingRateLimit.Limit))
	}

	r.Use(middleware.TimeoutMiddleware(timeout))
    
    // ... 注册 API 路由组
}
```

### 4.2 入口层注入
在 [cmd/server/main.go](file:///D:/download/project/bluebell/cmd/server/main.go) 中实例化并注入：

```go
func main() {
    // ... 初始化配置和基础资源
	rdb, err := redisrepo.Init(cfg)
	if err != nil {
		zap.L().Fatal("Init Redis failed", zap.Error(err))
	}
	defer redisrepo.Close(rdb)

    // ... 实例化 Service 和 Provider
	r, err := router.NewRouter(cfg.App.Mode, hp, cfg, tokenService, cacheRepos.TokenCache, rdb)
	if err != nil {
		zap.L().Fatal("init router failed", zap.Error(err))
	}
    // ... 启动 HTTP 服务
}
```
