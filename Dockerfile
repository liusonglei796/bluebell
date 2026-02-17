FROM golang:alpine AS builder

# 设置 Go 代理，加快下载速度 (针对中国用户优化)
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 编译应用
RUN CGO_ENABLED=0 GOOS=linux go build -o bluebell .

FROM alpine:latest

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/bluebell .

# 暴露端口
EXPOSE 8080

# 启动命令
CMD ["./bluebell", "-conf", "config.yaml"]
