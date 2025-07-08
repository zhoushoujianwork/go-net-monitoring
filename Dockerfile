# 多阶段构建 Dockerfile (无libpcap依赖版本)
FROM golang:1.23-alpine AS builder

# 安装构建依赖 (移除libpcap-dev)
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev

WORKDIR /app

# 优化：先复制依赖文件，利用Docker缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建参数
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT
ARG COMPONENT=server

# 构建指定组件
RUN echo "构建组件: ${COMPONENT}" && \
    CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-w -s" \
        -o ${COMPONENT} ./cmd/${COMPONENT}/

# 运行时镜像
FROM alpine:3.19

# 安装运行时依赖 (移除libpcap)
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    && addgroup -g 1000 netmon \
    && adduser -D -s /bin/sh -u 1000 -G netmon netmon \
    && mkdir -p /app/data \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# 复制二进制文件和配置
ARG COMPONENT=server
COPY --from=builder /app/${COMPONENT} /app/
COPY --from=builder /app/configs /app/configs
COPY --from=builder /app/docker/entrypoint.sh /app/

# 设置权限
RUN chown -R netmon:netmon /app && \
    chmod +x /app/entrypoint.sh

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep -f "${COMPONENT}" || exit 1

# 暴露端口
EXPOSE 8080

# 环境变量
ENV COMPONENT=${COMPONENT}
ENV LOG_LEVEL=info

# 使用非root用户
USER netmon

ENTRYPOINT ["/app/entrypoint.sh"]
