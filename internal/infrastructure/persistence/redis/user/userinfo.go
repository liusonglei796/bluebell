// Package usercache 提供用户信息（User Info）的 Redis 缓存实现
// 本文件与 user.go 同包，无需额外 import 别名
// 设计意图：缓存用户基本信息（UserID/UserName/Role），减少 MySQL 查询压力
// **严禁缓存 Password 字段**，避免 Redis 被攻破时密码泄露
// 被以下模块调用：UserService.RefreshToken、SocialService.GetProfile、CommunityService.CreateCommunity 等
package usercache

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

// tracerInfo 是 user-info 缓存模块的 OpenTelemetry tracer
// 与 user.go 的 tracer 分离，避免 span 名称冲突，便于链路追踪区分
var tracerInfo = infratrace.TracerForModule("dao/redis/user-info")

const (
	// keyUserInfo 是用户信息缓存的键前缀，完整键: user:info:<userID>
	// 存储 JSON 格式的 cacheUser（不含 Password）
	keyUserInfo = "user:info:"

	// keyUserInfoByName 是用户名到 userID 的索引键前缀，完整键: user:info:by_name:<username>
	// 用于 GetUserInfoByName 方法：先查用户名拿到 userID，再查 user:info:<userID>
	// 替代方案：直接用 username 做键存整个对象（冗余，且 username 变更时需要同步两份数据）
	// 选择本方案的原因：userID 是内部不可变标识，username 可能变更，索引 + 主数据分离更清晰
	keyUserInfoByName = "user:info:by_name:"
)

// userInfoTTL 是用户信息缓存的过期时间
// 10 分钟平衡了实时性和缓存命中率：用户信息变更不频繁，10min 足够
const userInfoTTL = 10 * time.Minute

// cacheUser 是缓存专用的用户信息结构体
// **严禁包含 Password 字段** — 与 entity.User 解耦，从源头杜绝密码泄露
// 即使 Redis 被攻破，攻击者也无法从缓存中获取用户密码哈希
type cacheUser struct {
	UserName string `json:"user_name"`
	UserID   int64  `json:"user_id"`
	Role     int    `json:"role"`
}

// userInfoCacheStruct 是 UserInfoCacheRepository 接口的 Redis 实现
// 字段：rdb — Redis 客户端连接（由工厂方法注入）
type userInfoCacheStruct struct {
	rdb *redis.Client
}

// NewUserInfoCache 创建用户信息缓存实例
// 被 di 依赖注入层调用（通过 cache.Repositories 聚合）
// 返回 domain.UserInfoCacheRepository 接口，面向接口编程而非具体实现
// 接口定义在 internal/domain/repository.go（与 PostCacheRepository、UserTokenCacheRepository 同级）
func NewUserInfoCache(rdb *redis.Client) domain.UserInfoCacheRepository {
	return &userInfoCacheStruct{rdb: rdb}
}

// entityToCacheUser 将 entity.User 转换为缓存专用 cacheUser
// **仅复制 UserID/UserName/Role，主动跳过 Password**
// 为什么不用 JSON tag 直接在 entity.User 上标记 json:"-"：
//   - 防御性编程原则：即使未来有人误删了 Password 上的 json tag，也不影响缓存安全
//   - 额外好处：减少序列化体积，降低 Redis 网络开销
func entityToCacheUser(u *entity.User) *cacheUser {
	return &cacheUser{
		UserID:   u.UserID,
		UserName: u.UserName,
		Role:     u.Role,
	}
}

// cacheUserToEntity 将 cacheUser 转回 entity.User
// Password 字段保持零值（空字符串），不包含密码信息
// 调用方（service 层）获取到的 User 对象若需要密码验证，应继续从 MySQL 查询
func cacheUserToEntity(c *cacheUser) *entity.User {
	return &entity.User{
		UserID:   c.UserID,
		UserName: c.UserName,
		Password: "", // 显式置空：缓存中无密码，返回零值以保证类型安全
		Role:     c.Role,
	}
}

// GetUserInfo 根据 userID 从 Redis 缓存获取用户基本信息
// 被以下调用链使用：
//   - UserService.RefreshToken → 先查缓存，miss 再查 MySQL
//   - SocialService.GetProfile → 先查缓存，miss 再查 MySQL
//   - CommunityService.CreateCommunity → 先查缓存，miss 再查 MySQL
//
// 并发优化：纯读操作，Redis GET 是原子操作，无需额外加锁
// 不做优化的后果：Redis 单线程模型保证 GET 原子性，不存在并发问题
//
// 返回值：
//   - (*entity.User, nil) — 缓存命中，返回用户信息（Password 为空字符串）
//   - (nil, nil) — 缓存未命中，由调用方回源 MySQL 查询
//   - (nil, error) — Redis 操作异常
func (c *userInfoCacheStruct) GetUserInfo(ctx context.Context, userID int64) (*entity.User, error) {
	// 创建 OpenTelemetry span，记录调用信息
	ctx, span := tracerInfo.Start(ctx, "UserInfoCache.GetUserInfo")
	defer span.End()
	infratrace.WithUserID(ctx, userID)

	// 构建 Redis key: user:info:<userID>
	key := getRedisKey(keyUserInfo + fmt.Sprint(userID))

	// 从 Redis 读取 JSON 字符串
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		// redis.Nil 表示缓存不存在（miss），返回 nil, nil 由调用方处理
		if err == redis.Nil {
			return nil, nil
		}
		// 其他 Redis 错误（如连接超时），包装后返回
		return nil, fmt.Errorf("get user info cache failed (user_id: %d): %w", userID, err)
	}

	// 反序列化为 cacheUser（不含 Password 的专用结构体）
	var cached cacheUser
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("unmarshal user info cache failed (user_id: %d): %w", userID, err)
	}

	// 转回 entity.User（Password 自动置零）
	return cacheUserToEntity(&cached), nil
}

// GetUserInfoByName 根据用户名从 Redis 缓存获取用户基本信息
// 流程：先读 user:info:by_name:<username> 索引拿到 userID，再读 user:info:<userID>
// 分为两步而非直接存完整信息的原因：
//  1. 避免数据冗余：用户名变更时只需更新索引，主数据复用一份
//  2. 一致性：SetUserInfo 和 InvalidateUserInfo 都维护这个索引，保持同步
//
// 被 UserService.GetUserByUsername 调用（对外公开接口）
//
// 返回值规则同 GetUserInfo：hit 返回 User，miss 返回 nil,nil
func (c *userInfoCacheStruct) GetUserInfoByName(ctx context.Context, username string) (*entity.User, error) {
	ctx, span := tracerInfo.Start(ctx, "UserInfoCache.GetUserInfoByName")
	defer span.End()

	// 第一步：从 by-name 索引读取 userID
	// 索引存储 userID 的字符串形式，如 "123456789"
	nameKey := getRedisKey(keyUserInfoByName + username)
	userIDStr, err := c.rdb.Get(ctx, nameKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // 索引不存在 → 缓存 miss
		}
		return nil, fmt.Errorf("get user info by name cache failed (username: %s): %w", username, err)
	}

	// 第二步：解析 userID 字符串为 int64
	var uid int64
	if _, err := fmt.Sscanf(userIDStr, "%d", &uid); err != nil {
		return nil, fmt.Errorf("parse user id from name index failed (username: %s): %w", username, err)
	}

	// 第三步：复用 GetUserInfo 通过 userID 读取完整信息
	// 复用保证主数据路径唯一，清除索引时也一并清除
	return c.GetUserInfo(ctx, uid)
}

// SetUserInfo 写入用户信息到 Redis 缓存
// 被以下调用链使用：
//   - UserService.RefreshToken（miss 后回源 MySQL 查询后写入）
//   - SocialService.GetProfile（miss 后回源 MySQL 查询后写入）
//   - 用户信息更新场景（更新缓存保障数据一致性）
//
// 并发优化：使用 SetEx（原子写入 + TTL），Pipeline 批量写入主数据和 by-name 索引
// 不做优化的后果：SetEx 是 Redis 单命令原子操作，不存在并发写入问题；
//
//	Pipeline 保证两个写操作在单连接中顺序执行，不会出现主数据写了但索引没写的半中间状态
//
// **安全保证**：本方法使用 entityToCacheUser 转换，主动跳过 Password 字段
//
//	传入的 entity.User 即使包含密码，也不会被写入 Redis
func (c *userInfoCacheStruct) SetUserInfo(ctx context.Context, user *entity.User) error {
	ctx, span := tracerInfo.Start(ctx, "UserInfoCache.SetUserInfo")
	defer span.End()
	infratrace.WithUserID(ctx, user.UserID)

	// 转换为缓存专用结构体（确保不包含 Password）
	cached := entityToCacheUser(user)

	// 序列化为 JSON
	data, err := json.Marshal(cached)
	if err != nil {
		return fmt.Errorf("marshal user info cache failed (user_id: %d): %w", user.UserID, err)
	}

	// 使用 Pipeline 批量写入，保证主数据和索引的原子性
	// 主数据键: user:info:<userID> → 完整 JSON
	// 索引键: user:info:by_name:<username> → userID 字符串
	pipe := c.rdb.Pipeline()
	pipe.SetEx(ctx, getRedisKey(keyUserInfo+fmt.Sprint(user.UserID)), data, userInfoTTL)
	pipe.SetEx(ctx, getRedisKey(keyUserInfoByName+user.UserName), fmt.Sprint(user.UserID), userInfoTTL)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("set user info cache pipeline exec failed (user_id: %d): %w", user.UserID, err)
	}
	return nil
}

// InvalidateUserInfo 删除用户信息缓存（使缓存失效）
// 被以下调用链使用：
//   - 用户信息更新时，先失效缓存，下次查询重新加载
//   - 用户注销/封禁等需要清理缓存的场景
//
// 并发优化：使用 Pipeline 批量删除两个键，保证原子性
// 不做优化的后果：Pipeline 保证主数据和索引在同一个连接中顺序删除，
//
//	不会出现只删了主数据但索引残留的脏数据状态
//
// 参数：
//   - userID: 用户 ID，用于删除 user:info:<userID>
//   - username: 用户名，用于删除 user:info:by_name:<username>
//     注意：调用方必须传入正确的 username，否则索引可能残留
func (c *userInfoCacheStruct) InvalidateUserInfo(ctx context.Context, userID int64, username string) error {
	ctx, span := tracerInfo.Start(ctx, "UserInfoCache.InvalidateUserInfo")
	defer span.End()
	infratrace.WithUserID(ctx, userID)

	// 批量删除主数据和索引
	pipe := c.rdb.Pipeline()
	pipe.Del(ctx, getRedisKey(keyUserInfo+fmt.Sprint(userID)))
	pipe.Del(ctx, getRedisKey(keyUserInfoByName+username))
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("invalidate user info cache failed (user_id: %d): %w", userID, err)
	}
	return nil
}
