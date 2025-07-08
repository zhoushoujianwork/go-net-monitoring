#!/bin/bash

# 测试环境一键启动和销毁脚本
# 用法: ./scripts/test-env.sh [start|stop|restart|status|logs]

set -e

# 配置变量
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="/tmp/netmon-logs"
PID_DIR="/tmp/netmon-pids"

# 服务配置
SERVER_CONFIG="configs/server-local.yaml"
AGENT_CONFIG="configs/agent.yaml"
SERVER_PORT=8080

# 创建必要目录
mkdir -p "$LOG_DIR" "$PID_DIR"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 检查端口是否被占用
check_port() {
    local port=$1
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        return 0  # 端口被占用
    else
        return 1  # 端口空闲
    fi
}

# 等待端口可用
wait_for_port() {
    local port=$1
    local timeout=${2:-30}
    local count=0
    
    log_info "等待端口 $port 可用..."
    while [ $count -lt $timeout ]; do
        if check_port $port; then
            log_info "端口 $port 已可用"
            return 0
        fi
        sleep 1
        count=$((count + 1))
    done
    
    log_error "等待端口 $port 超时"
    return 1
}

# 编译二进制文件
build_binaries() {
    log_info "🔨 编译二进制文件..."
    
    if ! go build -o bin/server ./cmd/server/; then
        log_error "Server编译失败"
        exit 1
    fi
    
    if ! go build -o bin/agent-ebpf ./cmd/agent-ebpf/; then
        log_error "eBPF Agent编译失败"
        exit 1
    fi
    
    log_info "✅ 编译完成"
}

# 创建Agent配置
create_agent_config() {
    cat > "$LOG_DIR/agent-simple.yaml" << EOF
monitor:
  interface: "en0"
  protocols:
    - "tcp"
    - "udp"
  report_interval: 10s
  buffer_size: 1000

reporter:
  server_url: "http://localhost:$SERVER_PORT/api/v1/metrics"
  timeout: 10s
  retry_count: 3
  batch_size: 100

log:
  level: "debug"
  format: "json"
  output: "stdout"
EOF
}

# 启动Server
start_server() {
    log_info "🖥️  启动Server..."
    
    # 检查端口
    if check_port $SERVER_PORT; then
        log_warn "端口 $SERVER_PORT 已被占用，尝试停止现有服务"
        stop_server
        sleep 2
    fi
    
    # 启动Server
    nohup ./bin/server --config "$SERVER_CONFIG" \
        > "$LOG_DIR/server.log" 2>&1 &
    
    local server_pid=$!
    echo $server_pid > "$PID_DIR/server.pid"
    
    # 等待Server启动
    if wait_for_port $SERVER_PORT 15; then
        log_info "✅ Server启动成功 (PID: $server_pid)"
    else
        log_error "❌ Server启动失败"
        cat "$LOG_DIR/server.log" | tail -10
        exit 1
    fi
}

# 启动eBPF Agent
start_agent() {
    log_info "🔍 启动eBPF Agent..."
    
    # 创建简化配置
    create_agent_config
    
    # 启动Agent
    nohup ./bin/agent-ebpf --debug --config "$LOG_DIR/agent-simple.yaml" \
        > "$LOG_DIR/agent.log" 2>&1 &
    
    local agent_pid=$!
    echo $agent_pid > "$PID_DIR/agent.pid"
    
    # 等待Agent启动
    sleep 3
    if kill -0 $agent_pid 2>/dev/null; then
        log_info "✅ eBPF Agent启动成功 (PID: $agent_pid)"
    else
        log_error "❌ eBPF Agent启动失败"
        cat "$LOG_DIR/agent.log" | tail -10
        exit 1
    fi
}

# 启动服务
start_services() {
    log_info "🚀 启动测试环境..."
    
    # 检查并编译
    build_binaries
    
    # 启动Server
    start_server
    
    # 启动eBPF Agent
    start_agent
    
    # 显示状态
    show_status
    
    log_info "✅ 测试环境启动完成!"
    log_info "📊 Server: http://localhost:$SERVER_PORT"
    log_info "📈 Metrics: http://localhost:$SERVER_PORT/metrics"
    log_info "💚 Health: http://localhost:$SERVER_PORT/health"
}

# 停止Server
stop_server() {
    local pid_file="$PID_DIR/server.pid"
    
    if [[ -f "$pid_file" ]]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            log_info "停止Server (PID: $pid)..."
            kill "$pid"
            sleep 2
            
            # 强制杀死
            if kill -0 "$pid" 2>/dev/null; then
                log_warn "强制停止Server..."
                kill -9 "$pid" 2>/dev/null || true
            fi
        fi
        rm -f "$pid_file"
    fi
    
    # 清理端口占用
    local pids=$(lsof -ti :$SERVER_PORT 2>/dev/null || true)
    if [[ -n "$pids" ]]; then
        log_warn "清理端口 $SERVER_PORT 占用..."
        echo "$pids" | xargs kill -9 2>/dev/null || true
    fi
}

# 停止Agent
stop_agent() {
    local pid_file="$PID_DIR/agent.pid"
    
    if [[ -f "$pid_file" ]]; then
        local pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            log_info "停止eBPF Agent (PID: $pid)..."
            kill "$pid"
            sleep 2
            
            # 强制杀死
            if kill -0 "$pid" 2>/dev/null; then
                log_warn "强制停止eBPF Agent..."
                kill -9 "$pid" 2>/dev/null || true
            fi
        fi
        rm -f "$pid_file"
    fi
    
    # 清理所有agent进程
    pkill -f "agent-ebpf" 2>/dev/null || true
}

# 停止服务
stop_services() {
    log_info "🛑 停止测试环境..."
    
    stop_agent
    stop_server
    
    log_info "✅ 测试环境已停止"
}

# 显示服务状态
show_status() {
    echo ""
    log_info "📊 服务状态:"
    
    # Server状态
    local server_pid_file="$PID_DIR/server.pid"
    if [[ -f "$server_pid_file" ]]; then
        local server_pid=$(cat "$server_pid_file")
        if kill -0 "$server_pid" 2>/dev/null; then
            echo -e "  🖥️  Server: ${GREEN}运行中${NC} (PID: $server_pid)"
            if check_port $SERVER_PORT; then
                echo -e "  📊 端口 $SERVER_PORT: ${GREEN}可用${NC}"
            else
                echo -e "  📊 端口 $SERVER_PORT: ${RED}不可用${NC}"
            fi
        else
            echo -e "  🖥️  Server: ${RED}已停止${NC}"
        fi
    else
        echo -e "  🖥️  Server: ${RED}未启动${NC}"
    fi
    
    # Agent状态
    local agent_pid_file="$PID_DIR/agent.pid"
    if [[ -f "$agent_pid_file" ]]; then
        local agent_pid=$(cat "$agent_pid_file")
        if kill -0 "$agent_pid" 2>/dev/null; then
            echo -e "  🔍 eBPF Agent: ${GREEN}运行中${NC} (PID: $agent_pid)"
        else
            echo -e "  🔍 eBPF Agent: ${RED}已停止${NC}"
        fi
    else
        echo -e "  🔍 eBPF Agent: ${RED}未启动${NC}"
    fi
    
    echo ""
}

# 显示日志
show_logs() {
    local service="$1"
    
    case "$service" in
        server)
            log_info "📋 Server日志:"
            tail -f "$LOG_DIR/server.log"
            ;;
        agent)
            log_info "📋 eBPF Agent日志:"
            tail -f "$LOG_DIR/agent.log"
            ;;
        *)
            log_info "📋 所有服务日志:"
            echo -e "\n${BLUE}=== Server日志 (最近10行) ===${NC}"
            tail -10 "$LOG_DIR/server.log" 2>/dev/null || echo "Server日志不存在"
            echo -e "\n${BLUE}=== Agent日志 (最近10行) ===${NC}"
            tail -10 "$LOG_DIR/agent.log" 2>/dev/null || echo "Agent日志不存在"
            ;;
    esac
}

# 清理文件
clean_files() {
    log_info "🧹 清理日志和PID文件..."
    
    rm -rf "$LOG_DIR" "$PID_DIR"
    mkdir -p "$LOG_DIR" "$PID_DIR"
    
    log_info "✅ 清理完成"
}

# 显示帮助信息
show_help() {
    cat << EOF
测试环境管理脚本

用法: $0 [命令]

命令:
  start     启动测试环境 (Server + eBPF Agent)
  stop      停止测试环境
  restart   重启测试环境
  status    查看服务状态
  logs      查看服务日志
  clean     清理日志和PID文件
  help      显示此帮助信息

示例:
  $0 start          # 启动测试环境
  $0 stop           # 停止测试环境
  $0 logs server    # 查看Server日志
  $0 logs agent     # 查看Agent日志

EOF
}

# 主函数
main() {
    cd "$PROJECT_ROOT"
    
    case "${1:-help}" in
        start)
            start_services
            ;;
        stop)
            stop_services
            ;;
        restart)
            stop_services
            sleep 2
            start_services
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs "$2"
            ;;
        clean)
            clean_files
            ;;
        help|*)
            show_help
            ;;
    esac
}

# 如果直接运行脚本，调用main函数
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
