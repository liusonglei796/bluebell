# 错误处理架构总结

## 概述

项目采用分层错误处理架构,遵循 **"Logic 层决定错误码,Controller 层透传响应"** 的设计原则。

## 核心组件

### 1. pkg/errorx 包

**职责**: 定义所有业务错误码和错误类型

**核心结构**:
```go
type CodeError struct {
    Code int    // 业务错误码
    Msg  string // 错误消息
}
```

**预定义错误**:
- `ErrInvalidParam` (1001) - 请求参数错误
- `ErrUserExist` (1002) - 用户已存在
- `ErrUserNotExist` (1003) - 用户不存在
- `ErrInvalidPassword` (1004) - 用户名或密码错误
- `ErrServerBusy` (1005) - 服务繁忙
- `ErrNeedLogin` (1006) - 需要登录
- `ErrInvalidToken` (1007) - 无效的Token
- `ErrNotFound` (1008) - 未找到资源

### 2. Controller 层错误处理函数

项目提供三个错误响应函数,各有明确的使用场景:

#### HandleError (推荐 - 业务逻辑错误)

**使用场景**: 处理从 Logic 层返回的错误

**特点**:
- 自动识别 `errorx.CodeError` 类型并返回对应错误码
- 对系统错误自动记录日志并返回 "服务繁忙"
- 符合 "Logic 层决定错误码" 的设计原则

**示例**:
```go
// Controller 层
data, err := logic.GetCommunityDetail(id)
if err != nil {
    HandleError(c, err)  // 自动处理业务错误和系统错误
    return
}
```

**使用统计**: 10 处使用
- `controller/user.go`: LoginHandler, RefreshTokenHandler
- `controller/community.go`: CommunityHandler, CommunityDetailHandler
- `controller/post.go`: CreatePostHandler, GetPostDetailHandler, GetPostListHandler
- `controller/vote.go`: PostVoteHandler (3 处)

#### ResponseError (参数校验和中间件)

**使用场景**:
1. 参数校验失败 (在调用 Logic 层之前)
2. 中间件认证失败
3. 不涉及 err 对象的错误场景

**特点**:
- 直接传入错误码
- 无需 err 对象
- 适用于 Controller 层独立判断的错误

**示例**:
```go
// 参数校验
if err := c.ShouldBindJSON(p); err != nil {
    ResponseError(c, errorx.CodeInvalidParam)
    return
}

// 中间件认证
if authHeader == "" {
    ResponseError(c, errorx.CodeNeedLogin)
    return
}
```

**使用统计**: 17 处使用
- 参数校验场景: 10 处
- 中间件认证: 5 处
- 函数定义: 1 处
- 其他: 1 处

#### ResponseErrorWithMsg (自定义错误消息)

**使用场景**:
1. Validator 翻译后的参数校验错误
2. 需要自定义错误消息的特殊场景

**特点**:
- 可覆盖默认错误消息
- 用于提供更详细的错误信息

**示例**:
```go
// Validator 翻译
errs, ok := err.(validator.ValidationErrors)
if ok {
    ResponseErrorWithMsg(c, errorx.CodeInvalidParam,
        removeTopStruct(errs.Translate(trans)))
    return
}

// 自定义消息
if parts[1] != redisToken {
    ResponseErrorWithMsg(c, errorx.CodeInvalidToken, "账号已在其他设备登录")
    return
}
```

**使用统计**: 6 处使用
- Validator 翻译: 4 处
- 自定义消息: 2 处

## 使用规范

### Logic 层规范

```go
func BusinessLogic(id int64) (*Data, error) {
    // 1. 调用 DAO 层
    data, err := mysql.GetData(id)
    if err != nil {
        // 系统错误: 记录详细日志 + 返回 ErrServerBusy
        zap.L().Error("mysql.GetData failed",
            zap.Int64("id", id),
            zap.Error(err))
        return nil, errorx.ErrServerBusy
    }

    // 2. 业务判断
    if data == nil {
        // 业务错误: 直接返回预定义错误
        return nil, errorx.ErrNotFound
    }

    return data, nil
}
```

### Controller 层规范

```go
func Handler(c *gin.Context) {
    // 1. 参数校验 - 使用 ResponseError
    p := new(ParamType)
    if err := c.ShouldBindJSON(p); err != nil {
        ResponseError(c, errorx.CodeInvalidParam)
        return
    }

    // 2. 调用 Logic 层 - 使用 HandleError
    data, err := logic.BusinessLogic(p)
    if err != nil {
        HandleError(c, err)  // 自动处理
        return
    }

    // 3. 返回成功响应
    ResponseSuccess(c, data)
}
```

## 架构优势

1. **单一数据源**: 所有错误码定义在 `pkg/errorx`,避免重复
2. **职责分离**: Logic 层决定错误类型,Controller 层只负责传递
3. **自动化**: HandleError 自动识别错误类型并处理
4. **可维护性**: 新增错误码只需在 errorx 包中定义
5. **一致性**: 统一的错误响应格式

## 迁移历史

### 已完成的优化

1. ✅ 创建 `pkg/errorx` 包
2. ✅ 删除 `pkg/errno` 冗余包
3. ✅ 移除 `controller/code.go` 中的重复定义
4. ✅ 迁移所有 Logic 层使用 errorx
5. ✅ 优化 Controller 层错误处理:
   - 业务逻辑错误改用 `HandleError`
   - 删除不必要的日志记录 (参数校验)
   - 统一使用 `errorx.CodeXXX` 常量

### 代码简化效果

**Before** (旧的错误处理):
```go
// Logic 层
if err != nil {
    zap.L().Error("business failed", zap.Error(err))
    return errors.New("帖子不存在")  // 返回字符串错误
}

// Controller 层 (10+ 行)
if err != nil {
    zap.L().Error("logic.CreatePost failed", zap.Error(err))
    if errors.Is(err,某个特定错误) {
        ResponseError(c, CodeXXX)
        return
    }
    ResponseError(c, CodeServerBusy)
    return
}
```

**After** (新的错误处理):
```go
// Logic 层
if err != nil {
    zap.L().Error("business failed", zap.Error(err))
    return errorx.ErrNotFound  // 返回语义化错误
}

// Controller 层 (1 行!)
if err != nil {
    HandleError(c, err)  // 自动处理所有情况
    return
}
```

## 最佳实践

1. **Logic 层**:
   - 使用预定义的 `errorx.ErrXXX` 返回业务错误
   - 系统错误记录日志后返回 `errorx.ErrServerBusy`
   - 不要返回原始 `errors.New()` 或 DAO 层错误

2. **Controller 层**:
   - 参数校验失败用 `ResponseError`
   - Logic 层错误用 `HandleError`
   - 特殊消息用 `ResponseErrorWithMsg`
   - 不要在 Controller 层记录业务日志

3. **错误码扩展**:
   - 在 `pkg/errorx/errorx.go` 添加新常量
   - 更新 `msgFlags` 映射
   - 定义预设错误实例 `ErrXXX`

## 文件清单

- `pkg/errorx/errorx.go` - 错误定义 (单一数据源)
- `controller/code.go` - 响应函数 (3 个)
- `docs/errorx_usage_guide.md` - 详细使用指南
- `docs/errorx_quick_reference.md` - 快速参考
- `scripts/test_errorx.sh` - 自动化测试脚本
