#!/bin/bash

# 测试环境变量修复

echo "🧪 测试环境变量修复..."

# 清理旧容器
echo "清理旧容器..."
docker rm -f netmon-agent-test 2>/dev/null || true

# 启动测试容器
echo "启动测试容器..."
docker run -d \
  --name netmon-agent-test \
  --privileged \
  --network host \
  -e COMPONENT=agent \
  -e DEBUG_MODE=false \
  -e SERVER_URL=http://192.168.3.233:8080/api/v1/metrics \
  go-net-monitoring:latest

# 等待容器启动
echo "等待容器启动..."
sleep 3

# 检查容器状态
echo "检查容器状态..."
if docker ps | grep -q netmon-agent-test; then
    echo "✅ 容器启动成功"
else
    echo "❌ 容器启动失败"
    docker logs netmon-agent-test
    exit 1
fi

# 检查配置文件
echo "检查生成的配置文件..."
echo "=== Agent配置文件 ==="
docker exec netmon-agent-test cat /app/configs/agent.yaml

# 检查日志
echo "=== 容器日志 (前20行) ==="
docker logs netmon-agent-test 2>&1 | head -20

# 检查是否还有debug日志
echo "=== 检查debug日志 ==="
if docker logs netmon-agent-test 2>&1 | grep -q '"level":"debug"'; then
    echo "❌ 仍然有debug日志，DEBUG_MODE=false没有生效"
else
    echo "✅ 没有debug日志，DEBUG_MODE=false生效"
fi

# 检查SERVER_URL是否生效
echo "=== 检查SERVER_URL ==="
if docker logs netmon-agent-test 2>&1 | grep -q "192.168.3.233:8080"; then
    echo "✅ SERVER_URL生效，正在连接192.168.3.233:8080"
else
    echo "❌ SERVER_URL没有生效，仍在连接localhost"
fi

# 清理测试容器
echo "清理测试容器..."
docker rm -f netmon-agent-test

echo "🎯 测试完成"
