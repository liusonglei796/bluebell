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
│   └── infrastructure/
│       └── snowflake/
│           └── snowflake.go  # Snowflake 封装
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

#### 文件: `internal/infrastructure/snowflake/snowflake.go`

```go
package snowflake

import (
	"bluebell/internal/config"
	"bluebell/pkg/errorx"
	"sync"
	"time"

	sf "github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
)

var (
	node          *sf.Node
	mu            sync.Mutex
	lastTimestamp int64 // 上次生成 ID 的毫秒时间戳
)

// Init 初始化雪花算法节点
func Init(cfg *config.Config) (err error) {
	sf.Epoch = cfg.Snowflake.StartTime.UnixNano() / 1000000
	node, err = sf.NewNode(cfg.Snowflake.MachineID)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "初始化雪花算法节点失败, machineID=%d", cfg.Snowflake.MachineID)
	}
	return
}

// GenID 生成 ID（带时钟回拨保护）
func GenID() int64 {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now().UnixMilli()

	// 核心防御逻辑：检查当前时间是否小于上次生成 ID 的时间
	if now < lastTimestamp {
		offset := lastTimestamp - now
		zap.L().Warn("clock backwards detected", zap.Int64("offset_ms", offset))

		// 策略 1：如果是小范围回拨（如 10ms 内），尝试休眠等待时钟追赶
		if offset <= 10 {
			time.Sleep(time.Duration(offset+1) * time.Millisecond)
			now = time.Now().UnixMilli()
		}

		// 策略 2：如果休眠后依然回拨，或者回拨幅度过大，直接报错/抛出异常
		if now < lastTimestamp {
			zap.L().Fatal("clock moved backwards too far, cannot generate ID",
				zap.Int64("last_timestamp", lastTimestamp),
				zap.Int64("current_now", now))
		}
	}

	lastTimestamp = now
	return node.Generate().Int64()
}
```

---

## 5. 进阶课题：应对时钟回拨 (Clock Backwards)

Snowflake 算法强依赖系统时间。在 Docker 或 K8s 容器化部署中，如果宿主机发生 NTP 时间校准，时钟可能会发生回退，导致生成的 ID 重复。

我们在代码中通过以下方式应对：

1. **自旋等待 (Spin Wait)**：对于 10ms 以内的微小回拨，让程序 `time.Sleep` 几毫秒，等时钟追赶上来。
2. **强制熔断 (Fail Fast)**：对于超过 10ms 的大范围回拨，为了绝对保证数据一致性，直接通过 `zap.L().Fatal` 拒绝服务。

> **面试 Tip**: “我的策略是：宁可暂时不可用，也绝不生成重复 ID。”

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
	"bluebell/internal/infrastructure/snowflake"
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

    // 5. 初始化 Snowflake (Viper 会自动将 RFC3339 格式的字符串解析为 time.Time)
    if err := snowflake.Init(cfg); err != nil {
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
	"bluebell/internal/infrastructure/snowflake"
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
