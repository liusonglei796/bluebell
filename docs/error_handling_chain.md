# 接口错误处理链路

## 全链路图

```
┌─────────────────────────────────────────────────────────────────┐
│                      客户端 (HTTP Response)                       │
│          {"code": 2001, "msg": "need login", "data": null}       │
└───────────────────────────┬─────────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────────┐
│  render 层 (internal/interfaces/http/render/)                    │
│                                                                  │
│  HandleError(c, err) ───┬── classifyError(err)  → HTTP 状态码    │
│                         │── codeFromError(err)  → 业务状态码      │
│                         │── metrics.RecordError → 监控打点        │
│                         └── Response{Code, Msg: err.Error(),     │
│                                              Data: nil}          │
│                                                                  │
│  HandleFail(c, 1001, msg) → 直接指定业务码+消息                   │
│     用于 validator 翻译错误等自定义 msg 场景                       │
│                                                                  │
│  HandleSuccess(c, data) → {code:200, msg:"success", data: ...}   │
└───────────────────────────┬─────────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────────┐
│  handler 层 (internal/interfaces/http/handler/)                   │
│                                                                  │
│  统一调用规则：                                                    │
│  ├─ 成功 → render.HandleSuccess(c, data)                         │
│  ├─ 业务错误 → render.HandleError(c, err)                        │
│  │  (err 来自 service/DAO 原样上抛)                               │
│  ├─ 未登录 → render.HandleError(c, entity.ErrNeedLogin)          │
│  ├─ 参数绑定失败(bind error) → render.HandleError(c, err)        │
│  └─ validator翻译错误 → render.HandleFail(c, CodeInvalidParam,   │
│                          translate.RemoveTopStruct(...))          │
└───────────────────────────┬─────────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────────┐
│  service/application 层 (internal/application/)                   │
│                                                                  │
│  repo 返回的 error 原路透传，不吞掉（仅记日志）                     │
│                                                                  │
│  例: CreateBookmark                                              │
│    postRepo.GetPostByID → err || nil                             │
│    if post == nil → return entity.ErrInvalidParam                │
│    bookmarkRepo.CreateBookmark → err || nil                      │
└───────────────────────────┬─────────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────────┐
│  DAO 层 (internal/infrastructure/persistence/mysql/)              │
│                                                                  │
│  GORM 错误: fmt.Errorf("create bookmark failed: %w", err)        │
│  使用 %w 包裹，支持 errors.Is/errors.As 向上穿透                   │
└───────────────────────────┬─────────────────────────────────────┘
                            │
┌───────────────────────────▼─────────────────────────────────────┐
│  领域层 (internal/domain/entity/errors.go)                        │
│                                                                  │
│  sentinel 错误定义（所有错误源头）：                                 │
│  ├─ ErrNeedLogin     = "need login"          → 2001              │
│  ├─ ErrInvalidToken  = "invalid token"       → 2002              │
│  ├─ ErrUnauthorized  = "unauthorized"        → 2003              │
│  ├─ ErrForbidden     = "forbidden operation" → 2004              │
│  ├─ ErrInvalidParam  = "invalid parameters"  → 1001              │
│  ├─ ErrNotFound      = "entity not found"    → 3001              │
│  ├─ ErrDuplicate     = "entity already exists" → 3002            │
│  ├─ ErrServerBusy    = "server is busy"      → 1002              │
│  └─ ErrRateLimitExceeded = "rate limit exceeded" → 1003          │
│                                                                  │
│  Wrap(sentinel, cause) 函数：用 wrappedError 包裹底层错误，        │
│  Error() 只显示 sentinel 消息，errors.Is() 匹配 sentinel          │
└─────────────────────────────────────────────────────────────────┘
```

## 三要素

每个 API 响应统一格式为：

```json
{
  "code": 200,
  "msg": "success",
  "data": {}
}
```

| 字段 | 说明 | 成功时 | 失败时 |
|------|------|--------|--------|
| `code` | 业务状态码（非 HTTP 状态码） | 200 | 1001/2001/3001 等 |
| `msg`  | 提示信息 | `"success"` | `err.Error()` 或翻译后的错误 map |
| `data` | 业务数据 | 具体数据 | `null` |

## 三类典型路径

### 路径 1：业务逻辑错误（ErrNeedLogin）

```
ctx.Get("UserIDKey") 不存在
  → handler: render.HandleError(c, entity.ErrNeedLogin)
    → classifyError  → HTTP 401
    → codeFromError → CodeNeedLogin = 2001
    → Response{Code:2001, Msg:"need login", Data:nil}
```

**相关文件：** `bookmark_handler/handler.go` 第 33 行、`community_handler/handler.go` 第 77 行等

### 路径 2：GORM 数据库错误

```
bookmarkRepo.CreateBookmark → db.Create 失败 (UNIQUE 约束等)
  → 返回 fmt.Errorf("create bookmark failed: %w", err)
  → service 原路透传
  → handler: render.HandleError(c, err)
    → classifyError: default → HTTP 500
    → codeFromError: default → CodeServerBusy = 1002
    → Response{Code:1002, Msg:"create bookmark failed: Error 1062 ...", Data:nil}
```

**相关文件：** DAO 层 `bookmarkdb/bookmark.go` 第 51 行、service 层 `bookmark_service.go` 第 75 行

### 路径 3：validator 翻译错误

```
ShouldBindJSON 失败 → validator.ValidationErrors
  → errors.As 匹配成功 → translate.Translate
  → handler: render.HandleFail(c, CodeInvalidParam, translatedErrs)
    → HTTP 400（硬编码）
    → Response{Code:1001, Msg:{"title":"标题为必填字段"}, Data:nil}
```

**相关文件：** `community_handler/handler.go` 第 54-55 行、`post_handler/handler.go` 多处

## 关键设计要点

### 双码分离

错误处理中有两层状态码，分别服务于不同目的：

| 层级 | 由谁决定 | 示例 | 用途 |
|------|----------|------|------|
| **HTTP 状态码** | `classifyError(err)` | 400/401/403/409/500 | 浏览器/网关/代理识别 |
| **业务状态码** | `codeFromError(err)` | 1001/2001/3001/200 | 前端逻辑判断 |

`classifyError` 和 `codeFromError` 都位于 `response.go`（第 24-72 行），分别对同一个 `err` 做 switch，生成不同的码。

### sentinel 穿透

- 所有业务错误定义为 `entity.ErrXXX` sentinel 错误（标准库 `errors.New`）
- 各层（DAO/service/handler）都不吞掉原始错误，原路透传
- `response.go` 中用 `errors.Is(err, entity.ErrXXX)` 做匹配，DAI 层的 `%w` 包裹不破坏链
- 兜底：未匹配到的错误走 `default` → HTTP 500 + CodeServerBusy(1002)

### HandleFail 旁路

`HandleFail` 不走 `codeFromError` 映射。它专门用于非领域错误场景（如 validator 翻译），因为这不是业务逻辑错误，而是参数格式问题。调用方直接指定业务码和消息。

### 错误码分类

| 范围 | 含义 | 错误码 |
|------|------|--------|
| 200 | 成功 | CodeSuccess |
| 1xxx | 通用错误 | 1001 参数错误、1002 服务繁忙、1003 限流 |
| 2xxx | 认证授权 | 2001 未登录、2002 令牌无效、2003 未授权、2004 无权限 |
| 3xxx | 资源错误 | 3001 不存在、3002 冲突/重复 |

### 业务状态码 ↔ sentinel 映射表

| 业务码 | 常量 | sentinel 错误 | HTTP 状态码 |
|--------|------|---------------|------------|
| 200    | CodeSuccess | - | 200 |
| 1001   | CodeInvalidParam | `ErrInvalidParam` | 400 |
| 1002   | CodeServerBusy | `ErrServerBusy` | 503 |
| 1003   | CodeRateLimit | `ErrRateLimitExceeded` | 429 |
| 2001   | CodeNeedLogin | `ErrNeedLogin` | 401 |
| 2002   | CodeInvalidToken | `ErrInvalidToken` | 401 |
| 2003   | CodeUnauthorized | `ErrUnauthorized` / `ErrNotLogin` | 401 |
| 2004   | CodeForbidden | `ErrForbidden` | 403 |
| 3001   | CodeNotFound | `ErrNotFound` | 404 |
| 3002   | CodeConflict | `ErrDuplicate` / `ErrUserExist` / `ErrVoteRepeated` / `ErrVoteTimeExpire` | 409 |

## 相关文件

| 文件 | 职责 |
|------|------|
| `internal/domain/entity/errors.go` | 定义所有 sentinel 错误 + Wrap 函数 |
| `internal/interfaces/http/render/code.go` | 定义业务状态码常量 + msg 映射 |
| `internal/interfaces/http/render/response.go` | HandleSuccess / HandleError / HandleFail + classifyError / codeFromError |
| `internal/interfaces/http/handler/*/handler.go` | 统一调用 render 函数，不再直接 c.JSON |
