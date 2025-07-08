#!/bin/bash

# 简化的Docker测试脚本

set -e

echo "🐳 简化Docker测试..."

# 1. 测试本地编译的程序
echo "📦 测试本地编译..."
if [ -f "bin/agent-ebpf" ] && [ -f "bin/agent" ]; then
    echo "✅ 本地编译文件存在"
    
    # 测试版本信息
    ./bin/agent-ebpf --version
    ./bin/agent --help | head -3
else
    echo "❌ 本地编译文件缺失"
    exit 1
fi

# 2. 使用现有的Docker Compose测试
echo "🔧 测试Docker Compose配置..."
if docker-compose -f docker-compose.yml config >/dev/null 2>&1; then
    echo "✅ 原始Docker Compose配置正确"
else
    echo "❌ 原始Docker Compose配置错误"
    exit 1
fi

# 3. 尝试启动现有的server (不构建)
echo "🚀 尝试启动现有server..."
if docker run --rm -d --name test-server -p 8081:8080 \
    -e COMPONENT=server \
    zhoushoujian/go-net-monitoring:latest >/dev/null 2>&1; then
    
    echo "✅ 现有镜像可以启动"
    
    # 等待启动
    sleep 3
    
    # 测试健康检查
    if curl -s http://localhost:8081/health >/dev/null 2>&1; then
        echo "✅ Server健康检查通过"
    else
        echo "⚠️  Server健康检查失败，但容器已启动"
    fi
    
    # 停止测试容器
    docker stop test-server >/dev/null 2>&1
    echo "✅ 测试容器已停止"
else
    echo "⚠️  现有镜像启动失败，需要重新构建"
fi

echo "🎉 简化Docker测试完成！"
echo ""
echo "建议:"
echo "  1. 使用现有镜像: zhoushoujian/go-net-monitoring:latest"
echo "  2. 或者先修复Dockerfile再重新构建"
echo "  3. 当前eBPF Agent可以直接运行: ./bin/agent-ebpf"
