# Bluebell 可观测性栈文档

## 项目架构概览

Bluebell 采用现代的可观测性架构，通过 OpenTelemetry（OTEL）统一收集应用的 **traces（追踪）、metrics（指标）、logs（日志）**。

```
┌─────────────────┐
│   应用程序      │
│  (Go/Python)    │
└────────┬────────┘
         │
    ┌────┴────┐
    │          │
    ↓ OTLP     ↓ Pyroscope SDK (直连)
    │          │
    ┌──────────┴──────┐
    │ Grafana Alloy   │  ← OTLP 数据采集和路由
    └──┬──┬──┬────────┘
       │  │  │
       ↓  ↓  ↓
   ┌──────┬──────┬──────┐
   │      │      │      │
   ↓      ↓      ↓      ↓
 Mimir  Tempo  Loki  Pyroscope
(指标)  (追踪) (日志) (性能分析)
   │      │      │      │
   └──────┴──────┴──────┘
           │
           ↓ Prometheus/各种 API
      ┌─────────────┐
      │  Grafana    │
      │  Dashboard  │
      └─────────────┘
```

---

## 核心概念

### OTEL 与 OTLP 的区别

| 概念 | 说明 | 类比 |
|------|------|------|
| **OTEL（OpenTelemetry）** | 开源项目框架，定义如何收集遥测数据 | 快递公司 |
| **OTLP（OpenTelemetry Protocol）** | 数据传输协议，支持 gRPC 和 HTTP | 快递规范 |

### OTLP 通信方式

```yaml
应用程序
  ├─ gRPC  → Alloy:4317  (高性能，二进制)
  └─ HTTP  → Alloy:4318  (兼容性好，文本)
```

---

## 各组件详解

### 1. Grafana Alloy（数据采集器）

**作用**：替代 OTel Collector，统一接收和路由遥测数据

**配置文件**：
- `alloy-config.alloy` — 有注释版本（推荐）
- `config.alloy` — 无注释版本（重复，可删除）

**监听端口**：
- `4317` — OTLP gRPC 接收器
- `4318` — OTLP HTTP 接收器
- `12345` — 自身指标导出

**处理流程**：
```
接收 → 批处理 → Transform（去重trace_id） → 导出
```

---

### 2. Mimir（指标存储）

**作用**：长期指标存储，完全兼容 Prometheus API

**特点**：
- ✅ 兼容 Prometheus 查询语言（PromQL）
- ✅ 支持长期存储（年级别）
- ✅ 高可用、分布式

**访问方式**：
- Grafana 查询：`http://mimir:9009/prometheus`
- Alloy 推送：`http://mimir:9009/api/v1/push`

**配置**：`mimir-config.yaml`

---

### 3. Tempo（分布式追踪）

**作用**：存储和查询分布式追踪数据（trace）

**特点**：
- 关联 logs 和 metrics
- 快速故障定位
- 支持 trace_id 关联

**监听端口**：`3200`

**接收协议**：OTLP gRPC (4317)

**配置**：`tempo-config.yaml`

---

### 4. Loki（日志聚合）

**作用**：收集和查询日志

**特点**：
- 标签索引（不是全文搜索）
- 轻量级、高效
- 自动提取 trace_id

**监听端口**：`3100`

**接收协议**：OTLP HTTP (`/otlp` 路径)

**配置**：`loki-config.yaml`

---

### 5. Pyroscope（性能分析）

**作用**：持续性能分析（Profiling），分析 CPU、内存使用情况

**特点**：
- 火焰图可视化
- 长期性能数据存储
- 快速性能瓶颈定位

**接入方式**：应用直接集成 Pyroscope SDK，**不需要 Alloy**

**监听端口**：`4040`

**配置**：Grafana 数据源配置（无需 Alloy 路由）

---

### 6. Grafana（可视化）

**作用**：统一的监控和日志查询界面

**数据源配置**：
```yaml
datasources:
  - name: Mimir           # 指标（用 Prometheus 类型）
    type: prometheus
    url: http://mimir:9009/prometheus
    
  - name: Tempo           # 追踪
    type: tempo
    url: http://tempo:3200
    
  - name: Loki            # 日志
    type: loki
    url: http://loki:3100
    
  - name: Pyroscope       # 性能分析
    type: pyroscope
    url: http://pyroscope:4040
```

**仪表板配置**：`grafana-provisioning/`

---

## 数据流示例

### 场景：应用请求链路

```
1. 应用发送数据
   请求 ID: abc123
   ├─ OTLP → Alloy  (traces/metrics/logs)
   │  ├─ Trace: [Server] → [DB] → [Cache]
   │  ├─ Metrics: response_time=145ms, status=200
   │  └─ Logs: "User logged in", traceID: abc123
   │
   └─ Pyroscope SDK → Pyroscope (性能分析，直连)
      └─ CPU profiles, Memory profiles

2. Alloy 接收和路由（仅 OTLP 数据）
   ├─ traces  → Tempo  (分布式追踪)
   ├─ metrics → Mimir  (指标存储)
   └─ logs    → Loki   (日志存储)

3. Grafana 查询和展示
   ├─ PromQL: select response_time where traceID=abc123
   ├─ LogQL: {traceID="abc123"}
   ├─ TraceQL: select from tempo where traceID=abc123
   └─ Pyroscope: 同一请求的 CPU/内存火焰图
   
   结果：一个完整的链路视图，包括性能、日志、追踪、Profiling
```

---

## 访问模式解释

### Proxy（代理）模式

```
用户浏览器 → Grafana 服务器 → 后端（Mimir/Loki/etc）
            ↑
       Grafana 发送 HTTP 请求
```

**优点**：
- 后端无需暴露公网
- 安全，内网通信
- 支持内网地址

**配置示例**：
```yaml
access: proxy
url: http://mimir:9009/prometheus  # 内网地址
```

---

## 启动项目

### 前置条件

1. **Docker Desktop 必须运行**
   - Windows: 搜索 "Docker Desktop" 并启动
   - 验证：`docker --version`

2. **依赖文件**
   - `docker-compose.dev.yml` — 开发环境配置
   - 各个 `*-config.yaml` — 服务配置

### 启动命令

```bash
# 启动所有服务
docker compose -f docker-compose.dev.yml up -d --build

# 查看日志
docker compose -f docker-compose.dev.yml logs -f

# 停止服务
docker compose -f docker-compose.dev.yml down
```

### 服务访问

| 服务 | 地址 | 用途 |
|------|------|------|
| Grafana | http://localhost:3000 | 监控仪表板 |
| Alloy | http://localhost:12345 | 自身指标（开发用） |
| Mimir | http://localhost:9009 | 指标 API |
| Tempo | http://localhost:3200 | 追踪 API |
| Loki | http://localhost:3100 | 日志 API |
| Pyroscope | http://localhost:4040 | 性能分析 |

---

## 配置文件位置

```
bluebell/
├── alloy-config.alloy              ← Alloy 配置（有注释）
├── config.alloy                    ← Alloy 配置（无注释，可删除）
├── mimir-config.yaml               ← Mimir 配置
├── tempo-config.yaml               ← Tempo 配置
├── loki-config.yaml                ← Loki 配置
├── otel-collector-config.yaml      ← OTEL 配置（备用）
├── prometheus.yml                  ← Prometheus 配置（如果使用）
├── prometheus-alerts.yml           ← 告警规则
├── grafana-provisioning/
│   ├── datasources/datasources.yaml     ← Grafana 数据源
│   └── dashboards/dashboards.yaml       ← Grafana 仪表板
└── loadtest/                       ← 负载测试工具
    ├── run-hey-test.sh             ← HTTP 负载测试
    ├── alert-simulation.ps1        ← 告警模拟
    └── grafana-full-flow.js        ← Grafana 集成测试
```

---

## 常见问题

### Q: Prometheus 还在用吗？

**A**: 不用。项目已迁移到 **Mimir**（更强大）。

- Prometheus 只能存储 15 天数据
- Mimir 支持年级数据存储
- Grafana 通过 Prometheus API 查询 Mimir

### Q: 必须用 Alloy 吗？

**A**: 可以用其他方案，但 Alloy 是最简洁的：

```yaml
# 选项 1: Alloy（推荐）
应用 → Alloy → Mimir/Tempo/Loki

# 选项 2: OTel Collector
应用 → OTel Collector → Mimir/Tempo/Loki

# 选项 3: 分别接入（不推荐）
应用 → Prometheus
应用 → Jaeger  
应用 → Fluentd
```

### Q: VSCode 有 Alloy 插件吗？

**A**: 没有官方插件。推荐：

1. 安装 "YAML Language Support by Red Hat"
2. 在 VSCode 设置添加：
   ```json
   "files.associations": {
     "*.alloy": "hcl"
   }
   ```

### Q: Docker 报错无法连接？

**A**: Docker Desktop 没运行。解决方案：

1. 搜索并启动 "Docker Desktop"
2. 等待显示 "Docker is running"
3. 重新运行命令

---

## 扩展阅读

- [OpenTelemetry 官网](https://opentelemetry.io/)
- [Grafana Alloy 文档](https://grafana.com/docs/alloy/latest/)
- [Mimir 文档](https://grafana.com/docs/mimir/latest/)
- [Tempo 文档](https://grafana.com/docs/tempo/latest/)
- [Loki 文档](https://grafana.com/docs/loki/latest/)

---

**文档更新日期**: 2026-05-30  
**适用版本**: Bluebell v1.0+
