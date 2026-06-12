package postcache

import (
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	infratrace "bluebell/internal/infrastructure/trace"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// keyPostRemarksPrefix 是评论列表缓存的 key 前缀
// 完整 key: bluebell:post:remarks:<postID>
// 与 keyPrefix（"bluebell:"）拼接后使用：redisKey(keyPostRemarksPrefix + postIDStr)
// 设计为独立前缀，便于按模式删除或排查
const keyPostRemarksPrefix = "post:remarks:"

// tracerRemark 是 remark 缓存模块的 OpenTelemetry tracer 实例
// 与 post.go 中的 tracer 分离，避免跨文件共享 tracer 导致的 span 命名冲突
// 两个文件属于同一包（postcache），tracer 变量名不能重复，因此独立命名
var tracerRemark = infratrace.TracerForModule("dao/redis/remark")

// remarkCacheStruct 实现 domain.RemarkCacheRepository 接口
// 用于对帖子评论列表进行 Redis 缓存读写操作
// 缓存粒度：按 postID 整体缓存整个评论列表（含 Author 嵌套）
// replyTo 过滤仍在 service 层做（与现状一致），缓存层不做过滤
//
// 为什么按 postID 整体缓存而非逐条缓存：
// 帖子详情页需要展示完整的评论列表，逐条缓存需要 N 次 GET + 内存聚合，
// 网络开销大，整体缓存只需一次 GET，反序列化后直接使用。
type remarkCacheStruct struct {
	rdb *redis.Client
}

// NewRemarkCache 创建 RemarkCacheRepository 的 Redis 实现
// 接收 *redis.Client 并返回接口，由依赖注入层注入到 PostService
// 遵循 postcache 包的一致工厂模式（同 NewCache / NewCacheWithRefresher）
func NewRemarkCache(rdb *redis.Client) domain.RemarkCacheRepository {
	return &remarkCacheStruct{rdb: rdb}
}

// GetRemarks 从 Redis 读取帖子评论列表缓存
// 功能：通过 postID 从 Redis 获取缓存的 JSON 字符串，反序列化为 []*entity.Remark 返回
// 命中：返回完整的评论列表（含 Author 嵌套）
// 未命中：返回 nil, nil（由 service 层决定是否回查 MySQL 并回写缓存）
//
// 为什么需要此缓存：
// 帖子详情页是核心读接口，评论列表是详情页的二级请求。
// 每次直接查 MySQL 需要 JOIN posts + users 表，热门帖子数千条评论反复查询 users 表，数据库压力大。
// 缓存评论列表后，大多数请求只读 Redis，Redis GET 操作耗时 < 1ms，QPS 可达 10w+。
//
// 并发优化：Get 是 Redis 原子操作，天然线程安全，无需额外锁保护
// 不做优化的后果：无（Redis 单线程模型，GET 命令本身线程安全）
//
// 被以下调用：PostService.GetPostRemarks（application 层 cache-first 路径）
func (c *remarkCacheStruct) GetRemarks(ctx context.Context, postID int64) ([]*entity.Remark, error) {
	ctx, span := tracerRemark.Start(ctx, "RemarkCache.GetRemarks")
	defer span.End()

	postIDStr := strconv.FormatInt(postID, 10)
	key := redisKey(keyPostRemarksPrefix + postIDStr)

	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// cache miss：返回 nil, nil，由调用方决定是否回查 MySQL 并回写缓存
			return nil, nil
		}
		return nil, fmt.Errorf("get remarks cache failed (post_id: %d): %w", postID, err)
	}

	var remarks []*entity.Remark
	if err := json.Unmarshal([]byte(val), &remarks); err != nil {
		return nil, fmt.Errorf("unmarshal remarks cache failed (post_id: %d): %w", postID, err)
	}
	return remarks, nil
}

// SetRemarks 将帖子评论列表写入 Redis 缓存
// 功能：序列化 []*entity.Remark 为 JSON，通过 SetEx 写入 Redis
// TTL 设计：3 分钟，平衡了命中率与新鲜度——太短命中率低，太长新增评论后用户看不到最新列表
//
// 不缓存空切片：如果 MySQL 返回空列表，不写缓存，避免 Redis 中堆积无意义的空缓存条目
// 大小评估：典型帖子评论数 < 1000，JSON 序列化后 < 500KB，Redis 单条 String 最大 512MB，完全可接受
//
// 并发优化：SetEx 是 Redis 原子操作，多个 goroutine 同时写入不会导致数据损坏
// 不做优化的后果：无（Redis 单线程处理写入，SetEx 天然原子，最终写入结果一致）
//
// 被以下调用：PostService.GetPostRemarks cache-miss 回写路径
func (c *remarkCacheStruct) SetRemarks(ctx context.Context, postID int64, remarks []*entity.Remark) error {
	ctx, span := tracerRemark.Start(ctx, "RemarkCache.SetRemarks")
	defer span.End()

	// 不缓存空切片：MySQL 返回空列表时跳过写入，
	// 避免 Redis 中存储大量无价值的空列表条目浪费内存
	if len(remarks) == 0 {
		return nil
	}

	data, err := json.Marshal(remarks)
	if err != nil {
		return fmt.Errorf("marshal remarks failed (post_id: %d): %w", postID, err)
	}

	postIDStr := strconv.FormatInt(postID, 10)
	key := redisKey(keyPostRemarksPrefix + postIDStr)

	if err := c.rdb.SetEx(ctx, key, string(data), 3*time.Minute).Err(); err != nil {
		return fmt.Errorf("set remarks cache failed (post_id: %d): %w", postID, err)
	}
	return nil
}

// InvalidateRemarks 删除指定帖子的评论列表缓存
// 功能：写操作（新增评论/删除帖子）后主动清理评论缓存，保证下次读请求走 cache-miss 回源 MySQL
//
// 为什么需要：
// 1. RemarkPost 新增评论后，缓存中的评论列表缺少最新评论，需要失效让下次读取重新加载
// 2. DeletePost 删除帖子后，评论列表不再有意义，需要清理避免残留
//
// 缓存一致性策略：Cache-Aside 模式的失效（invalidation）而非更新（update），
// 因为新增评论后更新缓存需要 append 操作，并发场景容易丢失数据。
// 失效后下次读请求走 cache-miss → 查 MySQL → 回写缓存，保证最终一致。
//
// 并发优化：Del 是 Redis 原子操作，无需额外同步
// 不做优化的后果：无（删除不存在的 key 返回 0，不会报错或影响数据）
//
// 被以下调用：
//   - PostService.RemarkPost（新增评论后失效）
//   - PostService.DeletePost（帖子删除时一并失效，通过 cacheStruct.DeletePost 内联 Del 调用）
func (c *remarkCacheStruct) InvalidateRemarks(ctx context.Context, postID int64) error {
	ctx, span := tracerRemark.Start(ctx, "RemarkCache.InvalidateRemarks")
	defer span.End()

	postIDStr := strconv.FormatInt(postID, 10)
	key := redisKey(keyPostRemarksPrefix + postIDStr)

	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("invalidate remarks cache failed (post_id: %d): %w", postID, err)
	}
	return nil
}
