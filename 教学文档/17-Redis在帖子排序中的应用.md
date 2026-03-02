# 第17章:Redis在帖子排序中的应用

> **本章导读**
>
> 上一章我们学习了投票系统的业务规则和参数设计,但核心的**投票如何实时更新分数**、**帖子如何按分数排序**还是黑盒。
>
> 本章将揭开Redis的神秘面纱,深入讲解ZSet数据结构、Pipeline原子性、投票功能的完整实现,以及如何利用Redis实现高性能的帖子排序。

---

## 📚 本章目标

学完本章,你将掌握:

1. 理解Redis ZSet的数据结构和应用场景
2. 掌握Redis Key的设计规范
3. 实现投票功能的完整Redis逻辑
4. 理解Redis Pipeline的原子性保证
5. 学习批量查询的性能优化
6. 实现基于Redis的帖子排序
7. 掌握ZSet的常用命令和操作
8. 理解时间窗口限制的实现原理

---

## 1. Redis数据结构选型

### 1.1 为什么选择ZSet?

**ZSet (Sorted Set) = 有序集合**

**核心特性:**
- **有序**: 自动按score(分数)排序
- **唯一**: member(成员)不重复
- **高效**: 查询、插入、删除都是O(log N)

---

**其他数据结构对比:**

| 数据结构 | 有序? | 去重? | 排序性能 | 适合投票吗? |
|---------|------|------|---------|-----------|
| **String** | ❌ | - | - | ❌ 无法排序 |
| **List** | ✅ | ❌ | O(N log N) | ❌ 允许重复 |
| **Set** | ❌ | ✅ | - | ❌ 无序 |
| **Hash** | ❌ | ✅ | O(N log N) | ❌ 需要额外排序 |
| **ZSet** | ✅ | ✅ | O(log N) | ✅ 完美匹配! |

---

### 1.2 ZSet在投票系统中的应用

**需求1: 存储帖子分数 (按热度排序)**
```redis
ZSet: bluebell:post:score
Member: 帖子ID (如 "123456789")
Score: 帖子分数 (如 1735820432.0)

示例:
ZADD bluebell:post:score 1735820432 "123456789"
ZADD bluebell:post:score 1735819500 "987654321"

→ 自动按score降序排列,实现热度排行榜
```

---

**需求2: 存储帖子发布时间 (按时间排序)**
```redis
ZSet: bluebell:post:time
Member: 帖子ID
Score: 发布时间戳

示例:
ZADD bluebell:post:time 1735776000 "123456789"

→ 自动按时间降序排列,实现最新帖子列表
```

---

**需求3: 存储用户投票记录**
```redis
ZSet: bluebell:post:voted:{post_id}
Member: 用户ID
Score: 投票方向 (1:赞成, -1:反对)

示例:
# 帖子123456789的投票记录
ZADD bluebell:post:voted:123456789 1 "user001"   # user001投赞成票
ZADD bluebell:post:voted:123456789 -1 "user002"  # user002投反对票
ZADD bluebell:post:voted:123456789 1 "user003"   # user003投赞成票

→ 每个帖子一个ZSet,记录所有投票用户
```

---

## 2. Redis Key设计规范

### 2.1 Key命名规范

**为什么需要规范?**
```redis
# ❌ 不好: key命名混乱
post:score       ← 缺少命名空间
postScore        ← 驼峰命名不统一
post_score       ← 下划线和冒号混用
```

**✅ Bluebell的命名规范:**
```
命名格式: {项目名}:{模块}:{业务}:{ID}

示例:
bluebell:post:score                  ← 帖子分数排行榜
bluebell:post:time                   ← 帖子时间排行榜
bluebell:post:voted:123456789        ← 帖子123456789的投票记录
bluebell:user:token:user001          ← 用户user001的Token
```

**规范要点:**
1. **统一前缀**: `bluebell:` (项目命名空间)
2. **使用冒号**: `:` 分隔层级
3. **小写字母**: 全部使用小写
4. **语义化**: 一看就懂业务含义

---

### 2.2 Key常量定义

**dao/redis/keys.go**
```go
package redis

const (
	// KeyPrefix 统一的命名空间前缀
	KeyPrefix = "bluebell:"

	// KeyPostTimeZSet 帖子发布时间ZSet
	// 用于按时间排序获取帖子列表
	KeyPostTimeZSet = "post:time"

	// KeyPostScoreZSet 帖子分数ZSet
	// 用于按热度排序获取帖子列表
	KeyPostScoreZSet = "post:score"

	// KeyPostVotedZSetPrefix 用户投票记录ZSet前缀
	// 完整key: bluebell:post:voted:{post_id}
	// 用于记录每个帖子的所有投票用户
	KeyPostVotedZSetPrefix = "post:voted:"
)

// getRedisKey 拼接完整的Redis Key
// 统一管理Key的生成,避免硬编码
func getRedisKey(key string) string {
	return KeyPrefix + key
}
```

---

### 2.3 Key设计的最佳实践

**原则1: 统一管理**
```go
// ❌ 不好: key硬编码在业务代码中
func VoteForPost(postID string) error {
	rdb.ZIncrBy(ctx, "bluebell:post:score", 432, postID) // 硬编码
}

// ✅ 好: 使用常量和函数
func VoteForPost(postID string) error {
	rdb.ZIncrBy(ctx, getRedisKey(KeyPostScoreZSet), 432, postID)
}
```

**好处:**
- 修改key只需改一处
- 避免拼写错误
- 代码可读性高

---

**原则2: 语义化命名**
```redis
# ❌ 不好: 缩写难懂
bluebell:p:s        ← 什么意思?
bluebell:pv:123     ← pv是什么?

# ✅ 好: 一目了然
bluebell:post:score         ← 帖子分数
bluebell:post:voted:123     ← 帖子123的投票记录
```

---

**原则3: 层级清晰**
```redis
# ❌ 不好: 平铺结构
bluebell:post_time
bluebell:post_score
bluebell:user_token

# ✅ 好: 层级结构
bluebell:post:time
bluebell:post:score
bluebell:user:token
```

**好处:**
- 方便使用 `KEYS bluebell:post:*` 查询
- 层级关系一目了然
- 便于监控和管理

---

## 3. 投票功能完整实现

### 3.1 创建帖子时的初始化

**dao/redis/vote.go**
```go
// CreatePost 创建帖子时初始化 Redis 数据
// 在发帖时调用,设置帖子的初始分数和发布时间
func CreatePost(postID, communityID int64) error {
	// 使用 Pipeline 保证两个操作的原子性
	pipeline := rdb.TxPipeline()

	// 1. 将帖子发布时间存入 ZSet
	// key: bluebell:post:time
	// score: 当前时间戳
	// member: postID
	pipeline.ZAdd(ctx, getRedisKey(KeyPostTimeZSet), redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: postID,
	})

	// 2. 将帖子初始分数存入 ZSet
	// key: bluebell:post:score
	// score: 初始分数(发布时间戳)
	// member: postID
	// 注意: 初始分数设置为发布时间戳,这样新帖子会排在前面
	pipeline.ZAdd(ctx, getRedisKey(KeyPostScoreZSet), redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: postID,
	})

	// 3. 执行 Pipeline
	// 返回两个命令的执行结果
	_, err := pipeline.Exec(ctx)
	return err
}
```

**为什么初始分数是时间戳?**
```
帖子A: 发布于 2025-01-01 00:00:00
  时间戳 = 1735689600
  初始分数 = 1735689600
  → 还没人投票,按时间排序

帖子B: 发布于 2025-01-01 12:00:00
  时间戳 = 1735732800
  初始分数 = 1735732800
  → 时间戳更大,排在前面

帖子A获得100票:
  分数 = 1735689600 + 100*432 = 1735732800
  → 现在和帖子B分数相同! (100票抵消了12小时的时间差)
```

---

### 3.2 投票核心逻辑

**dao/redis/vote.go**
```go
// 投票相关常量
const (
	// 一周的秒数,超过一周的帖子不允许投票
	OneWeekInSeconds = 7 * 24 * 3600

	// 每一票的分数权重: 86400秒/天 ÷ 200票 = 432分/票
	// 含义: 一个帖子需要200张赞成票才能在热榜上"续命"一天
	ScorePerVote = 432
)

// 投票相关错误
var (
	ErrVoteTimeExpire = errors.New("投票时间已过")
	ErrVoteRepeated   = errors.New("不允许重复投票")
)

// VoteForPost 为帖子投票
// 参数:
//   userID: 投票用户ID (字符串格式)
//   postID: 目标帖子ID (字符串格式)
//   value: 投票值 (1:赞成, -1:反对, 0:取消投票)
func VoteForPost(userID, postID string, value float64) error {
	// 1. 判断投票时间限制
	// 从 Redis 的 ZSet 中获取帖子的发布时间戳
	postTime := rdb.ZScore(ctx, getRedisKey(KeyPostTimeZSet), postID).Val()

	// 如果当前时间距离发帖时间超过一周,不允许投票
	if float64(time.Now().Unix())-postTime > OneWeekInSeconds {
		return ErrVoteTimeExpire
	}

	// 2. 查询用户之前对该帖子的投票记录
	// key: bluebell:post:voted:{post_id}
	// 该 ZSet 的 member 是 userID, score 是投票值(1/-1/0)
	oldValue := rdb.ZScore(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), userID).Val()

	// 3. 如果新旧投票值相同,直接返回(避免重复投票)
	if value == oldValue {
		return ErrVoteRepeated
	}

	// 4. 计算分数变化
	// op: 操作方向 (1表示加分, -1表示减分)
	var op float64
	if value > oldValue {
		op = 1 // 例如: 从0到1, 从-1到0, 从-1到1 都是加分
	} else {
		op = -1 // 例如: 从1到0, 从0到-1, 从1到-1 都是减分
	}

	// diff: 新旧投票值的差值绝对值
	// 例如: 从1变为-1, diff=2; 从0变为1, diff=1
	diff := math.Abs(value - oldValue)

	// 5. 使用 Redis Pipeline 保证原子性
	// 需要同时更新两个 ZSet: 帖子分数表 和 用户投票记录表
	pipeline := rdb.TxPipeline()

	// 5.1 更新帖子的总分数
	// key: bluebell:post:score
	// 分数变化 = 操作方向 * 差值 * 单票分数
	pipeline.ZIncrBy(ctx, getRedisKey(KeyPostScoreZSet), op*diff*ScorePerVote, postID)

	// 5.2 更新用户的投票记录
	if value == 0 {
		// 如果是取消投票,从 ZSet 中删除该用户记录
		pipeline.ZRem(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), userID)
	} else {
		// 否则,添加或更新用户的投票记录
		pipeline.ZAdd(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), redis.Z{
			Score:  value,  // 1 或 -1
			Member: userID, // 用户ID
		})
	}

	// 6. 执行 Pipeline 中的所有命令
	_, err := pipeline.Exec(ctx)
	return err
}
```

---

### 3.3 投票逻辑分步详解

**Step1: 检查时间窗口**
```go
// ZSCORE bluebell:post:time {postID}
postTime := rdb.ZScore(ctx, getRedisKey(KeyPostTimeZSet), postID).Val()

if float64(time.Now().Unix())-postTime > OneWeekInSeconds {
	return ErrVoteTimeExpire
}
```

**示例:**
```
当前时间: 2025-01-08 00:00:00 (1735862400)
帖子发布时间: 2025-01-01 00:00:00 (1735689600)
时间差: 1735862400 - 1735689600 = 172800 秒 = 2天

2天 < 7天 → 允许投票 ✅

如果时间差 > 604800秒(7天) → 禁止投票 ❌
```

---

**Step2: 查询旧投票记录**
```go
// ZSCORE bluebell:post:voted:{postID} {userID}
oldValue := rdb.ZScore(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), userID).Val()
```

**可能的结果:**
- `oldValue = 1` → 之前投了赞成票
- `oldValue = -1` → 之前投了反对票
- `oldValue = 0` (或Redis返回Nil) → 没有投过票

---

**Step3: 检查是否重复投票**
```go
if value == oldValue {
	return ErrVoteRepeated
}
```

**示例:**
```
旧值 = 1 (赞成)
新值 = 1 (赞成)
→ 重复投票,返回错误 ❌

旧值 = 1 (赞成)
新值 = -1 (反对)
→ 改票,继续执行 ✅
```

---

**Step4: 计算分数变化**
```go
var op float64
if value > oldValue {
	op = 1  // 加分
} else {
	op = -1 // 减分
}

diff := math.Abs(value - oldValue)
```

**示例计算:**

**Case 1: 从未投票(0) → 赞成(1)**
```
value = 1, oldValue = 0
value > oldValue → op = 1
diff = |1 - 0| = 1
分数变化 = 1 * 1 * 432 = +432
```

**Case 2: 从赞成(1) → 反对(-1)**
```
value = -1, oldValue = 1
value < oldValue → op = -1
diff = |-1 - 1| = 2
分数变化 = -1 * 2 * 432 = -864
```

**Case 3: 从反对(-1) → 取消(0)**
```
value = 0, oldValue = -1
value > oldValue → op = 1
diff = |0 - (-1)| = 1
分数变化 = 1 * 1 * 432 = +432
```

---

**Step5: Pipeline原子更新**
```go
pipeline := rdb.TxPipeline()

// 更新帖子分数
pipeline.ZIncrBy(ctx, getRedisKey(KeyPostScoreZSet), op*diff*ScorePerVote, postID)

// 更新投票记录
if value == 0 {
	pipeline.ZRem(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), userID)
} else {
	pipeline.ZAdd(ctx, getRedisKey(KeyPostVotedZSetPrefix+postID), redis.Z{
		Score:  value,
		Member: userID,
	})
}

_, err := pipeline.Exec(ctx)
```

**为什么用Pipeline?**
- 保证两个操作的**原子性**
- 减少网络往返次数(1次RTT)
- 提高性能

---

## 4. Redis Pipeline详解

### 4.1 什么是Pipeline?

**传统方式: 每条命令一次网络往返**
```
客户端                       Redis服务器
  |                             |
  |--- ZINCRBY score 432 123 -->|
  |<-- OK --------------------- |  ← RTT1
  |                             |
  |--- ZADD voted:123 1 u001 -->|
  |<-- OK --------------------- |  ← RTT2
  |                             |

总耗时 = RTT1 + RTT2 = 2ms + 2ms = 4ms
```

---

**Pipeline方式: 批量发送,一次往返**
```
客户端                       Redis服务器
  |                             |
  |--- ZINCRBY score 432 123 -->|
  |--- ZADD voted:123 1 u001 -->|
  |<-- OK, OK ----------------- |  ← RTT
  |                             |

总耗时 = RTT = 2ms (性能提升50%)
```

---

### 4.2 Pipeline的使用方式

**基本用法:**
```go
// 1. 创建Pipeline
pipeline := rdb.TxPipeline() // 或 rdb.Pipeline()

// 2. 添加命令(不会立即执行)
pipeline.ZIncrBy(ctx, "key1", 432, "member1")
pipeline.ZAdd(ctx, "key2", redis.Z{Score: 1, Member: "member2"})
pipeline.Del(ctx, "key3")

// 3. 批量执行
cmders, err := pipeline.Exec(ctx)
if err != nil {
	// 处理错误
}
```

---

### 4.3 Pipeline vs TxPipeline

**Pipeline (普通管道):**
```go
pipeline := rdb.Pipeline()
```
- 批量执行命令
- **非原子**: 命令之间可能被其他客户端的命令插入
- 性能最高

---

**TxPipeline (事务管道):**
```go
pipeline := rdb.TxPipeline()
```
- 批量执行命令
- **原子性**: 所有命令要么全部成功,要么全部失败
- 相当于 `MULTI + 命令 + EXEC`
- 性能略低于Pipeline

---

**什么时候用TxPipeline?**
```go
// ✅ 需要原子性: 投票功能
pipeline := rdb.TxPipeline()
pipeline.ZIncrBy(ctx, "post:score", 432, "123")    // 更新分数
pipeline.ZAdd(ctx, "post:voted:123", redis.Z{...}) // 更新投票记录
// → 两个操作必须同时成功,否则数据不一致

// ❌ 不需要原子性: 批量查询
pipeline := rdb.Pipeline()
for _, postID := range postIDs {
	pipeline.ZScore(ctx, "post:score", postID) // 只是查询,不修改
}
```

---

## 5. 帖子排序实现

### 5.1 按时间排序

**dao/redis/vote.go**
```go
// GetPostIDsInOrder 按照指定顺序获取帖子ID列表
// orderKey: "time" 或 "score"
// page: 页码(从1开始)
// size: 每页数量
func GetPostIDsInOrder(orderKey string, page, size int64) ([]string, error) {
	// 1. 确定查询的 Redis Key
	key := getRedisKey(KeyPostTimeZSet)
	if orderKey == "score" {
		key = getRedisKey(KeyPostScoreZSet)
	}

	// 2. 计算分页的起始和结束位置
	// Redis ZSet 的索引从0开始
	start := (page - 1) * size
	end := start + size - 1

	// 3. 按分数从大到小查询 (ZREVRANGE)
	// 返回的是帖子ID列表 (字符串数组)
	return rdb.ZRevRange(ctx, key, start, end).Result()
}
```

---

### 5.2 Redis命令对应关系

**ZREVRANGE详解:**
```redis
# 语法
ZREVRANGE key start stop [WITHSCORES]

# 含义
# 返回有序集合中,指定区间内的成员
# 按照score从大到小排序 (REV = Reverse)

# 示例数据
ZADD bluebell:post:time 1735776000 "post1"
ZADD bluebell:post:time 1735775000 "post2"
ZADD bluebell:post:time 1735774000 "post3"

# 查询前2条
ZREVRANGE bluebell:post:time 0 1
# 返回: ["post1", "post2"]

# 查询第3-5条 (start=2, stop=4)
ZREVRANGE bluebell:post:time 2 4
# 返回: ["post3"]
```

---

**对比ZRANGE:**
```redis
# ZRANGE: 从小到大排序 (旧帖子在前)
ZRANGE bluebell:post:time 0 -1
# 返回: ["post3", "post2", "post1"]

# ZREVRANGE: 从大到小排序 (新帖子在前)
ZREVRANGE bluebell:post:time 0 -1
# 返回: ["post1", "post2", "post3"]
```

---

### 5.3 分页查询示例

**场景: 每页10条,查询第2页**
```go
page := 2
size := 10

start := (2 - 1) * 10 = 10
end := 10 + 10 - 1 = 19

// Redis命令
ZREVRANGE bluebell:post:score 10 19
// 返回: 第11-20条帖子ID
```

---

**场景: 查询热榜前100**
```go
page := 1
size := 100

start := 0
end := 99

ZREVRANGE bluebell:post:score 0 99
// 返回: 前100个帖子ID
```

---

## 6. 批量查询投票数据

### 6.1 问题场景

**需求:** 帖子列表中,每个帖子需要显示投票数

**❌ 不好的做法: N+1查询**
```go
for _, postID := range postIDs {  // 假设有10个帖子
	// 每个帖子都查询一次Redis
	votes := rdb.ZCount(ctx, "bluebell:post:voted:"+postID, "1", "1").Val()
	// 总共: 1 + 10 = 11次查询!
}
```

---

**✅ 好的做法: 使用Pipeline批量查询**
```go
// GetPostsVoteData 批量获取多个帖子的投票数(赞成票数)
// 使用 Redis Pipeline 提高性能
func GetPostsVoteData(ids []string) (data []int64, err error) {
	// 使用 Pipeline 减少 RTT (Round Trip Time)
	pipeline := rdb.Pipeline()

	// 1. 组装 Pipeline 命令
	for _, id := range ids {
		key := getRedisKey(KeyPostVotedZSetPrefix + id)
		// ZCount 计算分数在 [1, 1] 之间的数量,即赞成票的数量
		pipeline.ZCount(ctx, key, "1", "1")
	}

	// 2. 执行 Pipeline
	cmders, err := pipeline.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// 3. 获取结果
	data = make([]int64, 0, len(cmders))
	for _, cmder := range cmders {
		// 类型断言,从 cmder 中拿到 IntCmd 的结果
		v := cmder.(*redis.IntCmd).Val()
		data = append(data, v)
	}
	return
}
```

---

### 6.2 ZCount命令详解

**语法:**
```redis
ZCOUNT key min max
```

**含义:**
- 统计score在 [min, max] 范围内的成员数量

**示例:**
```redis
# 假设帖子123的投票记录:
ZADD bluebell:post:voted:123 1 "user001"   # 赞成
ZADD bluebell:post:voted:123 -1 "user002"  # 反对
ZADD bluebell:post:voted:123 1 "user003"   # 赞成
ZADD bluebell:post:voted:123 -1 "user004"  # 反对
ZADD bluebell:post:voted:123 1 "user005"   # 赞成

# 统计赞成票数
ZCOUNT bluebell:post:voted:123 1 1
# 返回: 3

# 统计反对票数
ZCOUNT bluebell:post:voted:123 -1 -1
# 返回: 2

# 统计总投票数
ZCOUNT bluebell:post:voted:123 -1 1
# 返回: 5
```

---

### 6.3 Pipeline批量查询的性能对比

**场景: 查询10个帖子的投票数**

| 方案 | 查询次数 | 网络往返 | 耗时 |
|------|---------|---------|------|
| **循环查询** | 10次 | 10次RTT | 10 * 2ms = 20ms |
| **Pipeline** | 10次 | 1次RTT | 2ms |
| **提升** | - | 减少9次RTT | 快10倍 |

**场景: 查询100个帖子的投票数**

| 方案 | 查询次数 | 网络往返 | 耗时 |
|------|---------|---------|------|
| **循环查询** | 100次 | 100次RTT | 100 * 2ms = 200ms |
| **Pipeline** | 100次 | 1次RTT | 2ms |
| **提升** | - | 减少99次RTT | 快100倍 |

---

## 7. Logic层集成

### 7.1 投票业务逻辑

**logic/vote.go**
```go
package logic

import (
	"bluebell/dao/redis"
	"bluebell/models"
	"strconv"

	"go.uber.org/zap"
)

// VoteForPost 投票业务逻辑
// 参数:
//   userID: 投票用户ID
//   p: 投票参数(包含帖子ID和投票方向)
func VoteForPost(userID int64, p *models.ParamVoteData) error {
	// 记录投票操作日志
	zap.L().Debug("VoteForPost",
		zap.Int64("userID", userID),
		zap.Int64("postID", p.PostID),
		zap.Int8("direction", p.Direction))

	// 调用 Redis 层执行投票逻辑
	// 将 int64 类型的 ID 转换为 string (Redis 中统一使用 string)
	// 将 int8 类型的 direction 转换为 float64 (Redis ZSet 的 score 是 float64)
	return redis.VoteForPost(
		strconv.FormatInt(userID, 10),
		strconv.FormatInt(p.PostID, 10),
		float64(p.Direction),
	)
}
```

**为什么要类型转换?**
```go
// Redis ZSet 的 score 是 float64
// Redis 的 key 和 member 都是 string

// Controller 传来的是: int64 和 int8
userID int64 = 123456
postID int64 = 789012
direction int8 = 1

// 转换为 Redis 需要的类型
userID string = "123456"
postID string = "789012"
direction float64 = 1.0
```

---

### 7.2 获取排序后的帖子列表

**logic/post.go**
```go
// GetPostList2 升级版获取帖子列表
// 从 Redis 获取排序后的 ID,再从 MySQL 查询详情,最后组装投票数据
func GetPostList2(p *models.ParamPostList) (data []*models.ApiPostDetail, err error) {
	// 1. 从 Redis 查询帖子 ID 列表(已按时间或分数排序)
	ids, err := redis.GetPostIDsInOrder(p.Order, p.Page, p.Size)
	if err != nil {
		return
	}

	// 2. 处理空数据
	if len(ids) == 0 {
		zap.L().Warn("redis.GetPostIDsInOrder() return 0 data")
		// 返回空切片而不是 nil
		data = make([]*models.ApiPostDetail, 0)
		return
	}

	// 3. 根据 ID 列表从 MySQL 查询帖子详细信息(保持顺序)
	posts, err := mysql.GetPostListByIDs(ids)
	if err != nil {
		return
	}

	// 4. 使用 Pipeline 批量查询每个帖子的投票数据
	voteData, err := redis.GetPostsVoteData(ids)
	if err != nil {
		return
	}

	// 5. 组装数据:填充作者、社区、投票数据
	data = make([]*models.ApiPostDetail, 0, len(posts))
	for idx, post := range posts {
		// 查询作者信息
		user, err := mysql.GetUserByID(post.AuthorID)
		if err != nil {
			continue
		}

		// 查询社区信息
		community, err := mysql.GetCommunityDetailByID(post.CommunityID)
		if err != nil {
			continue
		}

		// 组装最终数据
		postDetail := &models.ApiPostDetail{
			AuthorName:      user.Username,
			CommunityDetail: community,
			Post:            post,
			VoteNum:         voteData[idx], // 填充投票数
		}
		data = append(data, postDetail)
	}

	return
}
```

---

### 7.3 数据流转图

```
用户请求 GET /api/v1/posts?order=score&page=1&size=10
                    ↓
        [Controller] 参数解析
                    ↓
           [Logic] GetPostList2
                    ↓
    ┌───────────────┴───────────────┐
    ↓                               ↓
[Redis]                         [MySQL]
获取排序后的ID列表                   根据ID列表查询详情
ZREVRANGE post:score 0 9          SELECT * FROM post WHERE id IN (...)
返回: [123,456,789,...]            返回: [{Post},{Post},...]
    ↓                               ↓
[Redis Pipeline]                  └→ 组装数据
批量查询投票数                           ↓
ZCOUNT voted:123 1 1             [{Author+Community+Post+VoteNum}, ...]
ZCOUNT voted:456 1 1                   ↓
ZCOUNT voted:789 1 1             [Controller] 返回JSON
返回: [10,5,20,...]                    ↓
    └──────────────┬────────────────┘
                   ↓
            用户看到帖子列表
```

---

## 8. 常见问题

### Q1: 为什么不直接在MySQL中排序?

**A:**

**MySQL排序的问题:**
```sql
-- 每次请求都要重新计算和排序
SELECT * FROM post
ORDER BY (score + (UNIX_TIMESTAMP(create_time) - UNIX_TIMESTAMP(NOW()))/200)
LIMIT 10 OFFSET 0;

-- 问题:
-- 1. 无法使用索引(表达式计算)
-- 2. 全表扫描 + 排序 (性能差)
-- 3. 每次查询都要计算分数 (CPU密集)
```

**Redis排序的优势:**
- ZSet天然有序,查询是O(log N)
- 分数实时更新,查询时不需要计算
- 内存操作,延迟低

---

### Q2: Redis宕机了怎么办?

**A:**

**降级方案:**
```go
func GetPostList2(p *models.ParamPostList) (data []*models.ApiPostDetail, err error) {
	// 1. 尝试从Redis获取
	ids, err := redis.GetPostIDsInOrder(p.Order, p.Page, p.Size)
	if err != nil {
		// Redis失败,降级到MySQL
		zap.L().Warn("Redis failed, fallback to MySQL", zap.Error(err))
		return GetPostListFromMySQL(p) // 从MySQL按时间排序
	}

	// 2. 正常流程...
}
```

**生产环境方案:**
- Redis主从复制
- Redis哨兵模式
- Redis集群

---

### Q3: 如何保证Redis和MySQL的数据一致性?

**A:**

**Bluebell的策略:**
1. **MySQL是主数据源** (Source of Truth)
2. **Redis是缓存** (可以丢失,可以重建)
3. **发帖时同步写入**:
   - 先写MySQL (失败则整体失败)
   - 再写Redis (失败只记录日志,不影响主流程)
4. **定时同步**: 凌晨从MySQL重建Redis数据

---

### Q4: Pipeline能保证事务吗?

**A:**

**TxPipeline:**
```go
pipeline := rdb.TxPipeline() // 带事务
// 所有命令要么全部成功,要么全部失败
```

**Pipeline:**
```go
pipeline := rdb.Pipeline() // 不带事务
// 命令独立执行,部分成功部分失败
```

**投票功能必须用TxPipeline:**
```go
// ✅ 正确: 分数和投票记录必须同时更新
pipeline := rdb.TxPipeline()
pipeline.ZIncrBy(...)  // 更新分数
pipeline.ZAdd(...)     // 更新投票记录
```

---

## 9. 本章总结

### 9.1 核心知识点

| 知识点 | 说明 |
|--------|------|
| **ZSet数据结构** | 有序、去重、高效,完美匹配投票场景 |
| **Key设计规范** | 统一前缀、冒号分隔、语义化命名 |
| **投票实现** | 时间检查 → 旧值查询 → 重复检测 → 分数计算 → Pipeline更新 |
| **Pipeline** | 批量执行、减少RTT、原子性保证 |
| **TxPipeline** | 事务管道,保证多个操作的原子性 |
| **批量查询** | 使用Pipeline避免N+1问题 |
| **帖子排序** | ZREVRANGE按分数降序查询 |

---

### 9.2 Redis命令汇总

| 命令 | 作用 | 示例 |
|------|------|------|
| **ZADD** | 添加成员到ZSet | `ZADD key score member` |
| **ZSCORE** | 查询成员分数 | `ZSCORE key member` |
| **ZINCRBY** | 增加成员分数 | `ZINCRBY key increment member` |
| **ZREM** | 删除成员 | `ZREM key member` |
| **ZCOUNT** | 统计范围内成员数 | `ZCOUNT key min max` |
| **ZREVRANGE** | 降序查询成员 | `ZREVRANGE key start stop` |

---

### 9.3 性能优化总结

| 优化点 | 方案 | 效果 |
|-------|------|------|
| **减少RTT** | 使用Pipeline批量执行 | 10倍提升 |
| **原子性** | 使用TxPipeline | 保证一致性 |
| **避免N+1** | 批量查询投票数据 | 100倍提升 |
| **内存排序** | 使用ZSet存储分数 | O(log N)查询 |

---

## 10. 延伸阅读

- [Redis ZSet内部实现原理](https://redis.io/docs/data-types/sorted-sets/)
- [Redis Pipeline性能测试](https://redis.io/docs/manual/pipelining/)
- [Redis事务机制](https://redis.io/docs/manual/transactions/)
- [分布式系统的CAP定理](https://en.wikipedia.org/wiki/CAP_theorem)

---

## 📖 下一章预告

现在我们已经实现了全站帖子的投票和排序,但还缺少一个重要功能:**按社区筛选帖子**!

用户希望:
- 只看"Go语言"社区的帖子
- 按社区进行内容过滤
- 统一的接口设计

下一章,我们将学习:
- 如何设计统一的帖子查询接口
- 按社区筛选帖子的实现
- 条件查询的最佳实践
- 接口参数的扩展性设计

让内容更有序、更精准! 🎯

---

**📖 下一章: [第18章:按社区筛选帖子实现](./18-按社区筛选帖子实现.md)**
