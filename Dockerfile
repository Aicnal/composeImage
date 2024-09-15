# 使用官方的Go镜像作为基础镜像
FROM golang:1.20 AS builder

# 设置工作目录
WORKDIR /app

# 复制当前目录内容到工作目录
COPY . .

# 下载依赖
RUN go mod download

# 编译Go程序
RUN go build -o composeImage .

# 使用一个更小的基础镜像来运行编译后的二进制文件
FROM alpine:latest

# 安装运行二进制文件所需的依赖项
RUN apk add --no-cache libc6-compat

# 设置工作目录
WORKDIR /app

# 从构建阶段复制编译后的二进制文件
COPY --from=builder /app/composeImage .

# 设置执行权限
RUN chmod +x composeImage

# 运行程序
CMD ["./composeImage", "-input", "/input", "-output", "/output", "-quality", "90", "-workers", "4"]