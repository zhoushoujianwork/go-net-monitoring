# 多阶段构建Dockerfile
FROM golang:1.21-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git make libpcap-dev gcc musl-dev

# 设置工作目录
WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN make build

# 运行时镜像
FROM alpine:latest

# 安装运行时依赖
RUN apk add --no-cache libpcap ca-certificates tzdata

# 创建非root用户
RUN addgroup -g 1001 -S netmon && \
    adduser -u 1001 -S netmon -G netmon

# 创建必要目录
RUN mkdir -p /app/bin /app/configs /app/logs && \
    chown -R netmon:netmon /app

# 复制二进制文件
COPY --from=builder /app/bin/agent /app/bin/
COPY --from=builder /app/bin/server /app/bin/

# 复制配置文件
COPY --from=builder /app/configs/ /app/configs/

# 设置权限
RUN chmod +x /app/bin/agent /app/bin/server

# 设置工作目录
WORKDIR /app

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 默认运行server
CMD ["/app/bin/server", "--config", "/app/configs/server.yaml"]
