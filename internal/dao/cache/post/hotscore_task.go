package postcache

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// HotScoreRefresherConfig 热度刷新配置
type HotScoreRefresherConfig struct {
	RefreshInterval time.Duration // 刷新间隔（默认 5 分钟）
	BatchSize       int64         // 每批处理的帖子数（默认 500）
	Enabled         bool          // 是否启用
}

// DefaultHotScoreRefresherConfig 默认配置
// [防御] 提供合理的默认值，避免调用方忘记传 config 时出现零值行为
var DefaultHotScoreRefresherConfig = HotScoreRefresherConfig{
	RefreshInterval: 5 * time.Minute,
	BatchSize:       500,
	Enabled:         true,
}

// HotScoreRefresher 定时刷新 Gravity 分数
type HotScoreRefresher struct {
	rdb    *redis.Client
	cache  *cacheStruct
	config *HotScoreRefresherConfig
	stopCh chan struct{} // [防御] struct{} 零内存开销，比 chan bool 更语义化
	doneCh chan struct{} // [防御] 用于 Stop() 等待 goroutine 完全退出，防止 goroutine 泄漏
}

// NewHotScoreRefresher 创建热度刷新器
func NewHotScoreRefresher(rdb *redis.Client, cache *cacheStruct, config *HotScoreRefresherConfig) *HotScoreRefresher {
	// [防御] config 为 nil 时用默认配置，避免后续访问 config.RefreshInterval 等字段时出现零值 panic
	// time.Duration 零值是 0，会导致 ticker 无限循环触发
	if config == nil {
		config = &DefaultHotScoreRefresherConfig
	}
	return &HotScoreRefresher{
		rdb:    rdb,
		cache:  cache,
		config: config,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// Start 启动刷新循环
func (r *HotScoreRefresher) Start() {
	// [防御] 支持通过配置禁用，方便测试环境和紧急关闭
	if !r.config.Enabled {
		log.Println("[HotScoreRefresher] disabled, not starting")
		// [防御] 必须关闭 doneCh，否则 Stop() 会永久阻塞在 <-r.doneCh
		close(r.doneCh)
		return
	}

	log.Printf("[HotScoreRefresher] started, interval: %v, batch size: %d",
		r.config.RefreshInterval, r.config.BatchSize)

	go r.runLoop()
}

// Stop 停止刷新循环，等待退出
func (r *HotScoreRefresher) Stop() {
	close(r.stopCh)
	// [防御] 阻塞等待 goroutine 退出，确保不会在刷新进行中关闭 Redis 连接
	// 这是 main.go 中 defer Stop() 的核心安全保障
	<-r.doneCh
	log.Println("[HotScoreRefresher] stopped")
}

// runLoop 运行刷新循环
func (r *HotScoreRefresher) runLoop() {
	// [防御] 无论正常退出还是 panic，都关闭 doneCh 通知 Stop() 返回
	defer close(r.doneCh)

	// 立即执行一次：服务启动时立刻刷新，而不是等第一个 interval
	// [防御] 如果 Redis 中已有旧数据（比如重启前），启动时立刻刷新避免排行榜短暂失真
	r.refreshAll()

	ticker := time.NewTicker(r.config.RefreshInterval)
	// [防御] ticker 必须 Stop，否则 goroutine 泄漏 + 定时器资源泄漏
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			// [防御] 收到停止信号后直接 return，defer 会关闭 doneCh
			return
		case <-ticker.C:
			r.refreshAll()
		}
	}
}

// refreshAll 全量刷新所有帖子的 Gravity 分数
func (r *HotScoreRefresher) refreshAll() {
	startTime := time.Now()
	totalProcessed := 0

	log.Println("[HotScoreRefresher] starting batch refresh...")

	// [防御] 设置 2 分钟超时，防止：
	// 1. Redis 响应慢导致整个 refreshAll 永久阻塞
	// 2. 大量帖子时全量刷新耗时过长，影响正常投票请求的 Redis 性能
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var start int64 = 0
	for {
		// [防御] ZRange 按 start/end 索引分批取帖子 ID，避免一次取全量导致：
		// 1. Redis 内存峰值过高（百万级帖子时 ZRange 0 -1 会返回所有 key）
		// 2. 单次响应超时
		postIDs, err := r.rdb.ZRange(ctx, redisKey(keyPostTimeZSet), start, start+r.config.BatchSize-1).Result()
		if err != nil {
			// [防御] ZRange 失败时 break 而不是 panic
			// 可能是 Redis 连接断开或 context 超时，记录日志后退出即可
			log.Printf("[HotScoreRefresher] ZRange error: %v", err)
			break
		}
		if len(postIDs) == 0 {
			break
		}

		// [防御] 单批刷新失败只记日志，不中断整个循环
		// 即使这批帖子刷新失败，下一批仍然可以继续
		if err := r.cache.batchRefreshGravityScores(ctx, postIDs); err != nil {
			log.Printf("[HotScoreRefresher] batch refresh error: %v", err)
		}

		totalProcessed += len(postIDs)
		start += r.config.BatchSize

		// [防御] 最后一批帖子数量 < BatchSize 说明已经取完了，提前退出
		// 避免多一次 ZRange 调用返回空结果再 break
		if int64(len(postIDs)) < r.config.BatchSize {
			break
		}
	}

	log.Printf("[HotScoreRefresher] completed, processed %d posts, took %v",
		totalProcessed, time.Since(startTime))
}
