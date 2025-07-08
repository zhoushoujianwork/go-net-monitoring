#!/bin/bash

# eBPF 路径配置测试脚本
# 用于验证 eBPF 程序路径解析功能

set -e

echo "=== eBPF 路径配置测试 ==="

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "项目根目录: $PROJECT_ROOT"

# 1. 检查 eBPF 程序文件是否存在
echo -e "\n1. 检查 eBPF 程序文件..."

EBPF_PATHS=(
    "bpf/programs/xdp_monitor.c"
    "bpf/programs/xdp_monitor_linux.c"
    "bin/bpf/xdp_monitor.o"
    "bin/bpf/xdp_monitor_linux.o"
)

for path in "${EBPF_PATHS[@]}"; do
    if [[ -f "$path" ]]; then
        echo "✓ 找到: $path"
    else
        echo "✗ 缺失: $path"
    fi
done

# 2. 检查配置文件中的 eBPF 配置
echo -e "\n2. 检查配置文件..."

if [[ -f "configs/agent.yaml" ]]; then
    echo "✓ 配置文件存在: configs/agent.yaml"
    
    if grep -q "ebpf:" "configs/agent.yaml"; then
        echo "✓ 找到 eBPF 配置段"
        echo "eBPF 配置内容:"
        grep -A 10 "ebpf:" "configs/agent.yaml" | head -10
    else
        echo "✗ 未找到 eBPF 配置段"
    fi
else
    echo "✗ 配置文件不存在: configs/agent.yaml"
fi

# 3. 创建测试用的 eBPF 程序文件（如果不存在）
echo -e "\n3. 创建测试文件..."

mkdir -p bin/bpf

# 创建一个简单的测试文件
if [[ ! -f "bin/bpf/xdp_monitor.o" ]]; then
    echo "创建测试文件: bin/bpf/xdp_monitor.o"
    echo "# Test eBPF program" > "bin/bpf/xdp_monitor.o"
fi

if [[ ! -f "bin/bpf/xdp_monitor_linux.o" ]]; then
    echo "创建测试文件: bin/bpf/xdp_monitor_linux.o"
    echo "# Test eBPF program for Linux" > "bin/bpf/xdp_monitor_linux.o"
fi

# 4. 测试不同的配置场景
echo -e "\n4. 测试配置场景..."

# 创建测试配置文件
TEST_CONFIG_DIR="configs/test"
mkdir -p "$TEST_CONFIG_DIR"

# 场景1: 使用绝对路径
cat > "$TEST_CONFIG_DIR/agent-absolute-path.yaml" << EOF
server:
  host: "localhost"
  port: 8080

monitor:
  interface: "eth0"
  protocols: ["tcp", "udp"]
  report_interval: "10s"
  buffer_size: 1000

ebpf:
  program_path: "$PROJECT_ROOT/bin/bpf/xdp_monitor.o"
  enable_fallback: true

reporter:
  server_url: "http://localhost:8080/api/v1/metrics"
  timeout: "10s"

log:
  level: "info"
  format: "json"
EOF

echo "✓ 创建测试配置: 绝对路径"

# 场景2: 使用相对路径
cat > "$TEST_CONFIG_DIR/agent-relative-path.yaml" << EOF
server:
  host: "localhost"
  port: 8080

monitor:
  interface: "eth0"
  protocols: ["tcp", "udp"]
  report_interval: "10s"
  buffer_size: 1000

ebpf:
  program_path: "bin/bpf/xdp_monitor.o"
  fallback_paths:
    - "bpf/xdp_monitor.o"
    - "bin/bpf/xdp_monitor_linux.o"
  enable_fallback: true

reporter:
  server_url: "http://localhost:8080/api/v1/metrics"
  timeout: "10s"

log:
  level: "info"
  format: "json"
EOF

echo "✓ 创建测试配置: 相对路径"

# 场景3: 使用不存在的路径（测试回退）
cat > "$TEST_CONFIG_DIR/agent-fallback.yaml" << EOF
server:
  host: "localhost"
  port: 8080

monitor:
  interface: "eth0"
  protocols: ["tcp", "udp"]
  report_interval: "10s"
  buffer_size: 1000

ebpf:
  program_path: "/nonexistent/path/xdp_monitor.o"
  fallback_paths:
    - "bin/bpf/xdp_monitor.o"
    - "bin/bpf/xdp_monitor_linux.o"
  enable_fallback: true

reporter:
  server_url: "http://localhost:8080/api/v1/metrics"
  timeout: "10s"

log:
  level: "info"
  format: "json"
EOF

echo "✓ 创建测试配置: 回退测试"

# 5. 验证配置文件语法
echo -e "\n5. 验证配置文件语法..."

for config_file in "$TEST_CONFIG_DIR"/*.yaml; do
    if command -v yq >/dev/null 2>&1; then
        if yq eval '.' "$config_file" >/dev/null 2>&1; then
            echo "✓ 配置文件语法正确: $(basename "$config_file")"
        else
            echo "✗ 配置文件语法错误: $(basename "$config_file")"
        fi
    else
        echo "⚠ yq 未安装，跳过语法检查"
        break
    fi
done

# 6. 显示使用说明
echo -e "\n6. 使用说明..."

cat << EOF

测试配置文件已创建在 $TEST_CONFIG_DIR/ 目录下：

1. agent-absolute-path.yaml  - 使用绝对路径
2. agent-relative-path.yaml  - 使用相对路径和备用路径
3. agent-fallback.yaml       - 测试回退机制

使用方法:
# 测试绝对路径配置
./bin/agent -config $TEST_CONFIG_DIR/agent-absolute-path.yaml

# 测试相对路径配置
./bin/agent -config $TEST_CONFIG_DIR/agent-relative-path.yaml

# 测试回退机制
./bin/agent -config $TEST_CONFIG_DIR/agent-fallback.yaml

注意事项:
- 确保 eBPF 程序文件存在于指定路径
- 在容器环境中使用绝对路径更可靠
- 开发环境可以使用相对路径
- 启用 enable_fallback 可以在 eBPF 加载失败时使用模拟模式

EOF

echo "=== 测试完成 ==="
