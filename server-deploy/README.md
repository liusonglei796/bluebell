# Bluebell 部署指南 - 单机服务器

## 环境要求

- Ubuntu 20.04+ / Debian 11+
- Docker 24+
- Docker Compose v2
- 2核4G 内存 minimum

## 部署步骤

### 1. 安装依赖

```bash
# 安装 Docker
curl -fsSL https://get.docker.com | sh

# 安装 Docker Compose
apt install docker-compose-plugin
```

### 2. 上传代码

```bash
# 在服务器上克隆仓库
git clone https://github.com/your-org/bluebell.git
cd bluebell
```

### 3. 配置

```bash
# 创建数据目录
mkdir -p data/mysql data/redis data/rabbitmq data/elasticsearch

# 可选: 修改 .env.prod 配置
# vim .env.prod
```

### 4. 部署

```bash
# 完整部署
./server-deploy/deploy.sh

# 或手动执行
docker compose build --parallel
docker compose up -d
```

### 5. 查看日志

```bash
docker compose logs -f
```

## 服务访问

| 服务 | 地址 |
|------|------|
| Web 应用 | http://你的服务器IP |
| RabbitMQ | http://你的服务器IP:15672 |
| Elasticsearch | http://你的服务器IP:9200 |
| Jaeger | http://你的服务器IP:16686 |

## 管理命令

```bash
# 部署/更新
./server-deploy/deploy.sh

# 重启
./server-deploy/restart.sh

# 清理 (删除所有数据!)
./server-deploy/cleanup.sh

# 查看状态
docker compose ps

# 查看日志
docker compose logs -f [服务名]
```

## 备份

```bash
# 备份数据目录
tar -czf bluebell-backup-$(date +%Y%m%d).tar.gz data/
```