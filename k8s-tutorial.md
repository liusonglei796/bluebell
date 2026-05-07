# Bluebell 项目 Kubernetes (K8s) 部署与学习指南

本指南旨在帮助你从零开始，在 Multipass 本地虚拟机环境中使用 K3s 部署 `bluebell` 全栈项目，并深入理解 K8s 的核心概念。

---

## 1. 基础环境准备

### 1.1 虚拟机准备 (Multipass)
为了省内存，我们使用了单节点 K3s 实例。
- **配置**：Ubuntu 24.04, 2 CPU, 4GB RAM, 20GB Disk.
- **原因**：8GB 的主机建议分配 4GB 给虚拟机，以保证宿主机和虚拟机都能稳定运行。

### 1.2 K3s 安装
K3s 是轻量级的 K8s 发行版，非常适合本地学习。
```bash
curl -sfL https://get.k3s.io | sh -
```

---

## 2. 镜像管理策略

在 K8s 中，Pod 无法直接访问你本地磁盘上的镜像。你需要将镜像导入到 K8s 的容器运行时（K3s 默认使用 containerd）。

### 2.1 镜像构建 (宿主机)
- **后端**：使用 `Dockerfile` 编译 Go 并打包。
- **前端**：使用 `Dockerfile.k8s` (基于 Nginx)，将 `npm run build` 后的 `dist` 目录打包。

### 2.2 镜像传输与导入
这是本地模拟中最关键的一步：
1. **导出**：`docker save -o image.tar image:tag`
2. **传输**：`multipass transfer image.tar vm-name:/home/ubuntu/`
3. **导入**：`sudo k3s ctr images import /home/ubuntu/image.tar`

---

## 3. K8s 核心资源详解

项目的所有部署文件位于 `k8s-deploy/` 目录下。

### 3.1 Deployment (部署)
定义了应用的副本数、镜像和环境变量。
- **避坑指南**：K8s 会自动给 Pod 注入名为 `SERVICE_NAME_PORT` 的环境变量。如果你的服务名叫 `redis`，它会注入 `REDIS_PORT=tcp://...`，这会冲突并导致 Go 程序无法解析端口。
- **解决方法**：在 Deployment 的 `spec.template.spec` 中设置 `enableServiceLinks: false`。

### 3.2 Service (服务)
提供稳定的 IP 和端口供其他 Pod 访问。
- **MySQL/Redis/ES/RabbitMQ**：通过 Service Name (如 `mysql`, `redis`) 进行集群内通信。

### 3.3 Ingress (入口网关)
K3s 自带 Traefik 作为 Ingress Controller。
- 我们配置了 `/api` 转发到后端，`/` 转发到前端，实现前后端同域访问。

---

## 4. 实操步骤回顾

1. **部署数据库与中间件**：
   ```bash
   kubectl apply -f k8s-deploy/db.yaml
   kubectl apply -f k8s-deploy/middleware.yaml
   ```
2. **部署应用层**：
   ```bash
   kubectl apply -f k8s-deploy/app.yaml
   ```
3. **部署网关**：
   ```bash
   kubectl apply -f k8s-deploy/ingress.yaml
   ```

---

## 5. 常用排障命令

- **查看 Pod 状态**：`kubectl get pods`
- **查看实时日志**：`kubectl logs -f deployment/bluebell-app`
- **查看 Pod 详情 (报错排查)**：`kubectl describe pod <pod-name>`
- **进入容器内部**：`kubectl exec -it <pod-name> -- /bin/sh`

---

## 6. 进阶练习建议
1. **持久化存储**：目前数据库使用的是 `emptyDir`，重启后数据会丢失。尝试学习 `PersistentVolume (PV)` 和 `PVC` 来持久化数据。
2. **扩缩容**：尝试运行 `kubectl scale deployment bluebell-app --replicas=3`，观察流量负载。
3. **ConfigMap**：尝试将 `config.toml` 的内容放入 K8s 的 `ConfigMap` 中，通过挂载卷的方式注入到容器里。
