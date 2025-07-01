#!/bin/bash

# 快速测试脚本
set -e

echo "🚀 开始测试网络监控系统..."

# 构建项目
echo "📦 构建项目..."
make build

# 启动Server
echo "🖥️  启动Server..."
./bin/server --config configs/server.yaml &
SERVER_PID=$!
sleep 2

# 检查Server是否启动成功
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ Server启动成功"
else
    echo "❌ Server启动失败"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
fi

# 启动测试模式的Agent
echo "🤖 启动测试Agent..."
TEST_MODE=true ./bin/agent --config configs/agent.yaml &
AGENT_PID=$!
sleep 5

# 检查指标
echo "📊 检查指标..."
if curl -s http://localhost:8080/metrics | grep -q "network_"; then
    echo "✅ 指标暴露正常"
else
    echo "❌ 指标暴露异常"
fi

# 检查API
echo "🔍 检查API..."
if curl -s http://localhost:8080/api/v1/status | grep -q "running"; then
    echo "✅ API响应正常"
else
    echo "❌ API响应异常"
fi

# 清理
echo "🧹 清理进程..."
kill $AGENT_PID 2>/dev/null || true
kill $SERVER_PID 2>/dev/null || true

echo "🎉 测试完成！"
echo ""
echo "📋 使用说明:"
echo "1. 启动Server: ./bin/server --config configs/server.yaml"
echo "2. 启动Agent (测试模式): TEST_MODE=true ./bin/agent --config configs/agent.yaml"
echo "3. 启动Agent (真实模式): sudo ./bin/agent --config configs/agent.yaml"
echo "4. 查看指标: curl http://localhost:8080/metrics"
echo "5. 查看状态: curl http://localhost:8080/api/v1/status"
