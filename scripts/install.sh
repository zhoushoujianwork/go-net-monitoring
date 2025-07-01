#!/bin/bash

set -e
set -u

###########################################
# go-net-monitoring installer for webinstall.dev
###########################################

# 默认配置
PKG_NAME="go-net-monitoring"
PKG_VERSION="${WEBI_VERSION:-latest}"
PKG_COMPONENT="${1:-agent}"  # agent 或 server

# GitHub 仓库信息
GITHUB_USER="your-username"  # 替换为你的GitHub用户名
GITHUB_REPO="go-net-monitoring"
GITHUB_URL="https://github.com/${GITHUB_USER}/${GITHUB_REPO}"

# 安装目录
WEBI_PKG_DIR="${WEBI_PKG_DIR:-$HOME/.local/opt/${PKG_NAME}}"
WEBI_PKG_BIN="${WEBI_PKG_BIN:-$HOME/.local/bin}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
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

# 检测系统信息
detect_system() {
    local os=""
    local arch=""
    
    # 检测操作系统
    case "$(uname -s)" in
        Linux*)     os="linux";;
        Darwin*)    os="darwin";;
        *)          log_error "不支持的操作系统: $(uname -s)"; exit 1;;
    esac
    
    # 检测架构
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64";;
        arm64|aarch64)  arch="arm64";;
        *)              log_error "不支持的架构: $(uname -m)"; exit 1;;
    esac
    
    echo "${os}-${arch}"
}

# 获取最新版本
get_latest_version() {
    if [ "$PKG_VERSION" = "latest" ]; then
        log_info "获取最新版本信息..."
        PKG_VERSION=$(curl -s "https://api.github.com/repos/${GITHUB_USER}/${GITHUB_REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [ -z "$PKG_VERSION" ]; then
            log_error "无法获取最新版本信息"
            exit 1
        fi
    fi
    log_info "版本: $PKG_VERSION"
}

# 下载并安装
install_package() {
    local system_info=$(detect_system)
    local download_url="https://github.com/${GITHUB_USER}/${GITHUB_REPO}/releases/download/${PKG_VERSION}/${PKG_NAME}-${PKG_COMPONENT}-${PKG_VERSION}-${system_info}.tar.gz"
    
    log_info "下载 ${PKG_NAME}-${PKG_COMPONENT} ${PKG_VERSION} for ${system_info}..."
    log_info "下载地址: $download_url"
    
    # 创建临时目录
    local tmp_dir=$(mktemp -d)
    local tar_file="${tmp_dir}/${PKG_NAME}-${PKG_COMPONENT}.tar.gz"
    
    # 下载文件
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$download_url" -o "$tar_file"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$download_url" -O "$tar_file"
    else
        log_error "需要 curl 或 wget 来下载文件"
        exit 1
    fi
    
    # 验证下载
    if [ ! -f "$tar_file" ]; then
        log_error "下载失败"
        exit 1
    fi
    
    # 创建安装目录
    mkdir -p "$WEBI_PKG_DIR" "$WEBI_PKG_BIN"
    
    # 解压文件
    log_info "解压到 $WEBI_PKG_DIR..."
    tar -xzf "$tar_file" -C "$tmp_dir"
    
    # 查找解压后的目录
    local extracted_dir=$(find "$tmp_dir" -maxdepth 1 -type d -name "${PKG_NAME}-${PKG_COMPONENT}-*" | head -1)
    if [ -z "$extracted_dir" ]; then
        log_error "解压失败，找不到预期的目录"
        exit 1
    fi
    
    # 复制文件
    cp -r "$extracted_dir"/* "$WEBI_PKG_DIR/"
    
    # 创建符号链接
    local binary_name="${PKG_COMPONENT}"
    if [ -f "$WEBI_PKG_DIR/$binary_name" ]; then
        ln -sf "$WEBI_PKG_DIR/$binary_name" "$WEBI_PKG_BIN/$binary_name"
        chmod +x "$WEBI_PKG_BIN/$binary_name"
        log_success "已安装 $binary_name 到 $WEBI_PKG_BIN/$binary_name"
    else
        log_error "找不到二进制文件: $WEBI_PKG_DIR/$binary_name"
        exit 1
    fi
    
    # 清理临时文件
    rm -rf "$tmp_dir"
}

# 检查依赖
check_dependencies() {
    if [ "$PKG_COMPONENT" = "agent" ]; then
        # Agent 需要 root 权限进行网络监控
        log_warn "Agent 组件需要 root 权限才能进行网络监控"
        log_info "请使用 'sudo ${PKG_COMPONENT}' 运行"
        
        # 检查 libpcap
        if ! ldconfig -p 2>/dev/null | grep -q libpcap; then
            log_warn "未检测到 libpcap，可能需要安装："
            log_info "  Ubuntu/Debian: sudo apt-get install libpcap-dev"
            log_info "  CentOS/RHEL:   sudo yum install libpcap-devel"
            log_info "  macOS:         brew install libpcap"
        fi
    fi
}

# 显示使用说明
show_usage() {
    log_success "安装完成！"
    echo ""
    echo "📋 ��用说明:"
    echo ""
    
    if [ "$PKG_COMPONENT" = "agent" ]; then
        echo "🔧 配置文件位置:"
        echo "  $WEBI_PKG_DIR/configs/agent.yaml"
        echo ""
        echo "🚀 启动 Agent:"
        echo "  sudo $PKG_COMPONENT --config $WEBI_PKG_DIR/configs/agent.yaml"
        echo ""
        echo "📊 查看帮助:"
        echo "  $PKG_COMPONENT --help"
    elif [ "$PKG_COMPONENT" = "server" ]; then
        echo "🔧 配置文件位置:"
        echo "  $WEBI_PKG_DIR/configs/server.yaml"
        echo ""
        echo "🚀 启动 Server:"
        echo "  $PKG_COMPONENT --config $WEBI_PKG_DIR/configs/server.yaml"
        echo ""
        echo "📊 查看指标:"
        echo "  curl http://localhost:8080/metrics"
        echo ""
        echo "📋 查看帮助:"
        echo "  $PKG_COMPONENT --help"
    fi
    
    echo ""
    echo "📖 文档: ${GITHUB_URL}"
    echo "🐛 问题反馈: ${GITHUB_URL}/issues"
    echo ""
    echo "⚠️  注意: 请根据需要修改配置文件中的网络接口等设置"
}

# 主函数
main() {
    echo ""
    log_info "🚀 开始安装 ${PKG_NAME}-${PKG_COMPONENT}..."
    echo ""
    
    # 验证组件名称
    if [ "$PKG_COMPONENT" != "agent" ] && [ "$PKG_COMPONENT" != "server" ]; then
        log_error "无效的组件名称: $PKG_COMPONENT"
        log_info "支持的组件: agent, server"
        exit 1
    fi
    
    get_latest_version
    install_package
    check_dependencies
    show_usage
}

# 运行主函数
main "$@"
