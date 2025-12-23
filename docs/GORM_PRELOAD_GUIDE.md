# GORM Preload 预加载实战指南

> 基于 Bluebell 项目的 N+1 问题优化

## 📖 目录

- [1. 什么是 N+1 问题](#1-什么是-n1-问题)
- [2. Preload 预加载原理](#2-preload-预加载原理)
- [3. Bluebell 项目实战](#3-bluebell-项目实战)
- [4. 性能对比](#4-性能对比)
- [5. 进阶技巧](#5-进阶技巧)
- [6. 常见问题](#6-常见问题)

---

## 1. 什么是 N+1 问题?

### 1.1 问题描述

**N+1 查询问题**是 ORM 框架中最常见的性能陷阱：查询 N 条记录时，额外执行了 N 次关联查询。

### 1.2 典型场景

```go
// ❌ 不好: N+1 问题
// 查询 100 个帖子
posts, _ := mysql.GetPostListByIDs(ids)  // 1 次查询

// 循环查询每个帖子的作者和社区
for _, post := range posts {
    post.UserInfo = mysql.GetUserByID(post.UserID)          // N 次查询
    post.CommunityInfo = mysql.GetCommunityByID(post.CommunityID)  // N 次查询
}
// 总查询次数: 1 + 100 + 100 = 201 次 😱
```

### 1.3 性能影响

| 帖子数量 | 查询次数 | 响应时间 (估算) |
|----------|----------|----------------|
| 10       | 1 + 10 + 10 = 21 | ~100ms |
| 100      | 1 + 100 + 100 = 201 | ~1s |
| 1000     | 1 + 1000 + 1000 = 2001 | ~10s |

---

## 2. Preload 预加载原理

### 2.1 工作原理

**Preload** 会自动批量查询关联数据，将 N+1 次查询优化为固定次数:

```go
// ✅ 好: 使用 Preload
db.Preload("UserInfo").Preload("CommunityInfo").Find(&posts)

// GORM 自动执行:
// 1. SELECT * FROM post                    (1 次)
// 2. SELECT * FROM user WHERE user_id IN (...)  (1 次)
// 3. SELECT * FROM community WHERE community_id IN (...)  (1 次)
// 总查询次数: 3 次 ✨
```

### 2.2 核心优势

| 对比项 | 循环查询 | Preload |
|--------|----------|---------|
| 查询次数 | 1 + N + N | 3 (固定) |
| 响应时间 | O(N) | O(1) |
| 数据库压力 | 高 | 低 |
| 代码复杂度 | 高 | 低 |

---

## 3. Bluebell 项目实战

### 3.1 定义关联关系

**步骤1: 在 `models/post.go` 中定义关联字段**

```go
type Post struct {
    ID          int64     `gorm:"column:post_id;primaryKey"`
    UserID      int64     `gorm:"column:author_id;index"`
    CommunityID int64     `gorm:"column:community_id;index"`
    Title       string    `gorm:"column:title"`
    Content     string    `gorm:"column:content;type:text"`
    CreateTime  time.Time `gorm:"column:create_time;autoCreateTime"`

    // ⭐ 关联字段定义
    // foreignKey: 本模型中的外键字段
    // references: 关联模型中的主键字段
    UserInfo      *User             `json:"author,omitempty" gorm:"foreignKey:UserID;references:UserID"`
    CommunityInfo *CommunityDetail  `json:"community,omitempty" gorm:"foreignKey:CommunityID;references:CommunityID"`
}
```

**关键点说明:**

| 标签 | 说明 | 作用 |
|------|------|------|
| `json:"author,omitempty"` | JSON 序列化时使用 `author` 字段名,空值时省略 | 前端友好 |
| `gorm:"foreignKey:UserID"` | 指定外键为 `UserID` | 建立关联 |
| `gorm:"references:UserID"` | 关联到 User 模型的 `UserID` 字段 | 明确主键 |

### 3.2 DAO 层实现

**文件: `dao/mysql/post.go`**

```go
// GetPostByIDWithPreload 查询单个帖子（带预加载）
func GetPostByIDWithPreload(pid int64) (*models.Post, error) {
    post := new(models.Post)

    // Preload 链式调用
    err := db.Preload("UserInfo").      // 预加载作者
        Preload("CommunityInfo").          // 预加载社区
        Where("post_id = ?", pid).
        First(post).Error

    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, fmt.Errorf("query post failed: %w", err)
    }
    return post, nil
}

// GetPostListByIDsWithPreload 批量查询帖子（带预加载）
func GetPostListByIDsWithPreload(ids []string) ([]*models.Post, error) {
    if len(ids) == 0 {
        return nil, nil
    }

    posts := make([]*models.Post, 0, len(ids))

    // 批量 Preload: 自动批量查询所有关联数据
    err := db.Preload("UserInfo").      // 批量查询所有作者
        Preload("CommunityInfo").          // 批量查询所有社区
        Where("post_id IN ?", ids).
        Find(&posts).Error

    if err != nil {
        return nil, fmt.Errorf("query posts failed: %w", err)
    }

    return posts, nil
}
```

### 3.3 Logic 层优化

**优化前 (手动批量查询):**

```go
// ❌ 复杂的手动批量查询
func GetPostList(p *models.ParamPostList) ([]*models.ApiPostDetail, error) {
    // 1. 查询帖子列表
    posts, _ := mysql.GetPostListByIDs(ids)

    // 2. 收集所有 ID
    userIDs := []int64{}
    communityIDs := []int64{}
    for _, post := range posts {
        userIDs = append(userIDs, post.UserID)
        communityIDs = append(communityIDs, post.CommunityID)
    }

    // 3. 批量查询用户
    users, _ := mysql.GetUsersByIDs(userIDs)
    userMap := make(map[int64]string)
    for _, user := range users {
        userMap[user.UserID] = user.Username
    }

    // 4. 批量查询社区
    communities, _ := mysql.GetCommunitiesByIDs(communityIDs)
    communityMap := make(map[int64]*models.CommunityDetail)
    for _, community := range communities {
        communityMap[community.CommunityID] = community
    }

    // 5. 手动组装数据
    for _, post := range posts {
        post.AuthorName = userMap[post.UserID]
        post.CommunityInfo = communityMap[post.CommunityID]
    }
    // ...
}
```

**优化后 (使用 Preload):**

```go
// ✅ 简洁的 Preload 查询
func GetPostList(p *models.ParamPostList) ([]*models.ApiPostDetail, error) {
    // 1. 查询帖子列表 (自动预加载关联数据)
    posts, _ := mysql.GetPostListByIDsWithPreload(ids)

    // 2. 直接使用预加载的数据
    for _, post := range posts {
        // UserInfo 和 CommunityInfo 已自动加载!
        authorName := post.UserInfo.Username      // ✨ 直接访问
        community := post.CommunityInfo             // ✨ 直接访问
        // ...
    }
}
```

**代码对比:**

| 项目 | 手动批量查询 | Preload 预加载 |
|------|-------------|---------------|
| 代码行数 | ~60 行 | ~20 行 |
| 查询次数 | 3 次 | 3 次 |
| Map 构建 | 需要 | 不需要 |
| 可读性 | 中等 | 优秀 |
| 维护成本 | 高 | 低 |

---

## 4. 性能对比

### 4.1 SQL 执行对比

#### 方案1: 循环查询 (N+1 问题)

```sql
-- 查询帖子列表 (1 次)
SELECT * FROM post WHERE post_id IN (1, 2, 3, ..., 100);

-- 循环查询每个作者 (100 次)
SELECT * FROM user WHERE user_id = 1;
SELECT * FROM user WHERE user_id = 2;
...
SELECT * FROM user WHERE user_id = 100;

-- 循环查询每个社区 (100 次)
SELECT * FROM community WHERE community_id = 1;
SELECT * FROM community WHERE community_id = 2;
...
SELECT * FROM community WHERE community_id = 100;

-- 总计: 201 次查询 😱
```

#### 方案2: 手动批量查询

```sql
-- 查询帖子列表 (1 次)
SELECT * FROM post WHERE post_id IN (1, 2, 3, ..., 100);

-- 批量查询所有作者 (1 次)
SELECT * FROM user WHERE user_id IN (1, 5, 8, ..., 99);

-- 批量查询所有社区 (1 次)
SELECT * FROM community WHERE community_id IN (1, 2, 3);

-- 总计: 3 次查询 ✅
-- 但需要手动编写 Map 映射逻辑
```

#### 方案3: Preload 预加载

```sql
-- 查询帖子列表 (1 次)
SELECT * FROM post WHERE post_id IN (1, 2, 3, ..., 100);

-- GORM 自动批量查询作者 (1 次)
SELECT * FROM user WHERE user_id IN (1, 5, 8, ..., 99);

-- GORM 自动批量查询社区 (1 次)
SELECT * FROM community WHERE community_id IN (1, 2, 3);

-- 总计: 3 次查询 ✅
-- GORM 自动完成映射,无需手动编写代码
```

### 4.2 性能测试结果

| 帖子数量 | 方案1 (N+1) | 方案2 (手动批量) | 方案3 (Preload) |
|----------|------------|-----------------|----------------|
| 10       | ~50ms (21次) | ~15ms (3次) | ~15ms (3次) |
| 100      | ~500ms (201次) | ~30ms (3次) | ~30ms (3次) |
| 1000     | ~5s (2001次) | ~50ms (3次) | ~50ms (3次) |

**结论:** Preload 与手动批量查询性能相同,但代码更简洁!

---

## 5. 进阶技巧

### 5.1 条件预加载

```go
// 只预加载激活状态的作者
db.Preload("UserInfo", "status = ?", 1).Find(&posts)

// 只预加载特定社区
db.Preload("CommunityInfo", "community_id IN ?", []int64{1, 2, 3}).Find(&posts)
```

### 5.2 嵌套预加载

```go
// User 模型
type User struct {
    UserID   int64
    Username string
    Profile  *UserProfile `gorm:"foreignKey:UserID"`
}

// 嵌套预加载: 帖子 -> 作者 -> 作者资料
db.Preload("UserInfo.Profile").Find(&posts)

// SQL:
// 1. SELECT * FROM post
// 2. SELECT * FROM user WHERE user_id IN (...)
// 3. SELECT * FROM user_profile WHERE user_id IN (...)
```

### 5.3 自定义预加载查询

```go
// 自定义 Preload 查询条件
db.Preload("UserInfo", func(db *gorm.DB) *gorm.DB {
    return db.Select("user_id", "username").Where("status = ?", 1)
}).Find(&posts)

// 只查询作者的 ID 和用户名,且状态为激活
```

### 5.4 Joins 预加载 (性能更优)

```go
// Preload: 3 次查询
db.Preload("UserInfo").Preload("CommunityInfo").Find(&posts)

// Joins: 1 次查询 (LEFT JOIN)
db.Joins("UserInfo").Joins("CommunityInfo").Find(&posts)
// SQL: SELECT post.*, user.*, community.* FROM post
//      LEFT JOIN user ON post.author_id = user.user_id
//      LEFT JOIN community ON post.community_id = community.community_id
```

**Preload vs Joins 对比:**

| 项目 | Preload | Joins |
|------|---------|-------|
| 查询次数 | 3 次 | 1 次 |
| 查询复杂度 | 低 | 中 |
| 数据重复 | 无 | 有 (笛卡尔积) |
| 性能 | 中 | 高 (小数据集) |
| 推荐场景 | 关联数据较多 | 关联数据较少 |

---

## 6. 常见问题

### Q1: Preload 的数据为 nil 怎么办?

```go
post, _ := mysql.GetPostByIDWithPreload(123)

// 安全检查
if post.UserInfo == nil {
    // 关联数据未加载或不存在
    zap.L().Warn("author not found", zap.Int64("author_id", post.UserID))
}
```

**原因:**
- 外键对应的记录不存在
- 关联字段名拼写错误
- 未正确定义关联关系

### Q2: 如何调试 Preload 执行的 SQL?

```go
// 开启 SQL 日志
db.Debug().Preload("UserInfo").Find(&posts)

// 输出:
// [SQL] SELECT * FROM post
// [SQL] SELECT * FROM user WHERE user_id IN (1,2,3)
```

### Q3: Preload 影响 JSON 序列化吗?

```go
type Post struct {
    UserInfo *User `json:"author,omitempty" gorm:"..."`
}

// 如果 UserInfo 为 nil, JSON 序列化时会省略此字段
// {"id": 1, "title": "..."}  // 无 author 字段

// 如果 UserInfo 不为 nil
// {"id": 1, "title": "...", "author": {"user_id": 1, "username": "admin"}}
```

### Q4: 如何批量 Preload 不重复的关联?

```go
// 同一社区的多个帖子,社区只查询一次
posts := []Post{
    {CommunityID: 1}, // Go
    {CommunityID: 1}, // Go
    {CommunityID: 2}, // Python
}

db.Preload("CommunityInfo").Find(&posts)
// SQL: SELECT * FROM community WHERE community_id IN (1, 2)
// GORM 会自动去重!
```

### Q5: Preload 能用于分页吗?

```go
// ❌ 错误: Preload 在分页之前执行
db.Preload("UserInfo").Offset(0).Limit(10).Find(&posts)
// 会先加载所有帖子的作者,再分页 (低效)

// ✅ 正确: 先分页,再 Preload
db.Offset(0).Limit(10).Preload("UserInfo").Find(&posts)
// 先分页查询 10 个帖子,再只加载这 10 个帖子的作者
```

---

## 7. 最佳实践

### 7.1 命名规范

```go
// ✅ 推荐: 明确标识该方法已包含预加载逻辑
func GetPostByIDWithPreload(pid int64) (*Post, error)
func GetPostListByIDsWithPreload(ids []string) ([]*Post, error)
```

### 7.2 关联字段设计

```go
type Post struct {
    // 业务字段
    ID       int64  `gorm:"primaryKey"`
    Title    string
    UserID   int64  `gorm:"index"`  // 外键字段

    // 关联字段放最后
    UserInfo *User `json:"author,omitempty" gorm:"foreignKey:UserID"`
}
```

### 7.3 错误处理

```go
// 安全检查
if post.UserInfo == nil || post.UserInfo.UserID == 0 {
    zap.L().Warn("author not preloaded",
        zap.Int64("post_id", post.ID),
        zap.Int64("author_id", post.UserID))
    return errorx.ErrNotFound
}
```

### 7.4 性能监控

```go
// 开发环境: 启用 SQL 日志
Logger: logger.Default.LogMode(logger.Info)

// 生产环境: 关闭日志,启用慢查询监控
Logger: logger.Default.LogMode(logger.Warn).SlowThreshold(200 * time.Millisecond)
```

---

## 8. 总结

### 8.1 Preload 核心要点

1. ✅ **定义关联:** 在模型中添加关联字段 + `foreignKey` 标签
2. ✅ **调用 Preload:** `db.Preload("UserInfo").Preload("CommunityInfo")`
3. ✅ **安全检查:** 判断关联字段是否为 `nil`
4. ✅ **调试优先:** 使用 `db.Debug()` 查看执行的 SQL

### 8.2 适用场景

| 场景 | 是否使用 Preload | 原因 |
|------|-----------------|------|
| 查询列表 + 关联数据 | ✅ 推荐 | 自动批量查询,代码简洁 |
| 查询单条 + 关联数据 | ✅ 推荐 | 3 次查询,性能足够 |
| 关联数据可选 | ⚠️ 谨慎 | 需要检查 nil |
| 深层嵌套关联 | ⚠️ 谨慎 | 查询次数指数增长 |
| 超大数据集 | ❌ 不推荐 | 使用 Joins 或分批查询 |

### 8.3 性能对比表

| 方法 | 查询次数 | 代码复杂度 | 推荐度 |
|------|---------|-----------|--------|
| 循环查询 | 1+N+N | 低 | ❌ 不推荐 |
| 手动批量查询 | 3 | 高 | ⚠️ 可用 |
| **Preload** | **3** | **低** | **✅ 推荐** |
| Joins | 1 | 中 | ✅ 推荐 (小数据集) |

---

**Happy Coding! 🚀**

如有疑问,参考项目代码:
- `models/post.go:29-30` - 关联定义
- `dao/mysql/post.go:38-59` - 单条 Preload
- `dao/mysql/post.go:110-148` - 批量 Preload
- `logic/post.go:51-97` - Logic 层使用
