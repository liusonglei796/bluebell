# CLAUDE.md

这个文件为 Claude Code (claude.ai/code) 在该代码库中工作时提供指导。

## 项目概述

**Bluebell** 是一个基于 Go 的社区论坛后端服务(类似 Reddit),采用 Gin Web 框架和三层架构(Controller-Logic-DAO)设计。核心功能包括用户认证(JWT)、社区管理、帖子发布、Reddit 算法投票系统。

**技术栈**: Gin + MySQL(sqlx) + Redis(go-redis) + Zap 日志 + Viper 配置 + Snowflake ID + JWT 认证

## 常用开发命令

### 启动和编译
```bash
# 开发环境运行(热重载) - 推荐
air

# 直接运行
go run main.go

# 或使用 Makefile
make run

# 编译 Linux 静态二进制
make build

# 格式化代码
make gotool
```

### Swagger 文档
```bash
# 生成/更新 API 文档
swag init

# 访问文档(启动后)
# http://127.0.0.1:8080/swagger/index.html
```

### 数据库
```bash
# MySQL 连接
mysql -u root -p

# Redis 连接
redis-cli
```

## 核心架构设计

### 三层架构原则

**数据流**: HTTP 请求 → Controller → Logic → DAO → MySQL/Redis

```
Controller (控制器层):
  - 职责: 参数解析/校验、调用 Logic、封装响应
  - 位置: controller/*.go
  - 规范: 使用 ShouldBindJSON 绑定参数,统一调用 ResponseSuccess/ResponseError

Logic (业务逻辑层):
  - 职责: 业务规则实现、数据聚合、缓存策略、协调 DAO 层
  - 位置: logic/*.go
  - 规范: 不直接处理 HTTP 请求,返回纯数据或错误

DAO (数据访问层):
  - 职责: 封装 MySQL 查询(dao/mysql/)和 Redis 操作(dao/redis/)
  - 位置: dao/mysql/*.go, dao/redis/*.go
  - 规范: 只负责数据库操作,不包含业务逻辑
```

### 关键设计模式

**1. 认证机制 (JWT + Redis)**
- Access Token: 2小时有效期,用于 API 认证
- Refresh Token: 30天有效期,用于刷新 Access Token
- 单点登录: Redis 存储 active tokens(key: `bluebell:active_access_token:{userID}`)
- 中间件: `middlewares.JWTAuthMiddleware()` 拦截受保护路由

**2. 投票算法 (Reddit 式)**
```
帖子分数 = 发帖时间戳 + (赞成票 - 反对票) × 432

核心常量:
- ScorePerVote = 432 (每票权重)
- OneWeekInSeconds = 7 * 24 * 3600 (投票期限)

Redis 数据结构:
- bluebell:post:score (ZSet, score=分数, member=postID)
- bluebell:post:time (ZSet, score=时间戳, member=postID)
- bluebell:post:voted:{postID} (ZSet, score=投票方向±1, member=userID)
```

**3. N+1 问题优化**
- 问题: 查询帖子列表时,for 循环中查询作者和社区信息导致 N+1 次查询
- 解决: 使用 `sqlx.In()` 批量查询,将查询次数从 1+N+N 降低到 3 次
- 位置: logic/post.go 中的帖子列表查询逻辑

**4. Redis Pipeline 原子性**
- 场景: 投票时需要同时更新 `post:score` ZSet 和 `post:voted:{id}` ZSet
- 实现: 使用 `rdb.TxPipeline()` 确保两个操作要么同时成功,要么同时失败
- 位置: dao/redis/vote.go:VoteForPost()

## 文件结构规范

```
项目根目录/
├── main.go                  # 程序入口,初始化顺序: 配置→日志→MySQL→Redis→路由
├── config.yaml              # 配置文件(开发环境 mode: dev)
├── routers/routers.go       # 路由注册(公共路由 + JWT 认证路由组)
├── controller/              # HTTP 处理函数 + 参数校验 + Swagger 注释
├── logic/                   # 业务逻辑实现
├── dao/mysql/               # MySQL 操作(sqlx)
├── dao/redis/               # Redis 操作(go-redis)
├── models/                  # 数据模型和请求/响应参数
├── middlewares/             # Gin 中间件(JWT 认证)
├── pkg/                     # 公共工具(jwt, snowflake, errno)
├── logger/                  # Zap 日志初始化
└── settings/                # Viper 配置加载
```

## 编码约定

### 命名规范
- 文件名: 小写+下划线 (`user_controller.go`)
- 函数名: 大驼峰(公开)/小驼峰(私有) (`CreatePost` / `getUserID`)
- 常量: 大写+下划线 (`MAX_PAGE_SIZE`)
- Context Key: 驼峰常量 (`CtxUserIDKey`)

### 注释规范(中文)
- 公开函数必须有中文注释
- 复杂逻辑必须解释"为什么"(见 main.go 的初始化注释)
- Swagger 注释必须完整(@Summary, @Tags, @Param, @Success, @Router)

### 错误处理规范
```go
// 1. Controller 层统一使用 ResponseError
if err != nil {
    zap.L().Error("business failed", zap.Error(err))
    ResponseError(c, CodeServerBusy)
    return
}

// 2. Logic 层记录详细日志后返回错误
if err != nil {
    zap.L().Error("dao operation failed",
        zap.Int64("post_id", postID),
        zap.Error(err))
    return err
}

// 3. DAO 层直接返回原始错误
return db.Exec(sqlStr, args...)
```

### 统一响应格式
```go
// 成功
ResponseSuccess(c, data) // {code:1000, msg:"success", data:{...}}

// 失败
ResponseError(c, CodeInvalidParam) // {code:1001, msg:"请求参数错误", data:nil}

// 自定义消息
ResponseErrorWithMsg(c, code, "自定义错误信息")
```

## 关键业务流程

### 1. 添加新接口流程
```
Step 1: models/params.go 定义请求参数结构体(带 validator 标签)
Step 2: dao/mysql/*.go 实现数据库操作(使用 sqlx)
Step 3: logic/*.go 实现业务逻辑(调用 DAO,处理缓存)
Step 4: controller/*.go 添加 Handler(绑定参数,调用 Logic,返回响应)
Step 5: routers/routers.go 注册路由(区分公共/认证路由组)
Step 6: swag init 生成文档
```

### 2. 认证流程
```
注册: SignUpHandler → logic.SignUp → mysql.CheckUserExist + mysql.InsertUser
登录: LoginHandler → logic.Login → mysql.Login(验证密码) → jwt.GenToken(生成双 Token) → redis.SetUserToken(存储到 Redis)
刷新: RefreshTokenHandler → jwt.ParseRefreshToken → jwt.GenToken(生成新 Access Token)
鉴权: JWTAuthMiddleware 解析 Authorization Header → jwt.ParseToken → c.Set(CtxUserIDKey, userID)
```

### 3. 投票流程
```
请求: POST /api/v1/vote {post_id, direction}
Controller: PostVoteHandler 解析参数 → 获取 userID(从 Context)
Logic: VoteForPost 转换参数格式
DAO: redis.VoteForPost
  1. 检查投票时限(7天)
  2. 查询旧投票记录
  3. 计算分数变化 = |新值-旧值| × 432
  4. Pipeline 更新 post:score ZSet 和 post:voted:{id} ZSet
```

### 4. 帖子列表排序
```
Logic: GetPostList(order="time"|"score")
  1. 从 Redis ZSet 获取帖子 ID 列表(按时间/分数排序)
  2. 批量查询帖子详情(MySQL, 使用 sqlx.In 避免 N+1)
  3. 批量查询作者信息(MySQL)
  4. 批量查询社区信息(MySQL)
  5. 拼接完整数据返回
```

## Redis 键命名规范

```
用户 Token:
  bluebell:active_access_token:{userID}
  bluebell:active_refresh_token:{userID}

帖子排序:
  bluebell:post:time (ZSet, score=时间戳)
  bluebell:post:score (ZSet, score=分数)

投票记录:
  bluebell:post:voted:{postID} (ZSet, member=userID, score=±1)
```

## 依赖包使用说明

- `github.com/gin-gonic/gin`: Web 框架,使用 `c.ShouldBindJSON` 绑定参数
- `github.com/jmoiron/sqlx`: MySQL 操作,使用 `db.Get/Select/Exec`,支持 `sqlx.In` 批量查询
- `github.com/redis/go-redis/v9`: Redis 客户端,使用 `ZAdd/ZScore/ZRange` 操作有序集合
- `go.uber.org/zap`: 结构化日志,使用 `zap.L().Info/Error` 记录日志
- `github.com/spf13/viper`: 配置管理,自动读取 config.yaml
- `github.com/bwmarrin/snowflake`: 分布式 ID 生成器,`snowflake.GenID()`
- `github.com/golang-jwt/jwt/v5`: JWT Token 生成和解析
- `github.com/go-playground/validator/v10`: 请求参数校验(struct tag)

## 开发注意事项

1. **配置文件敏感信息**: config.yaml 包含数据库密码,请勿提交到公开仓库
2. **优雅关机**: main.go 已实现 5 秒超时优雅关机,不要直接 kill -9
3. **日志刷新**: main.go 使用 `defer zap.L().Sync()` 确保日志写入磁盘
4. **JWT 中间件**: 所有需要认证的路由必须放在 `authGroup` 组下
5. **Redis 缓存降级**: 帖子创建时 Redis 失败不影响主流程(见 logic/post.go:38-44)
6. **投票时限**: 超过 7 天的帖子不允许投票,返回 `ErrVoteTimeExpire`
7. **Pipeline 原子性**: 涉及多个 Redis 操作的场景(如投票)必须使用 Pipeline
8. **N+1 优化**: 批量查询时使用 `sqlx.In()` 而非循环查询

## 测试和调试

```bash
# 查看日志
tail -f bluebell.log

# 测试注册
curl -X POST http://127.0.0.1:8080/api/v1/signup \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"123456","re_password":"123456"}'

# 测试登录
curl -X POST http://127.0.0.1:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"123456"}'

# 测试认证接口(需要替换 TOKEN)
curl -H "Authorization: Bearer {ACCESS_TOKEN}" \
  http://127.0.0.1:8080/api/v1/community
```

## 常见问题排查

- **启动失败**: 检查 MySQL/Redis 是否启动,config.yaml 配置是否正确
- **Swagger 不显示**: 运行 `swag init`,确保 main.go 导入了 `_ "bluebell/docs"`
- **JWT 认证失败**: 检查 Token 是否过期,Header 格式是否为 `Bearer {token}`
- **投票失败**: 检查帖子是否超过 7 天,用户是否重复投票
- **热重载不生效**: 检查 .air.conf 是否排除了 tmp 和日志文件
