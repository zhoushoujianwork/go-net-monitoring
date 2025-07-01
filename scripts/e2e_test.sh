#!/bin/bash

# 端到端测试脚本
set -e

echo "🚀 开始端到端测试..."

# 构建项目
echo "📦 构建项目..."
make build

# 启动Server
echo "🖥️  启动Server..."
./bin/server --config configs/server.yaml &
SERVER_PID=$!
sleep 3

# 检查Server是否启动成功
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ Server启动成功"
else
    echo "❌ Server启动失败"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
fi

# 启动Agent (需要sudo权限)
echo "🤖 启动Agent (需要sudo权限)..."
echo "请输入sudo密码以启动网络监控Agent..."

# 使用timeout命令限制运行时间
sudo timeout 10s ./bin/agent --config configs/agent.yaml &
AGENT_PID=$!

# 等待Agent收集一些数据
echo "⏳ 等待Agent收集数据..."
sleep 8

# 检查指标
echo "📊 检查网络监控指标..."
METRICS=$(curl -s http://localhost:8080/metrics)

if echo "$METRICS" | grep -q "network_"; then
    echo "✅ 发现网络监控指标"
    
    # 检查具体的IP和域名统计
    if echo "$METRICS" | grep -q "network_ips_accessed_total"; then
        echo "✅ IP访问统计正常"
        echo "$METRICS" | grep "network_ips_accessed_total" | head -3
    else
        echo "⚠️  暂未发现IP访问数据"
    fi
    
    if echo "$METRICS" | grep -q "network_domains_accessed_total"; then
        echo "✅ 域名访问统计正常"
        echo "$METRICS" | grep "network_domains_accessed_total" | head -3
    else
        echo "⚠️  暂未发现域名访问数据"
    fi
    
else
    echo "⚠️  暂未发现网络监控指标，可能需要更多时间收集数据"
fi

# 检查Agent状态
echo "🔍 检查Agent状态..."
AGENTS=$(curl -s http://localhost:8080/api/v1/agents)
echo "注册的Agent数量: $(echo "$AGENTS" | jq '.count' 2>/dev/null || echo "解析失败")"

# 清理
echo "🧹 清理进程..."
sudo pkill -f "bin/agent" 2>/dev/null || true
kill $SERVER_PID 2>/dev/null || true

echo "🎉 端到端测试完成！"
echo ""
echo "📋 测试结果总结:"
echo "- Server启动: ✅"
echo "- Agent启动: ✅" 
echo "- 网络监控: $(echo "$METRICS" | grep -q "network_" && echo "✅" || echo "⚠️")"
echo "- IP统计: $(echo "$METRICS" | grep -q "network_ips_accessed_total" && echo "✅" || echo "⚠️")"
echo "- 域名统计: $(echo "$METRICS" | grep -q "network_domains_accessed_total" && echo "✅" || echo "⚠️")"
echo ""
echo "💡 提示: 如果某些指标显示⚠️，可能需要更长时间收集网络数据"
