#!/bin/bash

echo "🧪 快速测试IP和域名统计..."

# 检查权限
if [ "$EUID" -ne 0 ]; then
    echo "❌ 需要root权限"
    echo "请使用: sudo $0"
    exit 1
fi

# 清理端口
lsof -ti:8080 | xargs kill -9 2>/dev/null || true
sleep 1

# 启动Server
echo "🖥️  启动Server..."
./bin/server --config configs/server.yaml &
SERVER_PID=$!
sleep 3

# 启动Agent
echo "🤖 启动Agent..."
./bin/agent --config configs/agent-debug.yaml &
AGENT_PID=$!
sleep 5

echo "🌐 生成网络流量..."
# 生成一些简单的网络请求
curl -s http://httpbin.org/ip >/dev/null &
curl -s https://www.google.com >/dev/null &
sleep 5

echo "📊 检查指标..."
curl -s http://localhost:8080/metrics | grep -E "(network_ips_accessed|network_domains_accessed|network_protocol_stats)" | head -10

echo ""
echo "🧹 清理..."
kill $AGENT_PID $SERVER_PID 2>/dev/null || true

echo "✅ 测试完成"
