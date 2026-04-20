package postcache

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// HotScoreRefresherConfig 热度刷新配置
type HotScoreRefresherConfig struct {
	RefreshInterval time.Duration
	BatchSize       int64
	Enabled         bool
}

var DefaultHotScoreRefresherConfig = HotScoreRefresherConfig{
	RefreshInterval: 5 * time.Minute,
	BatchSize:       500,
	Enabled:         true,
}

// HotScoreRefresher 定时刷新 Gravity 分数
type HotScoreRefresher struct {
	rdb    *redis.Client
	cache  *cacheStruct // 假设你已经在别处定义了这个结构体
	config *HotScoreRefresherConfig

	stopCh   chan struct{}  // 用于发送停止信号
	wg       sync.WaitGroup // 用于等待后台任务彻底结束
	stopOnce sync.Once      // 保证 stopCh 只被关闭一次，防止 panic
}

func NewHotScoreRefresher(rdb *redis.Client, cache *cacheStruct, config *HotScoreRefresherConfig) *HotScoreRefresher {
	if config == nil {
		config = &DefaultHotScoreRefresherConfig
	}
	return &HotScoreRefresher{
		rdb:    rdb,
		cache:  cache,
		config: config,
		stopCh: make(chan struct{}),
	}
}

func (r *HotScoreRefresher) Start() {
	if !r.config.Enabled {
		return
	}

	log.Printf("[HotScoreRefresher] started, interval: %v, batch size: %d",
		r.config.RefreshInterval, r.config.BatchSize)

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		ticker := time.NewTicker(r.config.RefreshInterval)
		defer ticker.Stop()

		for {
			// ================== 1. 执行批量刷新逻辑 ==================
			startTime := time.Now()
			totalProcessed := 0
			log.Println("[HotScoreRefresher] starting batch refresh...")

			var start int64 = 0
			for {
				// 每次批处理前检查是否收到 channel 停止信号
				select {
				case <-r.stopCh:
					log.Println("[HotScoreRefresher] refresh canceled mid-way")
					return
				default:
				}

				// 注意：Redis 操作依然需要 context 来控制单次网络请求的超时（5秒）
				batchCtx, batchCancel := context.WithTimeout(context.Background(), 5*time.Second)

				postIDs, err := r.rdb.ZRange(batchCtx, "bluebell:post:zset:time", start, start+r.config.BatchSize-1).Result()

				if err != nil {
					batchCancel()
					log.Printf("[HotScoreRefresher] ZRange error: %v", err)
					break
				}
				if len(postIDs) == 0 {
					batchCancel()
					break // 本次全量数据处理完毕
				}

				// 假设你的 cache 方法也接收 context 用于超时控制
				if err := r.cache.batchRefreshGravityScores(batchCtx, postIDs); err != nil {
					log.Printf("[HotScoreRefresher] batch refresh error: %v", err)
				}

				batchCancel() // 主动释放 context 资源不能在无限循环中用defer
			

				totalProcessed += len(postIDs)
				start += r.config.BatchSize

				// 稍微休息，缓解 Redis 和 CPU 压力
				time.Sleep(10 * time.Millisecond)
			}

			log.Printf("[HotScoreRefresher] completed, processed %d posts, took %v", totalProcessed, time.Since(startTime))

			// ================== 2. 阻塞等待下一次触发或退出 ==================
			select {
			case <-r.stopCh:
				// 收到停止信号，安全退出整个大循环
				log.Println("[HotScoreRefresher] receiving stop signal, exiting loop...")
				return
			case <-ticker.C:
				// 定时器触发，继续下一次外层 for 循环
			}
		}
	}()
}

func (r *HotScoreRefresher) Stop() {
	// 使用 sync.Once 确保 close 操作只执行一次，哪怕外部调用了多次 Stop()
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
	
	r.wg.Wait() // 阻塞等待 goroutine 彻底退出，实现优雅关机
	log.Println("[HotScoreRefresher] stopped completely")
}
