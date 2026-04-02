# 构建阶段
FROM node:20-alpine AS builder

WORKDIR /app

# 复制依赖文件
COPY package*.json ./
RUN npm ci

# 复制源代码
COPY . .

# 构建生产版本
RUN npm run build

# 运行阶段 - 仅作为产物提供者
FROM alpine:latest

WORKDIR /app

# 从构建阶段复制 dist
COPY --from=builder /app/dist ./dist

# 设置启动命令：将产物拷贝到共享卷目录后退出
CMD sh -c "cp -r /app/dist/* /usr/share/nginx/html/ && echo 'Frontend assets delivered!'"
