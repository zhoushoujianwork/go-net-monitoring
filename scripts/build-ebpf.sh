#!/bin/bash

# eBPF 快速构建脚本 (国内优化版本)

set -e

echo "🚀 Starting eBPF build process..."

# 检查Docker是否运行
if ! docker info >/dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker first."
    exit 1
fi

# 构建Docker镜像
echo "🔨 Building Docker image with China mirrors..."
docker build -f docker/Dockerfile.ebpf-dev -t go-net-monitoring-ebpf-dev . \
    --build-arg HTTP_PROXY=${HTTP_PROXY:-} \
    --build-arg HTTPS_PROXY=${HTTPS_PROXY:-}

# 测试环境
echo "🧪 Testing eBPF compilation environment..."
docker-compose -f docker-compose.ebpf-dev.yml run --rm ebpf-test

# 编译eBPF程序
echo "🔧 Compiling eBPF programs..."
docker-compose -f docker-compose.ebpf-dev.yml run --rm ebpf-build

# 检查编译结果
if [ -f "bin/bpf/xdp_monitor_linux.o" ]; then
    echo "✅ eBPF program compiled successfully!"
    ls -la bin/bpf/
else
    echo "❌ eBPF compilation failed!"
    exit 1
fi

# 测试Go程序
echo "🧪 Testing Go program..."
if [ -f "bin/ebpf-agent" ]; then
    echo "✅ Go program ready!"
    ./bin/ebpf-agent --help
else
    echo "❌ Go program build failed!"
    exit 1
fi

echo "🎉 Build process completed successfully!"
echo ""
echo "Next steps:"
echo "  1. Run development environment: docker-compose -f docker-compose.ebpf-dev.yml run --rm ebpf-dev"
echo "  2. Test eBPF program: ./bin/ebpf-agent --debug --program bin/bpf/xdp_monitor_linux.o"
echo "  3. Check logs: docker-compose -f docker-compose.ebpf-dev.yml logs"
