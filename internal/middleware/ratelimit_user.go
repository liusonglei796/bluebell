package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"bluebell/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/juju/ratelimit"
	"go.uber.org/zap"
)

// UserLimiter 管理基于用户/IP维度的独立令牌桶集合
type UserLimiter struct {
	mu           sync.RWMutex
	buckets      map[string]*userBucketEntry
	fillInterval time.Duration
	capacity     int64
	idleTTL      time.Duration
	stopCh       chan struct{}
}

type userBucketEntry struct {
	bucket       *ratelimit.Bucket
	lastSeenNano atomic.Int64 // 纳秒时间戳，保障在 RLock 下安全更新防 data race
}

// NewUserLimiter 创建用户维度限流器
// fillInterval: 每个令牌的填充时间间隔 (如 500ms 表示 2 QPS)
// capacity: 令牌桶容量 (最大突发请求数)
// idleTTL: 桶闲置多久后自动回收释放内存 (如 10 分钟)
func NewUserLimiter(fillInterval time.Duration, capacity int64, idleTTL time.Duration) *UserLimiter {
	if idleTTL <= 0 {
		idleTTL = 10 * time.Minute
	}
	limiter := &UserLimiter{
		buckets:      make(map[string]*userBucketEntry),
		fillInterval: fillInterval,
		capacity:     capacity,
		idleTTL:      idleTTL,
		stopCh:       make(chan struct{}),
	}

	// 启动后台定时清理协程，防止长时间运行下 map 无限膨胀
	go limiter.cleanupLoop()

	return limiter
}

func (l *UserLimiter) cleanupLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.stopCh:
			return
		}
	}
}

func (l *UserLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	nowNano := time.Now().UnixNano()
	ttlNano := l.idleTTL.Nanoseconds()
	for key, entry := range l.buckets {
		if nowNano-entry.lastSeenNano.Load() > ttlNano {
			delete(l.buckets, key)
		}
	}
}

// Close 停止后台清理任务
func (l *UserLimiter) Close() {
	select {
	case <-l.stopCh:
	default:
		close(l.stopCh)
	}
}

// getBucket 获取或创建指定 Key 的令牌桶 (采用读写锁分离优化高并发)
func (l *UserLimiter) getBucket(key string) *ratelimit.Bucket {
	nowNano := time.Now().UnixNano()

	// 1. 快路径：先尝试读锁获取已存在的桶
	l.mu.RLock()
	entry, ok := l.buckets[key]
	if ok {
		entry.lastSeenNano.Store(nowNano)
		bucket := entry.bucket
		l.mu.RUnlock()
		return bucket
	}
	l.mu.RUnlock()

	// 2. 慢路径：写锁创建新桶 (Double Check 机制)
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry, ok = l.buckets[key]; ok {
		entry.lastSeenNano.Store(nowNano)
		return entry.bucket
	}

	bucket := ratelimit.NewBucket(l.fillInterval, l.capacity)
	newEntry := &userBucketEntry{
		bucket: bucket,
	}
	newEntry.lastSeenNano.Store(nowNano)
	l.buckets[key] = newEntry
	return bucket
}

// UserRateLimitMiddleware 创建基于用户维度的令牌桶限流中间件
// 优先提取 Context 中的 UserIDKey，若未登录则降级按 ClientIP 隔离
func UserRateLimitMiddleware(fillInterval time.Duration, capacity int64) gin.HandlerFunc {
	limiter := NewUserLimiter(fillInterval, capacity, 10*time.Minute)

	zap.L().Info("UserRateLimitMiddleware initialized",
		zap.Duration("fillInterval", fillInterval),
		zap.Int64("capacity", capacity),
		zap.Float64("ratePerUserSec", float64(time.Second)/float64(fillInterval)))

	return func(c *gin.Context) {
		var limitKey string
		if uidVal, exists := c.Get("UserIDKey"); exists {
			switch v := uidVal.(type) {
			case int64:
				limitKey = "user:" + strconv.FormatInt(v, 10)
			case string:
				limitKey = "user:" + v
			default:
				limitKey = fmt.Sprintf("user:%v", v)
			}
		} else {
			limitKey = "ip:" + c.ClientIP()
		}

		bucket := limiter.getBucket(limitKey)
		// juju/ratelimit 的 TakeAvailable 是内部线程安全的
		if bucket.TakeAvailable(1) < 1 {
			c.Header("Retry-After", "1")
			c.Header("X-RateLimit-Limit", "user-rate-limited")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":  1014,
				"msg":   model.ErrRateLimitExceeded.Error(),
				"error": model.ErrRateLimitExceeded.Error(),
			})
			return
		}

		c.Next()
	}
}
