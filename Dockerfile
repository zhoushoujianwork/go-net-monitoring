# 多阶段构建 Dockerfile
FROM golang:1.21-alpine AS builder

# 安装构建依赖 (合并到一个RUN层)
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    libpcap-dev

WORKDIR /app

# 优化：先复制依赖文件，利用Docker缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码 (放在最后，避免代码变更影响依赖缓存)
COPY . .

# 构建参数
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

# 并行构建两个二进制文件 (优化构建时间)
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT} -s -w" \
    -o agent ./cmd/agent & \
    CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT} -s -w" \
    -o server ./cmd/server & \
    wait

# 运行时镜像 - 使用更新的Alpine版本
FROM alpine:3.19

# 优化：合并所有安装步骤到一个层，减少镜像大小
RUN apk add --no-cache \
    ca-certificates \
    libpcap \
    tzdata \
    tcpdump \
    iproute2 \
    net-tools \
    procps \
    curl \
    wget \
    && addgroup -g 1000 netmon \
    && adduser -D -s /bin/sh -u 1000 -G netmon netmon \
    && mkdir -p /app/data \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# 复制二进制文件和配置
COPY --from=builder /app/agent /app/server /usr/local/bin/
COPY --from=builder /app/configs/ /app/configs/
COPY docker/entrypoint.sh /entrypoint.sh

# 设置权限 (合并到一个RUN层)
RUN chmod +x /entrypoint.sh \
    && chown -R netmon:netmon /app

# 暴露端口
EXPOSE 8080

# 优化健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD if [ "$COMPONENT" = "server" ]; then \
            wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1; \
        else \
            pgrep agent > /dev/null || exit 1; \
        fi

# 设置默认环境变量
ENV COMPONENT=server \
    LOG_LEVEL=info

# 使用启动脚本
ENTRYPOINT ["/entrypoint.sh"]
