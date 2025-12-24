# Bluebell 项目压力测试指南 (使用 go-wrk)

本文档旨在指导开发人员使用 [go-wrk](https://github.com/adjust/go-wrk) 对 Bluebell 社区论坛后端服务进行压力测试，以评估系统在高并发情况下的性能表现、识别瓶颈并验证系统稳定性。

## 1. 环境准备

### 1.1 安装 go-wrk

`go-wrk` 是一个轻量级的 HTTP 基准测试工具，使用 Go 编写，易于安装和使用。

```bash
go install github.com/adjust/go-wrk@latest
```

安装完成后，确保 `$GOPATH/bin` 在你的系统 `PATH` 环境变量中，通过以下命令验证：

```bash
go-wrk --help
```

### 1.2 调整应用配置 (至关重要)

在进行压力测试之前，**必须**调整 `config.yaml` 配置以获得真实的性能数据。Debug 模式下的日志输出会严重拖慢系统吞吐量。

修改 `config.yaml`:

```yaml
app:
  mode: "release" # 将 "dev" 改为 "release"

log:
  level: "error" # 将 "debug" 改为 "error" 或 "warn"，减少磁盘 I/O
```

### 1.3 启动服务

建议编译后运行二进制文件，而不是直接使用 `go run`：

```bash
make build
./bluebell
```

---

## 2. 获取测试凭证 (JWT Token)

由于 Bluebell 的核心接口（如社区、帖子、投票）都需要鉴权，压测前必须先获取一个有效的 JWT Token。

**步骤：**

1.  启动服务。
2.  注册或登录一个用户（确保数据库中存在该用户）。
3.  复制响应中的 `token`。

**示例 Curl 命令:** 

```bash
# 1. 注册 (如果用户不存在)
curl -X POST http://127.0.0.1:8080/api/v1/signup \
  -H "Content-Type: application/json" \
  -d '{ "username": "stress_user", "password": "password123", "re_password": "password123" }'

# 2. 登录获取 Token
curl -X POST http://127.0.0.1:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{ "username": "stress_user", "password": "password123" }'
```

*记下返回 JSON 中的 `token` 字段值，后续测试中将作为 `Authorization` 头使用。*

---

## 3. 压测场景与命令

以下场景假设服务运行在 `http://127.0.0.1:8080`，请将 `<YOUR_TOKEN>` 替换为实际获取的 Token。

**参数说明:** 
- `-c`: 并发连接数 (Connections)
- `-t`: 线程数 (Threads) [go-wrk 不需要像 wrk 那样关注线程]
- `-n`: 总请求数 (Requests)
- `-d`: 测试持续时间 (Duration)
- `-H`: HTTP Header
- `-M`: HTTP Method
- `-b`: HTTP Body

### 场景一：受保护的读接口 (轻量级)

**目标接口:** 获取社区列表 (`GET /api/v1/community`)
**测试目的:** 评估 Gin 框架路由 + JWT 中间件 + MySQL 简单查询的基准性能。

```bash
# 模拟 100 个并发连接，持续 10 秒
go-wrk -c 100 -d 10s -H "Authorization: Bearer <YOUR_TOKEN>" http://127.0.0.1:8080/api/v1/community
```

### 场景二：受保护的读接口 (复杂逻辑/缓存)

**目标接口:** 获取帖子列表 (`GET /api/v1/posts`)
**测试目的:** 评估 Redis (ZSet) + MySQL (批量查询) 的混合性能。这是系统中最核心的高频接口。
**参数:** `page=1`, `size=10`, `order=score` (按分数排序)

```bash
# 模拟 200 个并发连接，持续 20 秒
go-wrk -c 200 -d 20s -H "Authorization: Bearer <YOUR_TOKEN>" "http://127.0.0.1:8080/api/v1/posts?page=1&size=10&order=score"
```

### 场景三：计算密集型接口 (写入/计算)

**目标接口:** 用户登录 (`POST /api/v1/login`)
**测试目的:** 评估 bcrypt 密码校验对 CPU 的消耗。
**注意:** 此接口不需要 Token，但消耗大量 CPU 资源。

```bash
# 模拟 50 个并发 (bcrypt 计算很慢，并发过高会导致超时)
go-wrk -c 50 -d 10s -M POST -b '{"username":"stress_user","password":"password123"}' -H "Content-Type: application/json" http://127.0.0.1:8080/api/v1/login
```

### 场景四：写入接口 (数据库写入)

**目标接口:** 发布帖子 (`POST /api/v1/post`)
**测试目的:** 评估 MySQL 写入性能及 Snowflake ID 生成性能。
**注意:** 确保 `community_id` 存在（例如 id 为 1 的社区）。

```bash
# 模拟 100 个并发，持续 10 秒
go-wrk -c 100 -d 10s -M POST -b '{"title":"Stress Test","content":"This is a test post","community_id":1}' -H "Content-Type: application/json" -H "Authorization: Bearer <YOUR_TOKEN>" http://127.0.0.1:8080/api/v1/post
```

### 场景五：混合写入接口 (Redis + MySQL)

**目标接口:** 帖子投票 (`POST /api/v1/vote`)
**测试目的:** 评估 Redis 事务/Pipeline 性能及逻辑校验开销。
**注意:** 这里的 `post_id` 需要替换为实际存在的帖子 ID。重复对同一帖子投相同票可能会被业务逻辑拦截（返回错误或幂等成功），这取决于具体实现，但依然能测试系统负载。

```bash
# 对 ID 为 1 的帖子投赞成票 (direction=1)
go-wrk -c 100 -d 10s -M POST -b '{"post_id":1,"direction":1}' -H "Content-Type: application/json" -H "Authorization: Bearer <YOUR_TOKEN>" http://127.0.0.1:8080/api/v1/vote
```

---

## 4. 结果分析指南

`go-wrk` 执行完成后会输出如下指标，重点关注以下几项：

```text
Running 10s test @ http://127.0.0.1:8080/api/v1/community
  100 goroutine(s) running concurrently
   
  14523 requests in 9.9542125s, 6.54MB read
  Requests/sec:		1458.98  <-- 核心指标：RPS (每秒请求数/QPS)
  Transfer/sec:		672.54KB
  Avg Req Time:		68.54ms  <-- 平均延迟
  Fastest Request:	2.12ms
  Slowest Request:	543.12ms
  Number of Errors:	0        <-- 必须为 0，否则系统已过载
```

**判定标准:** 

1.  **RPS (Requests/sec)**: 
    - 简单读接口 (Scenario 1) 预期应在 5000+ (取决于机器配置)。
    - 复杂接口 (Scenario 2) 预期应在 2000+。
    - 登录接口 (Scenario 3) 通常较低 (几百)，受限于 bcrypt 成本。
    - 写入接口 (Scenario 4 & 5) 通常低于读接口，受限于数据库 I/O。

2.  **Latency (Avg Req Time)**: 
    - 理想情况下应保持在 100ms 以内。
    - 如果超过 500ms，说明用户体验已有明显卡顿。

3.  **Errors**: 
    - 如果出现 Connection Refused 或 Timeout，说明并发数 `-c` 设置过高，或系统文件句柄限制 (`ulimit`) 已达到瓶颈。

---

## 5. 性能调优建议

如果压测结果不理想，可以尝试以下优化方向：

1.  **数据库连接池 (`config.yaml`)**: 
    - 增加 `max_open_conns`。如果压测时出现 DB 连接等待错误，适当调大此值（如 200）。
    
2.  **Redis 连接池**: 
    - 增加 Redis `pool_size`。

3.  **系统级限制 (Linux)**: 
    - 检查文件句柄限制: `ulimit -n`。
    - 临时提升限制: `ulimit -n 65535`。

4.  **代码级优化**: 
    - 检查是否有 N+1 查询查询遗漏。
    - 确保热点数据已正确命中 Redis 缓存。
    - 使用 pprof 分析 CPU 和 内存 瓶颈: 
      ```go
      // 在 main.go 中引入
      import _ "net/http/pprof"
      
      // 启动一个独立的 pprof server
      go func() {
          http.ListenAndServe(":6060", nil)
      }()
      ```