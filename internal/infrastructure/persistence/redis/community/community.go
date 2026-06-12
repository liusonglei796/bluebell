// Package communitycache 提供 Community 聚合的 Redis 缓存实现
//
// 设计说明：
// 1. 缓存策略：Cache-Aside（旁路缓存），先读缓存，未命中再读数据库并回写
// 2. 并发安全：使用 singleflight.Group 防止缓存击穿合并回源请求
// 3. 容错设计：Redis 操作失败仅记录 warn 日志，不阻塞业务，让调用方回源到 MySQL
// 4. 空结果缓存：空列表写入 "[]" 避免缓存穿透，nil 实体不缓存（防止穿透攻击）
// 5. TTL 差异化：列表缓存 10min（低频变化），详情缓存 30min（几乎不变）
//
// 被以下包引用：
// - internal/infrastructure/persistence/redis/caches.go（聚合所有 Redis 仓储）
// - internal/application/community_service.go（业务层调用缓存）
package communitycache

import (
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/logger"
	infratrace "bluebell/internal/infrastructure/trace"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// tracer 是 communitycache 包的 OpenTelemetry 追踪器实例
// 每个方法创建独立 span，命名格式 RedisCommunityDAO.MethodName
// 用于在 Grafana Tempo 中追踪缓存操作的耗时和状态
var tracer = infratrace.TracerForModule("dao/redis/community")

const (
	// keyPrefix 是所有 Redis key 的通用前缀，用于命名空间隔离
	keyPrefix = "bluebell:"

	// keyCommunityList 社区列表缓存 key（不含前缀）
	// 存储格式：JSON String（[]*entity.Community 的 JSON 数组）
	keyCommunityList = "community:list"

	// keyCommunityDetailPrefix 社区详情缓存 key 前缀（不含前缀）
	// 实际 key: bluebell:community:detail:<id>
	// 存储格式：JSON String（单个 entity.Community 的 JSON 对象）
	keyCommunityDetailPrefix = "community:detail:"

	// ttlCommunityList 社区列表缓存 TTL
	// 设置为 10 分钟：社区列表不会频繁变化，10min 足够显著降低 MySQL 读压力
	// 为什么不设更长：管理员创建社区后，最长 10min 用户可见新社区，体验可接受
	ttlCommunityList = 10 * time.Minute

	// ttlCommunityDetail 社区详情缓存 TTL
	// 设置为 30 分钟：社区名称和简介几乎不变（创建后很少修改）
	// 为什么不设更长：未来可能支持编辑功能，30min 是安全折中
	ttlCommunityDetail = 30 * time.Minute
)

// sf 是 singleflight.Group 实例，用于防止缓存击穿
//
// 并发优化：使用 sf.Do 合并相同 key 的并发回源请求，
// 同一时刻只有一个 goroutine 执行函数体，其余 goroutine 共享结果
//
// 不做优化的后果：
//  1. 大量并发请求同时未命中缓存时（如服务刚启动、缓存刚过期），
//     所有请求穿透到 MySQL，导致数据库连接池被打满
//  2. 数据库 QPS 瞬时飙升，可能触发慢查询报警甚至宕机
//  3. 在 MySQL 恢复前，所有请求持续报错，造成级联故障
//
// 适用场景：社区列表（多个用户同时访问首页）和社区详情（高频详情页）
var sf singleflight.Group

// redisKey 拼接完整 Redis key
// 格式：bluebell:<子 key>
// 被本包所有 Redis 操作方法调用
func redisKey(key string) string {
	return keyPrefix + key
}

// cacheStruct 是 CommunityCacheRepository 接口的 Redis 实现
// 持有 redis.Client 实例，所有方法通过该客户端操作 Redis
type cacheStruct struct {
	rdb *redis.Client
}

// NewCommunityCache 创建社区缓存仓储实例
//
// 参数：
//   - rdb: *redis.Client（由 cache.Init 初始化，在 DI 阶段注入）
//
// 返回值：
//   - domain.CommunityCacheRepository 接口实现
//
// 被以下位置调用：
// - internal/infrastructure/persistence/redis/caches.go（NewRepositories 中注册）
// - 或在 DI 容器中手动注入
func NewCommunityCache(rdb *redis.Client) domain.CommunityCacheRepository {
	return &cacheStruct{rdb: rdb}
}

// GetCommunityList 从 Redis 获取社区列表缓存
//
// 实现策略：
// 1. 先尝试读取 Redis key bluebell:community:list
// 2. 使用 singleflight 合并并发读请求，减少重复 Redis 查询
// 3. 命中则 JSON 反序列化为 []*entity.Community 返回
// 4. 未命中或出错则返回 (nil, nil) 让调用方回源 MySQL
//
// 空结果处理：已缓存的空列表 "[]" 会正常反序列化为空切片返回，
// 避免缓存穿透（与 SetCommunityList 写入 "[]" 对应）
//
// 并发优化：sf.Do 确保同一时刻只有一个 goroutine 执行 Redis GET，
// 其余 goroutine 共享结果，减少无效网络往返
//
// 被以下位置调用：
// - community_service.go GetCommunityList（缓存优先读取）
func (c *cacheStruct) GetCommunityList(ctx context.Context) ([]*entity.Community, error) {
	ctx, span := tracer.Start(ctx, "RedisCommunityDAO.GetCommunityList")
	defer span.End()

	// 使用 singleflight 合并相同 key 的并发 Redis 查询
	// 回调函数闭包捕获 ctx 用于 Redis 操作
	val, err, _ := sf.Do(redisKey(keyCommunityList), func() (interface{}, error) {
		data, getErr := c.rdb.Get(ctx, redisKey(keyCommunityList)).Bytes()
		if getErr != nil {
			return nil, getErr // 可能为 redis.Nil（未命中）或网络错误
		}
		return data, nil
	})

	if err != nil {
		if err == redis.Nil {
			// 缓存未命中，返回 nil 让调用方回源
			span.AddEvent("cache_miss")
			return nil, nil
		}
		// Redis 网络错误等：记录 warn 日志，不阻塞业务
		// 设计考量：缓存是性能优化手段，不应因缓存故障导致业务不可用
		logger.WithContext(ctx).Warn("communitycache.GetCommunityList: redis GET failed",
			zap.Error(err),
		)
		span.AddEvent("redis_error")
		return nil, nil
	}

	// singleflight 返回的是 interface{}，需要断言为 []byte
	bytes, ok := val.([]byte)
	if !ok {
		logger.WithContext(ctx).Warn("communitycache.GetCommunityList: sf.Do returned non-bytes",
			zap.Any("type", fmt.Sprintf("%T", val)),
		)
		return nil, nil
	}

	// JSON 反序列化为社区列表
	var list []*entity.Community
	if err := json.Unmarshal(bytes, &list); err != nil {
		// 反序列化失败：可能缓存数据被手动修改或版本不匹配
		// 记录 warn 后返回 nil 让调用方回源，并让调用方决定是否重建缓存
		logger.WithContext(ctx).Warn("communitycache.GetCommunityList: JSON unmarshal failed",
			zap.Error(err),
		)
		return nil, nil
	}

	// 正常返回缓存数据（可能是空列表 []，来自空结果缓存）
	span.AddEvent("cache_hit")
	return list, nil
}

// SetCommunityList 将社区列表写入 Redis 缓存
//
// 序列化策略：直接 JSON 序列化 []*entity.Community 切片
// 选择原因：
// - 简单直接，无需额外的 cacheCommunity 结构体
// - entity.Community 字段均为导出字段，可直接序列化
// - 与 GetCommunityList 的反序列化对称
//
// 空结果缓存：当 list 为空切片时，写入 "[]" JSON 字符串
// 为什么要缓存空结果：防止缓存穿透攻击——恶意请求或空数据阶段，
// 每次查询都穿透到 MySQL，写入空列表可避免这种场景
//
// TTL: 10 分钟（由 ttlCommunityList 控制）
//
// 错误处理：写入失败仅记录 warn 日志，由调用方决定是否重试
//
// 被以下位置调用：
// - community_service.go GetCommunityList（缓存未命中时回写）
// - 数据迁移或预热脚本
func (c *cacheStruct) SetCommunityList(ctx context.Context, list []*entity.Community) error {
	ctx, span := tracer.Start(ctx, "RedisCommunityDAO.SetCommunityList")
	defer span.End()

	// 空切片安全处理：确保空列表写入 "[]" 而非 "null"
	// 这样 GetCommunityList 反序列化时得到空切片而非 nil
	if list == nil {
		list = make([]*entity.Community, 0)
	}

	bytes, err := json.Marshal(list)
	if err != nil {
		return fmt.Errorf("communitycache.SetCommunityList: JSON marshal failed: %w", err)
	}

	// 使用 SetEx 写入带 TTL 的缓存
	if err := c.rdb.SetEx(ctx, redisKey(keyCommunityList), bytes, ttlCommunityList).Err(); err != nil {
		logger.WithContext(ctx).Warn("communitycache.SetCommunityList: redis SETEX failed",
			zap.Error(err),
		)
		return fmt.Errorf("communitycache.SetCommunityList: redis SETEX failed: %w", err)
	}

	span.AddEvent("cache_set")
	return nil
}

// InvalidateCommunityList 删除社区列表缓存
//
// 失效时机：
// - 管理员创建新社区后（CreateCommunity）
// - 社区信息批量更新后（如未来支持的编辑功能）
//
// 设计考量：删除而非更新
// - 删除实现简单，下次读取时自动回源重建
// - 避免与数据库数据不一致（更新可能遗漏字段）
// - 适合低频变化的社区列表（创建操作远少于读取）
//
// 错误处理：删除失败记录 warn，不阻塞调用方
//
// 被以下位置调用：
// - community_service.go CreateCommunity（创建社区后失效）
func (c *cacheStruct) InvalidateCommunityList(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "RedisCommunityDAO.InvalidateCommunityList")
	defer span.End()

	if err := c.rdb.Del(ctx, redisKey(keyCommunityList)).Err(); err != nil {
		logger.WithContext(ctx).Warn("communitycache.InvalidateCommunityList: redis DEL failed",
			zap.Error(err),
		)
		return fmt.Errorf("communitycache.InvalidateCommunityList: %w", err)
	}

	span.AddEvent("cache_invalidated")
	return nil
}

// GetCommunityDetail 从 Redis 获取单个社区详情缓存
//
// 实现策略：
// 1. 构造 key: bluebell:community:detail:<id>
// 2. 使用 singleflight 合并同社区的并发读请求
// 3. 命中则 JSON 反序列化为 *entity.Community 返回
// 4. 未命中返回 (nil, nil)
//
// 安全设计：绝不缓存 nil 值（SetCommunityDetail 已保证）。
// 不存在社区的 ID 请求不会导致缓存穿透，因为 MySQL 返回 nil 时我们不写入缓存。
//
// 并发优化：sf.Do 使用 "community:detail:<id>" 作为 key，
// 同一社区详情的高频访问（如首页推荐列表中的多个社区）可合并 Redis 查询
//
// 被以下位置调用：
// - community_service.go GetCommunityDetail（缓存优先读取）
func (c *cacheStruct) GetCommunityDetail(ctx context.Context, id int64) (*entity.Community, error) {
	ctx, span := tracer.Start(ctx, "RedisCommunityDAO.GetCommunityDetail")
	defer span.End()

	key := redisKey(keyCommunityDetailPrefix + fmt.Sprint(id))

	// 使用 singleflight 合并相同社区的并发查询
	val, err, _ := sf.Do(key, func() (interface{}, error) {
		data, getErr := c.rdb.Get(ctx, key).Bytes()
		if getErr != nil {
			return nil, getErr
		}
		return data, nil
	})

	if err != nil {
		if err == redis.Nil {
			// 缓存未命中
			span.AddEvent("cache_miss")
			return nil, nil
		}
		// Redis 错误：记录 warn，返回 nil 让调用方回源
		logger.WithContext(ctx).Warn("communitycache.GetCommunityDetail: redis GET failed",
			zap.Int64("community_id", id),
			zap.Error(err),
		)
		span.AddEvent("redis_error")
		return nil, nil
	}

	bytes, ok := val.([]byte)
	if !ok {
		logger.WithContext(ctx).Warn("communitycache.GetCommunityDetail: sf.Do returned non-bytes",
			zap.Int64("community_id", id),
			zap.Any("type", fmt.Sprintf("%T", val)),
		)
		return nil, nil
	}

	// JSON 反序列化为单个社区实体
	var community entity.Community
	if err := json.Unmarshal(bytes, &community); err != nil {
		logger.WithContext(ctx).Warn("communitycache.GetCommunityDetail: JSON unmarshal failed",
			zap.Int64("community_id", id),
			zap.Error(err),
		)
		return nil, nil
	}

	span.AddEvent("cache_hit")
	return &community, nil
}

// SetCommunityDetail 将社区详情写入 Redis 缓存
//
// 安全设计：如果 c 为 nil，直接返回 nil 不写入缓存
// 为什么不缓存 nil：防止缓存穿透攻击——
// 攻击者反复请求不存在的社区 ID，如果不做判断会将 nil 缓存，
// 导致合法请求也返回"不存在"，且消耗 Redis 内存存储大量无用 key
//
// TTL: 30 分钟（由 ttlCommunityDetail 控制）
//
// 被以下位置调用：
// - community_service.go GetCommunityDetail（缓存未命中且 MySQL 查到数据时回写）
func (c *cacheStruct) SetCommunityDetail(ctx context.Context, community *entity.Community) error {
	ctx, span := tracer.Start(ctx, "RedisCommunityDAO.SetCommunityDetail")
	defer span.End()

	// 不缓存 nil 实体：防止缓存穿透
	if community == nil {
		span.AddEvent("skip_cache_nil")
		return nil
	}

	key := redisKey(keyCommunityDetailPrefix + fmt.Sprint(community.ID))

	bytes, err := json.Marshal(community)
	if err != nil {
		return fmt.Errorf("communitycache.SetCommunityDetail: JSON marshal failed (id: %d): %w", community.ID, err)
	}

	if err := c.rdb.SetEx(ctx, key, bytes, ttlCommunityDetail).Err(); err != nil {
		logger.WithContext(ctx).Warn("communitycache.SetCommunityDetail: redis SETEX failed",
			zap.Int64("community_id", community.ID),
			zap.Error(err),
		)
		return fmt.Errorf("communitycache.SetCommunityDetail: %w", err)
	}

	span.AddEvent("cache_set")
	return nil
}

// InvalidateCommunityDetail 删除单个社区详情缓存
//
// 失效时机：
// - 社区信息修改后（当前未实现编辑功能，预留）
// - 数据修复或管理操作后
//
// 设计考量：同 InvalidateCommunityList，使用删除而非更新策略
//
// 被以下位置调用：
// - 未来的社区编辑功能
// - 管理后台的缓存刷新操作
func (c *cacheStruct) InvalidateCommunityDetail(ctx context.Context, id int64) error {
	ctx, span := tracer.Start(ctx, "RedisCommunityDAO.InvalidateCommunityDetail")
	defer span.End()

	key := redisKey(keyCommunityDetailPrefix + fmt.Sprint(id))

	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		logger.WithContext(ctx).Warn("communitycache.InvalidateCommunityDetail: redis DEL failed",
			zap.Int64("community_id", id),
			zap.Error(err),
		)
		return fmt.Errorf("communitycache.InvalidateCommunityDetail (id: %d): %w", id, err)
	}

	span.AddEvent("cache_invalidated")
	return nil
}
