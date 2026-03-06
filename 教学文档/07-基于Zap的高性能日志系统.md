# 第07章:基于Zap的高性能日志系统

> **本章导读**
> 
> 在开发过程中，`fmt.Println` 是最简单的调试工具，但在生产环境中，我们需要一个高性能、结构化、支持日志切割的专业日志系统。
> 
> Bluebell 项目选择了 Uber 开源的 **Zap** 库，它是目前 Go 语言中最快的日志库之一。本章将讲解如何配置 Zap，使其在"开发环境"和"生产环境"下表现出不同的行为（环境隔离）。

---

## 📚 本章目标

学完本章，你将掌握：

1. 理解为什么 `fmt` 包不适合生产环境
2. 掌握 Zap 日志库的核心概念（Logger, SugaredLogger, Encoder, Core）
3. 实现日志的**环境隔离**（开发环境输出到控制台，生产环境输出到文件）
4. 使用 `lumberjack` 实现日志的**自动切割与归档**
5. 集成 Gin 框架，将 HTTP 请求日志接管给 Zap

---

## 1. 为什么选择 Zap?

### 1.1 标准库 log vs Zap

| 特性 | 标准库 log | Zap |
|------|-----------|-----|
| **性能** | 一般 | **极高**（零内存分配）|
| **结构化** | 不支持（只能存字符串） | **支持**（JSON 格式）|
| **日志级别** | 仅 Print/Fatal/Panic | Debug/Info/Warn/Error/DPanic/Panic/Fatal |
| **字段类型** | 弱类型 | **强类型**（避免反射）|

### 1.2 什么是"环境隔离"?

- **开发环境 (`dev`)**:
    - 我们希望日志直接输出到 **控制台**，方便实时看。
    - 格式最好是 **Console**（人类可读），带颜色。
    - 同时也记录到文件，防止控制台刷屏太快漏看。
- **生产环境 (`release`)**:
    - 我们希望日志只输出到 **文件**，控制台保持静默（提高性能）。
    - 格式必须是 **JSON**，方便日志收集系统（如 ELK, Loki）解析。
    - 必须支持 **日志切割**，防止单个日志文件无限膨胀占满磁盘。

---

## 2. 核心代码实现

### 2.1 配置文件 (`config.yaml`)

首先，在配置文件中定义日志相关的参数：

```yaml
# config.yaml
log:
  level: "debug"         # 日志级别
  file_name: "bluebell.log" # 日志文件名
  max_size: 100          # 单个文件最大 MB
  max_backups: 7         # 保留旧文件个数
  max_age: 30            # 保留旧文件天数
```

### 2.2 日志初始化 (`internal/infrastructure/logger/logger.go`)

这是本章的核心。我们需要编写 `Init` 函数，根据 `mode` 参数决定 Zap 的行为。

```go
package logger

import (
	"bluebell/settings"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2" // 需要 go get gopkg.in/natefinch/lumberjack.v2
)

// Init 初始化 Logger
func Init(cfg *settings.LogConfig, mode string) (err error) {
	if cfg == nil {
		return fmt.Errorf("logger.Init received nil config")
	}

	// 1. 获取日志写入器 (支持切割)
	writeSyncer := getLogWriter(
		cfg.FileName,
		cfg.MaxSize,
		cfg.MaxBackups,
		cfg.MaxAge,
	)

	// 2. 获取日志编码器 (JSON)
	encoder := getEncoder()

	// 3. 解析日志级别
	var level zapcore.Level
	if err = level.UnmarshalText([]byte(cfg.Level)); err != nil {
		return
	}

	var core zapcore.Core

    // 🔥 核心逻辑：根据模式决定 Core 的行为
	if mode == "dev" || mode == gin.DebugMode {
		// === 开发模式 ===
        
        // 控制台编码器 (Console 格式, 人类可读)
		consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

		// 创建两个 Core
        // Core 1: 输出到文件 (JSON)
		fileCore := zapcore.NewCore(encoder, writeSyncer, level)
        // Core 2: 输出到控制台 (Console)
		consoleCore := zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), zapcore.DebugLevel)

		// 使用 NewTee 将它们合并 (双重输出)
		core = zapcore.NewTee(fileCore, consoleCore)
	} else {
		// === 生产模式 ===
        // 只输出到文件 (JSON)
		core = zapcore.NewCore(encoder, writeSyncer, level)
	}

	// 4. 创建 Logger 实例
	// AddCaller: 在日志中显示文件名和行号
	lg := zap.New(core, zap.AddCaller())
    
	// 5. 替换全局 Logger
	zap.ReplaceGlobals(lg)
	return
}
```

### 2.3 日志切割 (`lumberjack`)

为了防止 `bluebell.log` 变成几十 GB 的巨型文件，我们使用 `lumberjack` 库进行切割。

```go
func getLogWriter(filename string, maxSize int, maxBackups int, maxAge int) zapcore.WriteSyncer {
	lumberjackLogger := &lumberjack.Logger{
		Filename:   filename,   // 日志文件路径
		MaxSize:    maxSize,    // 单个文件最大尺寸 (MB)
		MaxBackups: maxBackups, // 最多保留备份个数
		MaxAge:     maxAge,     // 最多保留天数
        Compress:   false,      // 是否压缩 (gzip)
	}
	return zapcore.AddSync(lumberjackLogger)
}
```

### 2.4 日志编码器 (`Encoder`)

```go
func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.TimeKey = "time"                          // 时间字段 Key
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   // ISO8601 时间格式
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // 大写级别 (INFO, ERROR)
	return zapcore.NewJSONEncoder(encoderConfig)            // 返回 JSON 编码器
}
```

---

## 3. 集成 Gin 框架

Gin 默认的日志中间件输出到标准输出，我们需要编写自定义中间件，让 Gin 的请求日志也走 Zap。

在 `internal/infrastructure/logger/logger.go` 中添加：

### 3.1 GinLogger 中间件

用于替代 `gin.Logger()`。

```go
// GinLogger 接收 Gin 框架默认的日志
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next() // 执行后续业务逻辑

		// 记录请求日志
		cost := time.Since(start)
		zap.L().Info("http request",
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.String("errors", c.Errors.ByType(gin.ErrorTypePrivate).String()),
			zap.Duration("cost", cost),
		)
	}
}
```

### 3.2 GinRecovery 中间件

用于替代 `gin.Recovery()`，捕获 Panic 并记录堆栈信息。

```go
// GinRecovery recover掉项目可能出现的panic
func GinRecovery(stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Check for broken pipe ... (省略部分网络错误检查代码) 
				
				// 记录堆栈信息
				zap.L().Error("[Recovery from panic]",
					zap.Any("error", err),
					zap.String("request", string(httputil.DumpRequest(c.Request, false))),
                // 如果 stack 为 true，打印堆栈
					zap.String("stack", string(debug.Stack())),
				)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}
```

### 3.3 注册中间件 (`routers/routers.go`)

```go
func SetupRouter(mode string) *gin.Engine {
    if mode == gin.ReleaseMode {
        gin.SetMode(gin.ReleaseMode)
    }
    
r := gin.New()
    // 🔥 使用我们自定义的中间件
    r.Use(logger.GinLogger(), logger.GinRecovery(true))
    
    // ... 注册路由 ...
    return r
}
```

---

## 4. 在 main.go 中使用

```go
func main() {
    // 1. 加载配置 ...
    
    // 2. 初始化日志
    // 传入配置和当前运行模式
    if err := logger.Init(settings.Conf.Log, settings.Conf.App.Mode); err != nil {
        fmt.Printf("init logger failed, err:%v\n", err)
        return
    }
    // 退出前将缓冲区日志刷盘
    defer zap.L().Sync()

    // 3. 业务逻辑 ...
    zap.L().Info("Server is starting...") // 使用 Zap 记录日志
}
```

---

## 5. 验证效果

### 5.1 开发模式 (`dev`)

修改 `config.yaml` 中 `app.mode: "dev"`。
启动项目，你会看到：
1.  **控制台**：有彩色的日志输出，方便调试。
2.  **文件 (`bluebell.log`)**：同时生成 JSON 格式的日志。

### 5.2 生产模式 (`release`)

修改 `config.yaml` 中 `app.mode: "release"`。
启动项目，你会看到：
1.  **控制台**：没有任何输出（静默）。
2.  **文件 (`bluebell.log`)**：只有这里有日志。

---

## 6. 最佳实践总结

1.  **强类型字段**: 尽量使用 `zap.String`, `zap.Int` 等强类型方法，避免使用 `zap.Any` (涉及反射，性能稍差)。
2.  **全局 Logger**: 使用 `zap.ReplaceGlobals` 替换全局 Logger 后，可以在任何地方通过 `zap.L()` 调用，无需层层传递 Logger 实例。
3.  **日志分级**:
    *   `Debug`: 开发调试信息 (SQL 语句、参数细节)。
    *   `Info`: 关键业务状态 (服务启动、请求处理)。
    *   `Warn`: 警告但不影响运行 (Redis 连接慢)。
    *   `Error`: 错误但可恢复 (数据库查询失败)。
    *   `Fatal`: 致命错误 (配置文件加载失败，会导致程序退出)。

---

## 7. 延伸阅读

*   [Uber Zap GitHub](https://github.com/uber-go/zap)
*   [Lumberjack Log Rolling](https://github.com/natefinch/lumberjack)
*   📖 下一章: [第08章:JWT认证与登录功能实现](./08-JWT认证与登录功能实现.md)

