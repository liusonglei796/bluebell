# Token Bucket (令牌桶) 限流实现文档

本文档整理了项目中现有的、基于 `github.com/juju/ratelimit` 实现的单机令牌桶限流的全部代码与配置。

---

## 1. 配置文件定义与结构体

### 1.1 配置文件

在项目的配置文件中通过 `ratelimit` 的配置项定义填充间隔和容量：

**`config.yaml`**:
```yaml
ratelimit:
  # 彻底关闭限流，观察物理瓶颈（开发/调试配置）
  fill_interval: "1ns"
  capacity: 999999999999
```

**`config.toml`** / **`config.docker.toml`**:
```toml
[ratelimit]
fill_interval = "10ms"
capacity = 200
```

### 1.2 Go 配置结构体
在 [internal/config/config.go](file:///D:/download/project/bluebell/internal/config/config.go) 中映射：

```go
type rateLimitConfig struct {
	FillInterval string `mapstructure:"fill_interval"`
	Capacity     int64  `mapstructure:"capacity"`
}

type Config struct {
	// ... 其他配置项
	RateLimit        *rateLimitConfig        `mapstructure:"ratelimit"`
}
```

---

## 2. 令牌桶限流中间件

实现文件为 [internal/middleware/ratelimit_token.go](file:///D:/download/project/bluebell/internal/middleware/ratelimit_token.go)：

```go
package middleware

import (
	"net/http"
	"sync/atomic"
	"time"

	"bluebell/internal/domain/entity"

	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
	"go.uber.org/zap"
)

var (
	totalRequests   int64 // 统计收到的请求总数
	limitedRequests int64 // 统计被限流拦截的请求总数
)

// RateLimitMiddleware 构造基于 github.com/juju/ratelimit 的单机全局令牌桶限流中间件
func RateLimitMiddleware(fillInterval time.Duration, capacity int64) gin.HandlerFunc {
	// 初始化令牌桶：每过 fillInterval 产生一个令牌，最大容量为 capacity
	bucket := ratelimit.NewBucket(fillInterval, capacity)

	zap.L().Info("RateLimitMiddleware initialized",
		zap.Duration("fillInterval", fillInterval),
		zap.Int64("capacity", capacity),
		zap.Float64("ratePerSecond", float64(time.Second)/float64(fillInterval)))

	return func(c *gin.Context) {
		atomic.AddInt64(&totalRequests, 1)

		// 尝试非阻塞式地获取 1 个令牌
		if bucket.TakeAvailable(1) < 1 {
			// 获取失败，被限流拦截
			atomic.AddInt64(&limitedRequests, 1)
			limited := atomic.LoadInt64(&limitedRequests)
			total := atomic.LoadInt64(&totalRequests)
			zap.L().Warn("Rate limit triggered",
				zap.Int64("limited", limited),
				zap.Int64("total", total),
				zap.Float64("rate", float64(limited)/float64(total)*100))

			// 设置响应头并拦截请求
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

## 3. 服务注册与集成

### 3.1 路由层集成
在 [internal/interfaces/http/router/router.go](file:///D:/download/project/bluebell/internal/interfaces/http/router/router.go) 中实例化并应用为全局中间件：

```go
func NewRouter(
	mode string,
	hp *handler.Provider,
	cfg *config.Config,
	tokenService domain.TokenService,
	tokenCache domain.UserTokenCacheRepository,
	rdb *redis.Client,
) (*gin.Engine, error) {

	r := gin.New()

	// 1. 解析填充时间间隔
	fillInterval, err := time.ParseDuration(cfg.RateLimit.FillInterval)
	if err != nil {
		return nil, fmt.Errorf("parse rate limit fill interval failed: %w", err)
	}

	timeout, err := time.ParseDuration(cfg.Timeout.Timeout)
	if err != nil {
		return nil, fmt.Errorf("parse request timeout failed: %w", err)
	}

	// 2. 注册中间件
	r.Use(
		otelgin.Middleware("bluebell"), 
		middleware.GinLogger(),
		middleware.GinRecovery(true),
		middleware.Cors(), 
		// 应用令牌桶限流中间件到全部路由（全局限流）
		middleware.RateLimitMiddleware(fillInterval, cfg.RateLimit.Capacity), 
	)

	// ... 其他中间件和路由定义
}
```

---

## 4. 令牌桶算法特性 analysis

### 4.1 算法特点
1. **支持应对突发流量**：令牌桶最多可存放 `capacity` 个令牌。当流量突发时，系统可在瞬间消耗完积攒的令牌（并发量最高为 `capacity`），随后请求速度将被限制在配置的产生速率内。
2. **平滑限流**：即使有瞬时流量过大，也能通过平滑地往桶里补充令牌，来维持稳定的出水速度，比漏桶算法更为灵活。

### 4.2 本项目现有的设计局限
1. **全局限流，非 IP/用户维度**：本项目现有的 `RateLimitMiddleware` 令牌桶是在内存中维护一个全局单例 Bucket，对所有 API 请求进行统一限流，而不是针对独立 IP 或 User 进行限制。如果需要针对单个 IP 限流，更适合采用 Redis 滑动窗口限流。
2. **单机限流，非分布式**：该令牌桶状态完全保存在当前运行的进程内存中，不通过外部存储（如 Redis）共享。当项目有多实例集群部署时，各实例之间的令牌状态是隔离的，因此总体流量上限等于 `实例数 * capacity`。
