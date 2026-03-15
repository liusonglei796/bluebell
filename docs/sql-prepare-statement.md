# SQL 预编译（Prepared Statements）使用指南

## 1. 什么是 SQL 预编译？

SQL 预编译（Prepared Statements）是一种数据库优化技术，它将 SQL 语句的编译和执行分开：

```
普通查询流程：
  客户端 → 发送SQL → 服务器解析 → 编译 → 执行 → 返回结果

预编译流程：
  1. 客户端 → 发送预编译模板 → 服务器编译 → 返回句柄ID
  2. 客户端 → 发送句柄ID + 参数 → 服务器执行 → 返回结果
```

## 2. 预编译的优势

| 优势 | 说明 |
|------|------|
| **减少解析开销** | SQL 语句只编译一次，后续执行复用编译结果 |
| **防止 SQL 注入** | 参数与 SQL 分离，天然防御注入攻击 |
| **减少网络传输** | 预编译模板只传输一次，后续只传参数 |
| **提升性能** | 高并发场景下性能提升 20%-50% |

## 3. GORM 中启用预编译

### 方式一：全局启用（推荐）

在 `internal/dao/database/init.go` 中配置：

```go
gormConfig := &gorm.Config{
    Logger:                                   newLogger,
    DisableForeignKeyConstraintWhenMigrating: true,
    PrepareStmt:                              true,  // ← 启用预编译
}

db, err := gorm.Open(mysql.Open(dsn), gormConfig)
```

### 方式二：针对单个查询启用

```go
// 使用 PreparedStmt 为 true 的 db 实例
db := db.Session(&gorm.Session{PrepareStmt: true})

// 之后的查询都会使用预编译
db.Find(&users)
```

## 4. 当前项目状态

✅ **已启用预编译**

文件：`internal/dao/database/init.go` 第68行

```go
gormConfig := &gorm.Config{
    // ...
    PrepareStmt: true,  // 已启用
}
```

## 5. 验证预编译是否生效

### 方式一：查看 MySQL 进程列表

```sql
-- 查看当前预编译语句
SELECT * FROM mysql.prepared_statements_instance;
```

### 方式二：开启 GORM SQL 日志

在 debug 模式下，GORM 会打印 SQL：

```
[2026/03/12 10:00:00]  [0.42ms]  [default] SELECT * FROM `community`  WHERE ... AND `deleted_at` IS NULL
```

如果启用了预编译，会看到类似 `Preparing` 和 `Executing` 的日志。

### 方式三：使用 pprof 分析

```
curl http://127.0.0.1:8080/debug/pprof/heap?debug=1
```

查看 `gorm.(*PreparedStmtDB).QueryContext` 的调用情况。

## 6. 预编译与连接池的关系

预编译和连接池是**互补**的：

| 组件 | 作用 |
|------|------|
| **连接池** | 复用数据库连接，避免频繁建立/断开连接 |
| **预编译** | 复用 SQL 编译结果，避免重复解析 SQL |

两者结合使用效果最佳：

```go
// 连接池配置
sqlDB.SetMaxOpenConns(200)   // 最大连接数
sqlDB.SetMaxIdleConns(30)    // 空闲连接数
sqlDB.SetConnMaxLifetime(2 * time.Hour)  // 连接生命周期

// 预编译配置
PrepareStmt: true
```

## 7. 注意事项

### 预编译的潜在问题

1. **连接池耗尽**
   - 如果预编译语句过多，可能占用连接池资源
   - 解决：合理设置 `MaxOpenConns`

2. **长时间占用连接**
   - 预编译语句会在连接上保持
   - 解决：设置合理的 `ConnMaxLifetime`

3. **内存占用**
   - 服务端需要存储预编译语句
   - MySQL 默认 `max_prepared_stmt_count` 为 16382

### 查看 MySQL 预编译上限

```sql
SHOW VARIABLES LIKE 'max_prepared_stmt_count';
```

如果需要更多：

```ini
# my.cnf
max_prepared_stmt_count = 65532
```

## 8. 性能对比测试

### 测试脚本

```go
package main

import (
    "testing"
    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

func BenchmarkNormalQuery(b *testing.B) {
    db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    for i := 0; i < b.N; i++ {
        var community model.Community
        db.First(&community, 1)
    }
}

func BenchmarkPreparedQuery(b *testing.B) {
    db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{
        PrepareStmt: true,
    })
    for i := 0; i < b.N; i++ {
        var community model.Community
        db.First(&community, 1)
    }
}
```

### 运行测试

```bash
go test -bench=. -benchmem -run=^$
```

## 9. 总结

| 项目 | 状态 | 说明 |
|------|------|------|
| 全局预编译 | ✅ 已启用 | `PrepareStmt: true` |
| 连接池 | ✅ 已配置 | 200 连接，30 空闲 |
| SQL 日志 | ✅ 已配置 | debug 模式开启 |

**当前项目已正确配置预编译，无需额外修改。**

如需进一步优化，建议添加 Redis 缓存。
