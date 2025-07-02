#!/bin/bash

# 测试 Docker Compose 部署脚本

set -e

echo "🚀 开始测试 Docker Compose 部署..."

# 清理之前的部署
echo "📦 清理之前的部署..."
docker-compose down -v 2>/dev/null || true

# 测试默认部署 (Redis 存储)
echo "🔧 测试默认部署 (Redis 存储)..."
docker-compose up -d

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 30

# 检查服务状态
echo "📊 检查服务状态..."
docker-compose ps

# 测试服务连接
echo "🔍 测试服务连接..."

# 测试 Redis
echo "  - 测试 Redis 连接..."
docker-compose exec -T redis redis-cli ping

# 测试 Server 健康检查
echo "  - 测试 Server 健康检查..."
curl -f http://localhost:8080/health || echo "健康检查失败"

# 测试指标端点
echo "  - 测试指标端点..."
curl -f http://localhost:8080/metrics | head -10

# 检查日志
echo "📝 检查服务日志..."
echo "=== Server 日志 ==="
docker-compose logs --tail=10 server

echo "=== Agent 日志 ==="
docker-compose logs --tail=10 agent

echo "=== Redis 日志 ==="
docker-compose logs --tail=10 redis

# 测试内存存储模式
echo "🧠 测试内存存储模式..."
docker-compose --profile memory up -d server-memory

# 等待服务启动
sleep 10

# 测试内存存储服务
echo "  - 测试内存存储服务..."
curl -f http://localhost:8081/health || echo "内存存储服务健康检查失败"
curl -f http://localhost:8081/metrics | head -5

# 清理测试
echo "🧹 清理测试环境..."
docker-compose down
docker-compose --profile memory down

echo "✅ Docker Compose 部署测试完成！"

echo "
📋 部署命令总结:
  默认部署 (Redis):     docker-compose up -d
  内存存储模式:         docker-compose --profile memory up -d server-memory agent
  完整监控栈:           docker-compose --profile monitoring up -d
  
🌐 服务访问地址:
  Server (Redis):       http://localhost:8080
  Server (Memory):      http://localhost:8081
  Prometheus:           http://localhost:9090
  Grafana:              http://localhost:3000 (admin/admin123)
"
