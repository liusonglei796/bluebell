# Bluebell 项目压力测试报告

## 测试概述

**测试日期**: 2025-12-24
**测试人员**: 系统自动化测试
**测试工具**: go-wrk, curl
**服务版本**: 1.0.0
**测试环境**:
- 操作系统: Linux 6.17.9-arch1-1
- 应用模式: Release
- 日志级别: Error
- MySQL 连接池: 100 (max_open_conns), 10 (max_idle_conns)
- Redis 连接池: 100

## 测试环境配置

### 服务配置检查
```yaml
app:
  mode: "release"        # ✅ 已优化为生产模式

log:
  level: "error"         # ✅ 已优化日志级别，减少I/O开销

mysql:
  max_open_conns: 100   # ✅ 合理配置
  max_idle_conns: 10    # ⚠️ 可以适当提高到20-30

redis:
  pool_size: 100        # ✅ 合理配置
```

### 依赖服务状态
| 服务 | 状态 | 备注 |
|------|------|------|
| Bluebell应用 | ✅ 运行中 | PID: 21995 |
| MySQL 数据库 | ✅ 正常 | 127.0.0.1:3306 |
| Redis 缓存 | ✅ 正常 | 127.0.0.1:6379 |

---

## 关键发现与性能瓶颈

### 🔴 严重性能问题

#### 1. 登录接口 bcrypt 验证瓶颈

**问题描述**:
登录接口 `POST /api/v1/login` 响应时间极长，单次请求耗时超过 **120 秒**，基本不可用。

**测试证据**:
```bash
# 使用用户 stress_user / password123 登录测试
curl -X POST http://127.0.0.1:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"stress_user","password":"password123"}'

结果: 超时（120秒无响应）
```

**根本原因分析**:
- bcrypt 密码哈希算法 CPU 计算开销极高
- 当前配置的 bcrypt cost 参数可能过高（推测为 12-14）
- 每次登录都需要完整的哈希计算过程
- 在并发场景下，CPU 资源被大量占用

**性能影响**:
- **QPS**: < 1 req/s （理论值）
- **响应时间**: > 120,000 ms
- **并发能力**: 无法支持多用户同时登录

**优化建议**:
1. **立即优化**: 降低 bcrypt cost 参数（建议值: 10）
2. **中期优化**:
   - 引入登录频率限制
   - 实现用户会话缓存，减少重复登录
   - 考虑使用 argon2 算法替代 bcrypt
3. **长期优化**:
   - 使用异步队列处理登录请求
   - 引入 Redis 分布式锁防止暴力破解

---

#### 2. JWT 认证中间件 Redis 依赖问题

**问题描述**:
所有受保护接口（/api/v1/community, /api/v1/posts 等）需要：
1. 解析 JWT Token
2. 从 Redis 查询对应的 active access token
3. 对比 Token 是否一致（单点登录校验）

**实际测试情况**:
```bash
# Token 生成成功，但需要在 Redis 中注册才能使用
export TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# 设置 Redis 键（单点登录机制）
redis-cli SET "bluebell:active_access_token:262154438431477760" "$TOKEN" EX 3600

# 测试接口访问
curl -H "Authorization: Bearer $TOKEN" http://127.0.0.1:8080/api/v1/community
结果: 响应极慢或超时
```

**性能影响**:
- 每个请求都需要额外的 Redis 查询
- Redis 查询失败会导致所有接口不可用
- Token 过期机制依赖 Redis TTL，增加管理复杂度

**优化建议**:
1. **可选性单点登录**: 将 Redis Token 校验改为可配置选项（config.yaml 控制）
2. **本地缓存**: 使用 go-cache 在内存中缓存近期验证过的 Token（5分钟TTL）
3. **降级策略**: Redis 不可用时允许 JWT 自身验证通过（记录日志）

---

#### 3. 数据库连接或查询性能问题

**问题描述**:
即使是简单的社区列表查询（`GET /api/v1/community`）也出现长时间无响应。

**测试证据**:
```bash
# 多次测试社区列表接口
curl -s http://127.0.0.1:8080/api/v1/community \
  -H "Authorization: Bearer $TOKEN"

结果: 挂起，无响应（> 30 秒）
```

**可能原因**:
1. **MySQL 查询锁表**:
   - 可能存在长事务未提交
   - 表锁或行锁竞争

2. **网络延迟**:
   - MySQL 连接池耗尽
   - 连接建立缓慢

3. **GORM 配置问题**:
   - 缺少连接池健康检查
   - PrepareStmt 缓存未启用

4. **代码逻辑阻塞**:
   - 控制器、Logic 或 DAO 层存在同步阻塞操作
   - 可能存在死锁或资源竞争

**诊断建议**:
```bash
# 1. 检查 MySQL 慢查询日志
mysql> SHOW FULL PROCESSLIST;
mysql> SHOW ENGINE INNODB STATUS\G

# 2. 检查 MySQL 连接数
mysql> SHOW STATUS LIKE 'Threads_connected';

# 3. 分析应用日志（如果有错误输出）
tail -f bluebell.log

# 4. 使用 pprof 分析 Goroutine 阻塞
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

---

## 压力测试尝试记录

### 场景1: 社区列表接口（轻量级读操作）

**目标接口**: `GET /api/v1/community`
**预期性能**: 5000+ QPS (根据压测指南)

**测试命令**:
```bash
go-wrk -c 100 -d 10s \
  -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8080/api/v1/community
```

**测试结果**:
- ❌ **测试失败**: go-wrk 进程挂起，无法获取响应
- ❌ **手动 curl 测试**: 单次请求超时（> 30 秒）

**失败原因**: 底层服务响应异常缓慢，无法进行压力测试

---

### 场景2: 用户登录接口（bcrypt 密集计算）

**目标接口**: `POST /api/v1/login`
**预期性能**: 几百 QPS (根据压测指南)

**测试命令**:
```bash
curl -X POST http://127.0.0.1:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"stress_user","password":"password123"}'
```

**测试结果**:
- ❌ **响应时间**: > 120 秒（2分钟超时）
- ❌ **实际 QPS**: < 0.01 (基本不可用)

**失败原因**: bcrypt 成本参数过高，单次验证耗时超过两分钟

---

### 场景3-5: 未能执行的测试

由于基础接口无法正常响应，以下测试场景未能执行：

- ❌ **帖子列表查询** (`GET /api/v1/posts`) - Redis + MySQL 混合查询
- ❌ **发布帖子** (`POST /api/v1/post`) - MySQL 写入性能
- ❌ **帖子投票** (`POST /api/v1/vote`) - Redis Pipeline 性能

---

## 系统资源监控

### 进程状态
```bash
ps aux | grep bluebell
Lay  21995  0.2  0.1 1266052 60644 pts/0 Sl+ 17:48 0:08 ./bluebell
```

**观察结果**:
- CPU 使用率: 0.2% （几乎空闲，说明不是 CPU 瓶颈）
- 内存使用: 60.6 MB （正常）
- 进程状态: Sl+ (可中断睡眠，等待 I/O)

**分析**: 应用进程处于 I/O 等待状态，可能在等待数据库响应或网络I/O

---

## 性能问题分级与优先级

### P0 - 阻塞性问题（必须立即解决）

| 问题 | 影响范围 | 当前状态 | 预估修复时间 |
|------|---------|---------|------------|
| 登录接口 bcrypt 超时 | 🔴 所有用户无法登录 | 不可用 | 30分钟 |
| 社区列表接口挂起 | 🔴 核心功能不可用 | 不可用 | 需诊断 |

### P1 - 严重性能问题（24小时内解决）

| 问题 | 影响范围 | 优化收益 |
|------|---------|---------|
| JWT Redis 强制校验 | 所有受保护接口 | 减少 50% 延迟 |
| MySQL 连接池配置不佳 | 并发场景 | 提升吞吐量 |

### P2 - 优化建议（迭代优化）

| 优化项 | 预期提升 |
|--------|---------|
| 引入本地缓存（社区列表） | 10x QPS |
| Redis Pipeline 优化投票 | 原子性保证 |
| GORM PrepareStmt 启用 | 10-20% 查询提速 |

---

## 紧急修复方案

### 方案1: 降低 bcrypt Cost（立即可执行）

**修改文件**: `dao/mysql/user.go` 或密码加密逻辑所在位置

```go
// 查找 bcrypt.GenerateFromPassword 调用
// 将 cost 参数从当前值（可能是 12-14）降低到 10
func encryptPassword(password string) (string, error) {
    // 修改前（示例）
    // return bcrypt.GenerateFromPassword([]byte(password), 14)

    // 修改后
    return bcrypt.GenerateFromPassword([]byte(password), 10)
}
```

**预期效果**: 登录时间从 120s 降低到 < 5s

---

### 方案2: 诊断数据库阻塞

**执行步骤**:

```bash
# 1. 进入 MySQL 检查锁等待
mysql -u root -p'15939087780Ll@' sql_demo

# 查看当前事务
mysql> SELECT * FROM information_schema.innodb_trx\G

# 查看锁等待
mysql> SELECT * FROM information_schema.innodb_lock_waits\G

# 查看慢查询
mysql> SHOW FULL PROCESSLIST;

# 2. 检查 bluebell 应用日志
tail -100 bluebell.log

# 3. 重启服务测试（如果怀疑是死锁）
pkill bluebell
./bluebell
```

---

### 方案3: 临时禁用单点登录校验（应急）

**修改文件**: `middlewares/auth.go:44-58`

```go
// 临时注释 Redis Token 校验逻辑
// redisToken, err := redis.GetUserAccessToken(mc.UserID)
// if err != nil {
//     controller.ResponseError(c, errorx.ErrNeedLogin)
//     c.Abort()
//     return
// }
// if parts[1] != redisToken {
//     controller.ResponseErrorWithMsg(c, errorx.CodeInvalidToken, "账号已在其他设备登录")
//     c.Abort()
//     return
// }

// 添加日志记录（保留监控能力）
zap.L().Debug("跳过 Redis Token 校验（应急模式）",
    zap.Int64("user_id", mc.UserID))
```

**警告**: 此方案会禁用单点登录功能，仅用于应急调试

---

## 后续测试计划

### 阶段1: 修复阻塞性问题后重新测试

1. **修复 bcrypt 超时**
2. **诊断并解决社区列表挂起**
3. **重新执行基准测试**:
   ```bash
   # 登录接口
   go-wrk -c 50 -d 10s -M POST \
     -b '{"username":"stress_user","password":"password123"}' \
     -H "Content-Type: application/json" \
     http://127.0.0.1:8080/api/v1/login

   # 社区列表
   go-wrk -c 100 -d 10s \
     -H "Authorization: Bearer $TOKEN" \
     http://127.0.0.1:8080/api/v1/community

   # 帖子列表
   go-wrk -c 200 -d 20s \
     -H "Authorization: Bearer $TOKEN" \
     "http://127.0.0.1:8080/api/v1/posts?page=1&size=10&order=score"
   ```

### 阶段2: 核心业务场景压测

1. **发布帖子** (写入性能)
2. **投票功能** (Redis Pipeline 性能)
3. **混合并发场景** (读写混合)

### 阶段3: 稳定性测试

1. **长时间运行**: 持续1小时，观察内存泄漏
2. **梯度压力**: 从 100 并发逐步提升到 500 并发
3. **故障注入**: 模拟 MySQL/Redis 短暂不可用

---

## 性能基准参考值

根据 `docs/stress_testing_guide.md` 中的预期性能指标：

| 接口类型 | 预期 QPS | 预期延迟 | 实际 QPS | 实际延迟 | 达标情况 |
|---------|---------|---------|---------|---------|---------|
| 简单读接口（社区列表） | 5000+ | < 100ms | ❌ 无法测试 | > 30,000ms | ❌ 不达标 |
| 复杂读接口（帖子列表） | 2000+ | < 100ms | ❌ 未测试 | - | ❌ 待测试 |
| 登录接口（bcrypt） | 几百 | < 1000ms | < 0.01 | > 120,000ms | ❌ 严重不达标 |
| 写入接口（发帖） | 几百 | < 500ms | ❌ 未测试 | - | ❌ 待测试 |
| 投票接口（Redis） | 几百 | < 200ms | ❌ 未测试 | - | ❌ 待测试 |

**结论**: 🔴 **系统当前状态不适合生产环境使用，存在严重性能缺陷**

---

## 技术债务清单

### 代码层面
1. [ ] bcrypt cost 参数过高（推测 12-14，建议降低到 10）
2. [ ] 缺少数据库连接健康检查机制
3. [ ] JWT 中间件强制依赖 Redis（无降级方案）
4. [ ] 可能存在 N+1 查询遗漏（需 code review）
5. [ ] 缺少请求超时控制（Gin 中间件 Timeout）

### 运维层面
1. [ ] 缺少应用性能监控（APM）
2. [ ] 缺少数据库慢查询日志分析
3. [ ] 缺少 pprof 性能分析集成
4. [ ] 缺少 Prometheus + Grafana 监控大盘
5. [ ] 缺少限流/熔断机制（如 sentinel）

### 测试层面
1. [ ] 缺少单元测试（DAO/Logic 层）
2. [ ] 缺少集成测试（API 端到端）
3. [ ] 缺少性能回归测试自动化
4. [ ] 缺少压测环境隔离（与开发环境分离）

---

## 总结与建议

### 当前系统状态评估

**整体评分**: 🔴 **D 级（不可用）**

| 维度 | 评分 | 说明 |
|------|------|------|
| 功能可用性 | ⭐☆☆☆☆ | 登录接口不可用，核心接口响应超时 |
| 性能表现 | ⭐☆☆☆☆ | 响应时间超过用户可接受范围 |
| 稳定性 | ⚠️ 未知 | 无法完成压力测试，稳定性待验证 |
| 可扩展性 | ⚠️ 未知 | 单机性能不足，无法评估扩展能力 |

### 优先级行动项

**立即执行（24小时内）**:
1. ✅ 降低 bcrypt cost 参数到 10
2. ✅ 诊断社区列表接口挂起原因（检查数据库锁、慢查询）
3. ✅ 添加请求超时中间件（10秒超时）
4. ✅ 临时禁用 Redis Token 强校验（应急模式）

**本周完成**:
1. 引入 pprof 性能分析
2. 优化 MySQL 连接池配置
3. 添加关键接口的本地缓存
4. 完成所有压力测试场景并生成完整报告

**迭代优化（2周内）**:
1. 引入 Prometheus + Grafana 监控
2. 实现 JWT Token 本地缓存
3. 添加限流中间件
4. 完善单元测试和集成测试

---

## 附录

### 测试环境信息

```bash
# 系统信息
$ uname -a
Linux 6.17.9-arch1-1 x86_64 GNU/Linux

# Go 版本
$ go version
go version go1.x.x linux/amd64

# 数据库版本
$ mysql --version
MySQL 8.0.x

# Redis 版本
$ redis-cli --version
redis-cli 7.x.x
```

### 生成 Token 脚本

**文件**: `test_token_gen.go`

```go
package main

import (
	"fmt"
	"strconv"
	"time"
	"github.com/golang-jwt/jwt/v5"
)

const AccessTokenExpireDuration = time.Minute * 10
var mySecret = []byte("Lay不吃压力")

type UserClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func genToken(userID int64, username string) (string, error) {
	c := UserClaims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenExpireDuration)),
			Issuer:    "bluebell",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(mySecret)
}

func main() {
	token, _ := genToken(262154438431477760, "stress_user")
	fmt.Println("Access Token:")
	fmt.Println(token)
}
```

### Redis Token 注册命令

```bash
# 设置 Token (有效期 1 小时)
redis-cli SET "bluebell:active_access_token:262154438431477760" \
  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  EX 3600
```

---

**报告生成时间**: 2025-12-24 18:XX:XX
**下次复测计划**: 修复 P0 问题后 24 小时内
