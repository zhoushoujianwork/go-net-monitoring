# 多阶段构建 Dockerfile (修复版本)
FROM golang:1.23-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    make

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

# 创建 bin/bpf 目录并复制必要文件
RUN mkdir -p /app/bin/bpf && \
    if [ -d "/app/bpf" ]; then \
        find /app/bpf -name "*.o" -exec cp {} /app/bin/bpf/ \; 2>/dev/null || true; \
    fi

# 构建指定组件
RUN echo "构建组件: ${COMPONENT}" && \
    CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-w -s" \
        -o ${COMPONENT} ./cmd/${COMPONENT}/

# 运行时镜像
FROM alpine:3.19

# 安装运行时依赖
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

# 创建 eBPF 目录并复制文件（如果存在）
RUN mkdir -p /opt/go-net-monitoring/bpf
COPY --from=builder /app/bin/bpf /opt/go-net-monitoring/bpf/ || true
COPY --from=builder /app/bpf/programs /opt/go-net-monitoring/bpf/programs/ || true

# 设置权限
RUN chown -R netmon:netmon /app && \
    chmod +x /app/entrypoint.sh && \
    chown -R netmon:netmon /opt/go-net-monitoring && \
    chmod 644 /opt/go-net-monitoring/bpf/*.o 2>/dev/null || true

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD pgrep -f "${COMPONENT}" || exit 1

# 暴露端口
EXPOSE 8080

# 环境变量
ENV PATH="/app:${PATH}"
ENV COMPONENT=${COMPONENT}

# 切换到非root用户
USER netmon

# 启动脚本
ENTRYPOINT ["/app/entrypoint.sh"]
