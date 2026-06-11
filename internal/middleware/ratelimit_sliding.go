package middleware

import (
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
	// WARNING: timeNow is a package-level variable and is not thread-safe for parallel tests.
	// timeNow is a package-level variable that can be stubbed in tests.
	timeNow        = time.Now
	processID      = uuid.NewString()
	requestCounter uint64
)

var slidingLimitScript = redis.NewScript(`
	local key = KEYS[1]
	local now = tonumber(ARGV[1])
	local window = tonumber(ARGV[2])
	local limit = tonumber(ARGV[3])
	local member = ARGV[4]
	local clear_before = now - window

	redis.call('ZREMRANGEBYSCORE', key, '-inf', clear_before)
	local current_requests = redis.call('ZCARD', key)

	if current_requests < limit then
		redis.call('ZADD', key, now, member)
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

		now := timeNow().UnixMilli()
		windowMs := window.Milliseconds()
		count := atomic.AddUint64(&requestCounter, 1)
		member := strconv.FormatInt(now, 10) + "-" + processID + "-" + strconv.FormatUint(count, 10)

		ctx := c.Request.Context()
		allowed, err := slidingLimitScript.Run(ctx, rdb, []string{key}, now, windowMs, limit, member).Int()
		if err != nil {
			// Fallback: fail-open if Redis fails
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
