# DAO 层错误处理策略指南

## 核心原则

**不是所有 DAO 错误都应该返回 ErrServerBusy!**

需要区分 3 种错误类型:

### 1️⃣ 业务错误 (Business Error)

**定义**: 用户操作违反业务规则,应该返回明确的错误码

**示例**:
- 用户不存在 → ErrUserNotExist
- 密码错误 → ErrInvalidPassword
- 投票时间过期 → ErrVoteTimeExpire (需要新增!)
- 重复投票 → ErrVoteRepeated (需要新增!)
- 社区不存在 → ErrNotFound

**处理方式**:
```go
// DAO 层定义预定义错误
var (
    ErrorUserNotExist = errors.New("用户不存在")
)

// Logic 层识别并转换
if errors.Is(err, mysql.ErrorUserNotExist) {
    return errorx.ErrUserNotExist  // 转换为 errorx 错误
}
```

### 2️⃣ 系统错误 (System Error)

**定义**: 基础设施故障,用户无法解决,统一返回 ErrServerBusy

**示例**:
- 数据库连接失败
- SQL 执行错误 (非 ErrNoRows)
- Redis 连接超时
- 网络错误

**处理方式**:
```go
// Logic 层统一处理
if err != nil {
    zap.L().Error("dao operation failed", zap.Error(err))
    return errorx.ErrServerBusy  // ✅ 正确
}
```

### 3️⃣ 空数据 (No Data)

**定义**: 查询无结果,不一定是错误

**处理方式**:
```go
// DAO 层返回 nil, nil
if err == sql.ErrNoRows {
    return nil, nil
}

// Logic 层判断业务逻辑
if data == nil {
    return errorx.ErrNotFound  // 业务错误
}
```

---

## 当前代码的问题

### ❌ 问题 1: 投票错误未区分

**当前代码** (logic/vote.go:46):
```go
return redis.VoteForPost(...)  // 直接返回,未处理
```

**问题**:
- `redis.ErrVoteTimeExpire` 被 `HandleError` 当成系统错误
- 用户看到 "服务繁忙" 而不是 "投票时间已过"

**修复方案**:
```go
err := redis.VoteForPost(
    strconv.FormatInt(userID, 10),
    strconv.FormatInt(p.PostID, 10),
    strconv.FormatInt(post.CommunityID, 10),
    float64(p.Direction),
)
if err != nil {
    // 区分业务错误和系统错误
    if errors.Is(err, redis.ErrVoteTimeExpire) {
        return errorx.ErrVoteTimeExpire  // 需要在 errorx 中新增
    }
    if errors.Is(err, redis.ErrVoteRepeated) {
        return errorx.ErrVoteRepeated  // 需要在 errorx 中新增
    }
    // 其他 Redis 错误才是系统错误
    zap.L().Error("redis.VoteForPost failed", zap.Error(err))
    return errorx.ErrServerBusy
}
return nil
```

**需要新增 errorx 错误**:
```go
// pkg/errorx/errorx.go
const (
    // ... 现有错误码
    CodeVoteTimeExpire = 1009
    CodeVoteRepeated   = 1010
)

var (
    // ... 现有错误
    ErrVoteTimeExpire = New(CodeVoteTimeExpire, "投票时间已过")
    ErrVoteRepeated   = New(CodeVoteRepeated, "不允许重复投票")
)
```

---

## 正确的错误处理流程

### 流程图

```
DAO 层返回错误
    ↓
Logic 层判断
    ↓
┌─────────────────────────────────────┐
│ 1. 是预定义业务错误?                │
│    (errors.Is 检查)                  │
│    ├─ 是 → 转换为 errorx.ErrXXX    │
│    └─ 否 → 继续                     │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│ 2. 是空数据 (nil)?                  │
│    ├─ 是 → 返回 errorx.ErrNotFound │
│    └─ 否 → 继续                     │
└─────────────────────────────────────┘
    ↓
┌─────────────────────────────────────┐
│ 3. 系统错误                         │
│    ├─ 记录日志                      │
│    └─ 返回 errorx.ErrServerBusy    │
└─────────────────────────────────────┘
```

### 代码模板

```go
// Logic 层标准错误处理模板
func SomeLogic(id int64) (*Data, error) {
    // 1. 调用 DAO 层
    data, err := mysql.GetData(id)

    // 2. 错误处理
    if err != nil {
        // 2.1 检查预定义业务错误
        if errors.Is(err, mysql.ErrDataNotFound) {
            return nil, errorx.ErrNotFound
        }

        // 2.2 系统错误: 记录日志 + 返回 ErrServerBusy
        zap.L().Error("mysql.GetData failed",
            zap.Int64("id", id),
            zap.Error(err))
        return nil, errorx.ErrServerBusy
    }

    // 3. 检查空数据 (如果 DAO 层返回 nil, nil)
    if data == nil {
        return nil, errorx.ErrNotFound
    }

    return data, nil
}
```

---

## 优化建议

### 立即优化 (高优先级)

**修复 logic/vote.go 的错误处理**:
1. 新增 `errorx.ErrVoteTimeExpire` 和 `errorx.ErrVoteRepeated`
2. 修改 `VoteForPost` 函数,区分业务错误和系统错误
3. 测试投票功能是否正常返回错误

### 长期优化 (中优先级)

**统一所有 Logic 层的错误处理**:
1. 审计所有 Logic 层函数
2. 确保业务错误都有对应的 errorx 错误
3. 确保系统错误都记录日志

---

## 回答你的问题

> 只要是DAO层的系统错误不管是什么错误都返回errorx.ErrServerBusy吗?

**答案**:

✅ **纯系统错误 (如数据库连接失败) → ErrServerBusy**
```go
return nil, errorx.ErrServerBusy
```

❌ **业务错误 (如用户不存在) → 对应的业务错误码**
```go
if errors.Is(err, mysql.ErrorUserNotExist) {
    return errorx.ErrUserNotExist  // 不是 ErrServerBusy!
}
```

⚠️ **空数据 (查询无结果) → 根据业务逻辑决定**
```go
if data == nil {
    return errorx.ErrNotFound  // 或者返回空数组
}
```

---

## 当前项目的问题汇总

| 文件 | 问题 | 影响 | 优先级 |
|------|------|------|--------|
| logic/vote.go | 未处理 redis 业务错误 | 投票错误显示 "服务繁忙" | **高** |
| pkg/errorx/errorx.go | 缺少投票相关错误码 | 无法返回正确错误 | **高** |
| logic/*.go | 部分函数未区分业务/系统错误 | 错误提示不准确 | 中 |

---

## 是否立即修复?

建议**立即修复 logic/vote.go**,因为:
1. 投票是核心功能
2. 错误提示不准确影响用户体验
3. 修复成本低 (新增 2 个错误码 + 修改 1 个函数)
