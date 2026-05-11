#!/bin/bash
# ============================================
# Bluebell 单机服务器重启脚本
# Usage: ./restart.sh
# ============================================

set -e

PROJECT_DIR=$(cd "$(dirname "$0")/.." && pwd)

echo "===== Bluebell 重启 ====="

cd "$PROJECT_DIR"

echo "停止服务..."
docker compose down

echo "重新构建并启动..."
docker compose build --parallel
docker compose up -d

echo ""
echo "服务状态:"
docker compose ps