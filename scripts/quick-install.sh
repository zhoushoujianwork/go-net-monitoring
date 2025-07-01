#!/bin/bash

###########################################
# go-net-monitoring 一键安装脚本
# 支持直接通过 curl 安装
###########################################

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m'

# 显示横幅
show_banner() {
    echo -e "${BLUE}${BOLD}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                  Go Network Monitoring                       ║"
    echo "║                     一键安装脚本                              ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

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

# 显示菜单
show_menu() {
    echo ""
    echo -e "${BOLD}请选择要安装的组件:${NC}"
    echo ""
    echo "1) Agent  - 网络流量监控代理 (需要root权限)"
    echo "2) Server - 数据聚合服务器"
    echo "3) Both   - 同时安装Agent和Server"
    echo "4) Exit   - 退出"
    echo ""
    echo -n "请输入选择 [1-4]: "
}

# 安装组件
install_component() {
    local component=$1
    log_info "开始安装 $component..."
    
    # 下载并执行安装脚本
    if curl -fsSL https://raw.githubusercontent.com/your-username/go-net-monitoring/main/scripts/install.sh | bash -s "$component"; then
        log_success "$component 安装完成!"
        return 0
    else
        log_error "$component 安装失败!"
        return 1
    fi
}

# 检查系统要求
check_requirements() {
    log_info "检查系统要求..."
    
    # 检查操作系统
    case "$(uname -s)" in
        Linux*)     log_info "检测到 Linux 系统" ;;
        Darwin*)    log_info "检测到 macOS 系统" ;;
        *)          log_error "不支持的操作系统: $(uname -s)"; exit 1 ;;
    esac
    
    # 检查架构
    case "$(uname -m)" in
        x86_64|amd64)   log_info "检测到 x86_64 架构" ;;
        arm64|aarch64)  log_info "检测到 ARM64 架构" ;;
        *)              log_error "不支持的架构: $(uname -m)"; exit 1 ;;
    esac
    
    # 检查必要工具
    if ! command -v curl >/dev/null 2>&1; then
        log_error "需要 curl 工具，请先安装"
        exit 1
    fi
    
    log_success "系统要求检查通过"
}

# 显示安装后说明
show_post_install_info() {
    echo ""
    echo -e "${GREEN}${BOLD}🎉 安装完成!${NC}"
    echo ""
    echo -e "${BOLD}📋 快速开始:${NC}"
    echo ""
    echo "1. 启动 Server (如果已安装):"
    echo "   server --config ~/.local/opt/go-net-monitoring/configs/server.yaml"
    echo ""
    echo "2. 启动 Agent (如果已安装，需要root权限):"
    echo "   sudo agent --config ~/.local/opt/go-net-monitoring/configs/agent.yaml"
    echo ""
    echo "3. 查看监控指标:"
    echo "   curl http://localhost:8080/metrics"
    echo ""
    echo -e "${BOLD}📖 更多信息:${NC}"
    echo "   文档: https://github.com/your-username/go-net-monitoring"
    echo "   问题反馈: https://github.com/your-username/go-net-monitoring/issues"
    echo ""
    echo -e "${YELLOW}⚠️  注意事项:${NC}"
    echo "   - Agent 需要 root 权限进行网络监控"
    echo "   - 请根据需要修改配置文件中的网络接口设置"
    echo "   - 确保防火墙允许相关端口通信"
    echo ""
}

# 主函数
main() {
    show_banner
    check_requirements
    
    # 如果有命令行参数，直接安装
    if [ $# -gt 0 ]; then
        case "$1" in
            "agent"|"server")
                install_component "$1"
                show_post_install_info
                exit 0
                ;;
            "both")
                install_component "server"
                install_component "agent"
                show_post_install_info
                exit 0
                ;;
            *)
                log_error "无效的参数: $1"
                log_info "支持的参数: agent, server, both"
                exit 1
                ;;
        esac
    fi
    
    # 交互式安装
    while true; do
        show_menu
        read -r choice
        
        case $choice in
            1)
                install_component "agent"
                break
                ;;
            2)
                install_component "server"
                break
                ;;
            3)
                install_component "server"
                install_component "agent"
                break
                ;;
            4)
                log_info "退出安装"
                exit 0
                ;;
            *)
                log_error "无效选择，请输入 1-4"
                ;;
        esac
    done
    
    show_post_install_info
}

# 运行主函数
main "$@"
