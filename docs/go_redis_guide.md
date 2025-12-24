# Go-Redis 实战指南 (基于 Bluebell 项目)

## 1. 简介

在 Bluebell 项目中，我们使用 `github.com/redis/go-redis/v9` 作为 Redis 客户端。Redis 在本项目中扮演着至关重要的角色，主要用于：
*   **缓存**: 存储用户的 Token (Access Token / Refresh Token) 以实现高性能的认证鉴权。
*   **排行榜/投票系统**: 利用 Redis 的 `ZSet` (Sorted Set) 数据结构实现高性能的帖子投票和热榜排序（基于时间和分数）。

本文档将结合项目中的实际代码 (`dao/redis` 目录)，讲解 `go-redis` 的核心用法和最佳实践。

## 2. 客户端初始化

Redis 客户端的初始化通常在项目启动时完成，并维护一个全局的单例对象供后续调用。

**代码位置**: `dao/redis/redis.go`

```go
package redis

import (
	"context"
	"fmt"
	"time"
	"bluebell/settings"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var (
	rdb *redis.Client // 全局 Redis 客户端实例
	ctx = context.Background()
)

// Init 初始化连接池
func Init(cfg *settings.RedisConfig) error {
	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password, // 密码，没有则留空
		DB:       cfg.DB,       // 使用的数据库编号
		PoolSize: cfg.PoolSize, // 连接池大小
	})

	// 使用带超时的 Context 进行 Ping 测试，防止程序启动时因 Redis 不可用而卡死
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("connect to redis failed: %w", err)
	}

	return nil
}

func Close() {
	_ = rdb.Close()
}
```

### 关键点
*   **连接池**: `go-redis` 内部自动维护连接池，因此 `rdb` 对象是并发安全的，不需要为每个请求创建新连接。
*   **Context**: 所有 Redis 操作都需要传入 `context.Context`，用于控制超时和取消。

## 3. 基础操作 (String/Key-Value)

String 是 Redis 最基本的数据类型，在本项目中主要用于存储用户的登录 Token。

**代码位置**: `dao/redis/user.go`

```go
// SetUserToken 存储 Token
// 演示了 Set 用法，包含过期时间 (expiration)
func SetUserToken(userID int64, aToken, rToken string, aExp, rExp time.Duration) error {
    // ...
    // rdb.Set(ctx, key, value, expiration)
    err := rdb.Set(ctx, getRedisKey(KeyUserAccessToken+fmt.Sprint(userID)), aToken, aExp).Err()
    // ...
}

// GetUserAccessToken 获取 Token
// 演示了 Get 用法
func GetUserAccessToken(userID int64) (string, error) {
    // rdb.Get(ctx, key).Result() 返回 (string, error)
    return rdb.Get(ctx, getRedisKey(KeyUserAccessToken+fmt.Sprint(userID))).Result()
}
```

## 4. 高级操作 (ZSet/Pipeline)

这是本项目最核心的 Redis 用法。我们使用 Sorted Set (ZSet) 来存储帖子的分数和发布时间，从而实现按热度或时间排序的列表。

### 4.1 ZSet 基本操作

`ZSet` 存储的是 `Member` (元素) 和 `Score` (分数) 的对。

**代码位置**: `dao/redis/vote.go`

```go
// 示例：获取帖子的发帖时间
// ZScore 返回指定 member 的 score
postTime := rdb.ZScore(ctx, getRedisKey(KeyPostTimeZSet), postID).Val()

// 示例：按分数分页获取帖子列表
// ZRangeArgs 支持复杂的范围查询，Rev: true 表示降序（分数高的在前）
ids, err := rdb.ZRangeArgs(ctx, redis.ZRangeArgs{
    Key:   key,
    Start: start, // 分页起始索引
    Stop:  end,   // 分页结束索引
    Rev:   true,  // 降序
}).Result()
```

### 4.2 Pipeline (管道)与事务

当需要执行一组 Redis 命令时，使用 Pipeline 可以显著减少网络往返时间 (RTT)。`TxPipeline` 还可以通过 `MULTI/EXEC` 指令包裹，提供简单的事务性（保证命令序列原子执行）。

**场景**: 用户给帖子投票时，需要同时更新：
1.  帖子全局分数 (`ZIncrBy`)
2.  社区内帖子分数 (`ZIncrBy`)
3.  用户对该帖子的投票记录 (`ZAdd` 或 `ZRem`)

**代码位置**: `dao/redis/vote.go`

```go
func VoteForPost(userID, postID, communityID string, value float64) error {
    // ... 前置逻辑省略 ...

    // 开启事务管道
    pipeline := rdb.TxPipeline()

    // 1. 更新全局帖子分数
    pipeline.ZIncrBy(ctx, getRedisKey(KeyPostScoreZSet), scoreDiff, postID)

    // 2. 更新社区帖子分数
    pipeline.ZIncrBy(ctx, getRedisKey(KeyCommunityPostScorePrefix+communityID), scoreDiff, postID)

    // 3. 更新用户投票记录
    if value == 0 {
        pipeline.ZRem(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), userID)
    } else {
        pipeline.ZAdd(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), redis.Z{
            Score:  value,
            Member: userID,
        })
    }

    // 执行管道中的所有命令
    _, err := pipeline.Exec(ctx)
    return err
}
```

---

## 5. 进阶实战：缓存模式 (Cache Aside)

在项目中，我们可以使用 Redis 来缓存那些查询频繁但更新较少的数据（如：社区列表）。

### 5.1 社区列表查询示例 (模拟代码)

```go
func GetCommunityList() (list []*models.Community, err error) {
    key := "bluebell:communities"
    
    // 1. 尝试从 Redis 获取缓存
    val, err := rdb.Get(ctx, key).Result()
    if err == nil {
        // 缓存命中，解析 JSON 并返回
        json.Unmarshal([]byte(val), &list)
        return
    }
    
    // 2. 缓存失效或不存在，查询数据库 (MySQL)
    list, err = mysql.GetAllCommunities()
    if err != nil {
        return nil, err
    }
    
    // 3. 将结果写入缓存，并设置过期时间 (如 10 分钟)
    data, _ := json.Marshal(list)
    rdb.Set(ctx, key, data, 10*time.Minute)
    
    return
}
```

---

## 6. 基础数据结构操作补充

### 6.1 Hash (哈希) - 适合存储对象
相比于把对象转成 JSON 存 String，Hash 可以单独修改或获取某个字段，节省流量且效率更高。

```go
func HashExample() {
    key := "user:1001"

    // 1. 设置多个字段 (HMSet 的现代写法)
    rdb.HSet(ctx, key, map[string]interface{}{
        "name": "Lay",
        "age":  25,
        "city": "Beijing",
    })

    // 2. 获取单个字段
    name := rdb.HGet(ctx, key, "name").Val()

    // 3. 字段自增
    rdb.HIncrBy(ctx, key, "age", 1) // 年龄加1

    // 4. 获取所有字段
    data := rdb.HGetAll(ctx, key).Val() // 返回 map[string]string
    fmt.Println(data["name"], data["age"])
}
```

### 6.2 List (列表) - 消息队列 / 动态
List 是有序的字符串列表，可以从两端插入或弹出。

```go
func ListExample() {
    key := "user:1001:notifications"

    // 1. 从左侧推入 (LPush)
    rdb.LPush(ctx, key, "msg1", "msg2", "msg3")

    // 2. 修剪列表 (只保留最新的100条)
    rdb.LTrim(ctx, key, 0, 99)

    // 3. 获取范围内的元素 (0 -1 表示获取全部)
    msgs := rdb.LRange(ctx, key, 0, 10).Val()

    // 4. 从右侧弹出 (RPop) - 实现 FIFO 队列
    lastMsg := rdb.RPop(ctx, key).Val()
}
```

### 6.3 Set (集合) - 去重 / 社交关系
Set 是无序的，且元素唯一。非常适合做点赞列表、关注列表。

```go
func SetExample() {
    key1 := "user:1:follow"
    key2 := "user:2:follow"

    // 1. 添加元素
    rdb.SAdd(ctx, key1, "user:10", "user:11", "user:12")
    rdb.SAdd(ctx, key2, "user:12", "user:13")

    // 2. 检查是否在集合中 (例如判断是否已点赞)
    isFollowing := rdb.SIsMember(ctx, key1, "user:10").Val()

    // 3. 求交集 (共同关注)
    common := rdb.SInter(ctx, key1, key2).Val() // 返回 ["user:12"]

    // 4. 获取集合所有成员
    all := rdb.SMembers(ctx, key1).Val()
}
```

### 6.4 Bitmap (位图) - 签到 / 活跃统计
位图非常节省空间，1 亿个用户一天的签到状态只需约 12MB。

```go
func BitmapExample() {
    key := "user:sign:202312" // 2023年12月的签到记录
    userID := 1001

    // 1. 签到：将第 23 位设置为 1 (代表12月23日签到)
    rdb.SetBit(ctx, key, int64(userID), 1)

    // 2. 查询是否签到
    status := rdb.GetBit(ctx, key, int64(userID)).Val()

    // 3. 统计总签到人数
    count := rdb.BitCount(ctx, key, nil).Val()
}
```

### 6.5 HyperLogLog - 统计基数 (UV)
用于统计不重复的元素个数（如每日访问人数），误差极小且占用内存固定（约 12KB）。

```go
func UVExample() {
    key := "page:uv:20231223"

    // 添加访问者 IP
    rdb.PFAdd(ctx, key, "192.168.0.1", "192.168.0.2", "192.168.0.1")

    // 获取去重后的总数
    count := rdb.PFCount(ctx, key).Val() // 结果为 2
}
```

---

## 7. 高级特性补充

### 7.1 Lua 脚本原子操作

当多个操作需要完全原子性且依赖逻辑判断时，Lua 脚本是最佳选择。

```go
var incrByScript = redis.NewScript(`
    local current = redis.call("get", KEYS[1])
    if current and tonumber(current) > tonumber(ARGV[1]) then
        return redis.call("incrby", KEYS[1], ARGV[2])
    else
        return nil
    end
`)

// 使用示例
func AtomicIncr(key string) {
    res, err := incrByScript.Run(ctx, rdb, []string{key}, "100", "1").Result()
    // ...
}
```

### 7.2 Redis Stream (消息队列进阶)

如果你需要更强大的消息队列（支持多消费者组、ACK、消息持久化），应使用 Stream。

```go
func StreamExample() {
    // 发送消息
    rdb.XAdd(ctx, &redis.XAddArgs{
        Stream: "mystream",
        Values: map[string]interface{}{"event": "post_created", "id": 123},
    })

    // 读取消息 (从头开始)
    msgs, _ := rdb.XRead(ctx, &redis.XReadArgs{
        Streams: []string{"mystream", "0"},
        Count:   10,
        Block:   0,
    }).Result()
}
```

### 7.3 分布式锁 (Redlock 思路)

```go
func SimpleLock(key string, ttl time.Duration) (string, bool) {
    val := uuid.New().String()
    // NX: Key 不存在时才设置; EX: 设置过期时间
    ok, err := rdb.SetNX(ctx, key, val, ttl).Result()
    if err != nil || !ok {
        return "", false
    }
    return val, true
}
```

---

## 8. 性能优化与故障排查

1.  **连接池优化**: 在高并发场景下，适当调大 `PoolSize`。
2.  **避免大 Key**: 一个 Hash 或 Set 中不要存储超过 10,000 个元素，否则会引起阻塞。
3.  **慢查询分析**: 使用 `slowlog get 128` 查看 Redis 端的慢查询。
4.  **序列化选型**: 存储对象时，如果对性能要求极高，考虑使用 `Protobuf` 或 `MessagePack` 代替 `JSON`。

---

## 9. 总结

`go-redis` 在 Bluebell 项目中通过 **Sorted Set** 完美解决了复杂的排序问题，并通过 **Pipeline** 降低了延迟。掌握好这些模式及其进阶特性（如 Lua、缓存策略），将使你的 Go 后端服务具备极高的吞吐能力和稳定性。