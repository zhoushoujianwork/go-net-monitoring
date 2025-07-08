#!/bin/bash

# Docker 集成测试脚本 (简化版本)

set -e

echo "🐳 Docker集成测试..."

# 检查Docker是否运行
if ! docker info >/dev/null 2>&1; then
    echo "❌ Docker未运行"
    exit 1
fi

# 测试现有Docker Compose
echo "📦 测试现有Docker Compose..."
if [ -f "docker-compose.yml" ]; then
    echo "✅ 找到docker-compose.yml"
    
    # 测试配置语法
    if docker-compose -f docker-compose.yml config >/dev/null 2>&1; then
        echo "✅ Docker Compose配置正确"
    else
        echo "❌ Docker Compose配置错误"
        exit 1
    fi
else
    echo "❌ 未找到docker-compose.yml"
    exit 1
fi

# 测试eBPF Docker Compose
echo "📦 测试eBPF Docker Compose..."
if [ -f "docker-compose.ebpf.yml" ]; then
    echo "✅ 找到docker-compose.ebpf.yml"
    
    # 测试配置语法
    if docker-compose -f docker-compose.ebpf.yml config >/dev/null 2>&1; then
        echo "✅ eBPF Docker Compose配置正确"
    else
        echo "❌ eBPF Docker Compose配置错误"
        docker-compose -f docker-compose.ebpf.yml config
        exit 1
    fi
else
    echo "❌ 未找到docker-compose.ebpf.yml"
    exit 1
fi

# 测试构建脚本
echo "🔧 测试构建脚本..."
if [ -f "scripts/docker-ebpf.sh" ]; then
    echo "✅ 找到docker-ebpf.sh"
    
    # 测试帮助命令
    if ./scripts/docker-ebpf.sh help >/dev/null 2>&1; then
        echo "✅ 构建脚本可执行"
    else
        echo "❌ 构建脚本执行失败"
        exit 1
    fi
else
    echo "❌ 未找到docker-ebpf.sh"
    exit 1
fi

# 检查必要文件
echo "📋 检查必要文件..."
files=(
    "bin/agent-ebpf"
    "bin/bpf/xdp_monitor_linux.o"
    "configs/agent.yaml"
    "monitoring/prometheus.yml"
)

for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "✅ $file"
    else
        echo "⚠️  $file (缺失，但可能不影响测试)"
    fi
done

echo "🎉 Docker集成测试完成！"
echo ""
echo "测试结果:"
echo "  ✅ Docker环境正常"
echo "  ✅ Docker Compose配置正确"
echo "  ✅ eBPF Docker Compose配置正确"
echo "  ✅ 构建脚本可执行"
echo ""
echo "下一步:"
echo "  1. 运行: ./scripts/docker-ebpf.sh up"
echo "  2. 或者: docker-compose -f docker-compose.ebpf.yml up"
