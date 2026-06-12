# Design Spec: Redis-Based Sliding Window Rate Limiter

## 1. Overview
The goal is to implement a distributed, IP-based sliding window rate limiter using Redis to protect the Bluebell application APIs. This rate limiter will run alongside the existing local Token Bucket rate limiter, and can be enabled/disabled via configuration independently.

## 2. Requirements & Goals
* **Distributed Rate Limiting**: Limit tracking must be stored in Redis so that multiple instances of the backend share the same rate limits per client IP.
* **IP-based Limit**: Limit is applied per client IP address (`c.ClientIP()`).
* **Sliding Window Algorithm**: Rather than a fixed-window (which suffers from bursts at the boundary), a sliding window provides smooth limit enforcement across any sliding duration (e.g., 100 requests per 60 seconds).
* **Atomicity**: The checks and increments must be atomic to prevent race conditions during concurrent client requests. This is achieved using a Redis Lua script.
* **Configurability**: Toggleable independently, with configurable window duration and request limit.

## 3. Configuration Design
We add a new config block `sliding_ratelimit` to the configuration structure.

### 3.1 Go Struct Updates
In [internal/config/config.go](file:///D:/download/project/bluebell/internal/config/config.go):
```go
type slidingRateLimitConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Window  string `mapstructure:"window"` // e.g., "60s", "1m"
	Limit   int64  `mapstructure:"limit"`  // max requests allowed per window
}

type Config struct {
    // ...
	RateLimit        *rateLimitConfig        `mapstructure:"ratelimit"`
	SlidingRateLimit *slidingRateLimitConfig `mapstructure:"sliding_ratelimit"`
    // ...
}
```

### 3.2 Configuration File Updates
We will add `sliding_ratelimit` to:
* [config.yaml](file:///D:/download/project/bluebell/config.yaml)
* [config.toml](file:///D:/download/project/bluebell/config.toml)
* [config.docker.toml](file:///D:/download/project/bluebell/config.docker.toml)

Example in `config.yaml`:
```yaml
sliding_ratelimit:
  enabled: true
  window: "60s"
  limit: 100
```

## 4. Redis Lua Script & Key Design
### 4.1 Key Structure
* Key: `bluebell:ratelimit:sliding:<ip>`
* Type: Sorted Set (ZSET)
* Member & Score: Timestamp of the request (Unix Nano / Milliseconds). Since members must be unique, we can use a microsecond/nanosecond-resolution timestamp, or append a random ID if collision is possible. Using unix millisecond timestamp along with a unique suffix (like `timestamp-uuid` or just unix nanosecond timestamp) ensures uniqueness.

### 4.2 Lua Script
```lua
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local clear_before = now - window

-- Remove expired entries older than the current window
redis.call('ZREMRANGEBYSCORE', key, 0, clear_before)

-- Count remaining requests in the window
local current_requests = redis.call('ZCARD', key)

if current_requests < limit then
    -- Add the request timestamp (member and score both = now)
    redis.call('ZADD', key, now, now)
    -- Refresh TTL so key expires automatically when inactive
    redis.call('EXPIRE', key, math.ceil(window / 1000) + 10)
    return 1 -- Allowed
else
    return 0 -- Rejected
end
```

## 5. Middleware Design
A new file [internal/middleware/ratelimit_sliding.go](file:///D:/download/project/bluebell/internal/middleware/ratelimit_sliding.go) will be created:
* Loads the Lua script using `redis.NewScript` on startup.
* Implements `SlidingRateLimitMiddleware(rdb *redis.Client, window time.Duration, limit int64) gin.HandlerFunc`.
* Retrieves the client IP using `c.ClientIP()`.
* Runs the script with:
  * Key: `bluebell:ratelimit:sliding:<ip>`
  * ARGV[1]: `now` (current unix epoch millisecond timestamp)
  * ARGV[2]: `window` (duration of sliding window in milliseconds)
  * ARGV[3]: `limit` (max capacity)
* If script returns `0`, abort the request with `http.StatusTooManyRequests (429)` and headers:
  * `Retry-After: 1`
  * `X-RateLimit-Limit: rate-limited`
  * Response JSON: `{"error": "rate limit exceeded"}`

## 6. Router & Main Integration
1. Update `NewRouter` in [internal/interfaces/http/router/router.go](file:///D:/download/project/bluebell/internal/interfaces/http/router/router.go) to accept `rdb *redis.Client` and conditionally apply the `SlidingRateLimitMiddleware` middleware.
2. Update `main.go` in [cmd/server/main.go](file:///D:/download/project/bluebell/cmd/server/main.go) to inject the `rdb` client into `NewRouter`.

## 7. Testing Strategy
* **Unit/Integration Test**: Write a test in `internal/middleware/ratelimit_sliding_test.go` using a mock or a real local Redis container/instance to verify requests are throttled correctly once the limit is hit, and recover after the window slides.
* **Manual Verification**: Run load test/scripts against local API server to verify rate limit headers and 429 response codes.
