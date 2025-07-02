# 多阶段构建 Dockerfile
FROM golang:1.21-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    libpcap-dev

WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建二进制文件
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT} -s -w" \
    -o agent ./cmd/agent

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT} -s -w" \
    -o server ./cmd/server

# 运行时镜像
FROM alpine:3.18

# 安装运行时依赖和网络监控工具
RUN apk add --no-cache \
    ca-certificates \
    libpcap \
    tzdata \
    tcpdump \
    iproute2 \
    net-tools \
    procps \
    && rm -rf /var/cache/apk/*

# 创建非root用户
RUN addgroup -g 1000 netmon && \
    adduser -D -s /bin/sh -u 1000 -G netmon netmon

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/agent /app/server /usr/local/bin/
COPY --from=builder /app/configs/ /app/configs/

# 复制启动脚本
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# 创建数据目录
RUN mkdir -p /app/data && chown -R netmon:netmon /app

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD if [ "$COMPONENT" = "server" ]; then \
            wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1; \
        else \
            pgrep agent > /dev/null || exit 1; \
        fi

# 设置默认环境变量
ENV COMPONENT=server
ENV CONFIG_FILE=/app/configs/server.yaml
ENV LOG_LEVEL=info

# 使用启动脚本
ENTRYPOINT ["/entrypoint.sh"]
