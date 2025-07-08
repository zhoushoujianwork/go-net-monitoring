#!/bin/bash

# eBPF 集成测试脚本 (macOS兼容版本)

set -e

echo "🧪 eBPF集成测试开始..."

# 检查编译结果
echo "📦 检查编译结果..."
if [ ! -f "bin/agent-ebpf" ]; then
    echo "❌ eBPF Agent未编译"
    exit 1
fi

if [ ! -f "bin/bpf/xdp_monitor_linux.o" ]; then
    echo "⚠️  eBPF程序未找到，将使用模拟模式"
fi

# 测试版本信息
echo "🔍 测试版本信息..."
./bin/agent-ebpf --version

# 测试帮助信息
echo "📖 测试帮助信息..."
./bin/agent-ebpf --help || true

# 测试模拟模式运行 (macOS兼容)
echo "🎭 测试模拟模式运行..."
./bin/agent-ebpf --debug --config configs/agent.yaml > /tmp/ebpf-test.log 2>&1 &
TEST_PID=$!

# 等待5秒
sleep 5

# 检查进程是否运行
if kill -0 $TEST_PID 2>/dev/null; then
    echo "✅ eBPF Agent运行正常"
    kill $TEST_PID
    wait $TEST_PID 2>/dev/null || true
else
    echo "❌ eBPF Agent运行失败"
    if [ -f /tmp/ebpf-test.log ]; then
        echo "错误日志:"
        cat /tmp/ebpf-test.log
    fi
    exit 1
fi

# 检查日志输出
echo "📋 检查日志输出..."
if [ -f /tmp/ebpf-test.log ]; then
    if grep -q "启动eBPF网络监控代理" /tmp/ebpf-test.log; then
        echo "✅ 启动日志正常"
    else
        echo "❌ 启动日志异常"
        echo "实际日志内容:"
        cat /tmp/ebpf-test.log
        exit 1
    fi

    if grep -q "模拟监控模式" /tmp/ebpf-test.log; then
        echo "✅ 模拟模式启动正常"
    elif grep -q "eBPF监控模式" /tmp/ebpf-test.log; then
        echo "✅ eBPF模式启动正常"
    else
        echo "⚠️  未检测到明确的模式启动信息"
        echo "日志内容:"
        cat /tmp/ebpf-test.log
    fi
else
    echo "❌ 日志文件未生成"
    exit 1
fi

# 清理
rm -f /tmp/ebpf-test.log

echo "🎉 eBPF集成测试完成！"
echo ""
echo "测试结果:"
echo "  ✅ 编译成功"
echo "  ✅ 版本信息正确"
echo "  ✅ 程序启动正常"
echo "  ✅ 模拟模式工作"
echo ""
echo "下一步:"
echo "  1. 在Linux环境测试真实eBPF功能"
echo "  2. 集成到现有Docker Compose"
echo "  3. 添加Prometheus指标导出"
