# 第11章:集成Swagger自动生成文档

> **本章导读**
>
> 编写接口文档是后端开发中最枯燥的工作之一。手动维护 Markdown 或 Word 文档不仅费时,还容易与代码脱节(代码改了,文档忘改)。
>
> 本章将介绍 **Swagger** —— 业界标准的 RESTful API 文档工具。通过在代码中写注释,就能自动生成美观、可交互、实时同步的在线接口文档。

---

## 📚 本章目标

学完本章,你将掌握:

1. 理解 Swagger/OpenAPI 规范及其在现代API开发中的价值
2. 安装 `swag` 命令行工具和 `gin-swagger` 库
3. 编写通用 API 信息注释(Title, Version, Host, Security)
4. 编写接口级注释(Summary, Param, Success, Failure)
5. 在 Gin 路由中集成 Swagger UI
6. 定义复杂响应模型和安全认证
7. 集成到开发工作流(Makefile, Git hooks)
8. 解决常见的文档生成错误和性能优化

---

## 1. 为什么需要 Swagger?

### 1.1 传统文档维护的痛点

在没有 Swagger 之前,我们通常这样维护 API 文档:

**❌ 手写 Markdown 文档**
```markdown
### POST /api/v1/login
请求参数:
- username: 用户名 (string, required)
- password: 密码 (string, required)

响应示例:
{
  "code": 1000,
  "msg": "success",
  "data": {
    "access_token": "eyJhbGc...",
    "refresh_token": "eyJhbGc..."
  }
}
```

**问题:**
1. 代码改了,忘记同步文档 → 文档过期
2. 没有参数校验规则说明 → 前端试错成本高
3. 无法直接测试接口 → 需要 Postman/curl
4. 响应格式变化 → 手动更新多处文档

**✅ Swagger 自动生成**
```go
// LoginHandler 处理用户登录请求
// @Summary 用户登录
// @Tags 用户相关
// @Param object body models.ParamLogin true "登录参数"
// @Success 200 {object} _ResponseLogin
// @Router /login [post]
func LoginHandler(c *gin.Context) {
    // ...
}
```

**优势:**
- 📝 **代码即文档**: 注释写在函数旁边,改代码时自然会更新
- 🔄 **实时同步**: `swag init` 一键更新,永不过期
- 🧪 **在线测试**: Swagger UI 支持直接发送请求
- 🌍 **全球通用**: OpenAPI 是行业标准,工具链丰富(Postman 可直接导入)

### 1.2 什么是 OpenAPI 规范?

**Swagger** 是一个工具集,而 **OpenAPI** 是它背后的规范标准。

```
OpenAPI 规范 (YAML/JSON)
       ↓
Swagger Tools (swag, gin-swagger)
       ↓
交互式文档 (Swagger UI)
```

**OpenAPI 2.0 vs 3.0**:
| 特性 | OpenAPI 2.0 (Swagger 2.0) | OpenAPI 3.0 |
|------|--------------------------|-------------|
| **格式** | JSON, YAML | JSON, YAML |
| **参数定义** | 分散在 `parameters` | 统一在 `requestBody` |
| **响应定义** | 简单 | 支持多种 MIME 类型 |
| **安全** | 简单 | 更灵活的安全方案 |
| **工具支持** | 广泛 | 逐渐普及 |

Bluebell 项目使用的是 **OpenAPI 2.0**(swag 默认),足够满足需求。

---

## 2. 环境准备

Swagger 在 Go 中主要由三部分组成:
1. **CLI 工具 (swag)**: 扫描代码注释生成 `docs` 文件夹
2. **中间件 (gin-swagger)**: 在 Gin 中提供 Web 页面服务
3. **静态资源 (files)**: Swagger UI 的前端文件

### 2.1 安装

在终端执行以下命令:

```bash
# 1. 安装 swag 命令行工具 (用于生成文档)
go install github.com/swaggo/swag/cmd/swag@latest

# 2. 下载 Gin 适配库
go get -u github.com/swaggo/gin-swagger
go get -u github.com/swaggo/files
```

**验证安装:**
```bash
swag -v
# 输出: swag version v1.8.12
```

**如果 `swag` 命令找不到**:
确保 `$GOPATH/bin` 在 `$PATH` 中:
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### 2.2 工作原理

```
1. 编写注释 → controller/*.go
        ↓
2. swag init → 扫描代码生成 docs/
        ↓
3. import _ "bluebell/docs" → 注册文档数据到内存
        ↓
4. r.GET("/swagger/*any", ...) → 启动 Web UI
        ↓
5. 浏览器访问 http://localhost:8080/swagger/index.html
```

---

## 3. 全局配置注释

### 3.1 main.go 中的通用信息

在 `main.go` 的 `main` 函数**上方**添加全局配置注释。

```go
package main

// @title bluebell项目接口文档
// @version 1.0
// @description Go语言实战项目——社区web框架

// @contact.name 技术支持
// @contact.url http://www.bluebell.com/support
// @contact.email support@bluebell.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host 127.0.0.1:8080
// @BasePath /api/v1

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

func main() {
    // ...
}
```

### 3.2 注解详解

| 注解 | 说明 | 示例 |
|------|------|------|
| `@title` | API 文档标题 | `bluebell项目接口文档` |
| `@version` | API 版本号 | `1.0` |
| `@description` | API 描述信息 | `Go语言实战项目——社区web框架` |
| `@termsOfService` | 服务条款链接 | `http://swagger.io/terms/` |
| `@contact.name` | 联系人姓名 | `技术支持` |
| `@contact.email` | 联系人邮箱 | `support@bluebell.com` |
| `@license.name` | 许可证名称 | `Apache 2.0` |
| `@host` | API 服务地址 | `127.0.0.1:8080` (不含 `http://`) |
| `@BasePath` | API 基础路径 | `/api/v1` |
| `@securityDefinitions` | 安全定义 | JWT 认证方案 |

### 3.3 安全定义 (Security Definitions)

**为什么需要?**
- 让 Swagger UI 知道这个 API 需要认证
- 提供 "Authorize" 按钮,输入一次 Token 全局生效
- 自动在请求头中添加 `Authorization: Bearer <token>`

**JWT 认证定义:**
```go
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
```

**解释:**
- `apikey`: 认证类型(其他类型: `basic`, `oauth2`)
- `ApiKeyAuth`: 安全方案名称(自定义,在接口注释中引用)
- `@in header`: Token 位置(header/query/cookie)
- `@name Authorization`: Header 字段名

---

## 4. 接口注释

### 4.1 基本示例 - 用户注册

在 `controller/user.go` 中:

```go
// SignUpHandler 处理用户注册请求
// @Summary 用户注册
// @Description 用户注册接口,支持用户名密码注册
// @Tags 用户相关
// @Accept application/json
// @Produce application/json
// @Param object body models.ParamSignUp true "注册参数"
// @Success 200 {object} ResponseData "注册成功"
// @Failure 1005 {object} ResponseData "用户已存在"
// @Failure 1007 {object} ResponseData "参数错误"
// @Router /signup [post]
func SignUpHandler(c *gin.Context) {
    // ...
}
```

### 4.2 注解完整说明

| 注解 | 必填 | 说明 | 示例 |
|------|------|------|------|
| `@Summary` | ✅ | 接口简述(显示在列表) | `用户注册` |
| `@Description` | ❌ | 接口详细描述 | `用户注册接口,支持用户名密码注册` |
| `@Tags` | ✅ | 接口分组 | `用户相关` (同组接口会折叠在一起) |
| `@Accept` | ✅ | 请求 Content-Type | `application/json` |
| `@Produce` | ✅ | 响应 Content-Type | `application/json` |
| `@Param` | ✅ | 请求参数 | 见下方详解 |
| `@Success` | ✅ | 成功响应 | `200 {object} ResponseData` |
| `@Failure` | ❌ | 失败响应 | `1005 {object} ResponseData` |
| `@Router` | ✅ | 路由路径和方法 | `/signup [post]` |
| `@Security` | ❌ | 需要认证 | `ApiKeyAuth` |

### 4.3 @Param 参数详解

**格式:**
```
@Param 参数名 参数位置 数据类型 是否必填 "描述信息" 其他属性(可选)
```

**参数位置 (paramType):**
- `path`: URL 路径参数 (`/post/:id` 中的 `:id`)
- `query`: URL 查询参数 (`/posts?page=1` 中的 `page`)
- `header`: HTTP Header (`Authorization`)
- `body`: 请求体 (JSON)
- `formData`: 表单数据 (multipart/form-data)

**数据类型 (dataType):**
- 基本类型: `string`, `int`, `integer`, `number`, `bool`, `boolean`
- 结构体类型: `models.ParamSignUp`(引用 Go 结构体)
- 数组类型: `[]string`, `array`

**示例:**

```go
// 1. Body 参数 (JSON 对象)
// @Param object body models.ParamSignUp true "注册参数"

// 2. URL 路径参数
// @Param id path string true "帖子ID"

// 3. Query 参数
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(10)

// 4. Header 参数
// @Param Authorization header string true "Bearer Token"

// 5. 数组参数
// @Param community_ids query []int false "社区ID列表" collectionFormat(multi)
```

### 4.4 需要认证的接口

对于需要 JWT 认证的接口,添加 `@Security` 注解:

```go
// CreatePostHandler 创建帖子
// @Summary 创建帖子
// @Description 创建帖子接口
// @Tags 帖子相关
// @Accept application/json
// @Produce application/json
// @Security ApiKeyAuth
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body models.ParamPost true "创建帖子参数"
// @Success 200 {object} ResponseData
// @Router /post [post]
func CreatePostHandler(c *gin.Context) {
    // ...
}
```

**效果:**
- 接口名称右侧会显示 🔒 锁图标
- Swagger UI 右上角出现 "Authorize" 按钮
- 点击按钮输入 Token 后,所有请求自动携带

### 4.5 路径参数示例

```go
// GetPostDetailHandler 获取帖子详情
// @Summary 获取帖子详情
// @Description 根据帖子ID获取详情
// @Tags 帖子相关
// @Accept application/json
// @Produce application/json
// @Security ApiKeyAuth
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path string true "帖子ID"
// @Success 200 {object} ResponseData{data=models.ApiPostDetail} "成功"
// @Failure 1007 {object} ResponseData "参数错误"
// @Failure 1004 {object} ResponseData "请先登录"
// @Router /post/{id} [get]
func GetPostDetailHandler(c *gin.Context) {
    // ...
}
```

**注意:**
- 路径参数使用 `{id}` 包裹(不是 `:id`)
- `@Param id path` 必须与路由中的 `{id}` 名称一致

### 4.6 Query 参数示例

```go
// GetPostListHandler 获取帖子列表
// @Summary 获取帖子列表
// @Description 支持按时间/分数排序、分页、社区过滤
// @Tags 帖子相关
// @Accept application/json
// @Produce application/json
// @Security ApiKeyAuth
// @Param Authorization header string true "Bearer 用户令牌"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(10) minimum(1) maximum(100)
// @Param order query string false "排序方式" Enums(time, score) default(time)
// @Param community_id query int false "社区ID"
// @Success 200 {object} ResponseData{data=[]models.ApiPostDetail}
// @Router /posts [get]
func GetPostListHandler(c *gin.Context) {
    // ...
}
```

**高级属性:**
- `default(1)`: 默认值
- `minimum(1)`: 最小值
- `maximum(100)`: 最大值
- `Enums(time, score)`: 枚举值
- `collectionFormat(multi)`: 数组格式(`?ids=1&ids=2&ids=3`)

---

## 5. 定义响应模型

### 5.1 为什么需要专门的响应模型?

在实际代码中,我们可能使用 `gin.H` 返回动态数据:
```go
ResponseSuccess(c, gin.H{"list": posts, "total": total})
```

但 Swagger 无法解析 `gin.H`,导致文档中响应结构显示为 `{}` (空对象)。

**解决方案:**
定义专门的结构体用于 Swagger 文档展示(实际代码可以不用它)。

### 5.2 创建文档模型文件

新建 `controller/docs_models.go`:

```go
package controller

import "bluebell/models"

// 以下结构体仅用于 Swagger 文档生成,实际代码不直接使用

// _ResponseLogin 登录响应
type _ResponseLogin struct {
	Code int    `json:"code"`    // 业务状态码
	Msg  string `json:"msg"`     // 提示信息
	Data struct {
		AccessToken  string `json:"access_token"`  // 访问令牌
		RefreshToken string `json:"refresh_token"` // 刷新令牌
	} `json:"data"`
}

// _ResponsePostList 帖子列表响应
type _ResponsePostList struct {
	Code int                     `json:"code"` // 业务状态码
	Msg  string                  `json:"msg"`  // 提示信息
	Data []*models.ApiPostDetail `json:"data"` // 帖子列表
}

// _ResponseCommunityList 社区列表响应
type _ResponseCommunityList struct {
	Code int                `json:"code"` // 业务状态码
	Msg  string             `json:"msg"`  // 提示信息
	Data []*models.Community `json:"data"` // 社区列表
}
```

### 5.3 在注释中引用

```go
// LoginHandler 处理用户登录请求
// @Summary 用户登录
// @Tags 用户相关
// @Param object body models.ParamLogin true "登录参数"
// @Success 200 {object} _ResponseLogin "登录成功"
// @Router /login [post]
func LoginHandler(c *gin.Context) {
    // ...
}
```

### 5.4 内联定义响应字段

如果只想在某个接口使用,可以用内联语法:

```go
// @Success 200 {object} ResponseData{data=models.ApiPostDetail} "成功"
```

**解释:**
- `ResponseData`: 外层统一响应结构
- `{data=models.ApiPostDetail}`: `data` 字段类型为 `models.ApiPostDetail`

**生成的 JSON 结构:**
```json
{
  "code": 1000,
  "msg": "success",
  "data": {
    "post_id": 123,
    "title": "帖子标题",
    "content": "帖子内容",
    ...
  }
}
```

---

## 6. 生成文档

### 6.1 运行 swag init

在项目根目录下执行:

```bash
swag init
```

**输出:**
```
2024/01/15 10:30:12 Generate swagger docs....
2024/01/15 10:30:12 Generate general API Info, search dir:./
2024/01/15 10:30:13 Generating models.ParamSignUp
2024/01/15 10:30:13 Generating models.ParamLogin
2024/01/15 10:30:13 create docs.go at docs/docs.go
2024/01/15 10:30:13 create swagger.json at docs/swagger.json
2024/01/15 10:30:13 create swagger.yaml at docs/swagger.yaml
```

**生成的文件:**
```
docs/
├── docs.go         # Go 代码,包含文档数据 (需 import)
├── swagger.json    # JSON 格式的 OpenAPI 文档
└── swagger.yaml    # YAML 格式的 OpenAPI 文档
```

### 6.2 ⚠️ 重要提醒

**每次修改注释后,都必须重新运行 `swag init`!**

否则 Swagger UI 显示的是旧文档,你会发现:
- 新增的接口不显示
- 修改的参数没生效
- 删除的接口还在

**开发流程:**
```bash
1. 修改 controller/*.go 注释
2. swag init           # 重新生成文档
3. go run main.go      # 重启服务
4. 刷新浏览器页面
```

### 6.3 集成到 Makefile

为了避免忘记生成文档,可以在 Makefile 中添加:

```makefile
.PHONY: swag
swag:
	swag init

.PHONY: run
run: swag
	go run main.go

.PHONY: build
build: swag
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bluebell
```

**使用:**
```bash
make run    # 自动生成文档并运行
make build  # 自动生成文档并构建
```

---

## 7. 集成到 Gin

### 7.1 引入 docs 包

在 `routers/routers.go` 中:

```go
package routers

import (
	"bluebell/controller"
	"bluebell/internal/infrastructure/logger"
	"bluebell/middlewares"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "bluebell/docs" // ⚠️ 匿名导入,触发 docs.go 的 init() 函数
)
```

**为什么要匿名导入?**

`docs/docs.go` 中包含 `init()` 函数:
```go
func init() {
	// 注册文档数据到 swag 内部的全局变量
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
```

如果不导入 `docs` 包,`init()` 不会执行,导致 Swagger UI 显示 "Failed to load API definition"。

### 7.2 注册 Swagger UI 路由

在 `SetupRouter` 函数中:

```go
func SetupRouter(mode string) *gin.Engine {
	if mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(logger.GinLogger(), logger.GinRecovery(true))

	// ========== Swagger 文档路由 ==========
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ========== API 路由组 ==========
	v1 := r.Group("/api/v1")
	// 公开路由
	v1.POST("/signup", controller.SignUpHandler)
	v1.POST("/login", controller.LoginHandler)

	// 需认证路由
	v1.Use(middlewares.JWTAuthMiddleware())
	{
		v1.POST("/post", controller.CreatePostHandler)
		// ...
	}

	return r
}
```

**路由说明:**
- 路径: `/swagger/*any` (固定写法)
- Handler: `ginSwagger.WrapHandler(swaggerFiles.Handler)`
- 效果: 访问 `/swagger/index.html` 显示文档页面

### 7.3 生产环境禁用 Swagger

Swagger UI 会暴露所有接口细节,生产环境应该禁用:

```go
func SetupRouter(mode string) *gin.Engine {
	r := gin.New()

	// 仅在开发/测试环境启用 Swagger
	if mode != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// ...
	return r
}
```

**或者使用环境变量:**
```go
import "os"

if os.Getenv("ENABLE_SWAGGER") == "true" {
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
```

---

## 8. 验证与测试

### 8.1 启动服务

```bash
go run main.go
```

### 8.2 访问 Swagger UI

打开浏览器,访问:
```
http://localhost:8080/swagger/index.html
```

你将看到一个漂亮的蓝色文档页面:
- 顶部: API 标题、版本、描述
- 左侧: 接口分组(Tags)
- 右侧: 接口列表

### 8.3 测试接口

**1. 测试公开接口 (如 /signup)**

点击 **POST /api/v1/signup**:
1. 点击 "Try it out" 按钮
2. 修改 Request body 参数:
   ```json
   {
     "username": "testuser",
     "password": "123456",
     "re_password": "123456"
   }
   ```
3. 点击 "Execute" 按钮
4. 查看 Response:
   ```json
   {
     "code": 1000,
     "msg": "success",
     "data": null
   }
   ```

**2. 测试需认证的接口 (如 /post)**

首先需要登录获取 Token:
1. 测试 **POST /api/v1/login**,复制返回的 `access_token`
2. 点击页面右上角 🔒 **Authorize** 按钮
3. 在弹窗中输入: `Bearer eyJhbGc...` (注意 `Bearer ` 前缀)
4. 点击 "Authorize",关闭弹窗
5. 现在所有接口请求都会自动携带 Token

### 8.4 导出文档

Swagger UI 支持导出为多种格式:

**方法1: 下载 JSON**
访问: `http://localhost:8080/swagger/doc.json`

**方法2: 使用 swagger-codegen 生成客户端代码**
```bash
# 安装 swagger-codegen
brew install swagger-codegen  # macOS
# 或
npm install -g swagger-codegen

# 生成 Python 客户端
swagger-codegen generate -i http://localhost:8080/swagger/doc.json \
  -l python -o ./client/python

# 生成 TypeScript (Axios) 客户端
swagger-codegen generate -i http://localhost:8080/swagger/doc.json \
  -l typescript-axios -o ./client/typescript
```

---

## 9. 高级特性

### 9.1 自定义 Swagger UI 配置

如果想修改 Swagger UI 的行为(如默认展开、深色主题等):

```go
import ginSwagger "github.com/swaggo/gin-swagger"

// 自定义配置
url := ginSwagger.URL("http://localhost:8080/swagger/doc.json") // 指定文档 URL
r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
```

**常用配置:**
```go
ginSwagger.DefaultModelsExpandDepth(-1)  // 隐藏 Models 部分
ginSwagger.DocExpansion("none")          // 默认折叠所有接口
ginSwagger.DeepLinking(true)             // 启用深度链接(可分享特定接口 URL)
```

### 9.2 多环境文档

如果你的 API 部署在多个环境(dev, test, prod),可以动态修改 `@host`:

**方法1: 使用 swag 命令参数**
```bash
swag init --parseDependency --parseInternal --host dev.bluebell.com
```

**方法2: 运行时动态修改**
```go
import "bluebell/docs"

func main() {
	// ...
	// 根据环境变量修改文档 Host
	if host := os.Getenv("SWAGGER_HOST"); host != "" {
		docs.SwaggerInfo.Host = host
	}
	// ...
}
```

### 9.3 分组管理(多个 API 版本)

如果你的项目有多个 API 版本(如 v1, v2):

```go
// main.go
// @title Bluebell API V1
// @version 1.0
// @BasePath /api/v1

// @title Bluebell API V2
// @version 2.0
// @BasePath /api/v2
```

**生成多份文档:**
```bash
swag init -g main.go --instanceName v1 -o docs/v1
swag init -g main_v2.go --instanceName v2 -o docs/v2
```

**注册多个路由:**
```go
import (
	_ "bluebell/docs/v1"
	_ "bluebell/docs/v2"
)

r.GET("/swagger/v1/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
	ginSwagger.InstanceName("v1")))
r.GET("/swagger/v2/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
	ginSwagger.InstanceName("v2")))
```

### 9.4 添加示例值

为了让文档更友好,可以为参数添加示例值:

```go
// ParamSignUp 用户注册参数
type ParamSignUp struct {
	Username   string `json:"username" binding:"required" example:"zhangsan"` // 示例: zhangsan
	Password   string `json:"password" binding:"required" example:"123456"`
	RePassword string `json:"re_password" binding:"required,eqfield=Password" example:"123456"`
}
```

Swagger UI 会自动填充这些示例值。

### 9.5 文件上传接口

```go
// UploadAvatarHandler 上传头像
// @Summary 上传头像
// @Tags 用户相关
// @Accept multipart/form-data
// @Produce application/json
// @Param avatar formData file true "头像文件"
// @Success 200 {object} ResponseData{data=string} "返回图片URL"
// @Router /upload/avatar [post]
func UploadAvatarHandler(c *gin.Context) {
	file, _ := c.FormFile("avatar")
	// ...
}
```

---

## 10. 常见问题与解决方案

### 10.1 文档生成失败

**问题1: `cannot find package "bluebell/docs"`**

**原因:** 忘记运行 `swag init`

**解决:**
```bash
swag init
```

---

**问题2: `ParseComment error...`**

**原因:** 注释格式错误

**常见错误格式:**
```go
// ❌ 错误: @Param 参数之间少空格
// @Param objectbody models.ParamSignUp true "注册参数"

// ✅ 正确: 每个部分用空格分隔
// @Param object body models.ParamSignUp true "注册参数"
```

**解决:** 检查注释格式,确保每个字段之间有空格

---

**问题3: `cannot find type definition: models.ParamSignUp`**

**原因:** swag 找不到引用的结构体

**解决:** 使用 `--parseDependency` 参数
```bash
swag init --parseDependency --parseInternal
```

或者在 Makefile 中固定:
```makefile
swag:
	swag init --parseDependency --parseInternal
```

---

### 10.2 Swagger UI 显示问题

**问题1: 访问 /swagger/index.html 显示 404**

**原因:** 忘记匿名导入 `docs` 包

**解决:**
```go
import _ "bluebell/docs"  // 必须有这一行
```

---

**问题2: Swagger UI 显示 "Failed to load API definition"**

**原因:** `docs.go` 没有被编译进二进制

**解决:**
1. 确保 `import _ "bluebell/docs"` 存在
2. 重新 `go build` (不要使用缓存的二进制)
3. 检查 `docs/docs.go` 是否存在

---

**问题3: 接口请求返回 404**

**原因:** Swagger 中的 `@Router` 路径与实际路由不匹配

**检查:**
```go
// 注释中
// @Router /post/{id} [get]

// 路由注册
v1.GET("/post/:id", controller.GetPostDetailHandler)  // ✅ 匹配

v1.GET("/posts/:id", controller.GetPostDetailHandler) // ❌ 不匹配
```

---

### 10.3 认证问题

**问题: 点击 "Authorize" 输入 Token 后,接口仍然返回 401**

**原因1:** Token 格式错误

**解决:** 必须输入 `Bearer <token>`,包含 `Bearer ` 前缀
```
Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**原因2:** Token 已过期

**解决:** 重新登录获取新 Token

**原因3:** `@securityDefinitions` 定义错误

**检查 main.go:**
```go
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
```

**检查 controller:**
```go
// @Security ApiKeyAuth
// @Param Authorization header string true "Bearer 用户令牌"
```

---

### 10.4 性能问题

**问题: `swag init` 耗时过长(>10秒)**

**原因:** swag 默认扫描所有 Go 文件,包括 vendor/

**解决1:** 排除不必要的目录
```bash
swag init --exclude vendor/,docs/,test/
```

**解决2:** 只扫描 controller 目录
```bash
swag init --generalInfo main.go --dir ./,./controller
```

---

## 11. 最佳实践

### 11.1 注释编写规范

**1. 注释与代码保持一致**
```go
// ❌ 错误: 注释说是 POST,路由却是 GET
// @Router /post/{id} [post]
func GetPostDetailHandler(c *gin.Context) {  // ← 函数名是 Get
	// ...
}

// ✅ 正确
// @Router /post/{id} [get]
func GetPostDetailHandler(c *gin.Context) {
	// ...
}
```

**2. 使用统一的响应模型**
```go
// ✅ 所有接口都返回 ResponseData
// @Success 200 {object} ResponseData
// @Failure 1007 {object} ResponseData
```

**3. 详细的参数描述**
```go
// ❌ 描述太简单
// @Param page query int false "页码"

// ✅ 详细描述
// @Param page query int false "页码,从1开始" default(1) minimum(1)
```

### 11.2 文档维护流程

**1. 开发新接口时:**
```
编写 Handler 代码 → 添加 Swagger 注释 → swag init → 测试接口 → 提交代码
```

**2. 修改接口时:**
```
修改 Handler 代码 → 同步更新 Swagger 注释 → swag init → 提交代码
```

**3. Code Review 检查项:**
- [ ] Swagger 注释是否完整?
- [ ] `@Param` 是否与实际参数一致?
- [ ] `@Success` 响应模型是否正确?
- [ ] 是否执行了 `swag init`?

### 11.3 Git 集成

**方法1: 提交 docs/ 到版本库**

**优点:** 团队成员拉代码后可直接看文档,无需 swag 工具

**缺点:** 每次提交都包含大量 auto-generated 文件

```bash
git add docs/
git commit -m "docs: update swagger docs"
```

**方法2: 使用 Git hooks 自动生成**

创建 `.git/hooks/pre-commit`:
```bash
#!/bin/sh
swag init
git add docs/
```

```bash
chmod +x .git/hooks/pre-commit
```

**方法3: 使用 .gitignore 忽略 docs/**

```
# .gitignore
docs/
```

**在 CI/CD 中生成文档:**
```yaml
# .github/workflows/build.yml
- name: Generate Swagger Docs
  run: |
    go install github.com/swaggo/swag/cmd/swag@latest
    swag init
```

### 11.4 文档版本管理

为每个版本生成文档快照:
```bash
# 发布 v1.0.0 时
swag init
cp -r docs docs-v1.0.0
git add docs-v1.0.0/
git commit -m "docs: add swagger docs for v1.0.0"
git tag v1.0.0
```

---

## 12. 实战练习

### 练习1: 为投票接口添加文档

**任务:** 为 `controller/vote.go` 的 `PostVoteHandler` 添加完整的 Swagger 注释

**要求:**
1. 接口需要认证(`@Security`)
2. 接受 `models.ParamVoteData` 作为请求体
3. 返回标准 `ResponseData`
4. 可能失败的错误码: 1007(参数错误), 1004(未登录)

**参考答案:**
```go
// PostVoteHandler 为帖子投票
// @Summary 帖子投票
// @Description 为帖子投赞成票(1)或反对票(-1),取消投票传0
// @Tags 投票相关
// @Accept application/json
// @Produce application/json
// @Security ApiKeyAuth
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body models.ParamVoteData true "投票参数"
// @Success 200 {object} ResponseData "投票成功"
// @Failure 1007 {object} ResponseData "参数错误"
// @Failure 1004 {object} ResponseData "请先登录"
// @Router /vote [post]
func PostVoteHandler(c *gin.Context) {
	// ...
}
```

---

### 练习2: 定义社区列表响应模型

**任务:** 在 `controller/docs_models.go` 中定义 `_ResponseCommunityList` 结构体

**要求:**
1. 包含标准的 `code`, `msg` 字段
2. `data` 字段为 `[]*models.Community` 数组

**参考答案:**
```go
// _ResponseCommunityList 社区列表响应
type _ResponseCommunityList struct {
	Code int                `json:"code"` // 业务状态码
	Msg  string             `json:"msg"`  // 提示信息
	Data []*models.Community `json:"data"` // 社区列表
}
```

---

### 练习3: 生产环境禁用 Swagger

**任务:** 修改 `routers/routers.go`,使 Swagger 仅在开发环境启用

**提示:**
- 通过 `mode` 参数判断环境
- `gin.ReleaseMode` 为生产模式

**参考答案:**
```go
func SetupRouter(mode string) *gin.Engine {
	if mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(logger.GinLogger(), logger.GinRecovery(true))

	// 仅在非生产环境启用 Swagger
	if mode != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// ... 其他路由 ...

	return r
}
```

---

## 13. 本章总结

### 13.1 核心知识点

| 知识点 | 说明 |
|--------|------|
| **Swagger/OpenAPI** | 业界标准的 API 文档规范,支持代码注释自动生成文档 |
| **swag 工具** | 扫描 Go 代码注释生成 OpenAPI 文档(JSON/YAML) |
| **gin-swagger** | Gin 框架的 Swagger UI 中间件 |
| **全局注释** | 在 `main.go` 定义 API 标题、版本、认证方案 |
| **接口注释** | 在 Handler 函数上方定义路由、参数、响应 |
| **@Param** | 定义请求参数(path/query/header/body) |
| **@Success/@Failure** | 定义响应状态码和结构体 |
| **@Security** | 标记接口需要认证(JWT) |
| **响应模型** | 定义专门的结构体用于文档展示 |
| **swag init** | 生成 `docs/` 文件夹,必须在修改注释后重新运行 |
| **匿名导入** | `import _ "bluebell/docs"` 触发文档注册 |

### 13.2 开发工作流

```
编写代码
  ↓
添加 Swagger 注释
  ↓
swag init
  ↓
启动服务
  ↓
访问 /swagger/index.html
  ↓
在线测试接口
  ↓
发现问题 → 修改代码 → 重复流程
```

### 13.3 为什么 Swagger 是必备技能?

1. **前后端协作**: 前端可以直接看文档开发,无需等待 Word 文档
2. **接口测试**: 无需 Postman,直接在浏览器测试
3. **自动生成客户端**: 使用 swagger-codegen 生成各语言客户端代码
4. **团队协作**: 新成员快速了解项目接口
5. **持续集成**: 在 CI/CD 中自动生成和发布文档

---

## 14. 延伸阅读

### 14.1 官方文档

- [Swag GitHub](https://github.com/swaggo/swag) - Swagger 注解完整手册
- [gin-swagger GitHub](https://github.com/swaggo/gin-swagger) - Gin 集成指南
- [OpenAPI 规范](https://swagger.io/specification/) - OpenAPI 3.0 官方规范

### 14.2 工具推荐

- **Swagger Editor**: 在线编辑 OpenAPI 文档 (https://editor.swagger.io/)
- **Swagger Codegen**: 生成客户端代码 (https://github.com/swagger-api/swagger-codegen)
- **Postman**: 支持直接导入 Swagger 文档
- **Insomnia**: 另一个支持 OpenAPI 的 API 测试工具

### 14.3 进阶话题

- **API 网关集成**: Kong, Tyk 等网关支持 OpenAPI 自动导入
- **Mock Server**: 使用 Swagger 生成 Mock 服务器
- **文档国际化**: 支持多语言 API 文档
- **自定义主题**: 修改 Swagger UI 的样式

---

## 15. 常见面试题

**Q1: 为什么要用 Swagger,而不是手写 Markdown 文档?**

**A:**
1. **代码即文档**: 注释写在代码旁边,改代码时自然会更新注释
2. **实时同步**: `swag init` 一键更新,永不过期
3. **在线测试**: Swagger UI 支持直接发送请求,无需 Postman
4. **工具链丰富**: 可以导出为 JSON/YAML,生成客户端代码,导入 Postman
5. **行业标准**: OpenAPI 是业界标准,团队成员都熟悉

---

**Q2: `@Param` 的五种参数位置是什么?分别用在什么场景?**

**A:**
| 位置 | 说明 | 示例场景 |
|------|------|----------|
| `path` | URL 路径参数 | `/post/:id` 中的 `id` |
| `query` | URL 查询参数 | `/posts?page=1` 中的 `page` |
| `header` | HTTP Header | `Authorization` Token |
| `body` | 请求体 | JSON 对象(`POST /signup`) |
| `formData` | 表单数据 | 文件上传 |

---

**Q3: 如何在生产环境禁用 Swagger?**

**A:**
```go
func SetupRouter(mode string) *gin.Engine {
	r := gin.New()

	// 仅在非生产环境启用 Swagger
	if mode != gin.ReleaseMode {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	return r
}
```

或者使用环境变量:
```go
if os.Getenv("ENABLE_SWAGGER") == "true" {
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
```

---

**Q4: `swag init` 生成的 `docs/` 文件夹应该提交到 Git 吗?**

**A:** 两种方案各有优劣:

**方案1: 提交 docs/**
- ✅ 优点: 团队成员拉代码后可直接运行,无需安装 swag 工具
- ❌ 缺点: 每次提交都包含大量 auto-generated 文件,增加 diff 复杂度

**方案2: 忽略 docs/,在 CI/CD 中生成**
- ✅ 优点: 保持仓库干净,避免提交自动生成的代码
- ❌ 缺点: 本地开发前必须先运行 `swag init`

**推荐:** 使用 Makefile 或脚本自动化 `swag init`,无论哪种方案都不影响开发体验。

---

**Q5: JWT 认证接口如何在 Swagger 中配置?**

**A:**

**步骤1: 在 main.go 定义安全方案**
```go
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
```

**步骤2: 在接口注释中引用**
```go
// @Security ApiKeyAuth
// @Param Authorization header string true "Bearer 用户令牌"
```

**步骤3: 在 Swagger UI 中使用**
1. 点击右上角 🔒 "Authorize" 按钮
2. 输入: `Bearer eyJhbGc...`
3. 所有接口请求自动携带此 Token

---

## 📖 下一章预告

在掌握了 Swagger 自动文档后,我们的开发效率已经提升了一大截。但每次修改代码后,还需要手动:
- `swag init` 生成文档
- `go run main.go` 重启服务
- 手动刷新浏览器

这些重复操作依然浪费时间。下一章,我们将学习 **Makefile** 和 **Air** 两大神器:
- Makefile 一键执行复杂命令
- Air 自动检测文件变化,热重载服务

让我们的开发工作流更加丝滑!

---

**📖 下一章: [第12章:高效开发工具Makefile与Air](./12-高效开发工具Makefile与Air.md)**

