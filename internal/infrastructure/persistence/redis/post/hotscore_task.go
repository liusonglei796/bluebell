package postcache

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ========== 热度刷新常量 ==========

const (
	defaultRefreshInterval = 5 * time.Minute
	defaultBatchSize       = 500
)

// HotScoreRefresher 定时刷新 Gravity 分数
type HotScoreRefresher struct {
	rdb      *redis.Client
	stopCh   chan struct{}  // 用于发送停止信号
	wg       sync.WaitGroup // 用于等待后台任务彻底结束
	stopOnce sync.Once      // 保证 stopCh 只被关闭一次，防止 panic
}

func NewHotScoreRefresher(rdb *redis.Client) *HotScoreRefresher {
	return &HotScoreRefresher{
		rdb:    rdb,
		stopCh: make(chan struct{}),
	}
}

func (r *HotScoreRefresher) Start() {
	log.Printf("[HotScoreRefresher] started, interval: %v, batch size: %d",
		defaultRefreshInterval, defaultBatchSize)

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		ticker := time.NewTicker(defaultRefreshInterval)
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
				postIDs, err := r.rdb.ZRange(batchCtx, redisKey(keyPostTimeZSet), start, start+defaultBatchSize-1).Result()
				if err != nil {
					batchCancel()
					log.Printf("[HotScoreRefresher] ZRange error: %v", err)
					break
				}
				if len(postIDs) == 0 {
					batchCancel()
					break // 本次全量数据处理完毕
				}

				if err := r.batchRefreshGravityScores(batchCtx, postIDs); err != nil {
					log.Printf("[HotScoreRefresher] batch refresh error: %v", err)
				}

				batchCancel() // 主动释放 context 资源不能在无限循环中用defer

				totalProcessed += len(postIDs)
				start += defaultBatchSize

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

// batchRefreshGravityScores 批量刷新帖子的 Gravity 分数到 ZSet
func (r *HotScoreRefresher) batchRefreshGravityScores(ctx context.Context, postIDs []string) error {
	// [防御] 空切片直接返回
	if len(postIDs) == 0 {
		return nil
	}

	pipe := r.rdb.Pipeline()

	for _, postID := range postIDs {
		result, err := r.rdb.HGetAll(ctx, redisKey(keyPostMetaPrefix+postID)).Result()
		// [防御] 单个帖子失败不影响其他帖子，不能因为一个帖子失败就 abort 整批
		if err != nil || len(result) == 0 {
			continue
		}

		// [防御] ParseInt 忽略 error，格式不对时降级为 0
		createTimeUnix, _ := strconv.ParseInt(result["create_time"], 10, 64)
		voteUp, _ := strconv.ParseInt(result["vote_up"], 10, 64)
		voteDown, _ := strconv.ParseInt(result["vote_down"], 10, 64)

		createTime := time.Unix(createTimeUnix, 0)
		score := CalculateGravityScore(voteUp, voteDown, createTime)

		pipe.ZAdd(ctx, redisKey(keyPostScoreZSet), redis.Z{
			Score:  score,
			Member: postID,
		})
	}

	// [防御] Pipeline 即使为空 Exec 是安全的
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("hotscore_refresher: pipeline exec failed: %w", err)
	}
	return nil
}

func (r *HotScoreRefresher) Stop() {
	// 使用 sync.Once 确保 close 操作只执行一次，哪怕外部调用了多次 Stop()
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})

	r.wg.Wait() // 阻塞等待 goroutine 彻底退出，实现优雅关机
	log.Println("[HotScoreRefresher] stopped completely")
}
