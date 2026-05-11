#!/bin/bash
# ============================================
# Bluebell 单机服务器部署脚本
# Usage: ./deploy.sh
# ============================================

set -e

APP_NAME="bluebell"
PROJECT_DIR=$(cd "$(dirname "$0")/.." && pwd)
DEPLOY_DIR="$PROJECT_DIR/server-deploy"

echo "===== Bluebell 部署脚本 ====="
echo "项目目录: $PROJECT_DIR"

# 检查 Docker 和 Docker Compose
if ! command -v docker &> /dev/null; then
    echo "错误: 未安装 Docker"
    exit 1
fi

if ! docker compose version &> /dev/null; then
    echo "错误: 未安装 Docker Compose v2"
    exit 1
fi

echo ""
echo "===== 1. 拉取最新代码 ====="
cd "$PROJECT_DIR"
git pull origin main

echo ""
echo "===== 2. 构建并启动服务 ====="
docker compose build --parallel
docker compose up -d

echo ""
echo "===== 3. 查看服务状态 ====="
docker compose ps

echo ""
echo "===== 部署完成! ====="
echo "访问 http://localhost 查看应用"