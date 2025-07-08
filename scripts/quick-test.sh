#!/bin/bash

# 快速测试脚本

set -e

echo "🚀 Quick eBPF test..."

# 只构建镜像（使用缓存）
echo "📦 Building Docker image..."
docker build -f docker/Dockerfile.ebpf-dev -t go-net-monitoring-ebpf-dev . --quiet

# 快速编译测试
echo "🔧 Quick compilation test..."
docker run --rm -v $(pwd):/workspace -w /workspace go-net-monitoring-ebpf-dev bash -c "
    echo '🧪 Testing environment...'
    cd bpf && make test-env
    echo '🔨 Building eBPF...'
    make clean && make all
    echo '📊 Results:'
    ls -la ../bin/bpf/ || echo 'No files found'
"

echo "✅ Quick test completed!"
