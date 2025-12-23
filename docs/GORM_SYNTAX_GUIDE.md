# GORM 语法糖完全教学指南

> 基于 Bluebell 项目实战的 GORM 使用教程

## 目录

- [1. 基础概念](#1-基础概念)
- [2. 模型定义](#2-模型定义)
- [3. 数据库连接](#3-数据库连接)
- [4. 查询操作](#4-查询操作)
- [5. 插入操作](#5-插入操作)
- [6. 更新操作](#6-更新操作)
- [7. 删除操作](#7-删除操作)
- [8. 高级查询](#8-高级查询)
- [9. 错误处理](#9-错误处理)
- [10. 性能优化](#10-性能优化)

---

## 1. 基础概念

### 1.1 什么是 GORM?

GORM 是 Go 语言的 ORM (Object-Relational Mapping) 库,它将数据库表映射为 Go 结构体,让你用面向对象的方式操作数据库。

**核心优势:**
- ✅ 类型安全,编译时检查
- ✅ 链式 API,代码简洁优雅
- ✅ 自动迁移,自动处理表结构
- ✅ 关联加载,解决 N+1 问题
- ✅ 钩子函数,在 CRUD 前后执行逻辑

### 1.2 GORM vs 原生 SQL vs sqlx

```go
// 原生 SQL (database/sql)
sqlStr := "SELECT * FROM user WHERE user_id = ?"
row := db.QueryRow(sqlStr, 123)
var user User
err := row.Scan(&user.UserID, &user.Username, &user.Password)

// sqlx (稍微简化)
sqlStr := "SELECT * FROM user WHERE user_id = ?"
var user User
err := db.Get(&user, sqlStr, 123)

// GORM (最简洁)
var user User
err := db.Where("user_id = ?", 123).First(&user).Error
```

---

## 2. 模型定义

### 2.1 基础模型定义

**项目示例: `models/user.go`**

```go
package models

import "gorm.io/gorm"

type User struct {
    // gorm 标签说明:
    // column:user_id    - 指定数据库列名
    // primaryKey        - 标记为主键
    UserID   int64  `json:"user_id,string" gorm:"column:user_id;primaryKey"`

    // uniqueIndex  - 创建唯一索引
    // size:64      - 字段长度限制
    // not null     - 非空约束
    Username string `json:"username" gorm:"column:username;uniqueIndex;size:64;not null"`

    // json:"-"     - JSON 序列化时忽略此字段(安全)
    Password string `json:"-" gorm:"column:password;size:255;not null"`
}

// TableName 自定义表名
// 为什么需要: GORM 默认会将 User 映射为 users (复数)
func (User) TableName() string {
    return "user"  // 明确指定表名为 user
}
```

### 2.2 完整的标签参考

**项目示例: `models/post.go`**

```go
type Post struct {
    // primaryKey - 主键
    ID          int64 `gorm:"column:post_id;primaryKey"`

    // index - 普通索引,提升查询性能
    AuthorID    int64 `gorm:"column:author_id;index;not null"`
    CommunityID int64 `gorm:"column:community_id;index;not null"`

    // default:1 - 默认值
    Status int32 `gorm:"column:status;default:1"`

    // size:128 - VARCHAR(128)
    Title   string `gorm:"column:title;size:128;not null"`

    // type:text - TEXT 类型,存储长文本
    Content string `gorm:"column:content;type:text;not null"`

    // autoCreateTime - 自动填充创建时间
    CreateTime time.Time `gorm:"column:create_time;autoCreateTime"`
}

func (Post) TableName() string {
    return "post"
}
```

### 2.3 常用 GORM 标签完整列表

| 标签 | 说明 | 示例 |
|------|------|------|
| `column:xxx` | 指定列名 | `gorm:"column:user_id"` |
| `primaryKey` | 主键 | `gorm:"primaryKey"` |
| `autoIncrement` | 自增 | `gorm:"autoIncrement"` |
| `size:xxx` | 字段长度 | `gorm:"size:255"` |
| `type:xxx` | 数据类型 | `gorm:"type:text"` |
| `not null` | 非空约束 | `gorm:"not null"` |
| `unique` | 唯一约束 | `gorm:"unique"` |
| `index` | 普通索引 | `gorm:"index"` |
| `uniqueIndex` | 唯一索引 | `gorm:"uniqueIndex"` |
| `default:xxx` | 默认值 | `gorm:"default:0"` |
| `autoCreateTime` | 自动创建时间 | `gorm:"autoCreateTime"` |
| `autoUpdateTime` | 自动更新时间 | `gorm:"autoUpdateTime"` |
| `-` | 忽略字段 | `gorm:"-"` |

---

## 3. 数据库连接

### 3.1 初始化连接

**项目示例: `dao/mysql/mysql.go`**

```go
package mysql

import (
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

var db *gorm.DB

func Init(cfg *settings.MysqlConfig) error {
    // 1. 构建 DSN (Data Source Name)
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
        cfg.User,
        cfg.Password,
        cfg.Host,
        cfg.Port,
        cfg.DbName,
    )

    // 2. GORM 配置
    gormConfig := &gorm.Config{
        // 日志级别: Silent(静默) / Error / Warn / Info
        Logger: logger.Default.LogMode(logger.Info),

        // 禁用外键约束迁移
        DisableForeignKeyConstraintWhenMigrating: true,

        // 预编译语句,提升性能
        PrepareStmt: true,
    }

    // 3. 连接数据库
    db, err = gorm.Open(mysql.Open(dsn), gormConfig)
    if err != nil {
        return err
    }

    // 4. 配置连接池
    sqlDB, _ := db.DB()
    sqlDB.SetMaxOpenConns(200)      // 最大打开连接数
    sqlDB.SetMaxIdleConns(10)       // 最大空闲连接数
    sqlDB.SetConnMaxLifetime(2 * time.Hour)      // 连接最大存活时间
    sqlDB.SetConnMaxIdleTime(10 * time.Minute)   // 连接最大空闲时间

    return nil
}
```

### 3.2 日志级别详解

```go
// Silent - 生产环境推荐,不输出任何 SQL
Logger: logger.Default.LogMode(logger.Silent)

// Error - 只输出错误 SQL
Logger: logger.Default.LogMode(logger.Error)

// Warn - 输出慢 SQL 和错误
Logger: logger.Default.LogMode(logger.Warn)

// Info - 开发环境推荐,输出所有 SQL
Logger: logger.Default.LogMode(logger.Info)
```

---

## 4. 查询操作

### 4.1 单条查询 - First / Last / Take

**项目示例: `dao/mysql/user.go:GetUserByID()`**

```go
// First - 按主键升序,取第一条
var user User
err := db.Where("user_id = ?", 123).First(&user).Error
// SQL: SELECT * FROM user WHERE user_id = 123 ORDER BY user_id LIMIT 1

// Last - 按主键降序,取第一条
err := db.Last(&user).Error
// SQL: SELECT * FROM user ORDER BY user_id DESC LIMIT 1

// Take - 不排序,随机取一条
err := db.Take(&user).Error
// SQL: SELECT * FROM user LIMIT 1
```

**关键点:**
- `First` 会自动添加 `ORDER BY 主键 ASC`
- 找不到记录返回 `gorm.ErrRecordNotFound`
- 必须用 `.Error` 获取错误

### 4.2 多条查询 - Find

**项目示例: `dao/mysql/user.go:GetUsersByIDs()`**

```go
// 查询所有
var users []User
db.Find(&users)
// SQL: SELECT * FROM user

// Where IN 查询
ids := []int64{1, 2, 3}
db.Where("user_id IN ?", ids).Find(&users)
// SQL: SELECT * FROM user WHERE user_id IN (1,2,3)

// 多条件查询
db.Where("status = ? AND age > ?", 1, 18).Find(&users)
// SQL: SELECT * FROM user WHERE status = 1 AND age > 18
```

### 4.3 条件查询 - Where 的多种用法

```go
// 1. 字符串条件 (推荐,防 SQL 注入)
db.Where("username = ?", "admin").First(&user)

// 2. Struct 条件 (只匹配非零值)
db.Where(&User{Username: "admin", Status: 1}).First(&user)
// SQL: SELECT * FROM user WHERE username = 'admin' AND status = 1

// 3. Map 条件
db.Where(map[string]interface{}{
    "username": "admin",
    "status":   0,  // Map 可以匹配零值
}).First(&user)

// 4. 多个 Where 链式调用 (AND 关系)
db.Where("age > ?", 18).
   Where("status = ?", 1).
   Find(&users)
// SQL: SELECT * FROM user WHERE age > 18 AND status = 1
```

### 4.4 Or 条件查询

```go
// Or 查询
db.Where("username = ?", "admin").
   Or("email = ?", "admin@example.com").
   First(&user)
// SQL: SELECT * FROM user WHERE username = 'admin' OR email = 'admin@example.com'

// 复杂 Or 条件
db.Where(
    db.Where("username = ?", "admin").Or("email = ?", "admin@example.com"),
).Where("status = ?", 1).Find(&users)
// SQL: SELECT * FROM user WHERE (username = 'admin' OR email = 'admin@example.com') AND status = 1
```

### 4.5 选择字段 - Select

**项目示例: `dao/mysql/community.go:GetCommunityList()`**

```go
// 只查询指定字段
db.Select("community_id", "community_name").Find(&communities)
// SQL: SELECT community_id, community_name FROM community

// 排除某些字段
db.Omit("password").Find(&users)
// SQL: SELECT user_id, username FROM user (排除 password)

// 查询单个字段值
var names []string
db.Model(&User{}).Pluck("username", &names)
// SQL: SELECT username FROM user
```

### 4.6 排序 - Order

**项目示例: `dao/mysql/post.go:GetPostListByCommunityID()`**

```go
// 单字段排序
db.Order("create_time DESC").Find(&posts)
// SQL: SELECT * FROM post ORDER BY create_time DESC

// 多字段排序
db.Order("status ASC, create_time DESC").Find(&posts)
// SQL: SELECT * FROM post ORDER BY status ASC, create_time DESC
```

### 4.7 分页 - Limit / Offset

**项目示例: `dao/mysql/post.go:GetPostListByCommunityID()`**

```go
// 典型的分页查询
page := 1
size := 10

db.Where("community_id = ?", communityID).
   Order("create_time DESC").
   Offset(int((page - 1) * size)).  // 跳过前 N 条
   Limit(int(size)).                 // 取 N 条
   Find(&posts)
// SQL: SELECT * FROM post WHERE community_id = 1
//      ORDER BY create_time DESC LIMIT 10 OFFSET 0
```

### 4.8 统计 - Count

**项目示例: `dao/mysql/user.go:CheckUserExist()`**

```go
// 统计记录数
var count int64
db.Model(&User{}).Where("username = ?", "admin").Count(&count)
// SQL: SELECT COUNT(*) FROM user WHERE username = 'admin'

// 统计总数
db.Model(&Post{}).Count(&count)
// SQL: SELECT COUNT(*) FROM post
```

### 4.9 原生 SQL 查询

```go
// 原生查询
type Result struct {
    Username string
    PostCount int
}
var results []Result
db.Raw(`
    SELECT u.username, COUNT(p.post_id) as post_count
    FROM user u
    LEFT JOIN post p ON u.user_id = p.author_id
    GROUP BY u.user_id
`).Scan(&results)

// 带参数的原生查询
db.Raw("SELECT * FROM user WHERE user_id = ?", 123).Scan(&user)
```

---

## 5. 插入操作

### 5.1 单条插入 - Create

**项目示例: `dao/mysql/user.go:InsertUser()`**

```go
// 插入单条记录
user := &User{
    UserID:   123,
    Username: "testuser",
    Password: "hashed_password",
}
err := db.Create(user).Error
// SQL: INSERT INTO user (user_id, username, password) VALUES (123, 'testuser', 'hashed_password')

// 插入后,user.ID 会被自动填充 (如果是自增主键)
fmt.Println(user.UserID)  // 自增的 ID
```

**项目示例: `dao/mysql/post.go:CreatePost()`**

```go
func CreatePost(post *models.Post) error {
    err := db.Create(post).Error
    if err != nil {
        return fmt.Errorf("insert post failed: %w", err)
    }
    return nil
}
```

### 5.2 批量插入 - CreateInBatches

```go
// 批量插入 (一次性插入多条)
users := []*User{
    {UserID: 1, Username: "user1"},
    {UserID: 2, Username: "user2"},
    {UserID: 3, Username: "user3"},
}
db.Create(&users)
// SQL: INSERT INTO user (user_id, username) VALUES
//      (1, 'user1'), (2, 'user2'), (3, 'user3')

// 分批插入 (每批 100 条,避免单次插入太多)
db.CreateInBatches(users, 100)
```

### 5.3 选择性插入 - Select / Omit

```go
// 只插入指定字段
db.Select("username", "email").Create(&user)
// SQL: INSERT INTO user (username, email) VALUES ('admin', 'admin@example.com')

// 忽略某些字段
db.Omit("password").Create(&user)
// SQL: INSERT INTO user (user_id, username) VALUES (123, 'admin')
```

---

## 6. 更新操作

### 6.1 更新单个字段 - Update

```go
// 更新单个字段
db.Model(&User{}).Where("user_id = ?", 123).Update("status", 1)
// SQL: UPDATE user SET status = 1 WHERE user_id = 123

// 更新多个字段 - Updates (struct)
db.Model(&User{}).Where("user_id = ?", 123).Updates(User{
    Username: "newname",
    Status:   1,
})
// SQL: UPDATE user SET username = 'newname', status = 1 WHERE user_id = 123

// 更新多个字段 - Updates (map,可更新零值)
db.Model(&User{}).Where("user_id = ?", 123).Updates(map[string]interface{}{
    "username": "newname",
    "status":   0,  // struct 零值会被忽略,map 不会
})
```

### 6.2 全局更新 (危险)

```go
// 不加 Where 更新所有记录 (GORM 会报错,需要加参数)
db.Model(&User{}).Update("status", 1)
// 报错: UPDATE statements without WHERE clauses not allowed

// 确实要更新所有,需要加 Where("1 = 1")
db.Model(&User{}).Where("1 = 1").Update("status", 1)
// SQL: UPDATE user SET status = 1
```

### 6.3 使用表达式更新

```go
// 字段自增
db.Model(&Post{}).Where("post_id = ?", 123).Update("view_count", gorm.Expr("view_count + ?", 1))
// SQL: UPDATE post SET view_count = view_count + 1 WHERE post_id = 123

// 批量自减
db.Model(&User{}).Where("status = ?", 1).Update("score", gorm.Expr("score - ?", 10))
// SQL: UPDATE user SET score = score - 10 WHERE status = 1
```

---

## 7. 删除操作

### 7.1 硬删除 - Delete

```go
// 根据主键删除
db.Delete(&User{}, 123)
// SQL: DELETE FROM user WHERE user_id = 123

// 根据条件删除
db.Where("status = ?", 0).Delete(&User{})
// SQL: DELETE FROM user WHERE status = 0

// 批量删除
db.Where("user_id IN ?", []int64{1, 2, 3}).Delete(&User{})
// SQL: DELETE FROM user WHERE user_id IN (1,2,3)
```

### 7.2 软删除 (推荐)

```go
// 在模型中添加 DeletedAt 字段
type User struct {
    UserID    int64
    Username  string
    DeletedAt gorm.DeletedAt `gorm:"index"`  // 软删除标记
}

// 软删除 (不会真正删除,只设置 deleted_at)
db.Delete(&user)
// SQL: UPDATE user SET deleted_at = '2025-12-23 16:00:00' WHERE user_id = 123

// 查询时自动过滤软删除的记录
db.Find(&users)
// SQL: SELECT * FROM user WHERE deleted_at IS NULL

// 查询包括软删除的记录
db.Unscoped().Find(&users)
// SQL: SELECT * FROM user

// 永久删除
db.Unscoped().Delete(&user)
// SQL: DELETE FROM user WHERE user_id = 123
```

---

## 8. 高级查询

### 8.1 子查询

```go
// 子查询
db.Where("user_id IN (?)",
    db.Model(&Post{}).Select("author_id").Where("status = ?", 1),
).Find(&users)
// SQL: SELECT * FROM user WHERE user_id IN
//      (SELECT author_id FROM post WHERE status = 1)
```

### 8.2 分组查询 - Group / Having

```go
type Result struct {
    CommunityID int64
    PostCount   int
}

var results []Result
db.Model(&Post{}).
   Select("community_id, COUNT(*) as post_count").
   Group("community_id").
   Having("COUNT(*) > ?", 10).
   Scan(&results)
// SQL: SELECT community_id, COUNT(*) as post_count FROM post
//      GROUP BY community_id HAVING COUNT(*) > 10
```

### 8.3 联表查询 - Joins

```go
// 左连接
type UserWithPost struct {
    User
    PostCount int
}

var results []UserWithPost
db.Model(&User{}).
   Select("user.*, COUNT(post.post_id) as post_count").
   Joins("LEFT JOIN post ON user.user_id = post.author_id").
   Group("user.user_id").
   Scan(&results)
// SQL: SELECT user.*, COUNT(post.post_id) as post_count FROM user
//      LEFT JOIN post ON user.user_id = post.author_id
//      GROUP BY user.user_id
```

### 8.4 预加载 - Preload (解决 N+1 问题)

```go
// 假设 Post 模型定义了关联
type Post struct {
    ID        int64
    AuthorID  int64
    Author    User  `gorm:"foreignKey:AuthorID"`  // 定义关联
}

// N+1 问题示例 (不推荐)
var posts []Post
db.Find(&posts)
for _, post := range posts {
    db.Where("user_id = ?", post.AuthorID).First(&post.Author)  // 循环查询,N+1
}

// 使用 Preload 解决 (推荐)
db.Preload("Author").Find(&posts)
// SQL1: SELECT * FROM post
// SQL2: SELECT * FROM user WHERE user_id IN (1,2,3...)  // 一次查询所有作者
```

**项目实际解决方案:**

由于项目未定义关联,使用手动批量查询:

```go
// logic/post.go 中的解决方案
// 1. 查询所有帖子
posts := GetPostListByIDs(ids)

// 2. 提取所有 AuthorID
authorIDs := make([]int64, 0, len(posts))
for _, post := range posts {
    authorIDs = append(authorIDs, post.AuthorID)
}

// 3. 批量查询作者 (避免 N+1)
authors := GetUsersByIDs(authorIDs)  // 1 次查询

// 4. 组装数据
```

### 8.5 事务处理

```go
// 自动事务
err := db.Transaction(func(tx *gorm.DB) error {
    // 1. 创建用户
    if err := tx.Create(&user).Error; err != nil {
        return err  // 返回错误,自动回滚
    }

    // 2. 创建帖子
    if err := tx.Create(&post).Error; err != nil {
        return err  // 自动回滚
    }

    return nil  // 提交事务
})

// 手动事务
tx := db.Begin()  // 开始事务

if err := tx.Create(&user).Error; err != nil {
    tx.Rollback()  // 回滚
    return err
}

if err := tx.Create(&post).Error; err != nil {
    tx.Rollback()
    return err
}

tx.Commit()  // 提交
```

### 8.6 锁机制

```go
// 悲观锁 - 排它锁 (FOR UPDATE)
db.Clauses(clause.Locking{Strength: "UPDATE"}).
   Where("user_id = ?", 123).
   First(&user)
// SQL: SELECT * FROM user WHERE user_id = 123 FOR UPDATE

// 共享锁 (FOR SHARE)
db.Clauses(clause.Locking{Strength: "SHARE"}).First(&user)
// SQL: SELECT * FROM user LIMIT 1 FOR SHARE
```

---

## 9. 错误处理

### 9.1 常见错误类型

**项目示例: `dao/mysql/user.go:GetUserByID()`**

```go
import (
    "errors"
    "gorm.io/gorm"
)

// 1. 记录不存在
err := db.Where("user_id = ?", 999).First(&user).Error
if errors.Is(err, gorm.ErrRecordNotFound) {
    // 找不到记录,返回 nil 而不是错误
    return nil, nil
}

// 2. 其他数据库错误
if err != nil {
    return nil, fmt.Errorf("query failed: %w", err)
}
```

### 9.2 错误判断完整示例

```go
func GetUserByID(uid int64) (*User, error) {
    user := &User{}
    err := db.Where("user_id = ?", uid).First(user).Error

    if err != nil {
        // 判断是否是记录不存在
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil  // 不是错误,只是查不到
        }

        // 其他错误 (连接断开、SQL语法错误等)
        return nil, fmt.Errorf("query user failed: %w", err)
    }

    return user, nil
}
```

### 9.3 常见 GORM 错误

| 错误 | 说明 | 如何判断 |
|------|------|----------|
| `gorm.ErrRecordNotFound` | 查不到记录 | `errors.Is(err, gorm.ErrRecordNotFound)` |
| `gorm.ErrInvalidTransaction` | 无效事务 | `errors.Is(err, gorm.ErrInvalidTransaction)` |
| `gorm.ErrNotImplemented` | 未实现的功能 | `errors.Is(err, gorm.ErrNotImplemented)` |
| `gorm.ErrMissingWhereClause` | 缺少 WHERE (全表更新/删除) | `errors.Is(err, gorm.ErrMissingWhereClause)` |
| `gorm.ErrInvalidData` | 无效数据 | `errors.Is(err, gorm.ErrInvalidData)` |

---

## 10. 性能优化

### 10.1 索引优化

```go
// 在模型中定义索引
type User struct {
    UserID   int64  `gorm:"primaryKey"`
    Username string `gorm:"uniqueIndex"`        // 唯一索引
    Email    string `gorm:"index"`              // 普通索引
    Status   int    `gorm:"index:idx_status"`   // 命名索引
}

// 复合索引
type Post struct {
    CommunityID int64     `gorm:"index:idx_community_time"`
    CreateTime  time.Time `gorm:"index:idx_community_time"`
}
// 会创建 (community_id, create_time) 复合索引
```

### 10.2 批量查询优化

**项目示例: `dao/mysql/post.go:GetPostListByIDs()`**

```go
// ❌ 不好: N+1 查询
for _, id := range ids {
    db.Where("post_id = ?", id).First(&post)  // 循环查询 N 次
}

// ✅ 好: 批量查询
db.Where("post_id IN ?", ids).Find(&posts)  // 1 次查询
```

### 10.3 选择必要字段

```go
// ❌ 不好: 查询所有字段 (包括大字段 content)
db.Find(&posts)
// SQL: SELECT post_id, title, content, ... FROM post  (content 可能很大)

// ✅ 好: 只查询需要的字段
db.Select("post_id", "title", "create_time").Find(&posts)
// SQL: SELECT post_id, title, create_time FROM post
```

### 10.4 连接池配置

**项目示例: `dao/mysql/mysql.go`**

```go
sqlDB, _ := db.DB()

// 最大打开连接数
// 为什么: 防止连接数过多压垮数据库
sqlDB.SetMaxOpenConns(200)

// 最大空闲连接数
// 为什么: 保持连接池,避免频繁创建/销毁连接
sqlDB.SetMaxIdleConns(10)

// 连接最大存活时间
// 为什么: 防止连接长时间未使用被服务端断开
sqlDB.SetConnMaxLifetime(2 * time.Hour)

// 连接最大空闲时间
// 为什么: 及时回收长时间空闲的连接
sqlDB.SetConnMaxIdleTime(10 * time.Minute)
```

### 10.5 预编译语句

```go
// GORM 配置开启预编译
gormConfig := &gorm.Config{
    PrepareStmt: true,  // 开启预编译,提升性能
}

// 原理:
// 第一次: PREPARE stmt FROM 'SELECT * FROM user WHERE user_id = ?'
// 后续: EXECUTE stmt USING 123  (复用已编译的语句)
```

### 10.6 分页查询优化

```go
// ❌ 不好: OFFSET 很大时性能差
db.Offset(10000).Limit(10).Find(&posts)
// SQL: SELECT * FROM post LIMIT 10 OFFSET 10000  (需要扫描 10010 条)

// ✅ 好: 使用 WHERE 过滤 (需要索引)
db.Where("post_id > ?", lastID).Limit(10).Find(&posts)
// SQL: SELECT * FROM post WHERE post_id > 10000 LIMIT 10  (利用索引)
```

---

## 11. 项目实战技巧

### 11.1 统一错误处理

**项目规范:**

```go
// DAO 层: 只返回错误,不打印日志
func GetUserByID(uid int64) (*User, error) {
    user := &User{}
    err := db.Where("user_id = ?", uid).First(user).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil  // 查不到返回 nil
        }
        return nil, fmt.Errorf("query user failed: %w", err)
    }
    return user, nil
}

// Logic 层: 记录详细日志
func GetUser(uid int64) (*User, error) {
    user, err := mysql.GetUserByID(uid)
    if err != nil {
        zap.L().Error("get user failed",
            zap.Int64("user_id", uid),
            zap.Error(err))
        return nil, err
    }
    return user, nil
}
```

### 11.2 避免 N+1 查询

**项目示例: `logic/post.go`**

```go
// ❌ 不好: N+1 查询
posts := GetPostList()
for _, post := range posts {
    post.Author = GetUserByID(post.AuthorID)      // N 次查询
    post.Community = GetCommunityByID(post.CommunityID)  // N 次查询
}

// ✅ 好: 批量查询
posts := GetPostList()

// 1. 提取所有 ID
authorIDs := extractAuthorIDs(posts)
communityIDs := extractCommunityIDs(posts)

// 2. 批量查询 (2 次查询)
authors := GetUsersByIDs(authorIDs)
communities := GetCommunitiesByIDs(communityIDs)

// 3. 组装数据
assemblePostDetails(posts, authors, communities)
```

### 11.3 字段安全性

```go
// 敏感字段不要序列化到 JSON
type User struct {
    UserID   int64  `json:"user_id"`
    Username string `json:"username"`
    Password string `json:"-"`  // ⭐ json:"-" 防止密码泄露
}

// API 返回时,password 字段会被忽略
c.JSON(200, user)  // {"user_id": 123, "username": "admin"}
```

### 11.4 表名映射规范

```go
// ⭐ 所有模型都必须实现 TableName()
func (User) TableName() string {
    return "user"  // 明确指定,避免 GORM 自动复数化为 users
}

func (Post) TableName() string {
    return "post"  // 而不是 posts
}

func (CommunityDetail) TableName() string {
    return "community"  // 而不是 community_details
}
```

---

## 12. 常见问题 FAQ

### Q1: First 和 Take 有什么区别?

```go
// First - 会自动排序 (ORDER BY 主键)
db.First(&user)
// SQL: SELECT * FROM user ORDER BY user_id LIMIT 1

// Take - 不排序,随机取一条
db.Take(&user)
// SQL: SELECT * FROM user LIMIT 1
```

**使用建议:** 查询单条记录用 `First`,性能更好且结果可预测。

### Q2: Where 中的 struct 和 map 有什么区别?

```go
// Struct - 零值会被忽略
db.Where(&User{Status: 0}).Find(&users)
// SQL: SELECT * FROM user  (status = 0 被忽略!)

// Map - 零值不会被忽略
db.Where(map[string]interface{}{"status": 0}).Find(&users)
// SQL: SELECT * FROM user WHERE status = 0  ✅
```

**使用建议:** 需要匹配零值时用 `map`,其他情况用字符串条件。

### Q3: 如何调试 SQL?

```go
// 方法1: 开启日志
gormConfig := &gorm.Config{
    Logger: logger.Default.LogMode(logger.Info),  // 打印所有 SQL
}

// 方法2: 单次查询开启调试
db.Debug().Where("user_id = ?", 123).First(&user)
// 会打印: SELECT * FROM user WHERE user_id = 123

// 方法3: 只看 SQL 不执行
sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
    return tx.Where("user_id = ?", 123).First(&user)
})
fmt.Println(sql)  // SELECT * FROM user WHERE user_id = 123
```

### Q4: 如何处理 NULL 值?

```go
// 使用 sql.NullXxx 类型
type User struct {
    UserID   int64
    Username string
    Email    sql.NullString  // 可为 NULL 的字段
    Age      sql.NullInt64   // 可为 NULL 的整数
}

// 使用指针类型 (推荐)
type User struct {
    UserID   int64
    Username string
    Email    *string  // nil 表示 NULL
    Age      *int     // nil 表示 NULL
}
```

### Q5: GORM 支持哪些数据库?

- MySQL / MariaDB
- PostgreSQL
- SQLite
- SQL Server
- TiDB
- ClickHouse
- ...

只需要更换驱动:

```go
import "gorm.io/driver/postgres"
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
```

---

## 13. 学习资源

### 官方文档
- 中文文档: https://gorm.io/zh_CN/docs/
- 英文文档: https://gorm.io/docs/

### 推荐阅读顺序

1. ✅ **本文档** - 基于实战的语法糖教程
2. 📖 **GORM 官方指南** - 连接、CRUD、关联
3. 🚀 **GORM 高级功能** - Hooks、Plugin、自定义类型
4. 💡 **性能优化** - 索引、连接池、批量操作

### 实战建议

1. **先从简单 CRUD 开始** - Create / Find / Update / Delete
2. **掌握 Where 条件查询** - 这是最常用的
3. **学会批量查询** - 避免 N+1 问题
4. **理解事务** - 保证数据一致性
5. **性能优化** - 索引、连接池、预编译

---

## 14. 总结

### GORM 核心优势

| 对比项 | 原生 SQL | sqlx | GORM |
|--------|----------|------|------|
| 代码量 | 多 | 中等 | 少 |
| 类型安全 | 弱 | 中 | 强 |
| 学习成本 | 低 | 低 | 中 |
| 功能丰富度 | 手动实现 | 中等 | 丰富 |
| 性能 | 最高 | 高 | 较高 |
| 推荐场景 | 复杂SQL | 简单项目 | **中大型项目** ✅ |

### 最佳实践

1. ✅ **所有模型实现 TableName()** - 避免表名错误
2. ✅ **使用字符串条件查询** - 防 SQL 注入
3. ✅ **批量查询避免 N+1** - Where IN 一次查询
4. ✅ **错误统一处理** - DAO 返回错误,Logic 记录日志
5. ✅ **敏感字段 json:"-"** - 防止密码泄露
6. ✅ **开发环境开启日志** - 方便调试 SQL
7. ✅ **生产环境关闭日志** - 提升性能

---

**Happy Coding! 🚀**

如果有任何疑问,欢迎参考项目代码:
- `dao/mysql/user.go` - 用户 CRUD 示例
- `dao/mysql/post.go` - 帖子查询示例
- `dao/mysql/community.go` - 社区查询示例
