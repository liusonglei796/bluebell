#!/bin/bash
# ============================================
# Bluebell 单机服务器清理脚本
# Usage: ./cleanup.sh
# ============================================

PROJECT_DIR=$(cd "$(dirname "$0")/.." && pwd)

echo "===== Bluebell 清理 ====="
echo "警告: 这将删除所有数据卷! 按 Ctrl+C 取消..."

sleep 3

cd "$PROJECT_DIR"

docker compose down -v
docker compose rm -f

echo ""
echo "清理完成。"