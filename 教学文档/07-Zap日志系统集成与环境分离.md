# 第07章:Zap日志系统集成与环境分离

> **本章导读**
>
> 在前面的章节中,我们实现了用户注册功能,但一直使用 `fmt.Println` 进行调试。这在开发阶段没问题,但在生产环境中,我们需要一个专业的日志系统来记录应用运行状态、排查问题、监控性能。
>
> Bluebell 项目选择了 Uber 开源的 **Zap** 库,它是 Go 语言中性能最高的日志库之一。本章将详细讲解如何集成 Zap,实现日志的**环境隔离**(开发环境和生产环境使用不同的日志策略),以及如何使用 **Lumberjack** 实现日志自动切割。

---

## 📚 本章目标

学完本章,你将掌握:

1. 理解为什么 `fmt.Println` 不适合生产环境
2. 掌握 Zap 日志库的核心概念 (Logger, Core, Encoder, WriteSyncer)
3. 实现日志的**环境隔离** (dev vs release)
4. 使用 Lumberjack 实现日志的自动切割与归档
5. 编写自定义 Gin 中间件,接管 HTTP 请求日志
6. 实现 Panic 恢复机制,防止服务崩溃

---

## 1. 为什么需要专业日志系统?

### 1.1 fmt.Println 的问题

```go
// ❌ 开发阶段常见的做法
func SignUp(p *models.ParamSignUp) error {
    fmt.Println("用户注册:", p.Username)

    if err := mysql.CheckUserExist(p.Username); err != nil {
        fmt.Println("检查用户失败:", err)
        return err
    }

    fmt.Println("注册成功")
    return nil
}
```

**问题分析:**

| 问题 | 说明 | 影响 |
|------|------|------|
| **无日志级别** | 无法区分 Debug/Info/Error | 生产环境全是噪音 |
| **无结构化** | 纯文本,难以解析 | 日志收集系统无法使用 |
| **无文件输出** | 只输出到控制台 | 服务重启后日志丢失 |
| **无时间戳** | 不知道何时发生 | 无法追溯问题 |
| **无调用位置** | 不知道哪个文件哪一行 | 排查困难 |
| **性能差** | fmt 包使用反射 | 高并发下性能瓶颈 |

### 1.2 生产环境日志需求

**必须满足的需求:**

1. **结构化输出** (JSON 格式,便于日志收集系统解析)
2. **日志分级** (Debug/Info/Warn/Error/Fatal)
3. **文件输出** (持久化存储,服务重启不丢失)
4. **日志切割** (防止单个文件过大)
5. **高性能** (零内存分配,不影响业务性能)
6. **调用栈信息** (记录文件名、行号、函数名)

### 1.3 常见日志库对比

| 特性 | 标准库 log | Logrus | Zap | Zerolog |
|------|-----------|--------|-----|---------|
| **性能** | 一般 | 慢 | **极快** | 极快 |
| **结构化** | ❌ | ✅ | ✅ | ✅ |
| **零分配** | ❌ | ❌ | **✅** | ✅ |
| **类型安全** | ❌ | ❌ | **✅** | ✅ |
| **社区** | 官方 | 大 | **大** | 中等 |
| **学习曲线** | 平缓 | 平缓 | 陡峭 | 陡峭 |

**Bluebell 选择 Zap 的理由:**

1. **性能最优**: 在高并发场景下,Zap 比 Logrus 快 4-10 倍
2. **类型安全**: 使用强类型字段 (`zap.String`, `zap.Int`),避免反射
3. **零内存分配**: 核心路径无内存分配,减少 GC 压力
4. **成熟稳定**: Uber 内部大规模使用,社区活跃

**性能基准测试 (官方数据):**

```
BenchmarkZap-8           10000000      118 ns/op       0 B/op        0 allocs/op
BenchmarkLogrus-8         3000000      542 ns/op     531 B/op       11 allocs/op
BenchmarkStdLog-8         5000000      295 ns/op      80 B/op        2 allocs/op
```

---

## 2. Zap 核心概念

### 2.1 Zap 的架构

```
┌─────────────────────────────────────────────────────────┐
│                    zap.Logger                           │
│  ┌────────────────────────────────────────────────┐    │
│  │           zapcore.Core (核心)                   │    │
│  │                                                 │    │
│  │  ┌─────────────┐  ┌──────────────┐  ┌────────┐│    │
│  │  │  Encoder    │  │ WriteSyncer  │  │ Level  ││    │
│  │  │(编码器)      │  │(写入器)       │  │(级别)  ││    │
│  │  │             │  │              │  │        ││    │
│  │  │- JSON       │  │- 文件        │  │- Debug ││    │
│  │  │- Console    │  │- 控制台      │  │- Info  ││    │
│  │  │             │  │- 网络        │  │- Warn  ││    │
│  │  │             │  │              │  │- Error ││    │
│  │  └─────────────┘  └──────────────┘  └────────┘│    │
│  └────────────────────────────────────────────────┘    │
│                                                         │
│  Options: AddCaller, AddStacktrace, AddCallerSkip...   │
└─────────────────────────────────────────────────────────┘
```

### 2.2 核心组件详解

#### 2.2.1 Encoder (编码器)

负责将日志条目格式化为特定格式。

**JSON Encoder** (生产环境推荐):

```json
{
  "level": "info",
  "time": "2025-12-08T17:30:00.123Z",
  "caller": "logic/user.go:42",
  "msg": "用户注册成功",
  "username": "zhangsan",
  "user_id": 239482394823948
}
```

**Console Encoder** (开发环境推荐):

```
2025-12-08T17:30:00.123Z  INFO  logic/user.go:42  用户注册成功  {"username": "zhangsan", "user_id": 239482394823948}
```

**对比:**

| Encoder | 优点 | 缺点 | 适用场景 |
|---------|------|------|---------|
| **JSON** | 易于机器解析,支持 ELK 等日志系统 | 人类阅读困难 | **生产环境** |
| **Console** | 人类友好,调试方便 | 难以解析,不规范 | **开发环境** |

#### 2.2.2 WriteSyncer (写入器)

负责将日志写入到目标位置。

```go
// 写入文件
zapcore.AddSync(&lumberjack.Logger{Filename: "app.log"})

// 写入控制台
zapcore.Lock(os.Stdout)

// 同时写入多个目标 (使用 Tee)
zapcore.NewTee(fileCore, consoleCore)
```

#### 2.2.3 Level (日志级别)

| 级别 | 数值 | 用途 | 示例 |
|------|------|------|------|
| **Debug** | -1 | 开发调试信息 | SQL 查询语句、函数参数 |
| **Info** | 0 | 重要业务事件 | 用户登录、订单创建 |
| **Warn** | 1 | 警告但不影响运行 | Redis 连接慢、API 调用超时 |
| **Error** | 2 | 错误但可恢复 | 数据库查询失败、第三方 API 错误 |
| **DPanic** | 3 | 开发环境 panic,生产环境 error | 不应该发生的情况 |
| **Panic** | 4 | 记录日志后 panic | 致命错误,无法继续运行 |
| **Fatal** | 5 | 记录日志后退出程序 | 配置文件加载失败 |

**级别过滤:**

```go
// 设置级别为 Info
// Debug 级别的日志不会输出,只输出 Info 及以上级别
core := zapcore.NewCore(encoder, ws, zapcore.InfoLevel)
```

#### 2.2.4 Core (核心)

Core 是 Encoder、WriteSyncer 和 Level 的组合。

```go
// 创建一个 Core
core := zapcore.NewCore(
    encoder,      // 如何编码
    writeSyncer,  // 写到哪里
    level,        // 什么级别
)

// 使用 Tee 合并多个 Core (类似 Unix 的 tee 命令)
core := zapcore.NewTee(
    fileCore,     // 写到文件
    consoleCore,  // 写到控制台
)
```

---

## 3. 环境隔离设计

### 3.1 为什么需要环境隔离?

**开发环境 (dev) 需求:**
- 控制台输出 (实时查看)
- Console 格式 (人类友好)
- Debug 级别 (详细信息)
- 同时输出到文件 (方便回溯)

**生产环境 (release) 需求:**
- 只输出到文件 (控制台无人看)
- JSON 格式 (日志收集系统)
- Info 级别 (减少噪音)
- 日志切割 (防止磁盘占满)

### 3.2 配置文件设计

```yaml
# config.yaml

app:
  mode: "dev"  # dev 或 release
  port: 8080

log:
  level: "debug"           # 日志级别
  file_name: "bluebell.log"  # 日志文件名
  max_size: 100            # 单个文件最大 MB
  max_backups: 7           # 保留旧文件个数
  max_age: 30              # 保留旧文件天数
```

**配置结构体:**

```go
// settings/settings.go

type AppConfig struct {
    Name    string `mapstructure:"name"`
    Mode    string `mapstructure:"mode"`    // dev 或 release
    Version string `mapstructure:"version"`
    Port    int    `mapstructure:"port"`
}

type LogConfig struct {
    Level      string `mapstructure:"level"`
    FileName   string `mapstructure:"file_name"`
    MaxSize    int    `mapstructure:"max_size"`
    MaxBackups int    `mapstructure:"max_backups"`
    MaxAge     int    `mapstructure:"max_age"`
}
```

---

## 4. Logger 初始化实现

### 4.1 完整的 Init 函数

```go
// logger/logger.go

package logger

import (
    "fmt"
    "os"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "gopkg.in/natefinch/lumberjack.v2"

    "bluebell/settings"
)

// Init 初始化 Logger
// 为什么: 日志组件需要根据配置(如文件路径、级别)进行初始化,才能正确输出日志
func Init(cfg *settings.LogConfig, mode string) (err error) {
    // 1️⃣ 参数校验
    if cfg == nil {
        return fmt.Errorf("logger.Init received nil config")
    }

    // 2️⃣ 获取日志写入器 (支持日志切割)
    // 为什么: Lumberjack 实现了自动日志轮转,防止单个日志文件过大
    writeSyncer := getLogWriter(
        cfg.FileName,
        cfg.MaxSize,
        cfg.MaxBackups,
        cfg.MaxAge,
    )

    // 3️⃣ 获取日志编码器 (JSON 格式)
    // 为什么: JSON 格式易于解析,适合日志收集系统
    encoder := getEncoder()

    // 4️⃣ 解析日志级别
    // 为什么: 配置文件中是字符串 "debug",需要转换为 zapcore.Level 类型
    var level zapcore.Level
    if err = level.UnmarshalText([]byte(cfg.Level)); err != nil {
        return fmt.Errorf("parse log level failed: %w", err)
    }

    // 5️⃣ 根据模式创建 Core
    var core zapcore.Core

    if mode == "dev" || mode == gin.DebugMode {
        // 🔥 开发模式: 双重输出 (文件 + 控制台)

        // 控制台编码器 (Console 格式,人类可读)
        consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

        // 创建两个 Core
        // Core 1: 输出到文件 (JSON 格式)
        fileCore := zapcore.NewCore(encoder, writeSyncer, level)

        // Core 2: 输出到控制台 (Console 格式)
        // zapcore.Lock(os.Stdout): 线程安全的标准输出
        // zapcore.DebugLevel: 控制台输出所有级别的日志
        consoleCore := zapcore.NewCore(
            consoleEncoder,
            zapcore.Lock(os.Stdout),
            zapcore.DebugLevel,
        )

        // 使用 NewTee 合并两个 Core
        // Tee: 分流器,日志会同时写入两个目标
        core = zapcore.NewTee(fileCore, consoleCore)

    } else {
        // 🔥 生产模式: 只输出到文件
        // 为什么: 生产环境控制台输出无人查看,且影响性能
        core = zapcore.NewCore(encoder, writeSyncer, level)
    }

    // 6️⃣ 创建 Logger 实例
    // zap.AddCaller(): 在日志中添加调用者的文件名和行号
    // 为什么: 方便定位代码位置,排查问题
    lg := zap.New(core, zap.AddCaller())

    // 7️⃣ 替换全局 Logger
    // 为什么: 替换后可以在任何地方使用 zap.L() 调用,无需传递 Logger 实例
    zap.ReplaceGlobals(lg)

    return nil
}
```

### 4.2 日志切割实现

```go
// logger/logger.go

// getLogWriter 获取日志写入器
// 为什么: 使用 lumberjack 库实现日志切割(Log Rotation),防止单个日志文件过大占满磁盘
func getLogWriter(filename string, maxSize int, maxBackups int, maxAge int) zapcore.WriteSyncer {
    lumberjackLogger := &lumberjack.Logger{
        Filename:   filename,   // 日志文件路径
        MaxSize:    maxSize,    // 单个日志文件最大大小(MB)
        MaxBackups: maxBackups, // 保留旧日志文件的最大个数
        MaxAge:     maxAge,     // 保留旧日志文件的最大天数
        Compress:   false,      // 是否压缩 (gzip)
    }

    // AddSync 将 lumberjack.Logger 包装为 zapcore.WriteSyncer
    return zapcore.AddSync(lumberjackLogger)
}
```

**Lumberjack 切割规则:**

```
# 假设配置:
# MaxSize: 100MB
# MaxBackups: 7
# MaxAge: 30天

bluebell.log           # 当前日志文件
bluebell-2025-12-08T17-30-00.log  # 自动归档的旧日志
bluebell-2025-12-07T10-15-30.log
bluebell-2025-12-06T14-20-45.log
...

# 切割触发条件 (满足任一条件即切割):
1. 当前文件大小 >= 100MB
2. 日志文件年龄 >= 30天

# 清理规则:
1. 保留最新的 7 个备份文件
2. 删除超过 30 天的备份文件
```

**为什么不用系统工具 (logrotate)?**

| 方案 | 优点 | 缺点 |
|------|------|------|
| **logrotate** | 系统级通用工具 | 需要额外配置,重命名时可能丢失日志 |
| **Lumberjack** | Go 原生,无缝集成 | 仅适用于 Go 应用 |

### 4.3 编码器配置

```go
// logger/logger.go

// getEncoder 获取日志编码器
// 为什么: 配置日志的输出格式,这里使用 JSON 格式,适合机器解析
func getEncoder() zapcore.Encoder {
    // 使用开发环境的默认配置作为基础
    encoderConfig := zap.NewDevelopmentEncoderConfig()

    // 自定义配置
    encoderConfig.TimeKey = "time"  // 时间字段的 key
    encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder  // 时间格式: 2025-12-08T17:30:00.123Z
    encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder  // 级别大写: INFO, ERROR
    encoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder  // 时间间隔单位: 秒
    encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder  // 调用者格式: logger/logger.go:42

    // 返回 JSON 编码器
    return zapcore.NewJSONEncoder(encoderConfig)
}
```

**EncoderConfig 配置项详解:**

| 配置项 | 说明 | 可选值 | 示例 |
|--------|------|--------|------|
| **TimeKey** | 时间字段名 | "time", "ts", "@timestamp" | "time": "2025-12-08T17:30:00.123Z" |
| **LevelKey** | 级别字段名 | "level", "lvl", "severity" | "level": "INFO" |
| **MessageKey** | 消息字段名 | "msg", "message" | "msg": "用户注册成功" |
| **CallerKey** | 调用者字段名 | "caller", "source" | "caller": "logic/user.go:42" |
| **EncodeTime** | 时间编码格式 | ISO8601, RFC3339, EpochMillis | ISO8601: "2025-12-08T17:30:00.123Z" |
| **EncodeLevel** | 级别编码格式 | Capital, Lower, Color | Capital: "INFO", Lower: "info" |
| **EncodeCaller** | 调用者编码格式 | Full, Short | Short: "logger/logger.go:42" |

---

## 5. Gin 中间件集成

### 5.1 为什么需要自定义中间件?

**Gin 默认的 Logger 中间件问题:**

```go
// Gin 默认日志输出
[GIN] 2025/12/08 - 17:30:00 | 200 |  120.345ms |  192.168.1.100 | POST     "/api/v1/signup"
```

**问题:**
- ❌ 输出到标准输出,无法持久化
- ❌ 格式固定,无法自定义
- ❌ 不支持结构化日志
- ❌ 无法与 Zap 集成

### 5.2 GinLogger 中间件实现

```go
// logger/logger.go

// GinLogger 是一个中间件构造函数,返回 gin.HandlerFunc 类型
// 为什么: Gin 默认的 Logger 中间件输出格式固定,无法直接对接 zap
//        我们需要自定义中间件将 Gin 的请求日志通过 zap 输出
func GinLogger() gin.HandlerFunc {
    // 返回一个匿名函数,这个函数是实际处理请求的逻辑
    return func(c *gin.Context) {
        // 1️⃣ 【请求前逻辑】
        // 记录请求进入的时间点,用于后续计算耗时
        start := time.Now()

        // 获取请求路径 (如 /api/v1/signup)
        path := c.Request.URL.Path

        // 获取查询参数 (如 ?id=1&name=abc)
        query := c.Request.URL.RawQuery

        // 2️⃣ 【核心转折点】
        // c.Next() 表示"放行"
        // 程序会暂停在这里,去执行后续的中间件和具体的 Controller
        // 等到 Controller 处理完并返回响应后,程序会回到这里继续往下执行
        c.Next()

        // 3️⃣ 【请求后逻辑】
        // 此时业务逻辑已经执行完毕,响应数据已经准备好发送给客户端

        // 计算总耗时 (当前时间 - 开始时间)
        cost := time.Since(start)

        // 使用 zap 的全局 Logger 记录一条 Info 级别的日志
        // "http request" 是这条日志的 Message (标题)
        zap.L().Info("http request",
            // 记录 HTTP 状态码 (如 200, 404, 500)
            // 注意: 因为是在 c.Next() 之后,所以能拿到 Controller 设置的状态码
            zap.Int("status", c.Writer.Status()),

            // 记录 HTTP 请求方法 (GET, POST, PUT 等)
            zap.String("method", c.Request.Method),

            // 记录请求路径 (如 /api/v1/login)
            zap.String("path", path),

            // 记录 URL 查询参数 (如 ?id=1&name=abc)
            zap.String("query", query),

            // 记录客户端 IP,Gin 会自动处理 X-Forwarded-For 等头信息
            zap.String("ip", c.ClientIP()),

            // 记录用户代理 (浏览器信息、Postman 等)
            zap.String("user-agent", c.Request.UserAgent()),

            // 记录请求耗时,Zap 会自动格式化时间 (如 120ms)
            zap.Duration("cost", cost),

            // 记录 Gin 上下文中挂载的错误
            // 如果你在 Controller 里调用了 c.Error(err),这里会把它记录下来
            // ErrorTypePrivate 通常是内部错误,不会直接返回给前端,但需要记日志
            zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
        )
    }
}
```

**日志输出示例:**

```json
{
  "level": "info",
  "time": "2025-12-08T17:30:00.123Z",
  "caller": "logger/logger.go:129",
  "msg": "http request",
  "status": 200,
  "method": "POST",
  "path": "/api/v1/signup",
  "query": "",
  "ip": "192.168.1.100",
  "user-agent": "Mozilla/5.0...",
  "cost": 0.120345,
  "errors": ""
}
```

### 5.3 GinRecovery 中间件实现

```go
// logger/logger.go

// GinRecovery 是一个中间件,用于捕获 panic 并恢复
// 为什么: 防止某个请求处理发生 panic 导致整个服务崩溃
//        同时记录 panic 的堆栈信息到日志中
func GinRecovery(stack bool) gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            // recover() 会捕获当前 goroutine 的 panic
            if rec := recover(); rec != nil {

                // 1️⃣ 检查是否是 broken pipe (客户端断开连接)
                // 为什么: 如果是 broken pipe,说明客户端已经断开了
                //        没必要返回 500 错误,只需记录日志
                var brokenPipe bool
                if err, ok := rec.(error); ok {
                    brokenPipe = isBrokenPipeError(err)
                }

                // 2️⃣ 获取请求信息 (用于日志)
                // 为什么: 记录 panic 发生时的请求内容,方便复现和排查
                httpRequest, _ := httputil.DumpRequest(c.Request, false)
                requestStr := string(httpRequest)

                // 3️⃣ 构建日志字段
                // 统一日志字段作用就是"打包证据"
                fields := []zap.Field{
                    zap.Any("error", rec),           // 案发原因 (如 "index out of range")
                    zap.String("request", requestStr),  // 案发现场 (用户发了什么参数)
                }

                // 4️⃣ 处理 broken pipe (只记录,不返回响应)
                if brokenPipe {
                    zap.L().Error("broken pipe",
                        append(fields, zap.String("path", c.Request.URL.Path))...,
                    )
                    c.Error(rec.(error))  // 包装为 error 类型
                    c.Abort()             // 终止后续中间件
                    return
                }

                // 5️⃣ 其他 panic,根据参数决定是否打印堆栈
                // 为什么: 堆栈信息能精确指出代码哪一行出错了
                if stack {
                    // debug.Stack() 获取完整的调用栈
                    fields = append(fields, zap.String("stack", string(debug.Stack())))
                }

                zap.L().Error("[Recovery from panic]", fields...)

                // 返回 500 错误给客户端
                c.AbortWithStatus(http.StatusInternalServerError)
            }
        }()

        // 继续执行后续中间件和 Controller
        c.Next()
    }
}

// isBrokenPipeError 检查错误链中是否包含 broken pipe
// 为什么: 判断是否是网络连接中断导致的错误
func isBrokenPipeError(err error) bool {
    if err == nil {
        return false
    }

    var opErr *net.OpError
    if errors.As(err, &opErr) {
        var syscallErr *os.SyscallError
        if errors.As(opErr.Err, &syscallErr) {
            msg := strings.ToLower(syscallErr.Error())
            return strings.Contains(msg, "broken pipe") ||
                strings.Contains(msg, "connection reset by peer")
        }
    }

    // 兜底检查
    msg := strings.ToLower(err.Error())
    return strings.Contains(msg, "broken pipe") ||
        strings.Contains(msg, "connection reset by peer")
}
```

**Panic 恢复流程:**

```
1. Controller 发生 panic
   ↓
2. defer recover() 捕获
   ↓
3. 判断是否 broken pipe
   ├─ 是 → 记录日志,不返回响应
   └─ 否 → 记录日志+堆栈,返回 500
   ↓
4. 服务继续运行,不崩溃
```

**为什么需要 Panic 恢复?**

```go
// ❌ 没有 Recovery 中间件
func BadHandler(c *gin.Context) {
    arr := []int{1, 2, 3}
    _ = arr[10]  // panic: index out of range
    // 整个服务崩溃,所有请求都无法处理!
}

// ✅ 有 Recovery 中间件
func GoodHandler(c *gin.Context) {
    arr := []int{1, 2, 3}
    _ = arr[10]  // panic
    // Recovery 捕获 panic,记录日志,返回 500
    // 服务继续运行,其他请求不受影响
}
```

---

## 6. 路由注册与使用

### 6.1 在 main.go 中初始化

```go
// main.go

package main

import (
    "fmt"

    "bluebell/logger"
    "bluebell/routers"
    "bluebell/settings"

    "go.uber.org/zap"
)

func main() {
    // 1️⃣ 加载配置
    if err := settings.Init("./config.yaml"); err != nil {
        fmt.Printf("init settings failed, err:%v\n", err)
        return
    }

    // 2️⃣ 初始化日志
    // 传入配置和当前运行模式
    if err := logger.Init(settings.Conf.Log, settings.Conf.App.Mode); err != nil {
        fmt.Printf("init logger failed, err:%v\n", err)
        return
    }

    // 退出前将缓冲区日志刷盘
    // 为什么: Zap 有内部缓冲,Sync() 确保所有日志都写入
    defer zap.L().Sync()

    // 3️⃣ 初始化其他组件...
    // mysql.Init()
    // redis.Init()
    // ...

    // 4️⃣ 注册路由
    r := routers.SetupRouter(settings.Conf.App.Mode)

    // 5️⃣ 启动服务
    zap.L().Info("Server is starting...",
        zap.String("version", settings.Conf.App.Version),
        zap.Int("port", settings.Conf.App.Port),
    )

    if err := r.Run(fmt.Sprintf(":%d", settings.Conf.App.Port)); err != nil {
        zap.L().Fatal("Server startup failed", zap.Error(err))
    }
}
```

### 6.2 在路由中注册中间件

```go
// routers/routers.go

package routers

import (
    "bluebell/controller"
    "bluebell/logger"
    "bluebell/middlewares"

    "github.com/gin-gonic/gin"
)

func SetupRouter(mode string) *gin.Engine {
    // 设置 Gin 的运行模式
    if mode == gin.ReleaseMode {
        gin.SetMode(gin.ReleaseMode)
    }

    // 创建 Gin 引擎
    // gin.New() 返回一个不带任何中间件的空引擎
    r := gin.New()

    // 🔥 注册我们自定义的中间件
    // 替代 gin.Logger() 和 gin.Recovery()
    r.Use(
        logger.GinLogger(),        // 请求日志中间件
        logger.GinRecovery(true),  // Panic 恢复中间件 (打印堆栈)
    )

    // 注册路由组
    v1 := r.Group("/api/v1")
    {
        // 公开路由
        v1.POST("/signup", controller.SignUpHandler)
        v1.POST("/login", controller.LoginHandler)

        // 认证路由 (需要 JWT)
        v1.Use(middlewares.JWTAuthMiddleware())
        {
            v1.GET("/community", controller.CommunityHandler)
            v1.POST("/post", controller.CreatePostHandler)
            // ...
        }
    }

    return r
}
```

### 6.3 在业务代码中使用

```go
// logic/user.go

package logic

import (
    "bluebell/dao/mysql"
    "bluebell/models"
    "bluebell/pkg/snowflake"

    "go.uber.org/zap"
)

func SignUp(p *models.ParamSignUp) error {
    // 使用 zap.L() 获取全局 Logger

    // Debug 级别: 详细的调试信息
    zap.L().Debug("SignUp called",
        zap.String("username", p.Username),
    )

    // 检查用户是否存在
    if err := mysql.CheckUserExist(p.Username); err != nil {
        // Error 级别: 错误但可恢复
        zap.L().Error("CheckUserExist failed",
            zap.String("username", p.Username),
            zap.Error(err),
        )
        return err
    }

    // 生成 UserID
    userID := snowflake.GenID()

    // Info 级别: 重要业务事件
    zap.L().Info("User registering",
        zap.String("username", p.Username),
        zap.Int64("user_id", userID),
    )

    // 构造 User 实例
    user := &models.User{
        UserID:   userID,
        Username: p.Username,
        Password: p.Password,
    }

    // 保存到数据库
    if err := mysql.InsertUser(user); err != nil {
        zap.L().Error("InsertUser failed",
            zap.String("username", p.Username),
            zap.Error(err),
        )
        return err
    }

    // Info 级别: 注册成功
    zap.L().Info("User registered successfully",
        zap.String("username", p.Username),
        zap.Int64("user_id", userID),
    )

    return nil
}
```

---

## 7. 测试与验证

### 7.1 开发模式测试

**修改 config.yaml:**

```yaml
app:
  mode: "dev"
  port: 8080

log:
  level: "debug"
  file_name: "bluebell.log"
  max_size: 100
  max_backups: 7
  max_age: 30
```

**启动项目:**

```bash
go run main.go

# 控制台输出 (Console 格式):
2025-12-08T17:30:00.123Z  INFO  main.go:42  Server is starting...  {"version": "v1.0.0", "port": 8080}
2025-12-08T17:30:00.456Z  INFO  logger/logger.go:129  http request  {"status": 200, "method": "POST", "path": "/api/v1/signup", ...}
```

**同时查看文件 bluebell.log (JSON 格式):**

```json
{
  "level": "info",
  "time": "2025-12-08T17:30:00.123Z",
  "caller": "main.go:42",
  "msg": "Server is starting...",
  "version": "v1.0.0",
  "port": 8080
}
{
  "level": "info",
  "time": "2025-12-08T17:30:00.456Z",
  "caller": "logger/logger.go:129",
  "msg": "http request",
  "status": 200,
  "method": "POST",
  "path": "/api/v1/signup",
  "query": "",
  "ip": "192.168.1.100",
  "user-agent": "curl/7.68.0",
  "cost": 0.120345,
  "errors": ""
}
```

### 7.2 生产模式测试

**修改 config.yaml:**

```yaml
app:
  mode: "release"
  port: 8080

log:
  level: "info"
  file_name: "bluebell.log"
  max_size: 100
  max_backups: 7
  max_age: 30
```

**启动项目:**

```bash
go run main.go

# 控制台: 无输出 (静默)
# 所有日志都在 bluebell.log 中
```

### 7.3 日志切割测试

**模拟大量日志:**

```go
func TestLogRotation() {
    for i := 0; i < 1000000; i++ {
        zap.L().Info("Test log rotation",
            zap.Int("index", i),
            zap.String("data", strings.Repeat("A", 1000)),  // 1KB 数据
        )
    }
}
```

**运行后查看文件:**

```bash
ls -lh *.log

# 输出:
# -rw-r--r--  1 user  staff  100M 12  8 17:30 bluebell.log
# -rw-r--r--  1 user  staff  100M 12  8 17:25 bluebell-2025-12-08T17-25-30.log
# -rw-r--r--  1 user  staff  100M 12  8 17:20 bluebell-2025-12-08T17-20-15.log
```

**验证:** 当 bluebell.log 达到 100MB 时,自动归档为带时间戳的文件,并创建新的 bluebell.log。

---

## 8. 常见问题 FAQ

### Q1: Logger 和 SugaredLogger 有什么区别?

**A:**

| 特性 | Logger | SugaredLogger |
|------|--------|---------------|
| **性能** | 极快 (零内存分配) | 稍慢 (使用反射) |
| **API** | 强类型 (`zap.String`, `zap.Int`) | 类似 Printf (`Infof`, `Debugf`) |
| **适用场景** | 高频日志,性能敏感 | 低频日志,方便快捷 |

**示例:**

```go
// Logger (推荐用于高频日志)
zap.L().Info("User registered",
    zap.String("username", "zhangsan"),
    zap.Int64("user_id", 12345),
)

// SugaredLogger (方便但性能稍差)
zap.S().Infof("User registered: %s, ID: %d", "zhangsan", 12345)
```

**建议:** Bluebell 项目统一使用 Logger (`zap.L()`),追求最佳性能。

---

### Q2: 如何在日志中隐藏敏感信息?

**A:**

```go
// ❌ 错误做法:直接记录密码
zap.L().Info("User login",
    zap.String("username", "zhangsan"),
    zap.String("password", "123456"),  // 泄露密码!
)

// ✅ 正确做法1:不记录敏感字段
zap.L().Info("User login",
    zap.String("username", "zhangsan"),
    // 不记录 password
)

// ✅ 正确做法2:记录脱敏后的信息
zap.L().Info("User login",
    zap.String("username", "zhangsan"),
    zap.String("password_hash", "***"),  // 只记录掩码
)
```

---

### Q3: 如何实现日志的异步写入?

**A:**

Zap 默认是同步写入,如果需要异步可以使用 `zapcore.BufferedWriteSyncer`:

```go
func getLogWriter(filename string, maxSize int, maxBackups int, maxAge int) zapcore.WriteSyncer {
    lumberjackLogger := &lumberjack.Logger{
        Filename:   filename,
        MaxSize:    maxSize,
        MaxBackups: maxBackups,
        MaxAge:     maxAge,
    }

    // 🔥 异步写入 (缓冲 256KB,每秒刷盘一次)
    return &zapcore.BufferedWriteSyncer{
        WS:   zapcore.AddSync(lumberjackLogger),
        Size: 256 * 1024,  // 256KB 缓冲区
        FlushInterval: time.Second,  // 每秒刷盘一次
    }
}
```

**权衡:**
- ✅ 性能更好 (减少磁盘 IO)
- ❌ 可能丢失日志 (程序崩溃时缓冲区未刷盘)

---

### Q4: 如何根据请求动态调整日志级别?

**A:**

使用 `zap.AtomicLevel`:

```go
// 全局变量
var atomicLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)

func Init(cfg *settings.LogConfig, mode string) error {
    // 使用 atomicLevel
    core := zapcore.NewCore(encoder, writeSyncer, atomicLevel)
    lg := zap.New(core, zap.AddCaller())
    zap.ReplaceGlobals(lg)
    return nil
}

// 动态调整级别
func SetLogLevel(level string) {
    var l zapcore.Level
    l.UnmarshalText([]byte(level))
    atomicLevel.SetLevel(l)
}

// 路由注册
r.PUT("/admin/log-level", func(c *gin.Context) {
    level := c.Query("level")
    SetLogLevel(level)
    c.JSON(200, gin.H{"msg": "ok"})
})
```

**使用:**

```bash
# 动态调整为 Debug 级别
curl -X PUT "http://localhost:8080/admin/log-level?level=debug"

# 调整回 Info 级别
curl -X PUT "http://localhost:8080/admin/log-level?level=info"
```

---

### Q5: 如何将日志发送到远程日志服务?

**A:**

创建自定义 WriteSyncer:

```go
type RemoteWriter struct {
    url string
}

func (w *RemoteWriter) Write(p []byte) (n int, err error) {
    // 发送日志到远程服务 (如 Loki, Elasticsearch)
    resp, err := http.Post(w.url, "application/json", bytes.NewReader(p))
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()
    return len(p), nil
}

func (w *RemoteWriter) Sync() error {
    return nil
}

// 使用
remoteWriter := &RemoteWriter{url: "http://loki:3100/loki/api/v1/push"}
core := zapcore.NewCore(encoder, zapcore.AddSync(remoteWriter), level)
```

---

## 9. 最佳实践总结

### 9.1 日志级别使用规范

| 级别 | 使用场景 | 示例 |
|------|---------|------|
| **Debug** | 详细的调试信息,帮助开发排查问题 | SQL 查询语句、函数入参、中间变量 |
| **Info** | 关键业务事件,记录系统正常运行的重要信息 | 用户登录、订单创建、服务启动 |
| **Warn** | 警告但不影响运行,需要关注但不紧急 | API 调用超时、缓存未命中、配置使用默认值 |
| **Error** | 错误但可恢复,需要及时处理 | 数据库查询失败、第三方 API 错误、文件读取失败 |
| **Fatal** | 致命错误,程序无法继续运行 | 配置文件加载失败、必需资源不可用 |

### 9.2 性能优化清单

- ✅ 使用 Logger 而不是 SugaredLogger
- ✅ 使用强类型字段 (`zap.String`, `zap.Int`)
- ✅ 避免在循环中频繁记录 Debug 日志
- ✅ 生产环境设置合适的日志级别 (Info 或 Warn)
- ✅ 定期清理旧日志文件

### 9.3 安全清单

- ✅ 不记录敏感信息 (密码、Token、身份证号)
- ✅ 记录用户操作时使用用户 ID 而非用户名
- ✅ 日志文件设置合适的权限 (644 或 600)
- ✅ 定期审计日志内容,确保合规

---

## 10. 课后练习

### 练习1: 实现请求 ID 追踪

**任务:** 为每个请求生成唯一 ID,并在所有日志中记录。

**提示:**
1. 在 GinLogger 中间件中生成 UUID
2. 将 UUID 存入 `gin.Context`
3. 在业务代码中获取 UUID 并记录

<details>
<summary>点击查看答案</summary>

```go
// 中间件
func GinLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 生成请求 ID
        requestID := uuid.New().String()
        c.Set("request_id", requestID)

        start := time.Now()
        c.Next()
        cost := time.Since(start)

        zap.L().Info("http request",
            zap.String("request_id", requestID),  // ← 记录请求 ID
            zap.Int("status", c.Writer.Status()),
            // ...
        )
    }
}

// 业务代码
func SignUp(c *gin.Context, p *models.ParamSignUp) error {
    requestID, _ := c.Get("request_id")

    zap.L().Info("User registering",
        zap.String("request_id", requestID.(string)),  // ← 记录请求 ID
        zap.String("username", p.Username),
    )

    // ...
}
```
</details>

### 练习2: 实现慢请求日志

**任务:** 对耗时超过 1 秒的请求记录 Warn 日志。

<details>
<summary>点击查看答案</summary>

```go
func GinLogger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        cost := time.Since(start)

        // 根据耗时选择日志级别
        if cost > time.Second {
            zap.L().Warn("slow request",  // ← Warn 级别
                zap.Int("status", c.Writer.Status()),
                zap.String("method", c.Request.Method),
                zap.String("path", c.Request.URL.Path),
                zap.Duration("cost", cost),
            )
        } else {
            zap.L().Info("http request",  // ← Info 级别
                // ...
            )
        }
    }
}
```
</details>

### 练习3: 实现按日期切割日志

**任务:** 修改配置,使日志每天生成一个新文件 (bluebell-2025-12-08.log)。

**提示:** 修改 Lumberjack 配置,设置 `MaxAge=1`。

---

## 11. 延伸阅读

- 📖 [Uber Zap GitHub](https://github.com/uber-go/zap)
- 📖 [Lumberjack Log Rolling](https://github.com/natefinch/lumberjack)
- 📖 [Zap 性能基准测试](https://github.com/uber-go/zap/blob/master/FAQ.md#performance)
- 📖 下一章: [第08章:用户登录功能实现](./08-用户登录功能实现.md)

---

**恭喜!** 你已经掌握了 Zap 日志系统的集成和环境隔离技术,能够构建专业级的日志体系。下一章我们将学习如何实现用户登录功能。🔐
