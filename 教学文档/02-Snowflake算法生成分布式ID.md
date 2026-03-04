# 第02章: Snowflake 算法生成分布式 ID

> **学习目标**: 理解为什么需要分布式 ID、掌握 Snowflake 算法原理、在 Go 项目中集成雪花算法。

---

## 📚 本章导读

在上一章,我们设计了用户表,其中有两个 ID 字段:

```sql
`id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID'
`user_id` BIGINT(20) NOT NULL COMMENT '用户ID (业务ID)'
```

**问题来了:** 为什么不直接用 `AUTO_INCREMENT` 的 `id` 作为业务 ID?

本章将深入探讨:
1. 自增 ID 的安全隐患
2. 分布式系统中 ID 生成的挑战
3. Snowflake 算法的设计原理
4. 在 Bluebell 项目中集成雪花算法

---

## 1. 自增 ID 的问题

### 1.1 安全问题: 可遍历性

**场景:** 假设我们直接暴露自增 ID:

```
# 用户注册成功,返回用户信息
POST /api/v1/signup
{
  "id": 1,
  "username": "alice"
}

# 获取用户资料
GET /api/v1/users/1  ✅ alice 的资料
GET /api/v1/users/2  ✅ bob 的资料
GET /api/v1/users/3  ✅ charlie 的资料
```

**风险:**
- ❌ 可以轻松获取所有用户信息
- ❌ 可以统计用户总量
- ❌ 可以分析用户增长趋势
- ❌ 可以针对特定用户进行攻击

### 1.2 信息泄露

```
今天注册的用户 ID: 10523
昨天注册的用户 ID: 10412

得出结论: 昨天新增用户 = 10523 - 10412 = 111 人
```

**竞争对手可以:**
- 分析你的用户增长速度
- 评估业务健康度

### 1.3 分布式系统的挑战

**单机系统:**
```
数据库A: 自增ID 1, 2, 3, 4, 5...  ✅ 没问题
```

**分布式系统:**
```
数据库A: 自增ID 1, 2, 3...
数据库B: 自增ID 1, 2, 3...  ❌ ID 冲突!
```

---

## 2. 分布式 ID 的要求

一个优秀的分布式 ID 生成方案应该满足:

| 要求 | 说明 |
|------|------|
| **全局唯一** | 不同机器生成的 ID 不会重复 |
| **趋势递增** | 新的 ID 比旧的大(有利于数据库索引) |
| **高性能** | 生成 ID 速度快,不成为瓶颈 |
| **高可用** | 服务要稳定,不能因为 ID 生成失败而停服 |
| **安全性** | 不可预测,不暴露业务信息 |

---

## 3. Snowflake 算法原理

### 3.1 ID 结构解析

Snowflake 生成的是一个 **64 位的 int64** 整数,结构如下:

```
┌─────────────────────────────────────────────────────────────────┐
│ 0 │     41 位时间戳      │ 10 位机器ID │     12 位序列号     │
├───┼─────────────────────┼─────────────┼─────────────────────┤
│ 1 │ 42 ................│ 10 ......... │ 12 ..................│
│bit│ timestamp (毫秒)   │ machine ID  │ sequence number     │
└───┴─────────────────────┴─────────────┴─────────────────────┘
```

---

## 4. 在 Bluebell 项目中集成

### 4.1 项目结构

```
bluebell/
├── internal/
│   └── snowflake/
│       └── snowflake.go  # Snowflake 封装
├── internal/config/
│   └── config.go         # 配置管理
├── config.yaml           # 配置文件
└── cmd/bluebell/main.go  # 初始化入口
```

### 4.2 安装依赖

```bash
go get github.com/bwmarrin/snowflake
```

### 4.3 实现代码

#### 文件: `internal/snowflake/snowflake.go`

```go
package snowflake

import (
	"bluebell/pkg/errorx"
	"time"

	sf "github.com/bwmarrin/snowflake"
)

var node *sf.Node

// Init 初始化雪花算法节点
func Init(startTime time.Time, machineID int64) (err error) {
	sf.Epoch = startTime.UnixNano() / 1000000
	node, err = sf.NewNode(machineID)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "初始化雪花算法节点失败, machineID=%d", machineID)
	}
	return
}

// GetID 生成 ID
func GetID() int64 {
	return node.Generate().Int64()
}

// GenID 生成 ID 的别名函数
func GenID() int64 {
	return GetID()
}
```

### 4.4 配置文件

#### 文件: `config.yaml`

```yaml
snowflake:
  start_time: "2024-01-01"  # 起始时间 (Epoch)
  machine_id: 1             # 机器ID (0-1023)
```

### 4.5 在 main.go 中初始化

#### 文件: `cmd/bluebell/main.go`

```go
package main

import (
	"bluebell/internal/config"
	"bluebell/internal/snowflake"
	"time"
	"go.uber.org/zap"
)

func main() {
    // 1. 加载配置
    cfg, err := config.Init("./config.yaml")
    if err != nil {
        panic(err)
    }

    // ...

    // 5. 初始化 Snowflake
    startTime, _ := time.Parse("2006-01-02", cfg.Snowflake.StartTime)
    if err := snowflake.Init(startTime, cfg.Snowflake.MachineID); err != nil {
        zap.L().Fatal("初始化雪花算法失败", zap.Error(err))
    }
}
```

---

## 5. 使用 Snowflake 生成用户 ID

### 5.1 在注册逻辑中使用

#### 文件: `internal/service/user/user_service.go`

```go
package user

import (
	"bluebell/internal/snowflake"
	"bluebell/internal/model"
	"context"
)

// SignUp 用户注册业务逻辑
func (s *UserService) SignUp(ctx context.Context, p *request.SignUpRequest) (err error) {
	// 1. 生成 UID (使用雪花算法)
	userID := snowflake.GenID()

	u := &model.User{
		UserID:   userID,
		UserName: p.Username,
		Passwd:   p.Password,
	}

	return s.userRepo.InsertUser(ctx, u)
}
```

---

**下一章:** [第03章: 用户注册业务流程设计](./03-用户注册业务流程设计.md)

**返回目录:** [README.md](./README.md)
