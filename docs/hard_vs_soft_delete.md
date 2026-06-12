# 硬删除 vs 软删除：完全指南

## 📚 目录
1. [核心概念](#核心概念)
2. [技术对比](#技术对比)
3. [判断决策树](#判断决策树)
4. [代码示例](#代码示例)
5. [实际案例](#实际案例)
6. [最佳实践](#最佳实践)

---

## 核心概念

### 🗑️ 硬删除（Hard Delete）

**定义**：从数据库中**物理删除**记录，彻底移除数据。

```sql
-- 硬删除：记录永久消失
DELETE FROM vote WHERE id = 1;
-- 之后无法查询恢复，除非有备份
```

**特点**：
- 记录真正从数据库消失
- 无法追踪删除历史
- 磁盘空间立即释放
- 查询速度最快（没有垃圾数据）

### 🔒 软删除（Soft Delete）

**定义**：添加 `deleted_at` 字段，**逻辑标记**为删除，但记录保留在数据库中。

```sql
-- 软删除：只标记删除时间
UPDATE vote SET deleted_at = NOW() WHERE id = 1;
-- 记录仍在数据库，但被标记为已删除
```

**特点**：
- 记录保留在数据库中
- 可追踪什么时候删除的
- 可随时恢复
- 查询需要额外条件 (`WHERE deleted_at IS NULL`)

---

## 技术对比

| 对比维度 | 硬删除 | 软删除 |
|--------|------|------|
| **数据保留** | ❌ 物理删除 | ✅ 逻辑标记 |
| **恢复能力** | ❌ 无法恢复* | ✅ 可完全恢复 |
| **查询性能** | ✅ 最快 | ⚠️ 需要额外条件 |
| **磁盘空间** | ✅ 立即释放 | ❌ 持续占用 |
| **审计追踪** | ❌ 无历史 | ✅ 完整历史 |
| **数据库大小** | ✅ 较小 | ❌ 累积增长 |
| **实现复杂度** | ✅ 简单 | ⚠️ 需要修改所有查询 |
| **业务逻辑** | ✅ 简单 | ⚠️ 所有查询都要过滤 |

*除非有备份或 binlog

---

## 判断决策树

```
是否需要数据恢复或审计？
│
├─ 是 → 用户可能误删（用户内容、订单、账户）
│       → 需要查看删除历史（财务、法规）
│       → 【使用软删除】
│
└─ 否 → 业务逻辑本身就是"删除"（投票取消、临时记录）
        → 不需要恢复的业务数据（会话、缓存）
        → 【使用硬删除】
```

### 快速判断表

| 数据类型 | 推荐方案 | 原因 |
|---------|--------|------|
| **用户账户** | 软删除 | 可能误删，需要恢复期 |
| **订单记录** | 软删除 | 财务审计必需，法规要求 |
| **用户评论** | 软删除 | 用户可能想编辑/恢复 |
| **投票记录** | 硬删除 | 取消投票就是删除，无需历史 |
| **收藏夹** | 硬删除 | 取消收藏 = 删除关系 |
| **会话/Token** | 硬删除 | 过期自动删，不需要历史 |
| **临时上传文件** | 硬删除 | 本来就是临时的 |
| **用户消息** | 软删除 | 用户可能误删，想恢复 |
| **后台日志** | 硬删除 | 定期清理，不需要保存 |

---

## 代码示例

### 场景1：投票系统（论坛风格） → 硬删除

```go
// 数据库表结构
type Vote struct {
    ID        uint   `gorm:"primaryKey"`
    UserID    int64
    PostID    int64
    Direction int8   // 1: 赞, -1: 踩
    CreatedAt time.Time
}

// 取消投票 - 硬删除
func (repo *VoteRepository) DeleteVote(ctx context.Context, userID, postID int64) error {
    return repo.db.Where("user_id = ? AND post_id = ?", userID, postID).
        Delete(&Vote{}).Error
}

// 查询当前投票状态 - 无需额外过滤
func (repo *VoteRepository) GetUserVote(ctx context.Context, userID, postID int64) (*Vote, error) {
    var vote Vote
    err := repo.db.Where("user_id = ? AND post_id = ?", userID, postID).
        First(&vote).Error
    // 记录不存在 = 未投票，无垃圾数据
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil
    }
    return &vote, err
}
```

**理由**：
- 用户取消投票 = 删除这条投票记录
- 无需投票历史
- 查询简单快速

---

### 场景2：用户账户 → 软删除

```go
// 数据库表结构
type User struct {
    ID        uint      `gorm:"primaryKey"`
    Email     string
    Username  string
    DeletedAt *time.Time `gorm:"index"` // 软删除标记
}

// 删除用户 - 软删除
func (repo *UserRepository) DeleteUser(ctx context.Context, userID uint) error {
    return repo.db.Model(&User{}).
        Where("id = ?", userID).
        Update("deleted_at", time.Now()).Error
}

// 查询活跃用户 - 必须过滤已删除
func (repo *UserRepository) GetUser(ctx context.Context, userID uint) (*User, error) {
    var user User
    err := repo.db.Where("id = ? AND deleted_at IS NULL", userID).
        First(&user).Error
    return &user, err
}

// GORM 的软删除支持（自动处理 deleted_at）
func (repo *UserRepository) GetActiveUsers(ctx context.Context) ([]User, error) {
    var users []User
    // GORM 自动为 deleted_at IS NULL 的结构体添加过滤
    err := repo.db.Where("deleted_at IS NULL").Find(&users).Error
    return users, err
}

// 真正永久删除（谨慎使用）
func (repo *UserRepository) PermanentlyDeleteUser(ctx context.Context, userID uint) error {
    return repo.db.Unscoped(). // 忽略软删除
        Where("id = ?", userID).
        Delete(&User{}).Error
}
```

**理由**：
- 用户可能误删，需要恢复期
- 财务、法律需要删除记录
- 需要追踪谁在什么时候删除了账户

---

### 场景3：用户评论 → 软删除

```go
// 数据库表结构
type Comment struct {
    ID        uint       `gorm:"primaryKey"`
    Content   string
    UserID    uint
    PostID    uint
    DeletedAt *time.Time `gorm:"index"`
}

// 删除评论 - 软删除
func (repo *CommentRepository) DeleteComment(ctx context.Context, commentID uint) error {
    return repo.db.Where("id = ?", commentID).
        Update("deleted_at", time.Now()).Error
}

// 查询帖子下的评论 - 自动过滤已删除
func (repo *CommentRepository) GetPostComments(ctx context.Context, postID uint) ([]Comment, error) {
    var comments []Comment
    err := repo.db.Where("post_id = ?", postID).Find(&comments).Error
    return comments, err
}

// 用户可以编辑评论（包括已删除的在30分钟内恢复）
func (repo *CommentRepository) RestoreComment(ctx context.Context, commentID uint) error {
    return repo.db.Model(&Comment{}).
        Where("id = ? AND deleted_at > DATE_SUB(NOW(), INTERVAL 30 MINUTE)", commentID).
        Update("deleted_at", nil).Error
}
```

**理由**：
- 用户可能误删
- 允许在一定时间内恢复
- 需要审计谁删除了什么

---

### 场景4：收藏关系 → 硬删除

```go
// 数据库表结构
type Bookmark struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    int64
    PostID    int64
    CreatedAt time.Time
}

// 取消收藏 - 硬删除
func (repo *BookmarkRepository) DeleteBookmark(ctx context.Context, userID, postID int64) error {
    return repo.db.Where("user_id = ? AND post_id = ?", userID, postID).
        Delete(&Bookmark{}).Error
}

// 查询用户的收藏 - 无需过滤垃圾
func (repo *BookmarkRepository) GetUserBookmarks(ctx context.Context, userID int64) ([]Bookmark, error) {
    var bookmarks []Bookmark
    err := repo.db.Where("user_id = ?", userID).Find(&bookmarks).Error
    return bookmarks, err
}
```

**理由**：
- 取消收藏 = 删除这条关系
- 不需要历史记录
- 业务逻辑简单

---

## 实际案例

### 案例1：电商平台（订单） → 必须软删除

```
用户下单 → 支付成功 → 删除订单？

❌ 硬删除会导致：
   - 财务记录丢失
   - 无法对账
   - 违反会计法规
   - 无法追踪退款

✅ 软删除：
   - 保留完整交易历史
   - 可追踪退款、售后
   - 合规审计
```

### 案例2：社交媒体（赞/踩） → 硬删除

```
用户赞了帖子 → 取消赞？

✅ 硬删除是对的：
   - 取消赞 = 删除投票记录
   - 无需投票历史
   - 查询快速（没垃圾数据）
   - 业务逻辑清晰

❌ 软删除会导致：
   - 垃圾数据堆积
   - 查询变复杂
   - 磁盘占用浪费
```

### 案例3：CMS 系统（文章） → 软删除

```
编辑发布文章 → 删除文章？

❌ 硬删除问题：
   - 用户可能误删
   - 无法恢复
   - 无法追踪删除历史

✅ 软删除方案：
   - 30天回收站，可恢复
   - 记录谁删除了什么
   - 用户可以"撤销"删除
```

---

## 最佳实践

### ✅ 软删除使用规则

1. **表结构**必须添加 `deleted_at` 字段

```sql
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;
CREATE INDEX idx_deleted_at ON users(deleted_at);
```

2. **所有查询**都要添加过滤条件

```go
// ❌ 错误：查询了已删除的用户
db.Find(&users)

// ✅ 正确：只查询活跃用户
db.Where("deleted_at IS NULL").Find(&users)

// ✅ 或使用 GORM scopes
db.Scopes(notDeleted).Find(&users)
```

3. **定期清理**已删除数据（30-90天后）

```sql
-- 30天后彻底删除用户
DELETE FROM users WHERE deleted_at < DATE_SUB(NOW(), INTERVAL 30 DAY);
```

4. **为敏感操作添加保护**

```go
// 删除前验证：是否真的要删除？
func (svc *UserService) DeleteUser(ctx context.Context, userID uint, reason string) error {
    // 记录删除原因到审计日志
    svc.auditLog.Record("user_deletion", userID, reason)
    
    // 软删除
    return svc.repo.DeleteUser(ctx, userID)
}
```

### ✅ 硬删除使用规则

1. **明确业务定义**：删除意味着什么

```go
// 清晰的方法名，说明是硬删除
func (repo *VoteRepository) DeleteVote(...) // 取消投票 = 删除记录
func (repo *BookmarkRepository) RemoveBookmark(...) // 取消收藏 = 删除关系
```

2. **关键操作前做备份**

```go
// 删除前备份数据（用于分析）
func (svc *TempFileService) CleanupExpiredFiles(ctx context.Context) error {
    files, _ := svc.repo.GetExpiredFiles(ctx)
    svc.archiveLog.Archive(files) // 备份到日志
    return svc.repo.DeleteExpiredFiles(ctx)
}
```

3. **考虑增加软删除字段**用于临时保护

```go
// 即使是硬删除的场景，也可以加个 scheduled_deletion 保护
type TemporaryFile struct {
    ID                  uint
    ScheduledDeletion   *time.Time // NULL=保留, 有值=N天后删除
}
```

---

## 实施建议

### 📋 新建表时的检查清单

```
[] 1. 这是用户产生的数据还是系统生成的？
      用户数据 → 考虑软删除
      系统生成 → 考虑硬删除

[] 2. 是否有法规/合规要求保留记录？
      是 → 必须软删除
      否 → 继续评估

[] 3. 用户是否可能误操作？
      是 → 软删除（提供恢复）
      否 → 可以硬删除

[] 4. 是否需要审计追踪？
      是 → 软删除
      否 → 继续评估

[] 5. 是否影响业务统计？
      是 → 软删除
      否 → 硬删除可行

[] 6. 数据量会很大吗？
      是 → 硬删除更高效
      否 → 两者都可以
```

### 🔄 迁移策略

**从硬删除 → 软删除**（较安全）：

```sql
-- 1. 添加 deleted_at 字段
ALTER TABLE votes ADD COLUMN deleted_at TIMESTAMP NULL DEFAULT NULL;

-- 2. 修改业务逻辑：DELETE 改成 UPDATE
-- 3. 修改所有查询：添加 WHERE deleted_at IS NULL
-- 4. 测试完成后部署
```

**从软删除 → 硬删除**（较危险）：

```sql
-- 1. 导出备份（备份整个表）
-- 2. 创建新表（不带 deleted_at）
-- 3. 迁移活跃数据：INSERT INTO new_table SELECT * FROM old_table WHERE deleted_at IS NULL
-- 4. 修改应用指向新表
-- 5. 观察一段时间，确认无误
-- 6. 删除旧表备份
```

---

## 总结

| 记住这一点 | 你就懂了 |
|---------|--------|
| **硬删除** = 记录**永久消失** | 投票、收藏、临时数据 |
| **软删除** = 记录**标记删除** | 用户、订单、评论 |
| **关键问题** | "用户需要恢复吗？" |
| **如果不确定** | 用软删除（更保险） |
| **性能关键时** | 用硬删除（更快） |

---

## 参考资源

- [GORM 软删除官方文档](https://gorm.io/docs/delete.html#Soft%20Delete)
- [MySQL 性能优化：清理大表](https://dev.mysql.com/doc/)
- [数据库设计规范](https://www.postgresql.org/docs/)
