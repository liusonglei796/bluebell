# Handler 层 Mock 测试与文档

本文档介绍如何在 `bluebell` 项目中为 Handler 层编写依赖 Mock 的单元测试。这里以 `user_handler` 为例，展示如何隔离 Service 层依赖进行测试。

## 1. 为什么需要 Mock？

在分层架构（Handler -> Service -> Repository）中，Handler 的职责主要是：
1. 接收和解析外部（HTTP）请求参数。
2. 参数校验。
3. 调用下层 Service 层的业务逻辑。
4. 格式化组装响应并返回。

如果在测试 Handler 时，直接调用真实的 Service 层，就会变成集成测试，还需要准备数据库、Redis 缓存等各种外部资源。这使得测试变慢，且不够稳定。
因此，我们可以通过模拟（Mock）Service 层的行为，专门测试 Handler 是否正确解析了参数，是否以预期的方式调用了 Service，并能否返回正确的 HTTP 响应。

## 2. 依赖注入与接口

在 `user_handler/handler.go` 中，`Handler` 结构体包含了一个 `svcdomain.UserService` 的接口：

```go
type Handler struct {
	userService svcdomain.UserService
}

func New(userService svcdomain.UserService) *Handler { ... }
```

得益于这里使用的是**接口**，我们在测试时可以传入一个自己实现的伪造对象（Mock），只要这个对象实现了 `svcdomain.UserService` 的三个方法（`SignUp`, `Login`, `RefreshToken`）即可。

## 3. 我们做了什么？

在 `internal/handler/user_handler/handler_test.go` 中：

### 3.1 创建 Mock 对象
我们手动实现了一个 `MockUserService`：

```go
type MockUserService struct {
	SignUpFunc       func(ctx context.Context, p *userreq.SignUpRequest) error
	LoginFunc        func(ctx context.Context, p *userreq.LoginRequest) (string, string, error)
	RefreshTokenFunc func(ctx context.Context, p *userreq.RefreshTokenRequest) (string, string, error)
}
```

针对业务方法如 `SignUp`，它会直接调用对应的闭包函数，这样我们在不同的测试用例里就能动态注入不同的返回值（例如：模拟成功、模拟账号已存在等错误）。

### 3.2 编写测试用例
我们利用表驱动测试（Table-Driven Tests）编写了 `TestSignUpHandler`：
- 使用 `gin.SetMode(gin.TestMode)` 消除运行时的日志噪音。
- 使用 `httptest.NewRecorder()` 记录 HTTP 响应。
- 通过 `bytes.NewBuffer` 构造带有 JSON 的 POST 请求 `http.NewRequest`。
- 测试了**正常注册**和**参数缺失导致校验失败**两种不同场景。

### 3.3 运行测试
执行普通的 Go 测试命令即可验证 Handler 层逻辑，不再依赖任何外部组件。

```bash
cd internal/handler/user_handler
go test -v .
```

你也可以选择在整个项目根路径下运行：
```bash
go test -v ./internal/handler/user_handler
```
