# 设计文档：Tempo 配置文件逐行注释

## 1. 背景与目标
为了帮助开发和运维人员更好地理解 Grafana Tempo 的单机配置结构，需要对 `d:\download\project\bluebell\tempo-config.yaml` 配置文件进行逐行/逐项的详细中文注释。注释应阐明每个组件（Server, Distributor, Ingester, Compactor, Storage）的核心概念及参数配置目的。

## 2. 详细设计
我们将直接在 `tempo-config.yaml` 文件中添加内联注释。为了便于阅读，注释统一采用 `# 注释内容` 的格式，并对齐在参数行的后方或单独作为一行写在配置块的上方。

配置内容对应的注释要点：
- **server**: 阐述 Tempo HTTP 服务的作用（主要是 metrics 监控和管理接口）。
- **distributor**: 解释分发器和接收协议（gRPC 端口 4317，HTTP 端口 4318）。
- **ingester**: 解释写入器组件、环配置（Ring）以及副本因子（replication_factor: 1 为单机模式）。
- **compactor**: 说明压缩器组件及内存哈希环（inmemory）。
- **storage**: 细化 Trace 的本地存储后端（local）、Block 块路径和 WAL（Write-Ahead Log）路径的作用。

## 3. 验收标准
- 配置文件格式正确，Tempo 可以正常读取。
- 每一行核心配置均有清晰直白的中文注释说明。
