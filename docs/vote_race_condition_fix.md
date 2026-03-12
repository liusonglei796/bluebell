# Redis 投票并发问题 - Lua 脚本解决方案

## 问题描述

### 原始代码的竞态条件

在 `internal/dao/cache/post/post.go` 的 `VoteForPost` 函数中，使用了经典的"先查-后判断-再写"模式：

```go
// 1. 先查：查询用户之前的投票值
oldValue := c.rdb.ZScore(ctx, redisKey(keyPostVotedZSetPrefix+postID), userID).Val()

// 2. 后判断：如果值相同就返回
if value == oldValue {
    return errorx.ErrVoteRepeated
}

// 3. 再写：更新分数和投票记录
pipeline := c.rdb.TxPipeline()
pipeline.ZIncrBy(ctx, redisKey(keyPostScoreZSet), scoreDiff, postID)
// ...
_, err := pipeline.Exec(ctx)
```

### 竞态场景

假设 User A 手机卡顿，瞬间发出 3 个相同的"赞成票"请求：

1. **请求1**: 查到 `direction=0`（未投票）→ 通过检查 → 执行加分
2. **请求2**: 查到 `direction=0`（未投票）→ 通过检查 → 执行加分  
3. **请求3**: 查到 `direction=0`（未投票）→ 通过检查 → 执行加分

**结果**: 帖子分数被错误地加了 3 次！

### 为什么 TxPipeline 无法解决

`TxPipeline` 只保证打包的多个写命令不会被其他 Redis 客户端插队，但**解决不了"读-判断-写"的竞态条件**。因为：
- 3 个请求在"读"的阶段是各自独立执行的
- 3 个请求都通过了"判断"阶段
- 3 个请求才进入"写"阶段

---

## 解决方案

### 使用 Lua 脚本（业界最推荐）

Redis 执行 Lua 脚本是**原子性**的。我们将整个"查询-判断-更新"逻辑全部写成一段 Lua 脚本发给 Redis 执行：

```lua
-- 1. 检查投票时间限制（一周）
local postTime = redis.call('ZSCORE', timeKey, postID)
if currentTime - postTime > 7 * 24 * 3600 then
    return 1
end

-- 2. 查询用户之前的投票值
local oldValue = redis.call('ZSCORE', votedKey, userID)

-- 3. 如果新旧投票值相同，直接返回（重复投票）
if value == oldValue then
    return 2
end

-- 4. 计算分数变化并执行更新
local scoreDiff = (value - oldValue) * scorePerVote
redis.call('ZINCRBY', scoreKey, scoreDiff, postID)
redis.call('ZINCRBY', communityScoreKey, scoreDiff, postID)

-- 5. 更新用户投票记录
if value == 0 then
    redis.call('ZREM', votedKey, userID)
else
    redis.call('ZADD', votedKey, value, userID)
end

return 0
```

### 为什么能解决竞态问题

- Redis 是单线程处理命令的
- 在执行这段 Lua 脚本时，其他的并发请求只能排队等待
- 整个"查-判断-写"过程原子执行，不存在竞态窗口

### 并发场景模拟

| 请求 | 执行过程 | 结果 |
|-----|---------|------|
| 请求1 | 获取锁 → 查到 oldValue=0 → value=1 → 执行加分 → 写入新值 → 释放锁 | +432 分 |
| 请求2 | 等待锁 → 获取锁 → 查到 oldValue=1 → value=1 → 判断相同 → 直接返回 | 无变化 |
| 请求3 | 等待锁 → 获取锁 → 查到 oldValue=1 → value=1 → 判断相同 → 直接返回 | 无变化 |

---

## 修改的文件

- `internal/dao/cache/post/post.go`

### 主要改动

1. **新增 Lua 脚本变量**（第33-83行）
   - 封装完整的投票逻辑为原子操作
   - 返回值设计：0=成功, 1=过期, 2=重复, 3=失败

2. **修改 VoteForPost 函数**（第190-225行）
   - 原来：多次 Redis 调用（查-判断-写）
   - 现在：单次 Lua 脚本调用

---

## 其他可行的解决方案（供参考）

| 方案 | 优点 | 缺点 |
|-----|------|------|
| **Lua 脚本（推荐）** | 原子性、性能高、减少网络往返 | 需要学习 Lua 语法 |
| 分布式锁 | 实现简单 | 性能开销大、可能死锁 |
| Redis 事务 (MULTI/EXEC) | 支持多个命令 | 不支持条件判断 |
| 乐观锁 (版本号) | 实现简单 | 需要额外的版本字段 |

---

## 测试建议

可以通过以下方式验证修复效果：

1. 使用 wrk 或 ab 工具模拟高并发请求
2. 观察数据库中的实际投票记录次数
3. 对比修复前后的分数变化
