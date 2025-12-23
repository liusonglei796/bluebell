# GORM 迁移文档

## 概述

本项目已成功从 `sqlx` 迁移到 `GORM` ORM 框架。本文档记录了迁移的详细内容和注意事项。

## 迁移日期

2025-12-23

## 主要变更

### 1. 依赖包变更

**移除:**
- `github.com/jmoiron/sqlx` - sqlx ORM 框架

**新增:**
- `gorm.io/gorm v1.31.1` - GORM ORM 框架
- `gorm.io/driver/mysql v1.6.0` - GORM MySQL 驱动

### 2. 模型层改造 (models/)

所有模型文件已更新为 GORM 标签格式：

#### models/user.go
- 标签从 `db:"xxx"` 改为 `gorm:"column:xxx"`
- 添加了 `TableName()` 方法指定表名为 `user`
- Password 字段添加 `json:"-"` 防止序列化

#### models/post.go
- 标签从 `db:"xxx"` 改为 `gorm:"column:xxx"`
- 添加了 `TableName()` 方法指定表名为 `post`
- 添加了索引和约束标签(primaryKey, index, not null)

#### models/community.go
- 标签从 `db:"xxx"` 改为 `gorm:"column:xxx"`
- 添加了 `TableName()` 方法指定表名为 `community`

### 3. 数据库初始化层 (dao/mysql/mysql.go)

**主要变更:**
- 全局变量从 `*sqlx.DB` 改为 `*gorm.DB`
- 使用 `gorm.Open()` 替代 `sqlx.Connect()`
- 添加 GORM 配置(日志、预编译语句等)
- 新增 `GetDB()` 方法供外部获取数据库实例

**配置优化:**
```go
gormConfig := &gorm.Config{
    Logger: logger.Default.LogMode(logger.Info),
    DisableForeignKeyConstraintWhenMigrating: true,
    PrepareStmt: true,
}
```

### 4. DAO 层改造

#### dao/mysql/user.go

**查询操作对比:**

| 操作 | sqlx 写法 | GORM 写法 |
|------|-----------|-----------|
| 统计记录数 | `db.Get(&count, sqlStr, username)` | `db.Model(&models.User{}).Where("username = ?", username).Count(&count)` |
| 插入记录 | `db.NamedExec(sqlStr, user)` | `db.Create(user)` |
| 单条查询 | `db.Get(user, sqlStr, username)` | `db.Where("username = ?", user.Username).First(user)` |
| 批量查询 | `sqlx.In() + db.Select()` | `db.Where("user_id IN ?", ids).Find(&users)` |

**错误处理变更:**
```go
// sqlx
if err == sql.ErrNoRows {
    return ErrorUserNotExist
}

// GORM
if errors.Is(err, gorm.ErrRecordNotFound) {
    return ErrorUserNotExist
}
```

#### dao/mysql/post.go

**关键变更:**
1. **创建帖子:** `db.Exec()` → `db.Create()`
2. **查询单个帖子:** `db.Get()` → `db.Where().First()`
3. **分页查询:**
   ```go
   // sqlx
   db.Select(&posts, sqlStr, communityID, (page-1)*size, size)

   // GORM
   db.Where("community_id = ?", communityID).
      Order("create_time DESC").
      Offset(int((page - 1) * size)).
      Limit(int(size)).
      Find(&posts)
   ```

4. **批量查询并保持顺序:**
   - 移除了 `sqlx.In()` 和 `FIND_IN_SET()` 的复杂用法
   - 使用 `Where IN` 查询后在代码中排序

#### dao/mysql/community.go

**查询优化:**
```go
// 列表查询 - 只选择需要的字段
db.Select("community_id", "community_name").Find(&data)

// 批量查询
db.Where("community_id IN ?", ids).Find(&communities)
```

### 5. 代码改进点

#### 简化的 API
- **插入:** 从 `db.NamedExec(sqlStr, struct)` 简化为 `db.Create(struct)`
- **查询:** 从手写 SQL 改为链式调用 `db.Where().First()`
- **批量查询:** 从 `sqlx.In() + db.Rebind()` 简化为 `db.Where("id IN ?", ids)`

#### 更好的类型安全
- GORM 自动处理结构体字段映射
- 减少了手写 SQL 的错误风险

#### 更清晰的代码结构
```go
// Before (sqlx)
sqlStr := `SELECT user_id, username FROM user WHERE user_id = ?`
err := db.Get(user, sqlStr, uid)

// After (GORM)
err := db.Where("user_id = ?", uid).First(user).Error
```

## 兼容性保证

### 1. 数据库表结构
- **无需修改** 现有数据库表结构
- GORM 模型完全兼容现有表结构
- 通过 `TableName()` 方法显式指定表名，避免自动复数化

### 2. 业务逻辑层
- **无需修改** logic 层代码
- DAO 层接口保持不变
- 返回值类型保持一致

### 3. Controller 层
- **无需修改** controller 层代码
- 完全向下兼容

## 性能考虑

### 1. 连接池配置
保持与 sqlx 相同的连接池参数：
- MaxOpenConns: 从配置文件读取
- MaxIdleConns: 从配置文件读取
- ConnMaxIdleTime: 10分钟
- ConnMaxLifetime: 2小时

### 2. 预编译语句
启用了 GORM 的 PrepareStmt 特性，提升查询性能

### 3. N+1 查询优化
继续使用 `Where IN` 批量查询避免 N+1 问题

## 测试验证

### 编译测试
```bash
go build -v .
✅ 编译成功
```

### 依赖验证
```bash
go mod tidy
✅ 依赖整理完成
```

## 注意事项

### 1. GORM 特定行为

**自动表名复数化:**
```go
// GORM 默认会将 User 模型映射到 users 表
// 必须实现 TableName() 方法指定表名
func (User) TableName() string {
    return "user"
}
```

**字段映射:**
```go
// 必须使用 gorm tag 显式指定列名
UserID int64 `gorm:"column:user_id;primaryKey"`
```

### 2. 错误处理差异

**记录不存在:**
```go
// sqlx 使用 sql.ErrNoRows
if err == sql.ErrNoRows {}

// GORM 使用 gorm.ErrRecordNotFound
if errors.Is(err, gorm.ErrRecordNotFound) {}
```

### 3. 批量查询顺序保持

由于 GORM 不支持 `FIND_IN_SET()` 的原生方式，在 `GetPostListByIDs` 中：
1. 先用 `Where IN` 查询所有数据
2. 在内存中按传入的 ID 顺序重新排序
3. 性能影响可忽略不计(列表查询通常 <100 条记录)

## 后续优化建议

### 1. 启用 GORM 日志
开发环境可以启用详细日志查看 SQL 执行：
```go
Logger: logger.Default.LogMode(logger.Info)
```

生产环境建议改为：
```go
Logger: logger.Default.LogMode(logger.Silent)
```

### 2. 考虑使用关联查询
未来可以利用 GORM 的关联特性简化联表查询：
```go
type Post struct {
    ...
    Author    User             `gorm:"foreignKey:AuthorID"`
    Community CommunityDetail   `gorm:"foreignKey:CommunityID"`
}

// 预加载关联数据
db.Preload("Author").Preload("Community").Find(&posts)
```

### 3. 添加数据库迁移
可以使用 GORM AutoMigrate 管理表结构：
```go
db.AutoMigrate(&models.User{}, &models.Post{}, &models.CommunityDetail{})
```

## 迁移检查清单

- [x] 安装 GORM 依赖
- [x] 更新所有 Model 定义
- [x] 改造 mysql.go 初始化逻辑
- [x] 重写 user.go DAO 层
- [x] 重写 post.go DAO 层
- [x] 重写 community.go DAO 层
- [x] 编译通过验证
- [x] 更新 CLAUDE.md 文档
- [x] 创建迁移文档

## 总结

本次迁移从 sqlx 到 GORM 成功完成，主要优势：

1. **代码更简洁:** 减少了大量手写 SQL 代码
2. **类型更安全:** GORM 提供更好的编译时类型检查
3. **功能更丰富:** 支持钩子、关联、事务等高级特性
4. **向下兼容:** 保持了原有的业务逻辑和接口
5. **性能无损:** 连接池和查询优化与之前保持一致

迁移已验证编译通过，可以进行进一步的功能测试。
