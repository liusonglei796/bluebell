FROM golang:1.24-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

COPY . .

# 编译应用
RUN CGO_ENABLED=0 GOOS=linux go build -o bluebell ./cmd/bluebell

FROM alpine:latest

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/bluebell .

# 复制配置文件
COPY config.docker.toml config.toml


# 启动命令
CMD ["./bluebell", "-conf", "config.toml"]
