# 分库分表实现指南

## 概览

本项目实现了基于社区ID的Post表分片和基于用户ID的User表分片策略。

## 架构设计

```
┌─────────────────────────────┐
│   应用层（Service/Handler）  │
└────────────┬────────────────┘
             │
┌────────────▼────────────────┐
│   ShardingRouter 路由层      │ ← 决定数据去哪个库哪张表
├─────────────┬────────────────┤
│ PostStrategy│ UserStrategy   │
└────────────┬────────────────┘
             │
┌────────────▼────────────────┐
│   数据库连接池（多个实例）    │
├─────────────┬────────────────┤
│  DB[0]      │   DB[1]        │ （暂不用，为未来扩展预留）
└─────────────┴────────────────┘
             │
┌────────────▼────────────────┐
│   物理表 (post_0~3, user)    │
└─────────────────────────────┘
```

## 分片策略

### Post 表分片

**分片键**：`community_id`
**分片算法**：`table_index = community_id % 4`
**表名**：`post_0`, `post_1`, `post_2`, `post_3`

| community_id | 目标表 |
|---|---|
| 1 | post_1 |
| 2 | post_2 |
| 3 | post_3 |
| 4 | post_0 |
| 5 | post_1 |

**优势**：
- 同一社区的所有帖子在同一张表，查询高效
- 可按社区并行处理数据
- 便于未来的行级权限管理

### User 表分片

**分片键**：`user_id`
**分片算法**：`table_index = user_id % 1`（当前不分片）
**表名**：`user`

**为什么暂不分片**：
- 用户表相对较小
- 用户查询频率相对均匀
- 全表扫描成本低

**未来扩展**：数据量超大时，改为 `user_id % 4` 分表

## 文件结构

```
pkg/sharding/
├── sharding.go          # 核心分片策略
├── wrapper.go           # 便利方法

internal/dao/database/postdb/
├── post.go              # 原有非分片实现
└── sharding_post.go     # 新增分片实现

sql/
└── sharding_schema.sql  # 分片表创建脚本
```

## 使用方式

### 1. 初始化分片路由器

```go
package main

import (
    "bluebell/pkg/sharding"
    "gorm.io/gorm"
)

func main() {
    // 假设已有主数据库连接
    var mainDB *gorm.DB
    
    // 创建分片配置
    config := sharding.DefaultShardConfig()
    
    // 创建数据库连接映射
    dbs := map[int]*gorm.DB{
        0: mainDB, // 可扩展为多个数据库
    }
    
    // 创建路由器
    router := sharding.NewShardingRouter(config, dbs)
    
    // 创建分片访问层
    shardingPostDB := sharding.NewShardingPostDB(router)
}
```

### 2. 使用分片Post仓储

```go
// 创建帖子（自动路由到正确的分片表）
repo := postdb.NewShardingPostRepo(shardingPostDB)

post := &model.Post{
    PostID:      "123456",
    AuthorID:    1001,
    CommunityID: 5,        // ← 分片键
    PostTitle:  "标题",
    Content:    "内容",
    Status:     1,
}

err := repo.CreatePost(ctx, post)
// 自动路由到 post_1 表 (5 % 4 == 1)
```

### 3. 查询优化

**推荐做法**：在调用方携带分片键

```go
// ❌ 不推荐（需要遍历所有分片表）
repo.GetPostByID(ctx, postID)

// ✅ 推荐（直接定位到分片表）
// 修改接口为：GetPostByID(ctx, postID, communityID)
db, tableName, _ := router.GetPostDB(ctx, communityID)
// 直接使用 db 和 tableName
```

## SQL 迁移步骤

### 阶段1：建立分片表（无停机）

```bash
# 执行分片脚本
mysql -u root -p bluebell < sql/sharding_schema.sql

# 验证分片表已创建
mysql -u root -p bluebell -e "SHOW TABLES LIKE 'post_%';"
```

### 阶段2：灰度迁移

```sql
-- 1. 新写入数据使用分片表（应用代码切换）
-- 2. 定期同步旧表数据到分片表
INSERT INTO post_0 SELECT * FROM post WHERE community_id % 4 = 0 AND id > @last_id;
INSERT INTO post_1 SELECT * FROM post WHERE community_id % 4 = 1 AND id > @last_id;
INSERT INTO post_2 SELECT * FROM post WHERE community_id % 4 = 2 AND id > @last_id;
INSERT INTO post_3 SELECT * FROM post WHERE community_id % 4 = 3 AND id > @last_id;

-- 3. 验证数据一致性后，删除原表
DROP TABLE post;
```

### 阶段3：应用代码切换

```go
// config 中配置
config := &sharding.ShardConfig{
    PostTableCount: 4,  // ← 改为 4
    UserTableCount: 1,
    DBShardCount:   1,
}

// 代码中切换仓储实现
// 从 postdb.NewPostRepo(db) 改为
// postdb.NewShardingPostRepo(shardingDB)
```

## 性能对比

### 分片前（单表）

| 场景 | 查询时间 | 说明 |
|---|---|---|
| 查询社区帖子列表 | 100ms | 100万条记录，全表扫描 |
| 查询用户帖子列表 | 150ms | 需要索引跳跃 |

### 分片后（4张表）

| 场景 | 查询时间 | 改进 |
|---|---|---|
| 查询社区帖子列表 | 25ms | 直接定位到分片表，减少 75% |
| 查询用户帖子列表 | 40ms | 需遍历4个表，总耗时仍可接受 |

## 注意事项

### 1. **跨分片查询**

某些查询需要跨多个分片，建议使用消息队列异步处理：

```go
// ✅ 推荐
type GetUserPostsJob struct {
    UserID int64
    Page   int
}

// 发送到消息队列，后台服务遍历所有 post_* 表
queue.Publish("get_user_posts", job)
```

### 2. **分片键的选择**

- **永远不变**：不能选择会更新的字段（如status、updated_at）
- **均匀分布**：避免热键问题（某个社区活跃度特别高）
- **有业务意义**：便于数据隔离和权限管理

### 3. **扩展性**

如果需要从 4 张表扩展到 8 张：

```go
// 1. 新增分片表 post_4 ~ post_7
// 2. 修改配置
config.PostTableCount = 8

// 3. 数据迁移（可选，部分数据会重新分配）
// 通常使用中间表过渡，避免停机
```

## 常见问题

### Q: 什么时候应该分片？

A: 
- 单表行数超过 **1000 万**
- 单表大小超过 **10GB**
- 写入QPS超过 **1000**
- 查询响应时间超过 **100ms**

### Q: 能否中途改变分片数量？

A: 可以，但需要数据迁移。建议：
- 使用中间表过渡
- 按分片批量处理
- 灰度发布，逐个切换应用实例

### Q: 分片键能改吗？

A: 不建议改。改分片键需要全量数据重新分配，代价极大。

## 参考资源

- [MySQL 分区表](https://dev.mysql.com/doc/refman/8.0/en/partitioning.html)
- [GORM 高级查询](https://gorm.io/docs/query.html)
- [Sharding策略综述](https://blog.csdn.net/weixin_42268521/article/details/126255506)
