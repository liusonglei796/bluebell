# Go 服务性能优化指南

## 1. 性能瓶颈分析

### 1.1 如何定位瓶颈

使用 **pprof** 和 **压测工具** 配合：

```bash
# 1. 压测
wrk -t4 -c100 -d30s http://127.0.0.1:8080/api/v1/community

# 2. 采集 CPU profile（压测期间）
go tool pprof -http=:9090 http://127.0.0.1:8080/debug/pprof/profile?seconds=30

# 3. 浏览器查看火焰图
http://localhost:9090
```

### 1.2 火焰图阅读方法

```
从下往上看 = 调用链
条越宽 = 耗时越多
最顶层 = 热点函数
```

### 1.3 常见热点模式

| 火焰图特征 | 问题诊断 | 解决方案 |
|------------|----------|----------|
| `cgocall` 最宽 | 数据库/网络I/O | **加缓存** |
| `execIO` 最宽 | 网络I/O等待 | 加缓存/优化连接 |
| `lock` 突出 | 锁竞争 | 无锁数据结构 |
| `gc` 占比高 | 内存分配过多 | 对象池/减少分配 |
| `regexp` 突出 | 正则表达式 | 预编译/缓存 |
| `json.Unmarshal` | JSON解析 | 减少解析 |

---

## 2. 性能优化方案

### 方案1：缓存优化（最有效）

**适用场景**：读多写少，数据变化不频繁

**优化效果**：减少数据库 I/O，QPS 提升 5-10 倍

**实现方式**：

```go
// 1. Redis 缓存
func (s *Service) GetCommunityList() ([]Community, error) {
    // 先查缓存
    cached, err := s.cache.Get("community:list")
    if err == nil && cached != "" {
        return unmarshal(cached)
    }
    
    // 缓存未命中，查数据库
    data, err := s.db.GetCommunityList()
    if err != nil {
        return nil, err
    }
    
    // 写入缓存（设置过期时间）
    s.cache.Set("community:list", marshal(data), 5*time.Minute)
    return data
}
```

```go
// 2. 本地内存缓存（更快）
var communityCache struct {
    data   []Community
    expire time.Time
    mu     sync.RWMutex
}

func GetCommunityList() ([]Community, error) {
    communityCache.mu.RLock()
    if time.Now().Before(communityCache.expire) {
        defer communityCache.mu.RUnlock()
        return communityCache.data, nil
    }
    communityCache.mu.RUnlock()
    
    // 缓存过期，重新加载
    data, _ := loadFromDB()
    
    communityCache.mu.Lock()
    communityCache.data = data
    communityCache.expire = time.Now().Add(5 * time.Minute)
    communityCache.mu.Unlock()
    
    return data
}
```

---

### 方案2：连接池优化

**适用场景**：高并发，频繁创建连接

**优化配置**：

```go
// MySQL 连接池
sqlDB.SetMaxOpenConns(200)      // 最大连接数
sqlDB.SetMaxIdleConns(30)       // 空闲连接数
sqlDB.SetConnMaxLifetime(2 * time.Hour)  // 连接生命周期

// Redis 连接池
redisClient := redis.NewClient(&redis.Options{
    PoolSize: 100,              // 连接池大小
    MinIdleConns: 10,           // 最小空闲连接
    DialTimeout: 5 * time.Second,
    ReadTimeout: 3 * time.Second,
    WriteTimeout: 3 * time.Second,
})
```

---

### 方案3：SQL 预编译

**适用场景**：重复查询，SQL 解析开销大

**已启用**：项目已配置 `PrepareStmt: true`

```go
// 文件：internal/dao/database/init.go
gormConfig := &gorm.Config{
    PrepareStmt: true,  // 已启用
}
```

---

### 方案4：索引优化

**适用场景**：查询慢，数据库全表扫描

```sql
-- 添加索引
CREATE INDEX idx_post_user ON post(user_id);
CREATE INDEX idx_post_create ON post(created_at);

-- 查看慢查询
SHOW VARIABLES LIKE 'slow_query_log';
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 1;
```

---

### 方案5：减少内存分配

**适用场景**：GC 压力大，内存占用高

```go
// ❌ 避免：频繁分配对象
func GetData() []Result {
    var results []Result
    for i := 0; i < 1000; i++ {
        results = append(results, Result{...})  // 不断扩容
    }
    return results
}

// ✅ 优化：预分配容量
func GetData() []Result {
    results := make([]Result, 0, 1000)  // 预分配
    for i := 0; i < 1000; i++ {
        results = append(results, Result{...})
    }
    return results
}

// ✅ 优化：使用对象池
var bufPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func Process() {
    buf := bufPool.Get().(*bytes.Buffer)
    defer bufPool.Put(buf)
    buf.Reset()
    // 使用 buf
}
```

---

### 方案6：批量查询

**适用场景**：N+1 查询问题

```go
// ❌ 避免：N+1 查询
func GetPostsWithUsers(postIDs []int64) []Post {
    posts := make([]Post, len(postIDs))
    for i, id := range postIDs {
        posts[i] = db.First(&Post{}, id)  // N 次查询
        posts[i].User = db.First(&User{}, posts[i].UserID)  // 又 N 次
    }
    return posts
}

// ✅ 优化：批量查询
func GetPostsWithUsers(postIDs []int64) []Post {
    // 1 次查询所有帖子
    db.Where("id IN ?", postIDs).Find(&posts)
    
    // 1 次查询所有用户
    userIDs := make([]int64, len(posts))
    for i, p := range posts {
        userIDs[i] = p.UserID
    }
    users := make(map[int64]User)
    db.Where("id IN ?", userIDs).Find(&userList)
    for _, u := range userList {
        users[u.ID] = u
    }
    
    // 内存组装
    for i := range posts {
        posts[i].User = users[posts[i].UserID]
    }
    return posts
}
```

---

## 3. 优化效果对比

| 优化方案 | 预期提升 | 实现难度 |
|----------|----------|----------|
| Redis 缓存 | 5-10x | ⭐⭐ |
| 本地缓存 | 10-50x | ⭐⭐ |
| 连接池 | 1.5-2x | ⭐ |
| 预编译 SQL | 1.2-1.5x | ⭐（已启用） |
| 索引 | 10-100x | ⭐⭐ |
| 批量查询 | 2-5x | ⭐⭐ |

---

## 4. 优化优先级

```
第一步：加缓存（效果最明显）
    ↓
第二步：优化连接池配置
    ↓
第三步：添加索引
    ↓
第四步：解决 N+1 问题
    ↓
第五步：减少内存分配
```

---

## 5. 监控指标

| 指标 | 正常范围 | 告警阈值 |
|------|----------|----------|
| QPS | >1000 | <500 |
| 响应时间 P99 | <100ms | >500ms |
| CPU 使用率 | <70% | >80% |
| 内存使用 | 稳定 | 持续增长 |
| Goroutine 数量 | <1000 | >5000 |
| MySQL 连接使用率 | <70% | >80% |

---

## 6. 你的项目优化建议

### 当前瓶颈
- `execIO` + `cgocall` 占比 80%
- 每次请求都查数据库

### 推荐方案

1. **社区列表接口加 Redis 缓存**
   - 数据变化不频繁
   - 访问频率高
   - 预计 QPS 从 3000 → 15000+

2. **帖子列表接口加缓存 + 分页**
   - 热门帖子缓存
   - 分页数据缓存

3. **用户信息加本地缓存**
   - 热点用户数据

---

## 7. 相关命令

```bash
# 压测
wrk -t4 -c100 -d30s http://127.0.0.1:8080/api/v1/community

# CPU profile
go tool pprof -http=:9090 http://127.0.0.1:8080/debug/pprof/profile?seconds=30

# 内存 profile  
go tool pprof -http=:9090 http://127.0.0.1:8080/debug/pprof/heap

# Goroutine 分析
go tool pprof http://127.0.0.1:8080/debug/pprof/goroutine
```
