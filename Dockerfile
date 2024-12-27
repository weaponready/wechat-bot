# 使用官方 Golang 镜像作为构建阶段
FROM golang:1.23 AS builder

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum（如果有）
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制项目文件
COPY . .

# 构建可执行文件
RUN go build -o main .

# 使用更小的镜像运行构建的二进制文件
FROM debian:bullseye-slim

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/main .

# 暴露服务端口（根据项目需要调整）
EXPOSE 8080

# 启动应用程序
CMD ["./main"]
