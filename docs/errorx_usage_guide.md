# Errorx 错误处理架构使用指南

## 📋 概述

本项目引入了 `pkg/errorx` 包，实现了 **"Logic 层决定错误码，Controller 层透传响应"** 的模式，极大简化了错误处理流程。

## 🏗️ 架构设计

### 核心思想

```
┌──────────────┐
│  Controller  │  → 只负责调用 HandleError，不关心错误码
├──────────────┤
│    Logic     │  → 决定返回业务错误(CodeError) 或 系统错误(ErrServerBusy)
├──────────────┤
│     DAO      │  → 返回原始错误(mysql.Error 或 redis.Error)
└──────────────┘
```

### 错误分类

- **业务错误**: 用户可感知的错误（用户不存在、密码错误等）→ 返回 `*errorx.CodeError`
- **系统错误**: 系统内部错误（DB 连接失败、Redis 故障等）→ Logic 层记录日志后返回 `errorx.ErrServerBusy`

## 📦 核心组件

### 1. pkg/errorx/errorx.go

#### CodeError 结构体

```go
type CodeError struct {
    Code int    // 业务错误码
    Msg  string // 错误消息
}
```

#### 预定义错误常量

```go
var (
    ErrInvalidParam    = New(CodeInvalidParam, "请求参数错误")
    ErrUserExist       = New(CodeUserExist, "用户名已存在")
    ErrUserNotExist    = New(CodeUserNotExist, "用户名不存在")
    ErrInvalidPassword = New(CodeInvalidPassword, "用户名或密码错误")
    ErrServerBusy      = New(CodeServerBusy, "服务繁忙")
    ErrNeedLogin       = New(CodeNeedLogin, "需要登录")
    ErrInvalidToken    = New(CodeInvalidToken, "无效的Token")
    ErrNotFound        = New(CodeNotFound, "资源不存在")
)
```

#### 辅助函数

```go
// 创建自定义错误
errorx.New(code, msg)

// 创建格式化错误
errorx.Newf(code, "user %s not found", username)

// 获取错误码对应的默认消息
errorx.GetMsg(code)
```

### 2. controller/code.go - HandleError 方法

```go
func HandleError(c *gin.Context, err error) {
    // 1. 尝试断言为 *errorx.CodeError 类型
    var codeErr *errorx.CodeError
    if errors.As(err, &codeErr) {
        // 业务错误：直接返回携带的错误码和消息
        c.JSON(http.StatusOK, gin.H{
            "code": codeErr.Code,
            "msg":  codeErr.Msg,
            "data": nil,
        })
        return
    }

    // 2. 系统错误：记录日志并返回服务繁忙
    zap.L().Error("system error occurred",
        zap.String("path", c.Request.URL.Path),
        zap.String("method", c.Request.Method),
        zap.Error(err),
    )
    c.JSON(http.StatusOK, gin.H{
        "code": errorx.CodeServerBusy,
        "msg":  errorx.GetMsg(errorx.CodeServerBusy),
        "data": nil,
    })
}
```

## 🔨 使用示例

### 完整示例：用户登录

#### Logic 层 (logic/user.go)

```go
func Login(p *models.ParamLogin) (string, string, error) {
    user := &models.User{
        Username: p.Username,
        Password: p.Password,
    }

    // 1. 调用 DAO 层验证用户登录
    err := mysql.CheckLogin(user)
    if err != nil {
        // 判断是否是业务错误
        if errors.Is(err, mysql.ErrorUserNotExist) {
            // 业务错误：返回带错误码的 CodeError
            return "", "", errorx.ErrUserNotExist
        }
        if errors.Is(err, mysql.ErrorInvalidPassword) {
            return "", "", errorx.ErrInvalidPassword
        }

        // 系统错误：记录详细日志并返回通用错误
        zap.L().Error("mysql.CheckLogin failed",
            zap.String("username", p.Username),
            zap.Error(err),
        )
        return "", "", errorx.ErrServerBusy
    }

    // 2. 生成 JWT Token
    aToken, rToken, err := jwt.GenToken(user.UserID, user.Username)
    if err != nil {
        zap.L().Error("jwt.GenToken failed",
            zap.Int64("user_id", user.UserID),
            zap.Error(err),
        )
        return "", "", errorx.ErrServerBusy
    }

    // 3. 存入 Redis
    err = redis.SetUserToken(user.UserID, aToken, rToken, ...)
    if err != nil {
        zap.L().Error("redis.SetUserToken failed",
            zap.Int64("user_id", user.UserID),
            zap.Error(err),
        )
        return "", "", errorx.ErrServerBusy
    }

    return aToken, rToken, nil
}
```

#### Controller 层 (controller/user.go)

```go
func LoginHandler(c *gin.Context) {
    var p models.ParamLogin

    // 1. 参数校验
    if err := c.ShouldBindJSON(&p); err != nil {
        errs, ok := err.(validator.ValidationErrors)
        if !ok {
            ResponseError(c, CodeInvalidParam)
            return
        }
        ResponseErrorWithMsg(c, CodeInvalidParam, removeTopStruct(errs.Translate(trans)))
        return
    }

    // 2. 业务处理
    aToken, rToken, err := logic.Login(&p)
    if err != nil {
        // 3. 错误处理：一行代码搞定！
        HandleError(c, err)
        return
    }

    // 4. 返回响应
    ResponseSuccess(c, map[string]string{
        "access_token":  aToken,
        "refresh_token": rToken,
    })
}
```

## 📊 对比：改造前后

### 改造前（Controller 层需要判断错误类型）

```go
// ❌ Controller 层需要导入 mysql 包，违反分层原则
import "bluebell/dao/mysql"

func LoginHandler(c *gin.Context) {
    aToken, rToken, err := logic.Login(&p)
    if err != nil {
        zap.L().Error("logic.Login failed", zap.Error(err))

        // 需要逐个判断错误类型
        if errors.Is(err, mysql.ErrorUserNotExist) {
            ResponseError(c, CodeUserNotExist)
            return
        }
        if errors.Is(err, mysql.ErrorInvalidPassword) {
            ResponseError(c, CodeInvalidPassword)
            return
        }
        ResponseError(c, CodeServerBusy)
        return
    }
    // ...
}
```

### 改造后（Controller 层只需透传）

```go
// ✅ Controller 层无需导入 mysql 包，职责清晰
func LoginHandler(c *gin.Context) {
    aToken, rToken, err := logic.Login(&p)
    if err != nil {
        // 一行代码搞定所有错误处理
        HandleError(c, err)
        return
    }
    // ...
}
```

## 🎯 最佳实践

### 1. Logic 层错误处理规范

```go
func SomeLogicFunc() error {
    // DAO 层调用
    err := mysql.SomeQuery()
    if err != nil {
        // 判断是否是业务错误
        if errors.Is(err, mysql.ErrorNotFound) {
            return errorx.ErrNotFound  // ✅ 业务错误，直接返回
        }

        // 系统错误：先记日志，再返回通用错误
        zap.L().Error("mysql.SomeQuery failed",
            zap.String("context", "业务上下文"),
            zap.Error(err),
        )
        return errorx.ErrServerBusy  // ✅ 系统错误，返回服务繁忙
    }
    return nil
}
```

### 2. 自定义业务错误

```go
// 场景：需要返回自定义的错误消息
func CheckUserPermission(userID int64, resourceID int64) error {
    if !hasPermission(userID, resourceID) {
        // 使用 New 创建自定义错误
        return errorx.New(errorx.CodeInvalidParam,
            fmt.Sprintf("用户 %d 无权访问资源 %d", userID, resourceID))
    }
    return nil
}

// 场景：使用格式化消息
func ValidateAge(age int) error {
    if age < 18 {
        return errorx.Newf(errorx.CodeInvalidParam,
            "年龄必须大于等于18岁，当前年龄: %d", age)
    }
    return nil
}
```

### 3. Controller 层统一处理

```go
func SomeHandler(c *gin.Context) {
    // 所有 Logic 层返回的错误都用 HandleError 处理
    if err := logic.SomeFunc(); err != nil {
        HandleError(c, err)
        return
    }
    ResponseSuccess(c, data)
}
```

## 🚀 优势总结

| 维度 | 改造前 | 改造后 |
|------|--------|--------|
| **代码量** | Controller 层需要 10+ 行错误判断 | 1 行 `HandleError(c, err)` |
| **分层原则** | Controller 依赖 DAO 层错误 | Controller 无需导入 DAO 包 |
| **职责清晰** | Controller 需要决定错误码 | Logic 层决定，Controller 透传 |
| **日志记录** | 散落在各处 | Logic 层统一记录 + HandleError 兜底 |
| **可维护性** | 新增错误类型需要改 Controller | 只需在 Logic 层处理 |
| **测试友好** | 难以 Mock DAO 错误 | 只需 Mock Logic 层返回 CodeError |

## 🔧 迁移指南

### 步骤 1：改造 Logic 层

```go
// 旧代码
func OldLogin(p *Param) error {
    if err := mysql.Check(); err != nil {
        return err  // ❌ 直接返回 DAO 错误
    }
}

// 新代码
func NewLogin(p *Param) error {
    if err := mysql.Check(); err != nil {
        if errors.Is(err, mysql.ErrorNotFound) {
            return errorx.ErrUserNotExist  // ✅ 转换为业务错误
        }
        zap.L().Error("...", zap.Error(err))
        return errorx.ErrServerBusy  // ✅ 转换为系统错误
    }
}
```

### 步骤 2：简化 Controller 层

```go
// 旧代码
func OldHandler(c *gin.Context) {
    if err := logic.Func(); err != nil {
        if errors.Is(err, mysql.ErrorA) {
            ResponseError(c, CodeA)
            return
        }
        if errors.Is(err, mysql.ErrorB) {
            ResponseError(c, CodeB)
            return
        }
        ResponseError(c, CodeServerBusy)
        return
    }
}

// 新代码
func NewHandler(c *gin.Context) {
    if err := logic.Func(); err != nil {
        HandleError(c, err)  // ✅ 一行搞定
        return
    }
}
```

## 📝 错误码映射表

| 错误码 | 常量名 | 消息 | 使用场景 |
|--------|--------|------|----------|
| 1000 | CodeSuccess | success | 成功响应 |
| 1001 | CodeInvalidParam | 请求参数错误 | 参数校验失败 |
| 1002 | CodeUserExist | 用户名已存在 | 注册时用户名重复 |
| 1003 | CodeUserNotExist | 用户名不存在 | 登录时用户不存在 |
| 1004 | CodeInvalidPassword | 用户名或密码错误 | 登录密码错误 |
| 1005 | CodeServerBusy | 服务繁忙 | 系统错误兜底 |
| 1006 | CodeNeedLogin | 需要登录 | 未认证访问 |
| 1007 | CodeInvalidToken | 无效的Token | Token 过期/错误 |
| 1008 | CodeNotFound | 资源不存在 | 查询资源不存在 |

## 🐛 调试技巧

### 查看错误堆栈

```go
// Logic 层记录详细日志
zap.L().Error("operation failed",
    zap.String("operation", "用户登录"),
    zap.Int64("user_id", userID),
    zap.Error(err),  // 会打印完整的错误堆栈
)
```

### 区分业务错误和系统错误

```go
// 在 HandleError 中已经自动处理：
// - CodeError → 业务错误，直接返回给客户端
// - 其他 error → 系统错误，记录日志后返回 CodeServerBusy
```

## 🎓 总结

通过引入 `errorx` 包，我们实现了：

1. ✅ **职责分离**: Logic 层决定错误码，Controller 层透传
2. ✅ **代码简化**: Controller 层错误处理从 10+ 行缩减到 1 行
3. ✅ **统一日志**: 系统错误在 Logic 层和 HandleError 双重记录
4. ✅ **易于扩展**: 新增错误类型只需在 Logic 层处理
5. ✅ **分层清晰**: Controller 无需依赖 DAO 层错误类型

**核心原则**: Logic 层是错误处理的决策者，Controller 层是错误响应的执行者！
