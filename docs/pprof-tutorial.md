# go-wrk + pprof 性能调优演示

## 环境说明

- 服务端口: 8080
- pprof 端口: 8080/debug/pprof/ (需要 debug 模式)

---

## 步骤1：启动服务（debug模式）

```bash
# 方式1：本地运行
cd bluebell
go run ./cmd/bluebell/main.go -conf ./config.local.toml

# 方式2：Docker运行（需先改为debug模式）
docker compose up -d --build
```

---

## 步骤2：压测工具

### 方式A：使用 wrk（推荐）

```bash
# 安装 wrk
# macOS
brew install wrk

# 基础压测
wrk -t4 -c100 -d30s http://127.0.0.1:8080/api/v1/community

# 参数说明：
# -t4   4个线程
# -c100 100个并发连接
# -d30s 持续30秒
```

### 方式B：使用 Go 压测工具

项目已自带：`cmd/bench/main.go`

```bash
cd bluebell
go run ./cmd/bench/main.go
```

---

## 步骤3：CPU Profile 分析

### 同时进行压测和采集

```bash
# 终端1：启动压测（后台运行）
wrk -t4 -c100 -d60s http://127.0.0.1:8080/api/v1/community &

# 终端2：采集 CPU profile（30秒）
go tool pprof -http=:9090 http://127.0.0.1:8080/debug/pprof/profile?seconds=30

# 浏览器访问 http://localhost:9090 查看火焰图
```

### 命令行分析

```bash
# 下载 profile
curl -o cpu.prof "http://127.0.0.1:8080/debug/pprof/profile?seconds=30"

# 命令行分析
go tool pprof cpu.prof

# 常用命令：
(pprof) top              # Top 20 热点函数
(pprof) top10            # Top 10
(pprof) list 函数名      # 查看具体函数源码
(pprof) web             # 生成调用图（需安装 GraphViz）
(pprof) pdf             # 生成 PDF 报告
```

---

## 步骤4：内存分析

```bash
# 查看内存堆
curl -o heap.prof "http://127.0.0.1:8080/debug/pprof/heap"

# 分析
go tool pprof -http=:9091 heap.prof
```

---

## 步骤5：Goroutine 分析

```bash
# 查看协程堆栈
curl "http://127.0.0.1:8080/debug/pprof/goroutine?debug=1"

# 或下载分析
curl -o goroutine.prof "http://127.0.0.1:8080/debug/pprof/goroutine"
go tool pprof goroutine.prof
```

---

## pprof 常用端点

| 端点 | 用途 |
|------|------|
| `/debug/pprof/profile` | CPU 采样（默认30秒） |
| `/debug/pprof/heap` | 内存堆分配 |
| `/debug/pprof/goroutine` | 协程堆栈 |
| `/debug/pprof/block` | 阻塞操作 |
| `/debug/pprof/mutex` | 锁竞争 |
| `/debug/pprof/trace` | 追踪（GC等） |

---

## 结果解读示例

```
Showing top 20 nodes out of 146
      flat  flat%   sum%        cum   cum%
     3.72s 80.69% 80.69%      3.73s 80.91%  runtime.cgocall
     0.42s  9.11% 89.22%      0.42s  9.11%  communitydb.GetCommunityList
```

- **flat**: 函数自身执行时间
- **cum**: 函数及其调用子函数的总时间
- **80.69%**: CPU 热点在 MySQL CGO 调用

---

## 常见性能问题

| 问题 | 表现 | 解决方案 |
|------|------|----------|
| CPU 高 | runtime.cgocall 高 | 数据库查询多，加缓存 |
| 内存高 | heap 持续增长 | 内存泄漏，检查对象未释放 |
| 协程阻塞 | goroutine 数量暴增 | 检查 channel、锁、IO |
| 锁竞争 | mutex 耗时高 | 减少共享变量，改用无锁 |
