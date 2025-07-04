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

# 检查依赖
check_dependencies() {
    log_info "检查构建依赖..."
    
    # 检查Go
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go未安装，请先安装Go 1.19+"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go版本: $GO_VERSION"
    
    # 检查libpcap (用于Agent构建)
    if pkg-config --exists libpcap 2>/dev/null; then
        log_success "libpcap可用，可构建Agent"
        HAS_LIBPCAP=true
    else
        log_warn "libpcap不可用，将跳过Agent构建"
        log_warn "安装方法:"
        log_warn "  Ubuntu/Debian: sudo apt-get install libpcap-dev"
        log_warn "  CentOS/RHEL:   sudo yum install libpcap-devel"
        log_warn "  macOS:         brew install libpcap"
        HAS_LIBPCAP=false
    fi
}

# 构建单个平台
build_platform() {
    local goos=$1
    local goarch=$2
    local platform_name="$goos-$goarch"
    
    log_info "构建 $platform_name..."
    
    mkdir -p bin dist
    
    # 构建Server (不需要CGO)
    log_info "构建Server ($platform_name)..."
    CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build \
        -ldflags "-s -w" \
        -o "bin/server-$platform_name" \
        ./cmd/server
    
    # 构建Agent (需要CGO和libpcap)
    if [ "$HAS_LIBPCAP" = true ] && [ "$goos" = "$(go env GOOS)" ]; then
        log_info "构建Agent ($platform_name)..."
        CGO_ENABLED=1 GOOS=$goos GOARCH=$goarch go build \
            -ldflags "-s -w" \
            -o "bin/agent-$platform_name" \
            ./cmd/agent
        AGENT_BUILT=true
    else
        log_warn "跳过Agent构建 ($platform_name) - 需要本地libpcap或相同操作系统"
        AGENT_BUILT=false
    fi
    
    # 创建压缩包
    cd bin
    if [ "$AGENT_BUILT" = true ]; then
        if [ "$goos" = "windows" ]; then
            zip -q "../dist/go-net-monitoring-$platform_name.zip" \
                "agent-$platform_name.exe" "server-$platform_name.exe"
        else
            tar -czf "../dist/go-net-monitoring-$platform_name.tar.gz" \
                "agent-$platform_name" "server-$platform_name"
        fi
        log_success "$platform_name 构建完成 (包含Agent和Server)"
    else
        if [ "$goos" = "windows" ]; then
            zip -q "../dist/go-net-monitoring-$platform_name.zip" \
                "server-$platform_name.exe"
        else
            tar -czf "../dist/go-net-monitoring-$platform_name.tar.gz" \
                "server-$platform_name"
        fi
        log_success "$platform_name 构建完成 (仅Server)"
    fi
    cd ..
}

# 构建Windows版本
build_windows() {
    log_info "构建Windows版本..."
    
    # Windows AMD64
    log_info "构建Windows AMD64..."
    CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build \
        -ldflags "-s -w" \
        -o "bin/server-windows-amd64.exe" \
        ./cmd/server
    
    # 注意: Windows Agent需要WinPcap/Npcap，跨平台构建复杂
    log_warn "Windows Agent需要WinPcap/Npcap，建议在Windows系统上构建"
    
    cd bin
    zip -q "../dist/go-net-monitoring-windows-amd64.zip" "server-windows-amd64.exe"
    cd ..
    
    log_success "Windows构建完成 (仅Server)"
}

# 显示构建结果
show_results() {
    log_info "构建结果:"
    
    echo "二进制文件:"
    ls -lh bin/ | grep -E "(agent|server)" | awk '{print "  " $9 " (" $5 ")"}'
    
    echo ""
    echo "发布包:"
    ls -lh dist/ | grep -E "\.(tar\.gz|zip)$" | awk '{print "  " $9 " (" $5 ")"}'
    
    echo ""
    log_info "使用方法:"
    echo "  1. 解压对应平台的发布包"
    echo "  2. 运行Server: ./server --config server.yaml"
    echo "  3. 运行Agent: sudo ./agent --config agent.yaml (需要root权限)"
}

# 主函数
main() {
    log_info "开始跨平台构建..."
    
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
                echo "  --all       构建所有平台"
                echo "  --linux     构建Linux版本"
                echo "  --darwin    构建macOS版本"
                echo "  --macos     构建macOS版本 (同--darwin)"
                echo "  --windows   构建Windows版本"
                echo "  --current   构建当前平台"
                echo "  --help      显示帮助信息"
                echo ""
                echo "示例:"
                echo "  $0 --all              # 构建所有平台"
                echo "  $0 --linux --darwin   # 构建Linux和macOS"
                echo "  $0 --current          # 构建当前平台"
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                exit 1
                ;;
        esac
    done
    
    # 默认构建当前平台
    if [ "$BUILD_ALL" = false ] && [ -z "$PLATFORMS" ]; then
        CURRENT_OS=$(go env GOOS)
        CURRENT_ARCH=$(go env GOARCH)
        PLATFORMS="$CURRENT_OS-$CURRENT_ARCH"
        log_info "未指定平台，构建当前平台: $CURRENT_OS/$CURRENT_ARCH"
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
        
        if [ "$goos" = "windows" ]; then
            build_windows
        else
            build_platform "$goos" "$goarch"
        fi
    done
    
    show_results
    log_success "跨平台构建完成！"
}

# 运行主函数
main "$@"
