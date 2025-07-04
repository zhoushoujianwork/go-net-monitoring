#!/bin/bash

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查构建依赖
check_dependencies() {
    log_info "检查构建依赖..."
    
    # 检查Go
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go未安装，请先安装Go 1.19+"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go版本: $GO_VERSION"
    
    # 检查必要的构建工具
    if ! command -v tar >/dev/null 2>&1; then
        log_error "tar命令未找到"
        exit 1
    fi
    
    if ! command -v zip >/dev/null 2>&1; then
        log_warn "zip命令未找到，Windows构建将跳过"
    fi
    
    log_success "构建环境检查完成"
}

# 构建单个平台
build_platform() {
    local goos=$1
    local goarch=$2
    local platform_name="$goos-$goarch"
    
    log_info "构建 $platform_name 平台..."
    
    mkdir -p bin dist
    
    # 设置构建参数
    local ldflags="-s -w -X main.version=${VERSION:-dev} -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.gitCommit=${GIT_COMMIT:-unknown}"
    
    # 构建Server (无CGO依赖)
    log_info "构建Server ($platform_name)..."
    if [ "$goos" = "windows" ]; then
        CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build \
            -ldflags "$ldflags" \
            -o "bin/server-$platform_name.exe" \
            ./cmd/server
    else
        CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build \
            -ldflags "$ldflags" \
            -o "bin/server-$platform_name" \
            ./cmd/server
    fi
    
    # 构建Agent (需要CGO，但强制构建)
    log_info "构建Agent ($platform_name)..."
    local agent_built=false
    
    if [ "$goos" = "windows" ]; then
        # Windows构建
        if CGO_ENABLED=1 GOOS=$goos GOARCH=$goarch go build \
            -ldflags "$ldflags" \
            -o "bin/agent-$platform_name.exe" \
            ./cmd/agent 2>/dev/null; then
            agent_built=true
            log_success "Agent构建成功 ($platform_name)"
        else
            log_warn "Agent构建失败 ($platform_name) - 需要Windows交叉编译环境"
        fi
    else
        # Linux/macOS构建
        if CGO_ENABLED=1 GOOS=$goos GOARCH=$goarch go build \
            -ldflags "$ldflags" \
            -o "bin/agent-$platform_name" \
            ./cmd/agent 2>/dev/null; then
            agent_built=true
            log_success "Agent构建成功 ($platform_name)"
        else
            log_warn "Agent构建失败 ($platform_name) - 需要交叉编译环境"
        fi
    fi
    
    # 创建发布包
    create_release_package "$goos" "$goarch" "$platform_name" "$agent_built"
}

# 创建发布包
create_release_package() {
    local goos=$1
    local goarch=$2
    local platform_name=$3
    local agent_built=$4
    
    log_info "创建发布包 ($platform_name)..."
    
    # 创建临时目录
    local temp_dir="temp-$platform_name"
    mkdir -p "$temp_dir"
    
    # 复制二进制文件
    if [ "$goos" = "windows" ]; then
        cp "bin/server-$platform_name.exe" "$temp_dir/"
        if [ "$agent_built" = true ]; then
            cp "bin/agent-$platform_name.exe" "$temp_dir/"
        fi
    else
        cp "bin/server-$platform_name" "$temp_dir/"
        if [ "$agent_built" = true ]; then
            cp "bin/agent-$platform_name" "$temp_dir/"
        fi
    fi
    
    # 复制配置文件
    cp -r configs "$temp_dir/"
    
    # 创建平台特定的README
    create_platform_readme "$goos" "$goarch" "$temp_dir" "$agent_built"
    
    # 创建启动脚本
    create_startup_scripts "$goos" "$temp_dir" "$agent_built"
    
    # 打包
    cd "$temp_dir"
    if [ "$goos" = "windows" ]; then
        if command -v zip >/dev/null 2>&1; then
            zip -r "../dist/go-net-monitoring-$platform_name.zip" .
        else
            tar -czf "../dist/go-net-monitoring-$platform_name.tar.gz" .
        fi
    else
        tar -czf "../dist/go-net-monitoring-$platform_name.tar.gz" .
    fi
    cd ..
    
    # 清理临时目录
    rm -rf "$temp_dir"
    
    if [ "$agent_built" = true ]; then
        log_success "$platform_name 发布包创建完成 (包含Agent和Server)"
    else
        log_warn "$platform_name 发布包创建完成 (仅Server，Agent需要在目标平台构建)"
    fi
}

# 创建平台特定的README
create_platform_readme() {
    local goos=$1
    local goarch=$2
    local temp_dir=$3
    local agent_built=$4
    
    cat > "$temp_dir/README.md" << EOF
# Go Network Monitoring - $goos/$goarch

## 快速开始

### 1. 安装运行时依赖

#### $goos 系统依赖:
EOF

    if [ "$goos" = "linux" ]; then
        cat >> "$temp_dir/README.md" << EOF
\`\`\`bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install libpcap0.8

# CentOS/RHEL
sudo yum install libpcap

# 或者开发版本 (如果需要重新编译)
sudo apt-get install libpcap-dev  # Ubuntu/Debian
sudo yum install libpcap-devel    # CentOS/RHEL
\`\`\`
EOF
    elif [ "$goos" = "darwin" ]; then
        cat >> "$temp_dir/README.md" << EOF
\`\`\`bash
# 使用Homebrew安装
brew install libpcap

# 检查安装
brew list libpcap
\`\`\`
EOF
    elif [ "$goos" = "windows" ]; then
        cat >> "$temp_dir/README.md" << EOF
1. 下载并安装 Npcap: https://npcap.com/
2. 或者安装 WinPcap: https://www.winpcap.org/
3. 重启系统以确保驱动加载

注意: 推荐使用Npcap，它是WinPcap的现代替代品。
EOF
    fi

    cat >> "$temp_dir/README.md" << EOF

### 2. 运行服务

#### 启动Server:
EOF

    if [ "$goos" = "windows" ]; then
        cat >> "$temp_dir/README.md" << EOF
\`\`\`cmd
# 使用默认配置
server-$goos-$goarch.exe --config configs/server.yaml

# 启用debug模式
server-$goos-$goarch.exe --config configs/server.yaml --debug
\`\`\`
EOF
    else
        cat >> "$temp_dir/README.md" << EOF
\`\`\`bash
# 使用默认配置
./server-$goos-$goarch --config configs/server.yaml

# 启用debug模式
./server-$goos-$goarch --config configs/server.yaml --debug
\`\`\`
EOF
    fi

    if [ "$agent_built" = true ]; then
        cat >> "$temp_dir/README.md" << EOF

#### 启动Agent (需要管理员权限):
EOF
        if [ "$goos" = "windows" ]; then
            cat >> "$temp_dir/README.md" << EOF
\`\`\`cmd
# 以管理员身份运行命令提示符，然后执行:
agent-$goos-$goarch.exe --config configs/agent.yaml

# 启用debug模式
agent-$goos-$goarch.exe --config configs/agent.yaml --debug
\`\`\`
EOF
        else
            cat >> "$temp_dir/README.md" << EOF
\`\`\`bash
# 需要root权限进行网络监控
sudo ./agent-$goos-$goarch --config configs/agent.yaml

# 启用debug模式
sudo ./agent-$goos-$goarch --config configs/agent.yaml --debug
\`\`\`
EOF
        fi
    else
        cat >> "$temp_dir/README.md" << EOF

#### Agent构建:
Agent需要在目标平台上构建，因为需要CGO和平台特定的libpcap库。

\`\`\`bash
# 在目标$goos系统上:
git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
cd go-net-monitoring
make build-agent
\`\`\`
EOF
    fi

    cat >> "$temp_dir/README.md" << EOF

### 3. 验证运行

\`\`\`bash
# 检查Server健康状态
curl http://localhost:8080/health

# 查看监控指标
curl http://localhost:8080/metrics
\`\`\`

### 4. 配置说明

- \`configs/server.yaml\` - Server配置文件
- \`configs/agent.yaml\` - Agent配置文件
EOF

    if [ "$goos" = "darwin" ]; then
        cat >> "$temp_dir/README.md" << EOF
- \`configs/agent-macos.yaml\` - macOS优化配置
EOF
    fi

    cat >> "$temp_dir/README.md" << EOF

### 5. 故障排查

#### 常见问题:
1. **权限错误**: Agent需要管理员/root权限
2. **网络接口**: 检查配置文件中的interface设置
3. **依赖缺失**: 确保已安装libpcap运行时库

#### 获取帮助:
- 项目文档: https://github.com/zhoushoujianwork/go-net-monitoring
- 问题报告: https://github.com/zhoushoujianwork/go-net-monitoring/issues

---
构建信息: $goos/$goarch - $(date)
EOF
}

# 创建启动脚本
create_startup_scripts() {
    local goos=$1
    local temp_dir=$2
    local agent_built=$3
    
    if [ "$goos" = "windows" ]; then
        # Windows批处理脚本
        cat > "$temp_dir/start-server.bat" << 'EOF'
@echo off
echo Starting Go Network Monitoring Server...
server-windows-amd64.exe --config configs/server.yaml
pause
EOF
        
        if [ "$agent_built" = true ]; then
            cat > "$temp_dir/start-agent.bat" << 'EOF'
@echo off
echo Starting Go Network Monitoring Agent...
echo Note: This requires Administrator privileges
agent-windows-amd64.exe --config configs/agent.yaml
pause
EOF
        fi
    else
        # Linux/macOS shell脚本
        cat > "$temp_dir/start-server.sh" << EOF
#!/bin/bash
echo "Starting Go Network Monitoring Server..."
./server-$goos-$goarch --config configs/server.yaml
EOF
        chmod +x "$temp_dir/start-server.sh"
        
        if [ "$agent_built" = true ]; then
            cat > "$temp_dir/start-agent.sh" << EOF
#!/bin/bash
echo "Starting Go Network Monitoring Agent..."
echo "Note: This requires root privileges"
sudo ./agent-$goos-$goarch --config configs/agent.yaml
EOF
            chmod +x "$temp_dir/start-agent.sh"
        fi
    fi
}

# 显示构建结果
show_results() {
    log_info "构建结果:"
    
    echo "二进制文件:"
    ls -lh bin/ 2>/dev/null | grep -E "(agent|server)" | awk '{print "  " $9 " (" $5 ")"}'
    
    echo ""
    echo "发布包:"
    ls -lh dist/ 2>/dev/null | grep -E "\.(tar\.gz|zip)$" | awk '{print "  " $9 " (" $5 ")"}'
    
    echo ""
    log_info "分发说明:"
    echo "  1. 将对应平台的发布包分发给用户"
    echo "  2. 用户解压后按照README.md说明安装依赖"
    echo "  3. 用户运行启动脚本或直接执行二进制文件"
    echo "  4. Agent需要管理员权限，Server无需特殊权限"
}

# 主函数
main() {
    log_info "开始构建通用二进制发布包..."
    
    # 获取版本信息
    VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
    GIT_COMMIT=${GIT_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
    
    log_info "版本信息: $VERSION ($GIT_COMMIT)"
    
    # 解析参数
    PLATFORMS=""
    BUILD_ALL=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --all)
                BUILD_ALL=true
                shift
                ;;
            --linux)
                PLATFORMS="$PLATFORMS linux-amd64 linux-arm64"
                shift
                ;;
            --darwin|--macos)
                PLATFORMS="$PLATFORMS darwin-amd64 darwin-arm64"
                shift
                ;;
            --windows)
                PLATFORMS="$PLATFORMS windows-amd64"
                shift
                ;;
            --current)
                CURRENT_OS=$(go env GOOS)
                CURRENT_ARCH=$(go env GOARCH)
                PLATFORMS="$CURRENT_OS-$CURRENT_ARCH"
                shift
                ;;
            --help)
                echo "用法: $0 [选项]"
                echo "选项:"
                echo "  --all       构建所有平台发布包"
                echo "  --linux     构建Linux发布包"
                echo "  --darwin    构建macOS发布包"
                echo "  --macos     构建macOS发布包 (同--darwin)"
                echo "  --windows   构建Windows发布包"
                echo "  --current   构建当前平台发布包"
                echo "  --help      显示帮助信息"
                echo ""
                echo "示例:"
                echo "  $0 --all              # 构建所有平台发布包"
                echo "  $0 --linux --darwin   # 构建Linux和macOS发布包"
                echo "  $0 --current          # 构建当前平台发布包"
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                exit 1
                ;;
        esac
    done
    
    # 默认构建所有平台
    if [ "$BUILD_ALL" = false ] && [ -z "$PLATFORMS" ]; then
        BUILD_ALL=true
        log_info "未指定平台，构建所有平台发布包"
    fi
    
    # 构建所有平台
    if [ "$BUILD_ALL" = true ]; then
        PLATFORMS="linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64"
    fi
    
    check_dependencies
    
    # 清理旧的构建产物
    rm -rf bin dist
    mkdir -p bin dist
    
    # 构建各个平台
    for platform in $PLATFORMS; do
        IFS='-' read -r goos goarch <<< "$platform"
        build_platform "$goos" "$goarch"
    done
    
    show_results
    log_success "通用二进制发布包构建完成！"
    log_info "发布包位于 dist/ 目录，可直接分发给用户使用"
}

# 运行主函数
main "$@"
