package redis

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	postResp "bluebell/internal/dto/response/post"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type localCacheItem struct {
	data      *postResp.DetailResponse
	expiresAt time.Time
}

// LocalPostCache L1 进程内存本地缓存（低延迟、零网络开销、由 Redis 6.0 CSC Bcast 广播失效驱动）
type LocalPostCache struct {
	mu    sync.RWMutex
	items map[string]localCacheItem
}

// NewLocalPostCache 创建本地缓存实例
func NewLocalPostCache() *LocalPostCache {
	return &LocalPostCache{
		items: make(map[string]localCacheItem, 1024),
	}
}

// Get 从本地内存读取
func (c *LocalPostCache) Get(id string) (*postResp.DetailResponse, bool) {
	c.mu.RLock()
	item, exists := c.items[id]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}
	if time.Now().After(item.expiresAt) {
		c.Delete(id)
		return nil, false
	}
	return item.data, true
}

// Set 写入本地内存
func (c *LocalPostCache) Set(id string, data *postResp.DetailResponse, ttl time.Duration) {
	if data == nil || id == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	// 简单容量清理防护：超过 5000 条时清理已过期的 key
	if len(c.items) > 5000 {
		now := time.Now()
		for k, v := range c.items {
			if now.After(v.expiresAt) {
				delete(c.items, k)
			}
		}
	}

	c.items[id] = localCacheItem{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
}

// Delete 删除本地内存单个 key
func (c *LocalPostCache) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, id)
}

// FlushAll 清空本地所有缓存（全量失效时触发）
func (c *LocalPostCache) FlushAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = make(map[string]localCacheItem, 1024)
}

// ============================================================================
// Redis 6.0+ 客户端缓存广播模式 (Client-Side Caching: CLIENT TRACKING on bcast)
// ============================================================================

// BcastTracker Redis 6.0+ 客户端缓存跟踪器（广播模式 BCAST）
// 原理：
// 1. 建立专用的 Invalidation 订阅连接，获取其 CLIENT ID 并订阅 `__redis__:invalidate` 频道；
// 2. 主连接执行 `CLIENT TRACKING on bcast redirect <subClientID> prefixes <prefixes...>`；
// 3. 当任何应用节点或 Redis 修改/删除匹配前缀的 Key 时，Redis 自动广播推送失效事件；
// 4. Tracker 收到通知后，精准且即时地清理本地 L1 缓存，实现跨实例毫秒级缓存一致性。
type BcastTracker struct {
	rdb        *goredis.Client
	localCache *LocalPostCache
	prefixes   []string
	subClient  *goredis.Client
	pubsub     *goredis.PubSub
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewBcastTracker 创建 Redis 客户端缓存广播跟踪器
func NewBcastTracker(rdb *goredis.Client, localCache *LocalPostCache, prefixes ...string) *BcastTracker {
	if len(prefixes) == 0 {
		prefixes = []string{redisKey(keyPostDetailPrefix)}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &BcastTracker{
		rdb:        rdb,
		localCache: localCache,
		prefixes:   prefixes,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动客户端缓存广播追踪监听
func (t *BcastTracker) Start() error {
	if t.rdb == nil || t.localCache == nil {
		return nil
	}

	opts := t.rdb.Options()
	t.subClient = goredis.NewClient(opts)

	// 1. 获取订阅连接的 Client ID
	clientID, err := t.subClient.ClientID(t.ctx).Result()
	if err != nil {
		zap.L().Warn("get redis client id failed, fallback to local TTL cache only", zap.Error(err))
		return fmt.Errorf("get subClient ID failed: %w", err)
	}

	// 2. 订阅 Redis 6+ 原生失效广播频道
	t.pubsub = t.subClient.Subscribe(t.ctx, "__redis__:invalidate")

	// 3. 在主连接上开启 CLIENT TRACKING 广播模式并重定向至订阅连接
	args := []interface{}{"CLIENT", "TRACKING", "on", "bcast", "redirect", clientID}
	for _, prefix := range t.prefixes {
		args = append(args, "prefixes", prefix)
	}

	if err := t.rdb.Do(t.ctx, args...).Err(); err != nil {
		zap.L().Warn("enable CLIENT TRACKING bcast failed (Redis may be < 6.0 or mock), fallback to TTL", zap.Error(err))
		_ = t.pubsub.Close()
		_ = t.subClient.Close()
		return fmt.Errorf("enable CLIENT TRACKING failed: %w", err)
	}

	zap.L().Info("Redis 6.0+ Client-Side Caching (BCAST mode) tracking started",
		zap.Int64("redirect_client_id", clientID),
		zap.Strings("prefixes", t.prefixes))

	// 4. 启动后台协程消费失效事件
	t.wg.Add(1)
	go t.listenInvalidations()

	return nil
}

// listenInvalidations 循环监听 Redis 推送的失效 Key
func (t *BcastTracker) listenInvalidations() {
	defer t.wg.Done()
	ch := t.pubsub.Channel()

	for {
		select {
		case <-t.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			t.handleInvalidateMessage(msg)
		}
	}
}

// handleInvalidateMessage 处理失效消息并从本地内存剔除
func (t *BcastTracker) handleInvalidateMessage(msg *goredis.Message) {
	if msg == nil {
		return
	}

	var keys []string
	if len(msg.PayloadSlice) > 0 {
		keys = msg.PayloadSlice
	} else if msg.Payload != "" {
		keys = []string{msg.Payload}
	}

	if len(keys) == 0 {
		// Redis 发送 nil 表示全量前缀失效
		t.localCache.FlushAll()
		return
	}

	for _, rawKey := range keys {
		for _, prefix := range t.prefixes {
			if strings.HasPrefix(rawKey, prefix) {
				id := strings.TrimPrefix(rawKey, prefix)
				t.localCache.Delete(id)
				zap.L().Debug("L1 LocalCache invalidated via Redis BCAST tracking",
					zap.String("key", rawKey),
					zap.String("id", id))
				break
			}
		}
	}
}

// Stop 停止追踪并释放资源
func (t *BcastTracker) Stop() {
	t.cancel()

	// 关闭客户端 tracking
	if t.rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = t.rdb.Do(ctx, "CLIENT", "TRACKING", "off").Err()
		cancel()
	}

	if t.pubsub != nil {
		_ = t.pubsub.Close()
	}
	if t.subClient != nil {
		_ = t.subClient.Close()
	}

	t.wg.Wait()
	zap.L().Info("Redis 6.0+ Client-Side Caching (BCAST mode) tracking stopped")
}
