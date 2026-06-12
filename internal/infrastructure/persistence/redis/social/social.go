// Package socialcache 提供社交模块（Social）的 Redis 缓存层实现
//
// 包含 4 个子缓存：
//  1. 关注数/粉丝数（follow_count/follower_count）
//  2. 是否关注（is_following）
//  3. 用户资料（UserProfile）
//  4. 活动流首屏（Activities first page）
//
// 缓存策略：Cache-Aside（旁路缓存）
// - 读操作先查缓存，命中直接返回，未命中返回哨兵值让调用方回源 MySQL
// - 写操作同步写入缓存（Set）或在写数据时失效缓存（Invalidate）
// - 失效时机由调用方（Service 层）在业务写操作后触发
//
// 并发安全：Redis 单线程模型保证所有操作的原子性，无需额外锁
// 不做优化的后果：无（Redis 单线程处理所有命令，GET/SET/DEL 天然线程安全）
//
// 被以下包引用：
// - internal/infrastructure/persistence/redis/caches.go（聚合所有 Redis 仓储，需手动注册）
// - internal/application/social_service.go（应用层通过接口调用缓存方法）
package socialcache

import (
	"bluebell/internal/domain"
	"bluebell/internal/domain/entity"
	infratrace "bluebell/internal/infrastructure/trace"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// tracer 是 socialcache 包的 OpenTelemetry 追踪器实例
// 每个方法创建独立 span，命名格式 RedisSocialDAO.MethodName
// 用于在 Grafana Tempo 中追踪所有社交缓存操作的耗时和状态
var tracer = infratrace.TracerForModule("dao/redis/social")

const (
	// keyPrefix 是所有 Redis key 的通用前缀，用于命名空间隔离，避免与其他项目 key 冲突
	keyPrefix = "bluebell:"

	// ============================
	// 子缓存 1：关注数/粉丝数
	// ============================

	// keyFollowerCountPrefix 粉丝数缓存 key 前缀（不含前缀）
	// 实际 key: bluebell:social:follower_count:<userID>
	// 值类型：String（整数字符串）
	// TTL：5 分钟 — 关注/取关相对高频，TTL 短可以减少数据不一致窗口
	keyFollowerCountPrefix = "social:follower_count:"

	// keyFollowingCountPrefix 关注数缓存 key 前缀（不含前缀）
	// 实际 key: bluebell:social:following_count:<userID>
	// 值类型：String（整数字符串）
	keyFollowingCountPrefix = "social:following_count:"

	// ttlFollowCount 关注数/粉丝数缓存 TTL
	// 5 分钟：平衡了命中率与数据一致性。关注操作不是秒级敏感数据，5min 内最终一致可接受
	ttlFollowCount = 5 * time.Minute

	// ============================
	// 子缓存 2：是否关注
	// ============================

	// keyIsFollowingPrefix 是否关注缓存 key 前缀（不含前缀）
	// 实际 key: bluebell:social:is_following:<followerID>:<followingID>
	// 值类型：String（"1" 表示已关注，"0" 表示未关注）
	// TTL：1 分钟 — 关注状态变化时希望尽快反映，1min 是合理的折中
	keyIsFollowingPrefix = "social:is_following:"

	// ttlIsFollowing 是否关注缓存 TTL
	// 1 分钟：关注/取关是最频繁的社交操作之一，短 TTL 可快速反映状态变化
	// 为什么不是更长：用户关注某人后希望立即看到状态变化，1min 已是最大容忍
	ttlIsFollowing = 1 * time.Minute

	// ============================
	// 子缓存 3：用户资料
	// ============================

	// keyProfilePrefix 用户资料缓存 key 前缀（不含前缀）
	// 实际 key: bluebell:social:profile:<userID>
	// 值类型：JSON String（entity.UserProfile 的 JSON 序列化）
	// TTL：30 分钟 — 用户资料修改频率低（头像/Bio/GitHub 绑定），长 TTL 可显著减少 MySQL 读压力
	keyProfilePrefix = "social:profile:"

	// ttlProfile 用户资料缓存 TTL
	// 30 分钟：资料信息几乎不变（头像上传/Bio 修改确实存在但非常低频）
	// 不做优化的后果：每次 GetProfile 都查 MySQL，UserProfile 通过 LEFT JOIN 查询用户表和资料表，
	// 高频访问（如首页用户卡片展示）会严重增加数据库负载
	ttlProfile = 30 * time.Minute

	// ============================
	// 子缓存 4：活动流首屏
	// ============================

	// keyActivitiesPrefix 活动流首屏缓存 key 前缀（不含前缀）
	// 实际 key: bluebell:social:activities:first_page:<userID>
	// 值类型：JSON String（[]*entity.Activity 的 JSON 序列化）
	// 缓存策略：只缓存第 1 页（page=1 && size<=20），其他页直接穿透到 MySQL
	//   - 为什么只缓存首屏：用户活跃动态按时间倒序，第 1 页访问频率远高于后续页
	//   - 分页查询命中率低（不同用户翻页习惯不同），缓存收益小
	// TTL：1 分钟 — 活动流实时性要求高（新帖子/新关注/新点赞应尽快出现）
	keyActivitiesPrefix = "social:activities:first_page:"

	// ttlActivities 活动流缓存 TTL
	// 1 分钟：用户期望看到最新的动态，1min 过期后自然从 MySQL 读取最新数据
	ttlActivities = 1 * time.Minute
)

// cacheStruct 是 SocialCacheRepository 接口的 Redis 实现
// 持有 *redis.Client 实例，所有方法通过该客户端操作 Redis
type cacheStruct struct {
	rdb *redis.Client
}

// NewSocialCache 创建社交缓存仓储实例
//
// 参数：
//   - rdb: *redis.Client（由 cache.Init 初始化，在 DI 阶段注入）
//
// 返回值：
//   - domain.SocialCacheRepository 接口实现
//
// 被以下位置调用：
// - internal/infrastructure/persistence/redis/caches.go（NewRepositories 中注册）
// - 或在 DI 容器中手动注入
func NewSocialCache(rdb *redis.Client) domain.SocialCacheRepository {
	return &cacheStruct{rdb: rdb}
}

// redisKey 拼接完整 Redis key
// 格式：bluebell:<子 key>
// 被本包所有 Redis 操作方法调用
func redisKey(key string) string {
	return keyPrefix + key
}

// ============================================================================
// 子缓存 1：关注数 / 粉丝数
// ============================================================================

// GetFollowerCount 从 Redis 获取用户粉丝数缓存
//
// 实现策略：
//  1. 构造 key: bluebell:social:follower_count:<userID>
//  2. 读取 String 值并解析为 int64
//  3. 缓存未命中返回 (-1, nil) — -1 是哨兵值，因为粉丝数不可能为负数
//  4. Redis 错误返回 (0, error)
//
// 调用方行为：
//   - 返回 -1 表示 cache miss，应回源 MySQL 查询真实粉丝数
//   - 返回 >=0 表示缓存命中，直接使用
//
// 并发优化：GET 是 Redis 原子操作，天然线程安全
// 不做优化的后果：无（Redis 单线程执行，多 goroutine 同时读不会竞争）
//
// 被以下位置调用：
//   - social_service.go GetProfile（组装用户资料时获取粉丝数）
func (c *cacheStruct) GetFollowerCount(ctx context.Context, userID int64) (int64, error) {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.GetFollowerCount")
	defer span.End()

	key := redisKey(keyFollowerCountPrefix + fmt.Sprint(userID))
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// 缓存未命中：返回 -1 哨兵值，调用方应回源 MySQL
			span.AddEvent("cache_miss")
			return -1, nil
		}
		return 0, fmt.Errorf("socialcache.GetFollowerCount (user_id: %d): %w", userID, err)
	}

	// 解析 Redis 中存储的整数字符串
	count, err := parseInt64(val)
	if err != nil {
		return 0, fmt.Errorf("socialcache.GetFollowerCount parse failed (user_id: %d): %w", userID, err)
	}

	span.AddEvent("cache_hit")
	return count, nil
}

// SetFollowerCount 将用户粉丝数写入 Redis 缓存
//
// 写入策略：
//   - 使用 SetEx 写入，TTL 为 ttlFollowCount（5 分钟）
//   - count 为 0 时正常写入（用户可能确实没有粉丝）
//   - count 为负数时不缓存（数据异常）
//
// 为什么不更新而是覆盖：关注数每次变化都是完全替换，不需要 INCRBY/DECRBY
// 原因：INCRBY 在高并发下可能计数不准确（topic 的重试、幂等性问题），
// 每次从 MySQL 读取真实值后整体覆写更可靠
//
// 并发优化：SetEx 是 Redis 原子操作，多个 goroutine 同时写入不会导致数据损坏
//
// 被以下位置调用：
//   - social_service.go GetProfile（缓存未命中时，从 MySQL 读取后回写）
func (c *cacheStruct) SetFollowerCount(ctx context.Context, userID int64, count int64) error {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.SetFollowerCount")
	defer span.End()

	// 异常数据不缓存：粉丝数不可能为负数
	if count < 0 {
		span.AddEvent("skip_negative_count")
		return nil
	}

	key := redisKey(keyFollowerCountPrefix + fmt.Sprint(userID))
	if err := c.rdb.SetEx(ctx, key, fmt.Sprint(count), ttlFollowCount).Err(); err != nil {
		return fmt.Errorf("socialcache.SetFollowerCount (user_id: %d): %w", userID, err)
	}

	span.AddEvent("cache_set")
	return nil
}

// GetFollowingCount 从 Redis 获取用户关注数缓存
//
// 实现策略与 GetFollowerCount 完全相同，只是 key 不同
// 被以下位置调用：
//   - social_service.go GetProfile（组装用户资料时获取关注数）
func (c *cacheStruct) GetFollowingCount(ctx context.Context, userID int64) (int64, error) {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.GetFollowingCount")
	defer span.End()

	key := redisKey(keyFollowingCountPrefix + fmt.Sprint(userID))
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			span.AddEvent("cache_miss")
			return -1, nil
		}
		return 0, fmt.Errorf("socialcache.GetFollowingCount (user_id: %d): %w", userID, err)
	}

	count, err := parseInt64(val)
	if err != nil {
		return 0, fmt.Errorf("socialcache.GetFollowingCount parse failed (user_id: %d): %w", userID, err)
	}

	span.AddEvent("cache_hit")
	return count, nil
}

// SetFollowingCount 将用户关注数写入 Redis 缓存
//
// 策略与 SetFollowerCount 相同
//
// 被以下位置调用：
//   - social_service.go GetProfile（缓存未命中时，从 MySQL 读取后回写）
func (c *cacheStruct) SetFollowingCount(ctx context.Context, userID int64, count int64) error {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.SetFollowingCount")
	defer span.End()

	if count < 0 {
		span.AddEvent("skip_negative_count")
		return nil
	}

	key := redisKey(keyFollowingCountPrefix + fmt.Sprint(userID))
	if err := c.rdb.SetEx(ctx, key, fmt.Sprint(count), ttlFollowCount).Err(); err != nil {
		return fmt.Errorf("socialcache.SetFollowingCount (user_id: %d): %w", userID, err)
	}

	span.AddEvent("cache_set")
	return nil
}

// InvalidateFollowCounts 删除用户关注数和粉丝数缓存
//
// 失效时机：
//   - 用户关注某人（followerID 的关注数 +1，followingID 的粉丝数 +1）
//   - 用户取消关注某人（followerID 的关注数 -1，followingID 的粉丝数 -1）
//
// 使用场景：
//   - 调用方在 FollowUser / UnfollowUser 中分别对 followerID 和 followingID 各调用一次
//   - 例如：用户 A 关注用户 B → 调用 InvalidateFollowCounts(A) 和 InvalidateFollowCounts(B)
//
// 为什么同时失效双方的计数：
//   - 关注事件不仅影响关注者的关注数，也影响被关注者的粉丝数
//   - 如果只失效一方，另一方的计数缓存会过时，导致 Profile 页面数据不一致
//
// 并发优化：DEL 是 Redis 原子操作，不会出现并发删除冲突
//
// 被以下位置调用：
//   - social_service.go FollowUser（关注成功后失效双方的计数）
//   - social_service.go UnfollowUser（取消关注后失效双方的计数）
func (c *cacheStruct) InvalidateFollowCounts(ctx context.Context, userID int64) error {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.InvalidateFollowCounts")
	defer span.End()

	// 使用 Pipeline 批量发送两个 DEL 命令，减少网络往返
	// 为什么 Pipeline 不是事务：删除单个用户的 follower_count 和 following_count
	// 不需要原子性（即使一个删除失败，下次读取会重新回源）
	pipe := c.rdb.Pipeline()
	pipe.Del(ctx, redisKey(keyFollowerCountPrefix+fmt.Sprint(userID)))
	pipe.Del(ctx, redisKey(keyFollowingCountPrefix+fmt.Sprint(userID)))
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("socialcache.InvalidateFollowCounts (user_id: %d): %w", userID, err)
	}

	span.AddEvent("cache_invalidated")
	return nil
}

// ============================================================================
// 子缓存 2：是否关注（IsFollowing）
// ============================================================================

// GetIsFollowing 从 Redis 获取"用户 A 是否关注用户 B"的缓存状态
//
// 实现策略：
//  1. 构造 key: bluebell:social:is_following:<followerID>:<followingID>
//  2. 读取 String 值，"1"=已关注，"0"=未关注
//  3. 缓存未命中返回 (false, nil)
//  4. Redis 错误返回 (false, error)
//
// 关于 miss 和 "未关注" 的模糊性：
//
//	返回 false 时，调用方无法区分"缓存未命中"和"缓存命中但未关注"。
//	这是设计上的折中——为了保持 (bool, error) 的简洁签。
//	调用方应始终在返回 false 时回源 MySQL 确认。
//	实际影响：缓存建立后（见 SetIsFollowing），绝大多数请求是缓存命中，
//	miss 仅发生在首次访问或缓存过期后的短暂窗口。
//
// 并发优化：GET 是 Redis 原子操作
//
// 被以下位置调用：
//   - social_service.go GetProfile（判断当前用户是否查看目标用户）
func (c *cacheStruct) GetIsFollowing(ctx context.Context, followerID, followingID int64) (bool, error) {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.GetIsFollowing")
	defer span.End()

	key := redisKey(fmt.Sprintf("social:is_following:%d:%d", followerID, followingID))
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// 缓存未命中：返回 (false, nil)，调用方应回源 MySQL
			span.AddEvent("cache_miss")
			return false, nil
		}
		return false, fmt.Errorf("socialcache.GetIsFollowing (%d->%d): %w", followerID, followingID, err)
	}

	span.AddEvent("cache_hit")
	return val == "1", nil
}

// SetIsFollowing 将"用户 A 是否关注用户 B"的状态写入 Redis 缓存
//
// 存储格式：String，"1"=已关注，"0"=未关注
// 选择 String 而非 Bool：Redis 没有原生 bool 类型，String "1"/"0" 是最通用的表示方式
// TTL：1 分钟（由 ttlIsFollowing 控制）
//
// 被以下位置调用：
//   - social_service.go GetProfile（缓存未命中时，从 MySQL 读取后回写）
func (c *cacheStruct) SetIsFollowing(ctx context.Context, followerID, followingID int64, value bool) error {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.SetIsFollowing")
	defer span.End()

	key := redisKey(fmt.Sprintf("social:is_following:%d:%d", followerID, followingID))
	strVal := "0"
	if value {
		strVal = "1"
	}

	if err := c.rdb.SetEx(ctx, key, strVal, ttlIsFollowing).Err(); err != nil {
		return fmt.Errorf("socialcache.SetIsFollowing (%d->%d): %w", followerID, followingID, err)
	}

	span.AddEvent("cache_set")
	return nil
}

// InvalidateIsFollowing 删除两个方向的是否关注缓存
//
// 为什么需要失效两个方向：
//   - 用户 A 关注用户 B 后，不仅 "A 是否关注 B" 的缓存需要失效，
//     "B 是否关注 A" 的缓存也需要失效（虽然 B 关注 A 的状态未变，
//     但高并发场景下缓存可能已过期或数据不一致，刷新更安全）
//   - 安全隐患：不失效则 Profile 页面可能展示已过时的 IsFollowing 状态
//
// 实现方式：直接 DEL 两个精确 key（而非 SCAN）
// 为什么不用 SCAN：SCAN 遍历可能匹配到无关的 key，且多了 O(N) 的网络开销。
// 直接 DEL 两个已知 key 更高效（O(1) 每个 DEL），且不会误删。
//
// 失效时机：
//   - FollowUser 成功后
//   - UnfollowUser 成功后
//
// 并发优化：DEL 是 Redis 原子操作
// 不做优化的后果：无（删除不存在的 key 返回 0，不会报错或阻塞）
//
// 被以下位置调用：
//   - social_service.go FollowUser（关注成功后失效）
//   - social_service.go UnfollowUser（取消关注后失效）
func (c *cacheStruct) InvalidateIsFollowing(ctx context.Context, followerID, followingID int64) error {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.InvalidateIsFollowing")
	defer span.End()

	// 构造两个方向的 key
	// 方向 1：followerID → followingID（关注者看到被关注者的状态）
	// 方向 2：followingID → followerID（被关注者看到关注者的状态）
	key1 := redisKey(fmt.Sprintf("social:is_following:%d:%d", followerID, followingID))
	key2 := redisKey(fmt.Sprintf("social:is_following:%d:%d", followingID, followerID))

	// 使用 Pipeline 批量删除，减少网络往返
	pipe := c.rdb.Pipeline()
	pipe.Del(ctx, key1)
	pipe.Del(ctx, key2)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("socialcache.InvalidateIsFollowing (%d->%d): %w", followerID, followingID, err)
	}

	span.AddEvent("cache_invalidated")
	return nil
}

// ============================================================================
// 子缓存 3：用户资料（UserProfile）
// ============================================================================

// GetProfile 从 Redis 获取用户资料缓存
//
// 实现策略：
//  1. 构造 key: bluebell:social:profile:<userID>
//  2. 命中则 JSON 反序列化为 *entity.UserProfile 返回
//  3. 未命中返回 (nil, nil)
//
// 为什么不缓存 nil 用户资料：
//   - 如果 MySQL 中不存在该用户的资料，我们不写入缓存
//   - 防止缓存穿透攻击：攻击者恶意请求不存在的 userID，
//     如果缓存 nil，后续合法请求也返回"不存在"
//   - 不存在用户的资料在 MySQL 中返回 nil，调用方会创建默认空资料上传
//
// 并发优化：GET + JSON 反序列化是纯内存操作，无竞争条件
//
// 被以下位置调用：
//   - social_service.go GetProfile（获取用户资料信息）
func (c *cacheStruct) GetProfile(ctx context.Context, userID int64) (*entity.UserProfile, error) {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.GetProfile")
	defer span.End()

	key := redisKey(keyProfilePrefix + fmt.Sprint(userID))
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			// 缓存未命中：返回 nil 让调用方回源 MySQL
			span.AddEvent("cache_miss")
			return nil, nil
		}
		return nil, fmt.Errorf("socialcache.GetProfile (user_id: %d): %w", userID, err)
	}

	var profile entity.UserProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("socialcache.GetProfile unmarshal failed (user_id: %d): %w", userID, err)
	}

	span.AddEvent("cache_hit")
	return &profile, nil
}

// SetProfile 将用户资料写入 Redis 缓存
//
// 安全设计：
//   - profile 为 nil 时不缓存（防止缓存穿透）
//   - profile.UserID 为 0 时不缓存（无效数据）
//
// TTL：30 分钟（由 ttlProfile 控制）
// 为什么设 30 分钟：用户资料（头像/Bio/GitHub）修改频率极低，
// 长缓存时间可显著减少 MySQL 读压力，尤其是在首页用户卡片高频展示场景
//
// 失效时机（由调用方负责）：
//   - UploadAvatar（上传头像）后调 InvalidateProfile
//
// 被以下位置调用：
//   - social_service.go GetProfile（缓存未命中且 MySQL 查到数据后回写）
func (c *cacheStruct) SetProfile(ctx context.Context, profile *entity.UserProfile) error {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.SetProfile")
	defer span.End()

	// 不缓存 nil 实体：防止缓存穿透攻击
	if profile == nil {
		span.AddEvent("skip_cache_nil")
		return nil
	}

	key := redisKey(keyProfilePrefix + fmt.Sprint(profile.UserID))
	data, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("socialcache.SetProfile marshal failed (user_id: %d): %w", profile.UserID, err)
	}

	if err := c.rdb.SetEx(ctx, key, data, ttlProfile).Err(); err != nil {
		return fmt.Errorf("socialcache.SetProfile (user_id: %d): %w", profile.UserID, err)
	}

	span.AddEvent("cache_set")
	return nil
}

// InvalidateProfile 删除用户资料缓存
//
// 失效时机：
//   - UploadAvatar（上传头像）后 — 头像 URL 变更，缓存需要刷新
//   - SaveUserProfile（更新个人资料）后 — Bio/GitHub 等信息更新
//
// 为什么使用删除而非更新：
//   - 删除后下次读取自动回源 MySQL 重构缓存，保证数据一致性
//   - 避免"更新遗漏字段"导致的数据不一致问题
//
// 被以下位置调用：
//   - social_service.go UploadAvatar（头像上传成功后失效）
//   - social_service.go SaveUserProfile（资料更新成功后失效）
func (c *cacheStruct) InvalidateProfile(ctx context.Context, userID int64) error {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.InvalidateProfile")
	defer span.End()

	key := redisKey(keyProfilePrefix + fmt.Sprint(userID))
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("socialcache.InvalidateProfile (user_id: %d): %w", userID, err)
	}

	span.AddEvent("cache_invalidated")
	return nil
}

// ============================================================================
// 子缓存 4：活动流首屏（Activities First Page）
// ============================================================================

// GetActivitiesFirstPage 从 Redis 获取活动流首屏缓存
//
// 缓存策略（重要）：
//   - 只有 page == 1 && size <= 20 时才查缓存
//   - 其他页（page > 1 或 size > 20）直接返回 (nil, nil) 穿透到 MySQL
//
// 为什么只缓存首屏：
//   - 长尾分布：第 1 页流量占活动流总请求量的 90%+，缓存收益最大
//   - 翻页命中率低：不同用户的翻页习惯和深度不同，第 2 页及之后的缓存命中率显著降低
//   - 缓存成本：后续页的 key 数量多但命中少，浪费 Redis 内存
//   - 一致性：活动流实时性要求高，缓存太多页容易导致数据不一致
//
// 并发优化：GET + JSON 反序列化是纯内存操作
//
// 被以下位置调用：
//   - social_service.go GetActivities（获取活动流列表）
func (c *cacheStruct) GetActivitiesFirstPage(ctx context.Context, userID int64, page, size int) ([]*entity.Activity, error) {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.GetActivitiesFirstPage")
	defer span.End()

	// 非首屏请求直接穿透，不查缓存
	// 设计原因：见上方"为什么只缓存首屏"说明
	if page != 1 || size > 20 {
		span.AddEvent("bypass_cache_not_first_page")
		return nil, nil
	}

	key := redisKey(keyActivitiesPrefix + fmt.Sprint(userID))
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			span.AddEvent("cache_miss")
			return nil, nil
		}
		return nil, fmt.Errorf("socialcache.GetActivitiesFirstPage (user_id: %d): %w", userID, err)
	}

	var activities []*entity.Activity
	if err := json.Unmarshal(data, &activities); err != nil {
		return nil, fmt.Errorf("socialcache.GetActivitiesFirstPage unmarshal failed (user_id: %d): %w", userID, err)
	}

	span.AddEvent("cache_hit")
	return activities, nil
}

// SetActivitiesFirstPage 将活动流首屏写入 Redis 缓存
//
// 写入条件：
//   - activities 为 nil 时不缓存（防止缓存穿透）
//   - 空切片（[]）正常缓存，表示该用户暂时没有动态
//
// TTL：1 分钟（由 ttlActivities 控制）
// 短 TTL 原因：活动流是高实时性数据，新发帖/新关注/新评论应尽快出现在用户时间线上。
// 1 分钟过期后自然回源 MySQL，保证数据新鲜度。
//
// 调用方注意：
//
//	调用此方法前应确保 page == 1 && size <= 20（不满足条件时调用此方法无意义）
//
// 被以下位置调用：
//   - social_service.go GetActivities（缓存未命中且数据来自 MySQL 首屏时回写）
func (c *cacheStruct) SetActivitiesFirstPage(ctx context.Context, userID int64, activities []*entity.Activity) error {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.SetActivitiesFirstPage")
	defer span.End()

	// 不缓存 nil 切片：防止缓存穿透
	// 但空切片（[]）可以缓存 — 表示该用户确实没有动态，避免反复查询 MySQL
	if activities == nil {
		span.AddEvent("skip_cache_nil")
		return nil
	}

	key := redisKey(keyActivitiesPrefix + fmt.Sprint(userID))
	data, err := json.Marshal(activities)
	if err != nil {
		return fmt.Errorf("socialcache.SetActivitiesFirstPage marshal failed (user_id: %d): %w", userID, err)
	}

	if err := c.rdb.SetEx(ctx, key, data, ttlActivities).Err(); err != nil {
		return fmt.Errorf("socialcache.SetActivitiesFirstPage (user_id: %d): %w", userID, err)
	}

	span.AddEvent("cache_set")
	return nil
}

// InvalidateActivities 删除活动流首屏缓存
//
// 失效时机（proactive invalidation — 主动失效）：
//   - FollowUser 成功后 — 用户的关注行为会在活动流生成新条目
//   - UnfollowUser 成功后 — 同上
//   - CreatePost 成功后 — 发帖行为会产生 "发布新帖" 的活动
//
// 被动失效：即使不主动调用，ttlActivities（1 分钟）到期后缓存自然过期
// 两种策略结合：
//   - 主动失效（本方法）：写操作后立即删除缓存，下次读回源 MySQL
//   - 被动失效（TTL 过期）：MQ consumer 写入活动时不做失效（无法确定哪个 Service 触发），
//     靠 1min TTL 自然过期，保证最终一致性
//
// 为什么有些场景不主动失效：
//
//	MQ consumer（mq/activity_consumer.go）写入活动时，Service 层并不知道何时写入新活动，
//	因此无法主动失效。1min TTL 过期机制在这些场景下起兜底作用。
//
// 被以下位置调用：
//   - social_service.go FollowUser（关注成功后失效）
//   - social_service.go UnfollowUser（取消关注后失效）
func (c *cacheStruct) InvalidateActivities(ctx context.Context, userID int64) error {
	ctx, span := tracer.Start(ctx, "RedisSocialDAO.InvalidateActivities")
	defer span.End()

	key := redisKey(keyActivitiesPrefix + fmt.Sprint(userID))
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("socialcache.InvalidateActivities (user_id: %d): %w", userID, err)
	}

	span.AddEvent("cache_invalidated")
	return nil
}

// parseInt64 将字符串安全转换为 int64
//
// 为什么需要此辅助函数：
//
//	Redis String 类型存储的整数字符串需要解析为 int64
//	fmt.Sprint(count) 写入 → Get().Result() 读取 → parseInt64 解析
//
// 被本包以下方法调用：
//   - GetFollowerCount
//   - GetFollowingCount
func parseInt64(s string) (int64, error) {
	n := int64(0)
	for _, c := range []byte(s) {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid integer string: %s", s)
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}
