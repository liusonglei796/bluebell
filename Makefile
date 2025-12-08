.PHONY: all build run gotool clean help
# 2. 定义变量
BINARY="bluebell"
# 3. 定义默认目标 'all'，它依赖 'gotool' 和 'build'
all: gotool build
# 4. 定义 'build' 目标
#    - CGO_ENABLED=0: 禁用 CGo，实现静态编译
#    - GOOS=linux: 指定目标操作系统为 Linux (实现跨平台编译)
#    - GOARCH=amd64: 指定目标架构
#    - ${BINARY}: 使用变量作为输出文件名
build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${BINARY}
# 5. 定义 'run' 目标，用于本地运行
#    - @ 符号：阻止 make 打印该命令本身，使输出更干净
run:
	@go run ./main.go ./config.yaml
# 6. 定义 'gotool' 目标，用于运行 Go 工具
#    - ./... 语法：表示当前目录及所有子目录
gotool:
	go fmt ./...
	go vet ./...
# 7. 定义 'clean' 目标，用于清理
clean:
	@if [ -f ${BINARY} ]; then rm ${BINARY}; fi
# 8. 定义 'help' 目标，用于显示帮助信息
help:
	@echo "make - 格式化 Go 代码, 并编译生成二进制文件"
	@echo "make build - 编译 Go 代码, 生成二进制文件"
	@echo "make run - 直接运行 Go 代码"
	@echo "make clean - 移除二进制文件"
	@echo "make gotool - 运行 Go 工具 'fmt' and 'vet'"