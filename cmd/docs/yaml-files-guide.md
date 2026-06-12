# YAML 配置文件完全指南

## 已注释的配置文件总览

### 核心可观测性配置文件

#### 1. **mimir-config.yaml** ✅ 完全注释
**用途**：指标存储和查询引擎配置

**关键部分**：
- `target: all` — 运行所有组件
- `multitenancy_enabled: false` — 单租户模式
- `blocks_storage` — 块存储配置
- `compactor` — 数据压缩和合并
- `ingester` — 指标接收和缓存

**主要参数**：
- HTTP 端口：9009
- gRPC 端口：9095
- 存储路径：/var/mimir/blocks

---

#### 2. **tempo-config.yaml** ✅ 完全注释
**用途**：分布式追踪后端配置

**关键部分**：
- `server` — HTTP/gRPC 监听配置
- `distributor.receivers` — OTLP 接收器（4317/4318）
- `ingester` — 追踪数据缓存和 WAL
- `compactor` — 数据块合并
- `storage.trace` — 追踪数据存储

**主要参数**：
- HTTP 端口：3200
- OTLP gRPC：4317
- OTLP HTTP：4318
- 存储路径：/var/tempo/blocks

---

#### 3. **loki-config.yaml** ✅ 完全注释
**用途**：日志聚合系统配置

**关键部分**：
- `auth_enabled: false` — 禁用身份验证
- `server` — HTTP/gRPC 监听
- `common.storage` — 文件系统存储
- `schema_config` — 日志架构版本
- `limits_config` — 限制和策略

**主要参数**：
- HTTP 端口：3100
- gRPC 端口：9096
- 存储路径：/loki/chunks
- 日志保留期：7 天

---

#### 4. **otel-collector-config.yaml** ✅ 完全注释
**用途**：OpenTelemetry 数据采集和路由配置

**关键部分**：
- `receivers.otlp` — OTLP 接收器（gRPC + HTTP）
- `processors.batch` — 批处理（性能优化）
- `exporters` — 导出到 Loki、Tempo、Mimir
- `service.pipelines` — 三条数据管道

**数据流**：
```
应用程序 → OTLP (4317/4318) → Batch 处理 → 
  ├─ traces → Tempo (4317)
  ├─ metrics → Mimir (9009)
  └─ logs → Loki (3100)
```

---

#### 5. **prometheus.yml** ✅ 完全注释
**用途**：Prometheus 全局配置和抓取规则

**关键部分**：
- `global.scrape_interval` — 指标抓取间隔
- `rule_files` — 告警规则文件
- `scrape_configs` — 抓取目标定义

**主要参数**：
- 抓取间隔：15 秒
- 告警规则：prometheus-alerts.yml

---

#### 6. **prometheus-alerts.yml** ✅ 部分注释
**用途**：告警规则定义

**告警类型**：
- `HighErrorRate` — 错误率 > 5%（严重）
- `HighLatencyP95` — P95 延迟 > 1s（警告）
- `HighLatencyP99` — P99 延迟 > 3s（严重）

---

### Grafana 配置文件

#### 7. **grafana-provisioning/datasources/datasources.yaml** ✅ 完全注释
**用途**：定义 Grafana 数据源

**数据源列表**：
- **Mimir** — 指标存储（Prometheus API 兼容）
- **Tempo** — 分布式追踪
- **Loki** — 日志聚合
- **Pyroscope** — 性能分析

**访问模式**：
- `access: proxy` — Grafana 代理转发（安全）

**连接地址**：
```yaml
- Mimir:    http://mimir:9009/prometheus
- Tempo:    http://tempo:3200
- Loki:     http://loki:3100
- Pyroscope: http://pyroscope:4040
```

---

#### 8. **grafana-provisioning/dashboards/dashboards.yaml** ✅ 完全注释
**用途**：定义仪表板加载方式

**关键配置**：
- `type: file` — 从文件系统加载
- `file_format: json` — 仪表板格式
- `path: /etc/grafana/dashboards` — 仪表板存储路径
- `updateIntervalSeconds: 10` — 更新检查频率

---

### Docker 编排文件

#### 9. **docker-compose.dev.yml** ✅ 完全注释
**用途**：开发环境容器编排

**包含的服务**：
- MySQL — 数据库
- Redis — 缓存
- RabbitMQ — 消息队列
- Elasticsearch — 搜索引擎
- Grafana Alloy — 数据采集
- Tempo — 追踪存储
- Loki — 日志存储
- Mimir — 指标存储
- Pyroscope — 性能分析
- Grafana — 可视化

**使用命令**：
```bash
docker compose -f docker-compose.dev.yml up -d --build
```

---

#### 10. **docker-compose.yml** ✅ 完全注释
**用途**：生产环境容器编排

**与 dev 版本的主要差异**：
- 可能有更多的优化配置
- 网络和安全配置不同
- 可能包含持久化和监控配置

---

## 配置文件之间的关系

```
┌─────────────────────────────────────┐
│   应用程序（Go/Python）              │
│   ├─ OTLP SDK → 4317/4318           │
│   └─ Pyroscope SDK → 4040           │
└──────────────┬──────────────────────┘
               │
        ┌──────▼─────────┐
        │ Grafana Alloy  │  ← otel-collector-config.yaml
        │ + OTel SDK     │
        └──┬───┬───┬─────┘
           │   │   │
       ┌───▼─┐ │   └─────────────┐
       │     │ │                 │
   ┌───▼──┐ │ │    ┌────────┐   │
   │Mimir │ │ │    │Pyroscope   │
   └──────┘ │ │    └────────┘   │
            │ │
        ┌───▼──┐  ┌─────────┐
        │Tempo │  │  Loki   │
        └──────┘  └─────────┘
            │       │
            └───┬───┘
                │
           ┌────▼─────────┐
           │Grafana       │  ← datasources.yaml
           │Dashboard     │     dashboards.yaml
           └──────────────┘
```

---

## 关键配置值对照表

| 配置项 | 文件 | 值 | 说明 |
|--------|------|-----|------|
| Mimir HTTP 端口 | mimir-config.yaml | 9009 | 指标查询和导入 |
| Tempo HTTP 端口 | tempo-config.yaml | 3200 | 追踪查询和导入 |
| Loki HTTP 端口 | loki-config.yaml | 3100 | 日志查询和导入 |
| OTLP gRPC 端口 | otel-collector-config.yaml | 4317 | 高性能协议 |
| OTLP HTTP 端口 | otel-collector-config.yaml | 4318 | 通用协议 |
| Prometheus 抓取间隔 | prometheus.yml | 15s | 指标采集频率 |
| Pyroscope 端口 | docker-compose | 4040 | 性能分析 |
| Grafana 仪表板检查间隔 | datasources.yaml | 10s | 文件变化检测 |

---

## 如何修改配置

### 修改端口

**例子**：将 Mimir 从 9009 改为 9090

1. **mimir-config.yaml**
   ```yaml
   server:
     http_listen_port: 9090  # 改这里
   ```

2. **otel-collector-config.yaml**
   ```yaml
   exporters:
     prometheusremotewrite:
       endpoint: http://mimir:9090/api/v1/push  # 也要改这里
   ```

3. **datasources.yaml**
   ```yaml
   datasources:
     - name: Mimir
       url: http://mimir:9090/prometheus  # 也要改这里
   ```

### 修改存储路径

**例子**：将 Tempo 存储从 /var/tempo 改为 /data/tempo

在 **tempo-config.yaml**：
```yaml
storage:
  trace:
    local:
      path: /data/tempo/blocks  # 改这里
    wal:
      path: /data/tempo/wal     # 也要改这里
```

---

## 常见问题

### Q: 可以同时修改多个文件的同一个值吗？

**A**: 可以，但要确保一致性。使用以下步骤：
1. 在主配置文件修改值
2. 在所有引用该值的文件也修改
3. 重启容器使配置生效

### Q: 如何验证配置语法？

**A**: 使用 YAML 验证工具：
```bash
# YAML 格式验证
python -m yaml <file>.yaml

# 或用在线工具：https://www.yamllint.com/
```

### Q: 生产环境应该修改哪些值？

**A**: 关键修改项：
- `multitenancy_enabled` → true（Mimir）
- `auth_enabled` → true（Loki）
- 存储后端从文件系统改为 S3/GCS
- 副本因子 `replication_factor` 增加
- 移除所有 `insecure: true` 的 TLS 配置

---

**文档更新日期**: 2026-05-30
