#!/bin/sh

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

# 显示启动信息
show_banner() {
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                  Go Network Monitoring                       ║"
    echo "║                    Docker Container                          ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
}

# 检查环境变量
check_environment() {
    log_info "检查环境变量..."
    
    # 设置默认值
    COMPONENT=${COMPONENT:-server}
    LOG_LEVEL=${LOG_LEVEL:-info}
    
    # 根据组件设置配置文件
    if [ "$COMPONENT" = "agent" ]; then
        CONFIG_FILE=${CONFIG_FILE:-/app/configs/agent.yaml}
        NETWORK_INTERFACE=${NETWORK_INTERFACE:-eth0}
        SERVER_URL=${SERVER_URL:-http://localhost:8080/api/v1/metrics}
    elif [ "$COMPONENT" = "server" ]; then
        CONFIG_FILE=${CONFIG_FILE:-/app/configs/server.yaml}
        SERVER_HOST=${SERVER_HOST:-0.0.0.0}
        SERVER_PORT=${SERVER_PORT:-8080}
    else
        log_error "无效的组件类型: $COMPONENT"
        log_info "支持的组件: agent, server"
        exit 1
    fi
    
    log_success "环境变量检查完成"
    log_info "组件类型: $COMPONENT"
    log_info "配置文件: $CONFIG_FILE"
    log_info "日志级别: $LOG_LEVEL"
}

# 生成配置文件
generate_config() {
    log_info "生成配置文件..."
    
    if [ "$COMPONENT" = "agent" ]; then
        cat > "$CONFIG_FILE" << EOF
server:
  host: "${SERVER_HOST:-localhost}"
  port: ${SERVER_PORT:-8080}

monitor:
  interface: "${NETWORK_INTERFACE:-eth0}"
  protocols:
    - "tcp"
    - "udp"
    - "http"
    - "https"
    - "dns"
  report_interval: "${REPORT_INTERVAL:-10s}"
  buffer_size: ${BUFFER_SIZE:-1000}
  filters:
    ignore_localhost: ${IGNORE_LOCALHOST:-true}
    ignore_ports:
      - 22    # SSH
      - 123   # NTP
    ignore_ips:
      - "127.0.0.1"
      - "::1"

reporter:
  server_url: "${SERVER_URL:-http://localhost:8080/api/v1/metrics}"
  timeout: "${REPORTER_TIMEOUT:-10s}"
  retry_count: ${RETRY_COUNT:-3}
  batch_size: ${BATCH_SIZE:-100}

log:
  level: "${LOG_LEVEL:-info}"
  format: "${LOG_FORMAT:-json}"
  output: "${LOG_OUTPUT:-stdout}"
EOF
    elif [ "$COMPONENT" = "server" ]; then
        cat > "$CONFIG_FILE" << EOF
server:
  host: "${SERVER_HOST:-0.0.0.0}"
  port: ${SERVER_PORT:-8080}

storage:
  type: "${STORAGE_TYPE:-memory}"
  retention: "${STORAGE_RETENTION:-24h}"

log:
  level: "${LOG_LEVEL:-info}"
  format: "${LOG_FORMAT:-json}"
  output: "${LOG_OUTPUT:-stdout}"
EOF
    fi
    
    log_success "配置文件生成完成: $CONFIG_FILE"
}

# 检查权限
check_permissions() {
    if [ "$COMPONENT" = "agent" ]; then
        log_info "检查网络监控权限..."
        
        # 检查是否以root权限运行
        if [ "$(id -u)" != "0" ]; then
            log_warn "Agent需要root权限进行网络监控"
            log_warn "请使用 --privileged 或 --cap-add=NET_ADMIN 运行容器"
        else
            log_success "权限检查通过"
        fi
        
        # 检查网络接口
        if ! ip link show "$NETWORK_INTERFACE" >/dev/null 2>&1; then
            log_warn "网络接口 $NETWORK_INTERFACE 不存在"
            log_info "可用的网络接口:"
            ip link show | grep -E '^[0-9]+:' | awk -F': ' '{print "  " $2}' | sed 's/@.*//'
            
            # 尝试使用第一个可用接口
            FIRST_INTERFACE=$(ip link show | grep -E '^[0-9]+:' | head -1 | awk -F': ' '{print $2}' | sed 's/@.*//')
            if [ -n "$FIRST_INTERFACE" ] && [ "$FIRST_INTERFACE" != "lo" ]; then
                log_warn "将使用接口: $FIRST_INTERFACE"
                NETWORK_INTERFACE="$FIRST_INTERFACE"
                # 重新生成配置文件
                generate_config
            fi
        fi
    fi
}

# 启动应用
start_application() {
    log_info "启动 $COMPONENT..."
    
    # 检查二进制文件是否存在
    BINARY_PATH="/usr/local/bin/$COMPONENT"
    if [ ! -f "$BINARY_PATH" ]; then
        log_error "二进制文件不存在: $BINARY_PATH"
        exit 1
    fi
    
    # 检查配置文件是否存在
    if [ ! -f "$CONFIG_FILE" ]; then
        log_error "配置文件不存在: $CONFIG_FILE"
        exit 1
    fi
    
    log_success "启动参数:"
    log_info "  二进制文件: $BINARY_PATH"
    log_info "  配置文件: $CONFIG_FILE"
    log_info "  用户: $(whoami)"
    
    # 显示配置文件内容（调试用）
    if [ "$LOG_LEVEL" = "debug" ]; then
        log_info "配置文件内容:"
        cat "$CONFIG_FILE" | sed 's/^/  /'
    fi
    
    # 启动应用
    exec "$BINARY_PATH" --config "$CONFIG_FILE"
}

# 信号处理
cleanup() {
    log_info "收到停止信号，正在清理..."
    if [ -n "$APP_PID" ]; then
        kill -TERM "$APP_PID" 2>/dev/null || true
        wait "$APP_PID" 2>/dev/null || true
    fi
    log_success "清理完成"
    exit 0
}

# 设置信号处理
trap cleanup TERM INT

# 主函数
main() {
    show_banner
    check_environment
    generate_config
    check_permissions
    start_application
}

# 如果有参数，直接执行
if [ $# -gt 0 ]; then
    exec "$@"
else
    main
fi
