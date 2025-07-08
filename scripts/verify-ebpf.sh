#!/bin/bash

# eBPF 程序验证脚本

set -e

echo "🔍 Verifying eBPF programs..."

# 检查编译结果
if [ ! -d "bin/bpf" ]; then
    echo "❌ eBPF programs not found. Run build first."
    exit 1
fi

echo "📁 eBPF programs found:"
ls -la bin/bpf/

# 使用 file 命令检查文件类型
for prog in bin/bpf/*.o; do
    if [ -f "$prog" ]; then
        echo "🔍 Checking $prog:"
        file "$prog"
        
        # 检查是否是有效的 BPF 对象文件
        if file "$prog" | grep -q "ELF.*BPF"; then
            echo "✅ Valid BPF object file"
        else
            echo "⚠️  May not be a valid BPF file"
        fi
        echo ""
    fi
done

# 测试Go程序编译（不运行）
echo "🔧 Testing Go program compilation..."
docker run --rm -v $(pwd):/workspace -w /workspace go-net-monitoring-ebpf-dev bash -c "
    export CGO_ENABLED=0
    go build -o bin/ebpf-agent-static ./cmd/ebpf-agent/
    echo '✅ Go program compiled successfully (static)'
    ls -la bin/ebpf-agent-static
"

echo "🎉 Verification completed!"
