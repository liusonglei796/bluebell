# 22-Docker Compose 生产环境部署实战

> **Bluebell 项目 Docker Compose 生产部署全流程** - 从服务器准备到服务上线

## 📋 本章概述

本章将带你完成 Bluebell 项目从零到一的生产环境部署，涵盖：

- 阿里云服务器准备与环境配置
- Docker 与 Docker Compose 安装
- 项目代码上传与配置
- Docker Compose 多服务编排
- 依赖版本兼容性处理（重点）
- 服务验证与域名配置
- 常见问题排查与解决

### 难度：⭐⭐⭐
### 重点：生产部署、Docker Compose、依赖管理

---

## 🏗️ 部署架构

```
┌─────────────────────────────────────────────────┐
│              阿里云 ECS (4核/8G)                  │
│                                                   │
│  ┌─────────────────────────────────────────┐    │
│  │         Nginx Gateway (:8081)           │    │
│  │  ┌───────────┐    ┌─────────────────┐  │    │
│  │  │ 前端静态   │    │  API 反向代理    │  │    │
│  │  │  资源文件   │    │  → bluebell:8080 │  │    │
│  │  └─────┬─────┘    └────────┬────────┘  │    │
│  └────────┼───────────────────┼────────────┘    │
│           │                   │                  │
│  ┌────────▼─────────┐ ┌──────▼───────────────┐ │
│  │  Frontend 容器    │ │  Bluebell Go 服务     │ │
│  │  (构建+退出)      │ │  (持续运行)           │ │
│  └──────────────────┘ └──┬────────┬──────────┘ │
│                          │        │              │
│  ┌───────────────────────▼──┐ ┌──▼────────────┐ │
│  │  MySQL 8.0 容器           │ │  Redis 容器    │ │
│  │  (数据持久化)             │ │  (缓存/排序)   │ │
│  └──────────────────────────┘ └───────────────┘ │
│                                                   │
│  网络: bluebell_net (bridge)                      │
└─────────────────────────────────────────────────┘
```

### 服务说明

| 服务 | 镜像 | 端口映射 | 说明 |
|------|------|----------|------|
| MySQL | `mysql:8.0` | `3307:3306` | 数据库，健康检查通过后启动后端 |
| Redis | `redis:alpine` | `6380:6379` | 缓存与热帖排序 |
| Bluebell | 自定义构建 | 仅内部网络 | Go 后端服务，不暴露外部端口 |
| Gateway | `nginx:alpine` | `8081:80` | Nginx 反向代理 + 前端静态资源 |
| Frontend | 自定义构建 | 仅共享卷 | 构建前端产物后自动退出 |

---

## 🖥️ 第一步：服务器准备

### 1.1 服务器要求

| 配置项 | 最低要求 | 推荐配置 |
|--------|----------|----------|
| CPU | 2 核 | 4 核 |
| 内存 | 4 GiB | 8 GiB |
| 磁盘 | 40 GB SSD | 80 GB SSD |
| 系统 | Ubuntu 20.04+ / CentOS 8+ / Alibaba Cloud Linux 3 | Alibaba Cloud Linux 3 |
| 网络 | 公网 IP + 安全组开放 8081 | 绑定域名 + HTTPS |

### 1.2 安全组配置

在阿里云控制台的安全组规则中，确保以下端口已放行：

| 端口 | 协议 | 用途 | 是否必须 |
|------|------|------|----------|
| 22 | TCP | SSH 远程管理 | ✅ |
| 8081 | TCP | 应用访问（HTTP） | ✅ |
| 443 | TCP | HTTPS（可选） | 推荐 |
| 80 | TCP | HTTP 重定向（可选） | 推荐 |

> **注意**：MySQL (3307) 和 Redis (6380) 端口不建议对公网开放，仅在内部 Docker 网络中使用。

### 1.3 SSH 连接服务器

```bash
# 使用密钥连接（推荐）
ssh -i /path/to/your-key.pem root@<服务器IP>

# 或使用密码连接
ssh root@<服务器IP>
```

---

## 🐳 第二步：安装 Docker 环境

### 2.1 安装 Docker

```bash
# 使用官方一键安装脚本
curl -fsSL https://get.docker.com | sh

# 启动 Docker 并设置开机自启
systemctl start docker
systemctl enable docker

# 验证安装
docker --version
# 输出示例: Docker version 26.1.3, build b72abbb
```

### 2.2 安装 Docker Compose

Docker Compose V2 已集成到 Docker CLI 中：

```bash
# 检查是否已内置
docker compose version
# 输出示例: Docker Compose version v2.27.0

# 如果未安装，手动安装
DOCKER_CONFIG=${DOCKER_CONFIG:-$HOME/.docker}
mkdir -p $DOCKER_CONFIG/cli-plugins
curl -SL https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64 \
  -o $DOCKER_CONFIG/cli-plugins/docker-compose
chmod +x $DOCKER_CONFIG/cli-plugins/docker-compose
```

### 2.3 配置 Docker 镜像加速（国内服务器）

```bash
# 编辑 Docker 配置
cat > /etc/docker/daemon.json << 'EOF'
{
  "registry-mirrors": [
    "https://docker.1ms.run",
    "https://docker.m.daocloud.io"
  ]
}
EOF

# 重启 Docker
systemctl daemon-reload
systemctl restart docker
```

---

## 📁 第三步：上传项目代码

### 3.1 创建项目目录

```bash
# 在服务器上创建项目目录
mkdir -p /opt/bluebell
```

### 3.2 上传代码

**方式一：使用 scp（适合小项目）**

```bash
# 从本地电脑上传
scp -i /path/to/key.pem -r \
  --exclude='.git' \
  --exclude='node_modules' \
  /local/path/to/bluebell/* root@<服务器IP>:/opt/bluebell/
```

> **注意**：scp 不支持 `--exclude`，实际使用时可以使用 rsync：
> ```bash
> rsync -avz --exclude='.git' --exclude='node_modules' \
>   --exclude='data' --exclude='*.exe' \
>   /local/path/to/bluebell/ root@<服务器IP>:/opt/bluebell/
> ```

**方式二：使用 Git（推荐）**

```bash
# 在服务器上直接拉取代码
cd /opt/bluebell
git clone <你的Git仓库地址> .
```

**方式三：使用 tar 打包上传**

```bash
# 本地打包（排除不需要的文件）
tar czf bluebell-deploy.tar.gz \
  --exclude='.git' \
  --exclude='node_modules' \
  --exclude='data' \
  --exclude='*.exe' \
  --exclude='.vscode' \
  bluebell/

# 上传到服务器
scp -i /path/to/key.pem bluebell-deploy.tar.gz root@<服务器IP>:/opt/

# 在服务器上解压
ssh root@<服务器IP>
cd /opt && tar xzf bluebell-deploy.tar.gz
```

### 3.3 确认目录结构

```bash
cd /opt/bluebell
ls -la
```

应包含以下关键文件：

```
/opt/bluebell/
├── Dockerfile              # Go 后端 Dockerfile
├── docker-compose.yml      # 服务编排配置
├── config.docker.toml      # Docker 环境配置
├── go.mod                  # Go 依赖声明
├── go.sum                  # Go 依赖校验
├── frontend/               # 前端项目
│   └── Dockerfile          # 前端构建 Dockerfile
├── nginx/
│   └── backend.conf        # Nginx 反向代理配置
├── sql/                    # 数据库初始化脚本
├── cmd/                    # 应用入口
├── internal/               # 业务代码
└── pkg/                    # 公共包
```

---

## ⚙️ 第四步：配置文件说明

### 4.1 config.docker.toml

这是 Docker 环境专用的配置文件，关键配置项：

```toml
[app]
name = "bluebell"
port = 8080              # Go 服务内部端口
mode = "release"         # 生产模式

[mysql]
host = "mysql"           # Docker 内部网络中的服务名
port = 3306              # 容器内部端口
passwd = "your_password" # 与 docker-compose.yml 中的 MYSQL_ROOT_PASSWORD 一致
db_name = "bluebell"
user = "root"

[redis]
host = "redis"           # Docker 内部网络中的服务名
port = 6379
password = ""
db_name = 1

[jwt]
secret = "bluebell_secret_key_change_in_prod"  # ⚠️ 生产环境务必修改！
access_expiry = "120m"
refresh_expiry = "168h"
```

> **安全提醒**：部署到生产环境时，务必修改 `jwt.secret` 为强随机字符串！

### 4.2 docker-compose.yml 核心配置

```yaml
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: your_password  # 与 config.docker.toml 保持一致
      MYSQL_DATABASE: bluebell
    volumes:
      - ./data/mysql:/var/lib/mysql        # 数据持久化
      - ./sql:/docker-entrypoint-initdb.d  # 初始化 SQL
    healthcheck:                           # 健康检查
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 5s
      timeout: 10s
      retries: 5

  redis:
    image: redis:alpine
    volumes:
      - ./data/redis:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 10s
      retries: 5

  bluebell:
    build:
      context: .
      dockerfile: Dockerfile
    depends_on:
      mysql:
        condition: service_healthy         # 等待 MySQL 就绪
      redis:
        condition: service_healthy         # 等待 Redis 就绪

  bluebell_gateway:
    image: nginx:alpine
    ports:
      - "8081:80"                          # 唯一对外暴露的端口
    volumes:
      - ./nginx/backend.conf:/etc/nginx/conf.d/default.conf:ro
      - frontend_assets:/usr/share/nginx/html:ro

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    volumes:
      - frontend_assets:/usr/share/nginx/html
    # 前端容器构建完静态资源后自动退出，这是正常行为
```

---

## 🚀 第五步：启动服务

### 5.1 一键启动

```bash
cd /opt/bluebell
docker compose up -d --build
```

参数说明：
- `-d`：后台运行（detached mode）
- `--build`：强制重新构建镜像

### 5.2 查看服务状态

```bash
docker compose ps
```

正常输出：

```
NAME               IMAGE               COMMAND                  SERVICE            STATUS
bluebell_app       bluebell-bluebell   "./bluebell -conf co…"   bluebell           Up
bluebell_gateway   nginx:alpine        "/docker-entrypoint.…"   bluebell_gateway   Up
bluebell_mysql     mysql:8.0           "docker-entrypoint.s…"   mysql              Up (healthy)
bluebell_redis     redis:alpine        "docker-entrypoint.s…"   redis              Up (healthy)
```

### 5.3 查看服务日志

```bash
# 查看所有服务日志
docker compose logs

# 查看特定服务日志
docker compose logs bluebell
docker compose logs mysql
docker compose logs gateway

# 实时跟踪日志
docker compose logs -f bluebell
```

---

## 🔧 第六步：常见问题与解决方案

### 6.1 Go 依赖版本兼容性问题 ⭐ 重点

这是部署过程中最常遇到的问题。Go 生态的版本依赖链可能导致构建失败。

#### 问题 1：`go.mod` 要求 Go 版本过高

**现象**：
```
go.mod requires go >= 1.25.0 (running go 1.24.13; GOTOOLCHAIN=local)
```

**原因**：`go mod tidy` 在本地执行时，如果本地 Go 版本较高（如 1.26），会自动将 `go.mod` 中的 `go` 指令升级到与本地匹配的版本。同时，某些第三方库的最新版本可能要求更新的 Go 版本。

**解决方案**：

```bash
# 方案 1：使用 GOTOOLCHAIN=local 防止自动升级
GOTOOLCHAIN=local go mod tidy

# 方案 2：手动指定 go.mod 中的 go 版本
# 编辑 go.mod，将 go 1.25.0 改为 go 1.22
# 然后使用匹配的 toolchain
GOTOOLCHAIN=go1.22.0 go mod tidy
```

#### 问题 2：第三方库要求 Go 版本高于 Docker 镜像

**现象**：
```
github.com/gin-contrib/timeout@v1.2.1 requires go >= 1.25.0
```

**原因**：某些依赖库的版本要求比 Docker 镜像中 Go 版本更高。

**解决方案**：降级依赖到兼容版本

```toml
# go.mod 中降级有问题的依赖
require (
    github.com/gin-gonic/gin v1.9.1          # 而非 v1.12.0
    github.com/gin-contrib/timeout v1.1.0    # 要求 Go >= 1.23
    github.com/gin-contrib/pprof v1.4.0      # 而非 v1.5.3
    github.com/go-playground/validator/v10 v10.16.0
)
```

然后在 Dockerfile 中使用匹配的 Go 版本：

```dockerfile
# Dockerfile
FROM golang:1.24-alpine AS builder
```

#### 问题 3：Go 模块下载超时

**现象**：
```
go mod download: context deadline exceeded
```

**解决方案**：在 Dockerfile 中配置国内代理

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct    # 添加这一行
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o bluebell ./cmd/bluebell
```

#### 问题 4：`filippo.io/edwards25519` 需要 Go >= 1.24

**现象**：
```
filippo.io/edwards25519@v1.2.0 requires go >= 1.24
```

**解决方案**：确保 Dockerfile 使用 `golang:1.24-alpine` 或更高版本。

### 6.2 前端构建问题

#### 问题：`npm ci` 失败

**现象**：
```
npm ERR! Could not resolve dependency
```

**原因**：`npm ci` 要求 `package-lock.json` 与 `package.json` 完全匹配。

**解决方案**：改用 `npm install`

```dockerfile
# frontend/Dockerfile
COPY package*.json ./
RUN npm install    # 改为 npm install 而非 npm ci
```

### 6.3 Dockerfile 被误覆盖

**问题**：上传文件时目标路径错误，导致 `Dockerfile`（Go 后端）被 `frontend/Dockerfile` 内容覆盖。

**解决方案**：
```bash
# 确认两个 Dockerfile 内容不同
cat /opt/bluebell/Dockerfile           # 应该是 golang:1.24-alpine
cat /opt/bluebell/frontend/Dockerfile  # 应该是 node:20-alpine

# 如果内容相同，重新上传正确的文件
scp -i /path/to/key.pem ./Dockerfile root@<服务器IP>:/opt/bluebell/Dockerfile
```

### 6.4 容器启动失败排查

```bash
# 1. 检查容器状态
docker compose ps

# 2. 查看失败容器的日志
docker logs bluebell_app

# 3. 检查容器内部网络
docker network inspect bluebell_bluebell_net

# 4. 手动进入容器调试
docker exec -it bluebell_app sh

# 5. 检查配置文件是否正确挂载
docker exec bluebell_app cat /app/config.toml
```

---

## 🌐 第七步：验证服务

### 7.1 本地验证

```bash
# 测试前端页面
curl http://<服务器IP>:8081/
# 应返回 HTML 内容

# 测试 API（需要登录）
curl http://<服务器IP>:8081/api/v1/posts
# 返回: {"code":1006,"msg":"需要登录"}

# 这个 1006 响应说明 API 路由正常工作，只是需要认证
```

### 7.2 浏览器验证

打开浏览器访问：`http://<服务器IP>:8081`

你应该能看到 Bluebell 的前端页面。

### 7.3 健康检查

```bash
# 检查 MySQL 健康
docker exec bluebell_mysql mysqladmin ping -h localhost

# 检查 Redis 健康
docker exec bluebell_redis redis-cli ping

# 检查所有容器状态
docker compose ps
```

---

## 🔗 第八步：域名配置

### 8.1 添加 DNS A 记录

在 DNS 服务商（如 Cloudflare）中添加 A 记录：

| 类型 | 名称 | 内容 | TTL |
|------|------|------|-----|
| A | bluebell | 47.113.144.229 | Auto |

### 8.2 验证 DNS 解析

```bash
nslookup bluebell.dpdns.org
# 应返回你的服务器 IP
```

### 8.3 Cloudflare 注意事项

- 如果开启 **Proxy（橙色云朵）**，Cloudflare 会代理流量，真实 IP 会被隐藏
- 如果暂时需要直接访问服务器 IP，可设为 **DNS only（灰色云朵）**
- DNS 传播通常需要 1-5 分钟

### 8.4 通过域名访问

配置完成后访问：`http://bluebell.dpdns.org:8081`

---

## 🛠️ 第九步：日常运维

### 9.1 重启服务

```bash
cd /opt/bluebell

# 重启所有服务
docker compose restart

# 重启单个服务
docker compose restart bluebell
```

### 9.2 更新代码

```bash
cd /opt/bluebell

# 拉取最新代码（如果使用 Git）
git pull

# 重新构建并启动
docker compose up -d --build

# 清理无用镜像
docker image prune -f
```

### 9.3 查看日志

```bash
# 实时查看所有服务日志
docker compose logs -f

# 查看最近 100 行
docker compose logs --tail=100 bluebell

# 导出日志到文件
docker compose logs bluebell > bluebell.log
```

### 9.4 停止服务

```bash
# 停止所有服务（保留数据卷）
docker compose stop

# 停止并删除容器（保留数据卷）
docker compose down

# 停止并删除容器和数据卷（⚠️ 会丢失数据）
docker compose down -v
```

### 9.5 数据库备份

```bash
# 备份 MySQL 数据
docker exec bluebell_mysql mysqldump -u root -p'15939087780Ll@' bluebell > backup_$(date +%Y%m%d).sql

# 恢复数据库
docker exec -i bluebell_mysql mysql -u root -p'15939087780Ll@' bluebell < backup_20260402.sql
```

---

## 📊 部署检查清单

部署完成后，确认以下项目：

- [ ] Docker 和 Docker Compose 已安装并正常运行
- [ ] 项目代码已上传到服务器
- [ ] `config.docker.toml` 中的密码与 `docker-compose.yml` 一致
- [ ] JWT Secret 已修改为生产环境的强随机字符串
- [ ] 所有容器正常运行（`docker compose ps`）
- [ ] MySQL 和 Redis 健康检查通过
- [ ] 通过 IP:8081 可以访问前端页面
- [ ] API 接口返回正常响应
- [ ] DNS 记录已配置并解析正确
- [ ] 阿里云安全组已放行 8081 端口
- [ ] 数据库初始化脚本已执行（`./sql/` 目录）
- [ ] 数据持久化目录已挂载（`./data/mysql` 和 `./data/redis`）

---

## 🔒 安全建议

1. **修改默认密码**：将 MySQL 密码和 JWT Secret 改为强随机字符串
2. **关闭不必要的端口**：MySQL (3307) 和 Redis (6380) 不应暴露到公网
3. **启用 HTTPS**：使用 Let's Encrypt 或 Cloudflare 的 SSL 证书
4. **定期更新镜像**：`docker compose pull && docker compose up -d`
5. **配置防火墙**：仅开放必要的端口
6. **定期备份**：设置定时任务备份数据库

---

## 📝 总结

通过本章学习，你掌握了：

- ✅ Docker Compose 多服务编排
- ✅ Go 项目的 Docker 化部署
- ✅ 前后端分离架构的容器化部署
- ✅ 依赖版本兼容性问题的排查与解决
- ✅ 生产环境的安全配置
- ✅ 日常运维操作（重启、更新、备份）

这是 Bluebell 项目教学文档系列的最后一章，涵盖了从开发到部署的完整生命周期。

---

*最后更新*：2026 年 04 月
*文档版本*：1.0
*部署环境*：Docker 26.1.3, Docker Compose v2.27.0, Go 1.24
