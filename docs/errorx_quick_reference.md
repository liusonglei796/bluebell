## Errorx 快速参考卡片 🚀

### 📦 核心组件

```go
// 1. pkg/errorx/errorx.go
type CodeError struct {
    Code int    // 业务错误码
    Msg  string // 错误消息
}

// 2. controller/code.go
func HandleError(c *gin.Context, err error)
```

---

### 🔑 预定义错误常量

| 常量 | 错误码 | 使用场景 |
|------|--------|----------|
| `errorx.ErrInvalidParam` | 1001 | 参数校验失败 |
| `errorx.ErrUserExist` | 1002 | 用户名已存在 |
| `errorx.ErrUserNotExist` | 1003 | 用户不存在 |
| `errorx.ErrInvalidPassword` | 1004 | 密码错误 |
| `errorx.ErrServerBusy` | 1005 | 系统错误兜底 |
| `errorx.ErrNeedLogin` | 1006 | 未认证 |
| `errorx.ErrInvalidToken` | 1007 | Token错误 |
| `errorx.ErrNotFound` | 1008 | 资源不存在 |

---

### 💡 使用模板

#### Logic 层（决定错误码）

```go
func SomeFunc() error {
    // 调用 DAO
    err := mysql.Query()
    if err != nil {
        // 1️⃣ 业务错误：直接返回 CodeError
        if errors.Is(err, mysql.ErrorNotFound) {
            return errorx.ErrNotFound
        }

        // 2️⃣ 系统错误：记录日志 + 返回 ErrServerBusy
        zap.L().Error("mysql.Query failed",
            zap.String("context", "具体业务上下文"),
            zap.Error(err),
        )
        return errorx.ErrServerBusy
    }
    return nil
}
```

#### Controller 层（透传响应）

```go
func SomeHandler(c *gin.Context) {
    // 业务处理
    data, err := logic.SomeFunc()
    if err != nil {
        HandleError(c, err)  // ✅ 一行搞定
        return
    }

    ResponseSuccess(c, data)
}
```

---

### 🛠️ 进阶用法

#### 自定义错误消息

```go
// 方式 1: 创建新错误
return errorx.New(errorx.CodeInvalidParam, "自定义消息")

// 方式 2: 格式化消息
return errorx.Newf(errorx.CodeInvalidParam, "用户 %d 无权限", userID)
```

#### 获取默认消息

```go
msg := errorx.GetMsg(errorx.CodeInvalidParam)
// 返回: "请求参数错误"
```

---

### ✅ 错误处理决策树

```
DAO 层返回错误
    │
    ├─ 是业务错误？(用户不存在、密码错误等)
    │   └─ YES → return errorx.ErrXXX
    │
    └─ 是系统错误？(DB连接失败、Redis故障等)
        └─ YES → zap.L().Error(...) + return errorx.ErrServerBusy
```

---

### 📊 改造前后对比

| 维度 | 改造前 | 改造后 |
|------|--------|--------|
| **代码行数** | 10+ 行 | 1 行 |
| **Controller 依赖** | 需要导入 `dao/mysql` | 无需导入 |
| **职责** | Controller 决定错误码 | Logic 决定错误码 |
| **日志** | Controller 记录 | Logic 记录 + HandleError 兜底 |

---

### 🎯 记住三原则

1. **Logic 层是决策者**: 决定返回业务错误还是系统错误
2. **Controller 层是执行者**: 只需透传，不关心错误码
3. **系统错误必须记日志**: Logic 层遇到 DB/Redis 错误要先记日志

---

### 📝 示例：完整的用户登录流程

```go
// Logic 层
func Login(p *ParamLogin) (string, string, error) {
    err := mysql.CheckLogin(user)
    if err != nil {
        if errors.Is(err, mysql.ErrorUserNotExist) {
            return "", "", errorx.ErrUserNotExist  // 业务错误
        }
        zap.L().Error("mysql.CheckLogin failed", zap.Error(err))
        return "", "", errorx.ErrServerBusy  // 系统错误
    }
    // ...
    return aToken, rToken, nil
}

// Controller 层
func LoginHandler(c *gin.Context) {
    aToken, rToken, err := logic.Login(&p)
    if err != nil {
        HandleError(c, err)  // 自动识别错误类型
        return
    }
    ResponseSuccess(c, map[string]string{
        "access_token":  aToken,
        "refresh_token": rToken,
    })
}
```

---

**详细文档**: `docs/errorx_usage_guide.md`
