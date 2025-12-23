# DAO 层错误处理优化方案

## 问题总结

### 1. 日志记录职责混乱
- ❌ DAO 层记录了 zap 日志 (community.go)
- ✅ 应该只由 Logic 层记录日志

### 2. 错误返回不一致
- ❌ 有时返回 `nil, nil` 表示"未找到"
- ❌ 有时返回 `nil, err` 表示系统错误
- ❌ 有时直接返回原始错误
- ✅ 应该统一返回语义

### 3. 业务错误定义分散
- ✅ mysql/user.go 有预定义错误 (ErrorUserExist, ErrorUserNotExist)
- ✅ redis/vote.go 有预定义错误 (ErrVoteTimeExpire)
- ❌ 其他文件缺少业务错误定义

## 优化原则

### DAO 层的唯一职责
```
执行数据库操作 + 包装错误 + 返回结果
```

**不应该做的事:**
- ❌ 记录日志 (zap.L())
- ❌ 业务逻辑判断
- ❌ 错误码转换

**应该做的事:**
- ✅ 包装系统错误 (fmt.Errorf)
- ✅ 返回预定义业务错误 (errors.New)
- ✅ 清晰的返回语义

## 具体优化建议

### 优化 1: 删除 DAO 层的所有日志记录

**Before** (dao/mysql/community.go):
```go
err = db.Get(community, sqlStr, id)
if err != nil {
    if err == sql.ErrNoRows {
        zap.L().Warn("there is no community in db", zap.Int64("community_id", id))
        return nil, nil
    }
    zap.L().Error("query community detail failed", zap.Error(err))
    return nil, err
}
```

**After**:
```go
err = db.Get(community, sqlStr, id)
if err != nil {
    if err == sql.ErrNoRows {
        return nil, ErrCommunityNotFound  // 预定义业务错误
    }
    return nil, fmt.Errorf("query community detail failed: %w", err)
}
```

**配套修改** (dao/mysql/community.go 顶部):
```go
var (
    ErrCommunityNotFound = errors.New("社区不存在")
)
```

### 优化 2: 统一"未找到"的处理方式

**方案 A: 返回预定义错误 (推荐)**
```go
// DAO 层
func GetCommunityDetailByID(id int64) (*models.CommunityDetail, error) {
    community := new(models.CommunityDetail)
    err = db.Get(community, sqlStr, id)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, ErrCommunityNotFound  // 清晰的业务错误
        }
        return nil, fmt.Errorf("query community detail failed: %w", err)
    }
    return community, nil
}

// Logic 层
func GetCommunityDetail(id int64) (*models.CommunityDetail, error) {
    data, err := mysql.GetCommunityDetailByID(id)
    if err != nil {
        if errors.Is(err, mysql.ErrCommunityNotFound) {
            return nil, errorx.ErrNotFound  // 转换为统一的业务错误
        }
        zap.L().Error("mysql.GetCommunityDetailByID failed", ...)
        return nil, errorx.ErrServerBusy
    }
    return data, nil
}
```

**方案 B: 返回 nil (简化版,当前在用)**
```go
// DAO 层
func GetUserByID(uid int64) (*models.User, error) {
    user := &models.User{}
    err := db.Get(user, sqlStr, uid)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, nil  // nil 表示"未找到",不是错误
        }
        return nil, fmt.Errorf("query user by id failed: %w", err)
    }
    return user, nil
}

// Logic 层
func GetUserByID(uid int64) (*models.User, error) {
    user, err := mysql.GetUserByID(uid)
    if err != nil {
        zap.L().Error("mysql.GetUserByID failed", ...)
        return nil, errorx.ErrServerBusy
    }
    if user == nil {
        return nil, errorx.ErrNotFound  // Logic 层判断业务逻辑
    }
    return user, nil
}
```

**对比**:
| 方案 | 优点 | 缺点 |
|------|------|------|
| A: 返回错误 | 语义清晰,类型安全 | 需要定义更多错误常量 |
| B: 返回 nil | 简单,减少错误定义 | 需要检查 nil (容易忘记) |

**推荐**: 方案 B (当前已经在用),但要统一执行

### 优化 3: 统一错误包装格式

**规则**:
```go
// 系统错误: 使用 fmt.Errorf 包装,带上下文信息
return nil, fmt.Errorf("operation failed: %w", err)

// 业务错误: 返回预定义错误
return nil, ErrUserNotExist

// 成功查询但无数据: 返回 nil, nil (仅限查询操作)
if err == sql.ErrNoRows {
    return nil, nil
}
```

### 优化 4: Redis 层预定义更多业务错误

**当前** (redis/vote.go):
```go
var (
    ErrVoteTimeExpire = errors.New("投票时间已过")
    ErrVoteRepeated   = errors.New("不允许重复投票")
)
```

**建议扩展** (redis/errors.go - 新建文件):
```go
package redis

import "errors"

// 业务错误定义
var (
    // 投票相关
    ErrVoteTimeExpire = errors.New("投票时间已过")
    ErrVoteRepeated   = errors.New("不允许重复投票")

    // 帖子相关
    ErrPostNotFound   = errors.New("帖子不存在")

    // 用户相关
    ErrTokenNotFound  = errors.New("Token不存在")
)
```

## 优化清单

### dao/mysql/community.go
- [ ] 删除 zap.L().Warn (2 处)
- [ ] 删除 zap.L().Error (1 处)
- [ ] 删除 zap 导入
- [ ] 统一返回 nil, nil 表示未找到

### dao/mysql/user.go
- [ ] 保持当前的预定义错误 (已经很好)
- [ ] 确保所有 fmt.Errorf 都使用 %w 包装

### dao/mysql/post.go
- [ ] 检查是否有日志记录
- [ ] 统一错误包装格式

### dao/redis/*.go
- [ ] 检查是否有日志记录 (除了初始化)
- [ ] 考虑创建 errors.go 集中定义业务错误

## 实施步骤

### Step 1: 清理 dao/mysql/community.go
删除所有日志记录,统一返回格式

### Step 2: 验证其他 DAO 文件
确保没有日志记录,错误包装统一

### Step 3: 更新 Logic 层
适配 DAO 层的变更

### Step 4: 测试
确保功能正常

## 最佳实践总结

### ✅ DAO 层应该做的

```go
// 1. 包装系统错误
if err != nil {
    return nil, fmt.Errorf("specific operation failed: %w", err)
}

// 2. 返回预定义业务错误
if err == sql.ErrNoRows {
    return nil, nil  // 或者 return nil, ErrNotFound
}

// 3. 直接返回特定错误
if count > 0 {
    return ErrorUserExist
}
```

### ❌ DAO 层不应该做的

```go
// 1. 不要记录日志
zap.L().Error("...")  // ❌

// 2. 不要做业务判断 (简单判断除外)
if user.Age < 18 {    // ❌ 这是业务逻辑
    return errors.New("未成年")
}

// 3. 不要直接返回原始错误
return err  // ❌ 应该包装: fmt.Errorf("...: %w", err)
```

## 预期收益

1. **职责清晰**: DAO 只负责数据访问,Logic 负责日志和业务判断
2. **日志集中**: 所有业务日志都在 Logic 层,便于统一管理
3. **错误可追踪**: 使用 %w 包装后可以用 errors.Is/As 追踪
4. **易于测试**: DAO 层纯粹,更容易编写单元测试
5. **减少重复**: 避免 DAO 和 Logic 层都记录相同错误

## 是否需要立即优化?

**建议**:
- ✅ **立即优化**: dao/mysql/community.go (影响小,收益大)
- ⏸️ **暂缓优化**: 其他文件 (当前没有明显问题,可渐进式优化)

**原因**:
- community.go 的日志记录是明显的职责混乱
- 其他文件的错误处理基本合理
- 避免一次性大规模重构带来的风险
