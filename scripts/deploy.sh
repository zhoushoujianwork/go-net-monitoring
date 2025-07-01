#!/bin/bash

# 网络监控系统部署脚本
# 使用方法: ./deploy.sh [agent|server|all]

set -e

# 配置变量
INSTALL_DIR="/opt/network-monitoring"
CONFIG_DIR="/etc/network-monitoring"
LOG_DIR="/var/log/network-monitoring"
USER="network-monitoring"
GROUP="network-monitoring"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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

# 检查是否为root用户
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "此脚本需要root权限运行"
        exit 1
    fi
}

# 创建用户和组
create_user() {
    if ! id "$USER" &>/dev/null; then
        log_info "创建用户: $USER"
        useradd -r -s /bin/false -d /nonexistent "$USER"
    else
        log_info "用户 $USER 已存在"
    fi
}

# 创建目录
create_directories() {
    log_info "创建目录结构"
    
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$CONFIG_DIR"
    mkdir -p "$LOG_DIR"
    
    # 设置权限
    chown -R "$USER:$GROUP" "$INSTALL_DIR"
    chown -R "$USER:$GROUP" "$LOG_DIR"
    chmod 755 "$CONFIG_DIR"
}

# 安装二进制文件
install_binaries() {
    local component=$1
    
    log_info "安装 $component 二进制文件"
    
    if [[ "$component" == "agent" || "$component" == "all" ]]; then
        if [[ -f "bin/agent" ]]; then
            cp bin/agent "$INSTALL_DIR/"
            chmod +x "$INSTALL_DIR/agent"
            ln -sf "$INSTALL_DIR/agent" /usr/local/bin/network-agent
            log_info "Agent安装完成"
        else
            log_error "Agent二进制文件不存在，请先编译"
            exit 1
        fi
    fi
    
    if [[ "$component" == "server" || "$component" == "all" ]]; then
        if [[ -f "bin/server" ]]; then
            cp bin/server "$INSTALL_DIR/"
            chmod +x "$INSTALL_DIR/server"
            ln -sf "$INSTALL_DIR/server" /usr/local/bin/network-server
            log_info "Server安装完成"
        else
            log_error "Server二进制文件不存在，请先编译"
            exit 1
        fi
    fi
}

# 安装配置文件
install_configs() {
    local component=$1
    
    log_info "安装配置文件"
    
    if [[ "$component" == "agent" || "$component" == "all" ]]; then
        if [[ -f "configs/agent.yaml" ]]; then
            cp configs/agent.yaml "$CONFIG_DIR/"
            log_info "Agent配置文件安装完成"
        fi
    fi
    
    if [[ "$component" == "server" || "$component" == "all" ]]; then
        if [[ -f "configs/server.yaml" ]]; then
            cp configs/server.yaml "$CONFIG_DIR/"
            log_info "Server配置文件安装完成"
        fi
    fi
}

# 创建systemd服务
create_systemd_service() {
    local component=$1
    
    log_info "创建 $component systemd服务"
    
    if [[ "$component" == "agent" || "$component" == "all" ]]; then
        cat > /etc/systemd/system/network-agent.service << EOF
[Unit]
Description=Network Monitoring Agent
After=network.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/local/bin/network-agent --config $CONFIG_DIR/agent.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF
        log_info "Agent systemd服务创建完成"
    fi
    
    if [[ "$component" == "server" || "$component" == "all" ]]; then
        cat > /etc/systemd/system/network-server.service << EOF
[Unit]
Description=Network Monitoring Server
After=network.target

[Service]
Type=simple
User=$USER
Group=$GROUP
ExecStart=/usr/local/bin/network-server --config $CONFIG_DIR/server.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF
        log_info "Server systemd服务创建完成"
    fi
    
    systemctl daemon-reload
}

# 启动服务
start_services() {
    local component=$1
    
    log_info "启动 $component 服务"
    
    if [[ "$component" == "agent" || "$component" == "all" ]]; then
        systemctl enable network-agent
        systemctl start network-agent
        log_info "Agent服务已启动"
    fi
    
    if [[ "$component" == "server" || "$component" == "all" ]]; then
        systemctl enable network-server
        systemctl start network-server
        log_info "Server服务已启动"
    fi
}

# 检查服务状态
check_services() {
    local component=$1
    
    log_info "检查 $component 服务状态"
    
    if [[ "$component" == "agent" || "$component" == "all" ]]; then
        if systemctl is-active --quiet network-agent; then
            log_info "Agent服务运行正常"
        else
            log_error "Agent服务启动失败"
            systemctl status network-agent
        fi
    fi
    
    if [[ "$component" == "server" || "$component" == "all" ]]; then
        if systemctl is-active --quiet network-server; then
            log_info "Server服务运行正常"
        else
            log_error "Server服务启动失败"
            systemctl status network-server
        fi
    fi
}

# 安装依赖
install_dependencies() {
    log_info "安装系统依赖"
    
    # 检测操作系统
    if [[ -f /etc/debian_version ]]; then
        # Debian/Ubuntu
        apt-get update
        apt-get install -y libpcap-dev
    elif [[ -f /etc/redhat-release ]]; then
        # RHEL/CentOS
        yum install -y libpcap-devel
    else
        log_warn "未知的操作系统，请手动安装libpcap开发包"
    fi
}

# 主函数
main() {
    local component=${1:-all}
    
    log_info "开始部署网络监控系统 - $component"
    
    # 检查参数
    if [[ "$component" != "agent" && "$component" != "server" && "$component" != "all" ]]; then
        log_error "无效的参数: $component"
        echo "使用方法: $0 [agent|server|all]"
        exit 1
    fi
    
    # 执行部署步骤
    check_root
    install_dependencies
    create_user
    create_directories
    install_binaries "$component"
    install_configs "$component"
    create_systemd_service "$component"
    start_services "$component"
    
    # 等待服务启动
    sleep 5
    check_services "$component"
    
    log_info "部署完成！"
    
    # 显示后续操作提示
    echo ""
    log_info "后续操作:"
    echo "1. 检查服务状态: systemctl status network-$component"
    echo "2. 查看日志: journalctl -u network-$component -f"
    echo "3. 修改配置: vim $CONFIG_DIR/$component.yaml"
    echo "4. 重启服务: systemctl restart network-$component"
    
    if [[ "$component" == "server" || "$component" == "all" ]]; then
        echo "5. 访问指标: http://localhost:8080/metrics"
        echo "6. 健康检查: http://localhost:8080/health"
    fi
}

# 执行主函数
main "$@"
