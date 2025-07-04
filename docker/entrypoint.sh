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
    DEBUG_MODE=${DEBUG_MODE:-false}
    
    # 根据DEBUG_MODE自动设置LOG_LEVEL，避免重复配置
    if [ "$DEBUG_MODE" = "true" ]; then
        LOG_LEVEL="debug"
    else
        LOG_LEVEL=${LOG_LEVEL:-info}
    fi
    
    # 根据组件设置配置文件路径
    if [ "$COMPONENT" = "agent" ]; then
        CONFIG_FILE=${CONFIG_FILE:-/app/configs/agent.yaml}
        NETWORK_INTERFACE=${NETWORK_INTERFACE:-eth0}
        SERVER_URL=${SERVER_URL:-http://localhost:8080/api/v1/metrics}
        
        # 从SERVER_URL解析SERVER_HOST和SERVER_PORT
        if echo "$SERVER_URL" | grep -q "://"; then
            # 提取协议后的部分
            URL_WITHOUT_PROTOCOL=$(echo "$SERVER_URL" | sed 's|^[^:]*://||')
            # 提取主机和端口部分（去掉路径）
            HOST_PORT=$(echo "$URL_WITHOUT_PROTOCOL" | sed 's|/.*||')
            
            if echo "$HOST_PORT" | grep -q ":"; then
                # 有端口号
                SERVER_HOST=$(echo "$HOST_PORT" | cut -d: -f1)
                SERVER_PORT=$(echo "$HOST_PORT" | cut -d: -f2)
            else
                # 没有端口号，使用默认端口
                SERVER_HOST="$HOST_PORT"
                if echo "$SERVER_URL" | grep -q "^https://"; then
                    SERVER_PORT=443
                else
                    SERVER_PORT=80
                fi
            fi
        else
            # 如果不是完整URL，使用默认值
            SERVER_HOST=${SERVER_HOST:-localhost}
            SERVER_PORT=${SERVER_PORT:-8080}
        fi
        
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
    log_info "Debug模式: $DEBUG_MODE"
    log_info "日志级别: $LOG_LEVEL"
    
    if [ "$COMPONENT" = "agent" ]; then
        log_info "Server URL: $SERVER_URL"
        log_info "Server Host: $SERVER_HOST"
        log_info "Server Port: $SERVER_PORT"
        log_info "Network Interface: $NETWORK_INTERFACE"
    fi
}

# 生成配置文件
generate_config() {
    log_info "生成配置文件..."
    
    # 使用临时目录来生成配置文件，避免只读文件系统问题
    TEMP_CONFIG_DIR="/tmp/netmon-config"
    TEMP_CONFIG_FILE="$TEMP_CONFIG_DIR/$(basename "$CONFIG_FILE")"
    
    # 创建临时配置目录
    mkdir -p "$TEMP_CONFIG_DIR"
    
    if [ "$COMPONENT" = "agent" ]; then
        cat > "$TEMP_CONFIG_FILE" << EOF
server:
  host: "${SERVER_HOST}"
  port: ${SERVER_PORT}

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
  server_url: "${SERVER_URL}"
  timeout: "${REPORTER_TIMEOUT:-10s}"
  retry_count: ${RETRY_COUNT:-3}
  batch_size: ${BATCH_SIZE:-100}

log:
  level: "${LOG_LEVEL:-info}"
  format: "${LOG_FORMAT:-json}"
  output: "${LOG_OUTPUT:-stdout}"
EOF
    elif [ "$COMPONENT" = "server" ]; then
        cat > "$TEMP_CONFIG_FILE" << EOF
http:
  host: "${SERVER_HOST:-0.0.0.0}"
  port: ${SERVER_PORT:-8080}
  read_timeout: "${READ_TIMEOUT:-30s}"
  write_timeout: "${WRITE_TIMEOUT:-30s}"
  enable_tls: ${ENABLE_TLS:-false}
  tls_cert_path: "${TLS_CERT_PATH:-}"
  tls_key_path: "${TLS_KEY_PATH:-}"
  debug: ${DEBUG_MODE:-false}

metrics:
  path: "${METRICS_PATH:-/metrics}"
  enabled: ${METRICS_ENABLED:-true}
  interval: "${METRICS_INTERVAL:-15s}"

storage:
  type: "${STORAGE_TYPE:-memory}"
  ttl: "${STORAGE_TTL:-1h}"
  max_entries: ${STORAGE_MAX_ENTRIES:-10000}
  
  # Redis配置 (当type为redis时使用)
  redis:
    host: "${REDIS_HOST:-localhost}"
    port: ${REDIS_PORT:-6379}
    password: "${REDIS_PASSWORD:-}"
    db: ${REDIS_DB:-0}
    pool_size: ${REDIS_POOL_SIZE:-10}
    timeout: "${REDIS_TIMEOUT:-5s}"

log:
  level: "${LOG_LEVEL:-info}"
  format: "${LOG_FORMAT:-json}"
  output: "${LOG_OUTPUT:-stdout}"
EOF
    fi
    
    # 更新配置文件路径为临时文件
    CONFIG_FILE="$TEMP_CONFIG_FILE"
    
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

# 检查配置文件
check_config_file() {
    log_info "检查配置文件..."
    
    # 检查是否需要重新生成配置文件
    REGENERATE_CONFIG=false
    
    # 如果是Agent且设置了SERVER_URL环境变量，强制重新生成配置
    if [ "$COMPONENT" = "agent" ] && [ -n "$SERVER_URL" ] && [ "$SERVER_URL" != "http://localhost:8080/api/v1/metrics" ]; then
        log_info "检测到自定义SERVER_URL，将重新生成配置文件"
        REGENERATE_CONFIG=true
    fi
    
    # 如果设置了DEBUG_MODE环境变量，强制重新生成配置
    if [ -n "$DEBUG_MODE" ]; then
        log_info "检测到DEBUG_MODE环境变量，将重新生成配置文件"
        REGENERATE_CONFIG=true
    fi
    
    # 检查原始配置文件是否存在且可读
    ORIGINAL_CONFIG_EXISTS=false
    if [ -f "$CONFIG_FILE" ] && [ -r "$CONFIG_FILE" ]; then
        ORIGINAL_CONFIG_EXISTS=true
    fi
    
    # 如果原始配置文件存在且不需要重新生成，则使用现有文件
    if [ "$ORIGINAL_CONFIG_EXISTS" = "true" ] && [ "$REGENERATE_CONFIG" = "false" ]; then
        log_success "使用现有配置文件: $CONFIG_FILE"
        
        # 只在debug模式下显示配置文件内容
        if [ "$DEBUG_MODE" = "true" ]; then
            log_info "配置文件内容:"
            cat "$CONFIG_FILE" | sed 's/^/  /'
        fi
    else
        if [ "$REGENERATE_CONFIG" = "true" ]; then
            log_info "重新生成配置文件以应用环境变量"
        else
            log_info "配置文件不存在或不可读，将生成默认配置"
        fi
        generate_config
        
        # 在debug模式下显示生成的配置文件内容
        if [ "$DEBUG_MODE" = "true" ]; then
            log_info "生成的配置文件内容:"
            cat "$CONFIG_FILE" | sed 's/^/  /'
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
    
    log_info "二进制文件: $BINARY_PATH"
    log_info "配置文件: $CONFIG_FILE"
    log_info "Debug模式: $DEBUG_MODE"
    log_info "用户: $(whoami)"
    
    log_success "正在启动应用..."
    
    # 直接执行命令，避免颜色代码问题
    if [ "$DEBUG_MODE" = "true" ]; then
        log_info "启动命令: $BINARY_PATH --debug --config $CONFIG_FILE"
        exec "$BINARY_PATH" --debug --config "$CONFIG_FILE"
    else
        log_info "启动命令: $BINARY_PATH --config $CONFIG_FILE"
        exec "$BINARY_PATH" --config "$CONFIG_FILE"
    fi
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
    check_config_file
    check_permissions
    start_application
}

# 如果有参数，直接执行传入的命令
if [ $# -gt 0 ]; then
    log_info "执行自定义命令: $*"
    exec "$@"
else
    main
fi
