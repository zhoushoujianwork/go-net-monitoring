#!/bin/bash

# 网络监控系统一键安装脚本
# 作者: Claude
# 版本: 1.0.0

set -e

# 配置变量
INSTALL_DIR="/opt/go-net-monitoring"
BIN_DIR="$INSTALL_DIR/bin"
BPF_DIR="$INSTALL_DIR/bpf"
CONFIG_DIR="$INSTALL_DIR/configs"
SYSTEMD_DIR="/etc/systemd/system"
PROFILE_DIR="/etc/profile.d"

# release 目录（后续会替换为 GitHub release URL）
RELEASE_DIR="release/linux-amd64"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# 检查系统要求
check_system() {
    log_info "检查系统要求..."
    
    # 检查是否是 Linux
    if [ "$(uname -s)" != "Linux" ]; then
        log_error "只支持 Linux 系统"
        exit 1
    fi
    
    # 检查是否有 root 权限
    if [ "$EUID" -ne 0 ]; then
        log_error "请使用 root 权限运行此脚本"
        exit 1
    fi
    
    # 检查系统架构
    ARCH=$(uname -m)
    if [ "$ARCH" != "x86_64" ]; then
        log_error "当前仅支持 x86_64 架构"
        exit 1
    fi
    
    log_success "系统检查通过"
}

# 检查 release 文件
check_release_files() {
    log_info "检查 release 文件..."
    
    # 检查必要文件
    REQUIRED_FILES=(
        "$RELEASE_DIR/agent-ebpf"
        "$RELEASE_DIR/bpf/xdp_monitor.o"
        "$RELEASE_DIR/bpf/xdp_monitor_linux.o"
        "$RELEASE_DIR/agent.yaml.example"
    )
    
    for file in "${REQUIRED_FILES[@]}"; do
        if [ ! -f "$file" ]; then
            log_error "缺少必要文件: $file"
            exit 1
        fi
    done
    
    log_success "release 文件检查通过"
}

# 创建安装目录
create_dirs() {
    log_info "创建安装目录..."
    
    mkdir -p "$BIN_DIR" "$BPF_DIR" "$CONFIG_DIR"
    
    log_success "目录创建完成"
}

# 安装文件
install_files() {
    log_info "安装文件..."
    
    # 复制二进制文件
    cp "$RELEASE_DIR/agent-ebpf" "$BIN_DIR/"
    chmod +x "$BIN_DIR/agent-ebpf"
    
    # 复制 eBPF 对象文件
    cp "$RELEASE_DIR/bpf"/*.o "$BPF_DIR/"
    chmod 644 "$BPF_DIR"/*.o
    
    # 复制配置文件
    cp "$RELEASE_DIR/agent.yaml.example" "$CONFIG_DIR/agent.yaml.example"
    if [ ! -f "$CONFIG_DIR/agent.yaml" ]; then
        cp "$RELEASE_DIR/agent.yaml.example" "$CONFIG_DIR/agent.yaml"
    fi
    
    log_success "文件安装完成"
}

# 配置系统环境
configure_system() {
    log_info "配置系统环境..."
    
    # 添加到系统 PATH
    cat > "$PROFILE_DIR/go-net-monitoring.sh" << EOF
export PATH=\$PATH:$BIN_DIR
export GO_NET_MONITORING_HOME=$INSTALL_DIR
EOF
    
    # 创建 systemd 服务
    cat > "$SYSTEMD_DIR/go-net-monitoring.service" << EOF
[Unit]
Description=Go Network Monitoring Agent
After=network.target

[Service]
Type=simple
Environment=GO_NET_MONITORING_HOME=$INSTALL_DIR
ExecStart=$BIN_DIR/agent-ebpf --config $CONFIG_DIR/agent.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
    
    # 重新加载 systemd
    systemctl daemon-reload
    
    log_success "系统环境配置完成"
}

# 启动服务
start_service() {
    log_info "启动服务..."
    
    systemctl enable go-net-monitoring
    systemctl start go-net-monitoring
    
    log_success "服务启动完成"
}

# 卸载程序
uninstall() {
    log_info "开始卸载程序..."
    
    # 停止并禁用服务
    systemctl stop go-net-monitoring || true
    systemctl disable go-net-monitoring || true
    
    # 删除服务文件
    rm -f "$SYSTEMD_DIR/go-net-monitoring.service"
    
    # 删除环境变量配置
    rm -f "$PROFILE_DIR/go-net-monitoring.sh"
    
    # 删除安装目录
    rm -rf "$INSTALL_DIR"
    
    # 重新加载 systemd
    systemctl daemon-reload
    
    log_success "卸载完成"
}

# 显示帮助信息
show_help() {
    cat << EOF
Go Network Monitoring 安装脚本

用法: $0 [命令]

命令:
  install    安装程序
  uninstall  卸载程序
  help       显示此帮助信息

示例:
  $0 install     # 安装程序
  $0 uninstall   # 卸载程序
EOF
}

# 安装主函数
do_install() {
    log_info "开始安装 Go Network Monitoring..."
    
    check_system
    check_release_files
    create_dirs
    install_files
    configure_system
    start_service
    
    log_success "安装完成！"
    echo ""
    echo "📋 安装信息:"
    echo "  安装目录: $INSTALL_DIR"
    echo "  配置文件: $CONFIG_DIR/agent.yaml"
    echo "  服务状态: $(systemctl is-active go-net-monitoring)"
    echo ""
    echo "🚀 使用说明:"
    echo "  1. 编辑配置: vi $CONFIG_DIR/agent.yaml"
    echo "  2. 重启服务: systemctl restart go-net-monitoring"
    echo "  3. 查看日志: journalctl -u go-net-monitoring"
    echo "  4. 卸载程序: $0 uninstall"
}

# 主函数
main() {
    case "${1:-help}" in
        install)
            do_install
            ;;
        uninstall)
            uninstall
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"
