#!/bin/bash

# eBPF Docker 管理脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# 颜色定义
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

# 显示帮助信息
show_help() {
    cat << EOF
eBPF Docker 管理脚本

用法: $0 [命令] [选项]

命令:
  build         构建eBPF Agent Docker镜像
  up            启动eBPF服务栈
  down          停止eBPF服务栈
  restart       重启eBPF服务栈
  logs          查看服务日志
  status        查看服务状态
  test          运行集成测试
  clean         清理Docker资源
  help          显示此帮助信息

选项:
  --debug       启用调试模式
  --monitoring  启用监控服务 (Prometheus + Grafana)
  --legacy      同时启动传统Agent (用于对比)
  --interface   指定网络接口 (默认: eth0)

示例:
  $0 build                    # 构建镜像
  $0 up --monitoring          # 启动服务和监控
  $0 up --debug --legacy      # 启动调试模式和传统Agent
  $0 logs ebpf-agent          # 查看eBPF Agent日志
  $0 test                     # 运行测试

EOF
}

# 构建Docker镜像
build_images() {
    log_info "构建eBPF Agent Docker镜像..."
    
    cd "$PROJECT_DIR"
    
    # 构建eBPF Agent镜像
    docker build -f docker/Dockerfile.ebpf-agent -t go-net-monitoring-ebpf:latest .
    
    log_info "镜像构建完成"
    docker images | grep go-net-monitoring
}

# 启动服务
start_services() {
    local profiles=""
    local env_file="$PROJECT_DIR/.env"
    
    # 处理选项
    while [[ $# -gt 0 ]]; do
        case $1 in
            --monitoring)
                profiles="$profiles --profile monitoring"
                shift
                ;;
            --legacy)
                profiles="$profiles --profile legacy"
                shift
                ;;
            --debug)
                echo "DEBUG_MODE=true" >> "$env_file"
                echo "LOG_LEVEL=debug" >> "$env_file"
                shift
                ;;
            --interface)
                echo "INTERFACE=$2" >> "$env_file"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    log_info "启动eBPF服务栈..."
    cd "$PROJECT_DIR"
    
    # 创建必要的目录
    mkdir -p logs monitoring/grafana/dashboards
    
    # 启动服务
    docker-compose -f docker-compose.ebpf.yml $profiles up -d
    
    log_info "服务启动完成"
    show_status
}

# 停止服务
stop_services() {
    log_info "停止eBPF服务栈..."
    cd "$PROJECT_DIR"
    
    docker-compose -f docker-compose.ebpf.yml --profile monitoring --profile legacy down
    
    log_info "服务已停止"
}

# 重启服务
restart_services() {
    log_info "重启eBPF服务栈..."
    stop_services
    sleep 2
    start_services "$@"
}

# 查看日志
show_logs() {
    local service="${1:-ebpf-agent}"
    
    log_info "查看 $service 服务日志..."
    cd "$PROJECT_DIR"
    
    docker-compose -f docker-compose.ebpf.yml logs -f "$service"
}

# 查看状态
show_status() {
    log_info "eBPF服务状态:"
    cd "$PROJECT_DIR"
    
    docker-compose -f docker-compose.ebpf.yml ps
    
    echo ""
    log_info "服务访问地址:"
    echo "  Server:     http://localhost:8080"
    echo "  Prometheus: http://localhost:9090 (如果启用)"
    echo "  Grafana:    http://localhost:3000 (如果启用, admin/admin123)"
}

# 运行测试
run_tests() {
    log_info "运行eBPF集成测试..."
    
    # 检查服务是否运行
    if ! docker-compose -f docker-compose.ebpf.yml ps | grep -q "Up"; then
        log_warn "服务未运行，先启动服务..."
        start_services
        sleep 10
    fi
    
    # 运行测试
    log_info "测试eBPF Agent健康状态..."
    if docker exec netmon-ebpf-agent pgrep -f agent-ebpf > /dev/null; then
        log_info "✅ eBPF Agent运行正常"
    else
        log_error "❌ eBPF Agent未运行"
        return 1
    fi
    
    # 测试Server连接
    log_info "测试Server连接..."
    if curl -s http://localhost:8080/health > /dev/null; then
        log_info "✅ Server连接正常"
    else
        log_error "❌ Server连接失败"
        return 1
    fi
    
    log_info "🎉 集成测试通过"
}

# 清理资源
clean_resources() {
    log_info "清理Docker资源..."
    cd "$PROJECT_DIR"
    
    # 停止服务
    docker-compose -f docker-compose.ebpf.yml --profile monitoring --profile legacy down -v
    
    # 清理镜像
    docker rmi go-net-monitoring-ebpf:latest 2>/dev/null || true
    
    # 清理临时文件
    rm -f .env
    
    log_info "清理完成"
}

# 主函数
main() {
    case "${1:-help}" in
        build)
            build_images
            ;;
        up|start)
            shift
            start_services "$@"
            ;;
        down|stop)
            stop_services
            ;;
        restart)
            shift
            restart_services "$@"
            ;;
        logs)
            show_logs "$2"
            ;;
        status)
            show_status
            ;;
        test)
            run_tests
            ;;
        clean)
            clean_resources
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

# 执行主函数
main "$@"
