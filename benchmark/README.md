# Bluebell k6 全链路混合压测指南

本目录提供了基于 **k6** 的全链路真实业务高并发压测套件，涵盖用户认证、热榜浏览、帖子详情多级缓存以及投票接口限流与熔断的综合检验。

---

## 📁 目录结构

| 文件 | 说明 |
| :--- | :--- |
| `k6_mixed_load.js` | **核心压测脚本**：定义 0~500 VUs 阶梯爬坡负载模型、8:2 读写分布行为、指标阈值及 HTML 报表生成。 |
| `seed.go` | **数据预热工具**：秒级预置 100 个用户和 50 篇帖子，预热 Redis 榜单并导出 `tokens.json`。 |
| `summary.html` | 压测结束后自动生成的交互式可视化分析报告（含延迟分位图、错误率等）。 |

---

## 🚀 快速开始

### 1. 安装 k6

- **Windows (Winget / Chocolatey)**:
  ```powershell
  winget install k6 --source winget
  # 或 choco install k6
  ```
- **macOS (Homebrew)**:
  ```bash
  brew install k6
  ```
- **Docker 运行 (无需本地安装)**:
  ```bash
  docker run --rm -i --net=host grafana/k6 run - <benchmark/k6_mixed_load.js
  ```

---

### 2. （可选）一键预置压测数据

你可以直接运行 Go 脚本预置 100 个用户与 50 篇帖子，避免压测时的冷启动耗时：

```powershell
go run ./benchmark/seed.go -conf ./config.yaml -users 100 -posts 50
```

> 💡 *若不执行此步骤，`k6_mixed_load.js` 的 `setup()` 阶段也会自动调用 API 注册并登录 50 个测试账号。*

---

### 3. 执行压测

确保 Bluebell 后端服务已启动（默认监听 `http://127.0.0.1:8080`）：

```powershell
# 执行默认阶梯加压测试
k6 run benchmark/k6_mixed_load.js

# 若后端运行在其他端口或远程服务器，可通过环境变量覆盖：
k6 run -e BASE_URL=http://127.0.0.1:8083/api/v1 benchmark/k6_mixed_load.js
```

---

## 📊 负载模型与阶段规划 (Load Stages)

脚本配置了标准的**阶梯递进式加压模型**，全程约 6.5 分钟：

```
VU (并发用户数)
 500 │                     ┌─────────┐
     │                    ╱           ╲
 200 │         ┌─────────┘             ╲
     │        ╱                         ╲
  50 │   ┌───┘                           ╲
   0 └───┴─────┴─────────┴─────────┴─────┴───▶ 时间 (t)
      30s 1m   30s  2m   30s  1m   30s
     预热 阶梯1     阶梯2    极限冲刺  收尾
```

1. **预热期 (0 ~ 50 VUs, 30s)**：建立连接池，填充进程内 L1 客户端缓存。
2. **阶梯 1 (50 VUs, 1m)**：基础吞吐观测，验证多级缓存命中率。
3. **阶梯 2 (200 VUs, 2m)**：日常高峰流量模拟，观察延迟与 CPU/内存波动。
4. **极限冲刺 (500 VUs, 1m)**：高压冲击，验证单用户令牌桶限流与 Redis 断路器。
5. **降压收尾 (500 ~ 0 VUs, 30s)**：观察资源回收与连接释放。

---

## 🎯 核心监控指标与阈值 (Thresholds)

| 指标项 | 含义 | 期望阈值 | 达标意义 |
| :--- | :--- | :--- | :--- |
| `http_req_failed` | 请求失败率（5xx/连接异常） | `< 1%` | 服务高可用，无服务级联雪崩 |
| `http_req_duration` | 综合接口响应时延 | `P95 < 300ms, P99 < 800ms` | 保证高并发下良好的交互体验 |
| `trend_posts_list_duration` | 热榜列表接口时延 | `P95 < 150ms` | 验证 Redis ZSet + 内存重排的高效性 |
| `trend_post_detail_duration`| 帖子详情接口时延 | `P95 < 100ms` | 验证 L1 客户端缓存 + L2 Redis 的加速效果 |
| `trend_vote_duration` | 投票接口时延 | `P95 < 200ms` | 验证纯 Redis Lua 内存化写入与极速响应 |
| `counter_vote_rate_limited` | 投票触发 429 的次数 | 自定义计数器 | 验证单用户令牌桶限流是否精准拦截刷票 |

---

## 📈 查看可视化 HTML 报表

压测执行完毕后，脚本会在当前工作目录下自动生成 **`summary.html`**。直接在浏览器中打开：

```powershell
start summary.html
```

报告内直观展示：
- 吞吐量曲线（RPS / QPS 随并发用户的变化）
- 响应时间百分位数分布（P50 / P90 / P95 / P99）
- 各业务接口独立耗时对比
- 成功率与限流拦截统计
