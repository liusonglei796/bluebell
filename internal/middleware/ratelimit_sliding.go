package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"bluebell/internal/backfront"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// SlidingWindowRateLimit 基于 Redis ZSET 的滑动窗口限流中间件
//
// 参数:
//   - rdb:         Redis 客户端实例
//   - window:      滑动窗口大小（如 1*time.Minute）
//   - maxRequests: 窗口内允许的最大请求数
//   - keyFunc:     自定义限流维度函数（返回客户端 IP、用户 ID 等）
//
// 使用示例:
//
//	// 按 IP 限流：每分钟最多 100 次请求
//	middleware.SlidingWindowRateLimit(rdb, time.Minute, 100, func(c *gin.Context) string {
//	    return c.ClientIP()
//	})
//
//	// 按用户 ID 限流：每分钟最多 50 次请求
//	middleware.SlidingWindowRateLimit(rdb, time.Minute, 50, func(c *gin.Context) string {
//	    userID, _ := c.Get("userID")
//	    return fmt.Sprintf("user:%v", userID)
//	})
func SlidingWindowRateLimit(rdb *redis.Client, window time.Duration, maxRequests int64, keyFunc func(c *gin.Context) string) gin.HandlerFunc {
	// Lua 脚本保证原子性：清理窗口外数据 → 检查数量 → 添加记录 → 设置过期时间
	// KEYS[1] = ZSET key
	// ARGV[1] = 窗口起始时间戳（秒级浮点数）
	// ARGV[2] = 当前时间戳（秒级浮点数，用作 score）
	// ARGV[3] = 唯一 member（纳秒时间戳 + 随机后缀）
	// ARGV[4] = 最大请求数
	// ARGV[5] = 过期时间（秒）= window * 2
	// 返回值: 0 = 允许通过, 1 = 限流触发
	const luaScript = `
		-- 清理窗口外的旧数据
		redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', ARGV[1])
		-- 获取当前窗口内的请求数量
		local count = redis.call('ZCARD', KEYS[1])
		-- 检查是否超过限制
		if count >= tonumber(ARGV[4]) then
			return 1
		end
		-- 添加当前请求记录
		redis.call('ZADD', KEYS[1], ARGV[2], ARGV[3])
		-- 设置过期时间，防止内存泄漏（窗口大小的 2 倍）
		redis.call('EXPIRE', KEYS[1], tonumber(ARGV[5]))
		return 0
	`

	// 预加载 Lua 脚本获取 SHA，提升后续执行效率
	sha := rdb.ScriptLoad(context.Background(), luaScript).Val()
	expireSeconds := int64(window.Seconds() * 2)

	return func(c *gin.Context) {
		key := fmt.Sprintf("bluebell:ratelimit:%s", keyFunc(c))

		now := time.Now()
		windowStart := float64(now.Add(-window).UnixNano()) / 1e9
		nowScore := float64(now.UnixNano()) / 1e9
		member := strconv.FormatInt(now.UnixNano(), 10) + randomHex(8)

		// 优先使用 EvalSha（已缓存脚本），失败时降级 Eval（自动加载脚本）
		result, err := rdb.EvalSha(
			context.Background(),
			sha,
			[]string{key},
			windowStart,
			nowScore,
			member,
			maxRequests,
			expireSeconds,
		).Int()

		if err == redis.Nil {
			// SHA 不存在（Redis 重启后脚本丢失），使用 Eval 重新加载
			result, err = rdb.Eval(
				context.Background(),
				luaScript,
				[]string{key},
				windowStart,
				nowScore,
				member,
				maxRequests,
				expireSeconds,
			).Int()
		}

		if err != nil {
			// Redis 异常时放行请求，避免影响正常业务
			c.Next()
			return
		}

		if result == 1 {
			c.Header("Retry-After", strconv.FormatInt(int64(window.Seconds()), 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, &backfront.ResponseData{
				Code: errorx.CodeRateLimitExceeded,
				Msg:  errorx.ErrRateLimitExceeded.Error(),
				Data: nil,
			})
			return
		}

		c.Next()
	}
}

// randomHex 生成指定字节长度的随机十六进制字符串
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// 降级方案：使用时间戳
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b)
}
