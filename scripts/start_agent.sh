#!/bin/bash

echo "🚀 启动网络监控Agent..."

# 检查是否以root权限运行
if [ "$EUID" -ne 0 ]; then
    echo "❌ 需要root权限进行网络数据包捕获"
    echo "请使用: sudo $0"
    exit 1
fi

# 构建项目
echo "📦 构建项目..."
make build

# 检查端口占用
if lsof -ti:8080 >/dev/null 2>&1; then
    echo "⚠️  端口8080被占用，尝试清理..."
    lsof -ti:8080 | xargs kill -9 2>/dev/null || true
    sleep 1
fi

# 启动Server
echo "🖥️  启动Server..."
./bin/server --config configs/server.yaml &
SERVER_PID=$!
sleep 3

# 检查Server是否启动成功
if ! curl -s http://localhost:8080/health >/dev/null; then
    echo "❌ Server启动失败"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
fi

echo "✅ Server启动成功"

# 设置信号处理
cleanup() {
    echo ""
    echo "🧹 清理进程..."
    kill $SERVER_PID 2>/dev/null || true
    exit 0
}

trap cleanup SIGINT SIGTERM

# 启动Agent
echo "🤖 启动Agent..."
echo "按 Ctrl+C 停止监控"
echo ""

# 使用exec替换当前进程，这样信号处理更直接
exec ./bin/agent --config configs/agent.yaml
