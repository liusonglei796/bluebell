FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 复制本地编译好的二进制文件 (注意：需要在本地先执行交叉编译)
# $env:CGO_ENABLED=0; $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o bluebell-linux .
COPY bluebell-linux ./bluebell

# 赋予执行权限
RUN chmod +x ./bluebell

# 暴露端口
EXPOSE 8080

# 启动命令
CMD ["./bluebell", "-conf", "config.yaml"]
