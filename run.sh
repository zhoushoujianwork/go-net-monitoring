#!/bin/bash

echo "🚀 启动网络监控系统..."

# 清理可能影响容器间通信的代理设置
echo "清理代理环境变量..."
unset http_proxy https_proxy HTTP_PROXY HTTPS_PROXY

# 设置容器内部通信的 no_proxy
export no_proxy="localhost,127.0.0.1,server,redis,prometheus,grafana"
export NO_PROXY="localhost,127.0.0.1,server,redis,prometheus,grafana"

# 使用主要的 Docker Compose 配置
echo "停止现有服务..."
docker-compose --profile monitoring down 2>/dev/null || true

echo "启动服务..."
docker-compose --profile monitoring up
