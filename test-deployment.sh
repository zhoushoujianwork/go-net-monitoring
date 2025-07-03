#!/bin/bash

# 简化的部署测试脚本
set -e

echo "=== 网络监控系统部署测试 ==="

# 配置
SERVER_URL="http://localhost:8080"
COMPOSE_FILE="docker-compose.yml"

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

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查服务状态
check_service() {
    local service_name="$1"
    local url="$2"
    local max_attempts=30
    local attempt=1
    
    log_info "检查 $service_name 服务状态..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$url" > /dev/null 2>&1; then
            log_success "$service_name 服务正常"
            return 0
        fi
        
        log_info "尝试 $attempt/$max_attempts - 等待服务启动..."
        sleep 2
        ((attempt++))
    done
    
    log_error "$service_name 服务启动失败"
    return 1
}

# 生成测试流量
generate_traffic() {
    log_info "生成测试流量..."
    
    # 生成一些HTTP请求
    for i in {1..5}; do
        curl -s http://httpbin.org/get > /dev/null 2>&1 || true
        curl -s https://api.github.com > /dev/null 2>&1 || true
        sleep 1
    done
    
    log_success "测试流量生成完成"
}

# 检查指标
check_metrics() {
    log_info "检查Prometheus指标..."
    
    local metrics=$(curl -s "$SERVER_URL/metrics" 2>/dev/null || echo "")
    
    if echo "$metrics" | grep -q "network_domains_accessed"; then
        log_success "✅ 域名访问指标正常"
    else
        log_warning "⚠️  域名访问指标未找到"
    fi
    
    if echo "$metrics" | grep -q "network_domain_bytes"; then
        log_success "✅ 域名流量指标正常"
    else
        log_warning "⚠️  域名流量指标未找到"
    fi
    
    # 显示一些示例指标
    log_info "示例指标:"
    echo "$metrics" | grep "network_domains_accessed" | head -3 || true
}

# 测试Agent重启恢复
test_agent_restart() {
    log_info "测试Agent重启恢复..."
    
    # 记录重启前的指标
    local before_metrics=$(curl -s "$SERVER_URL/metrics" 2>/dev/null || echo "")
    
    # 重启Agent
    log_info "重启Agent..."
    docker restart netmon-agent
    sleep 20
    
    # 检查重启后的指标
    local after_metrics=$(curl -s "$SERVER_URL/metrics" 2>/dev/null || echo "")
    
    if [ -n "$after_metrics" ] && echo "$after_metrics" | grep -q "network_domains_accessed"; then
        log_success "✅ Agent重启后指标恢复正常"
    else
        log_error "❌ Agent重启后指标异常"
        return 1
    fi
}

# 测试Server重启恢复
test_server_restart() {
    log_info "测试Server重启恢复..."
    
    # 重启Server
    log_info "重启Server..."
    docker restart netmon-server
    
    # 等待服务恢复
    if check_service "Server" "$SERVER_URL/health"; then
        log_success "✅ Server重启后恢复正常"
    else
        log_error "❌ Server重启后恢复失败"
        return 1
    fi
    
    # 检查数据是否持久化
    sleep 10
    local metrics=$(curl -s "$SERVER_URL/metrics" 2>/dev/null || echo "")
    if echo "$metrics" | grep -q "network_domains_accessed"; then
        log_success "✅ Server重启后数据持久化正常"
    else
        log_warning "⚠️  Server重启后数据可能丢失"
    fi
}

# 主测试流程
main() {
    log_info "开始部署测试..."
    
    # 1. 启动服务
    log_info "启动服务..."
    docker-compose -f $COMPOSE_FILE down -v 2>/dev/null || true
    docker-compose -f $COMPOSE_FILE up -d
    
    # 2. 检查服务状态
    if ! check_service "Server" "$SERVER_URL/health"; then
        log_error "服务启动失败"
        docker-compose -f $COMPOSE_FILE logs
        exit 1
    fi
    
    # 3. 生成测试流量
    generate_traffic
    sleep 15  # 等待数据上报
    
    # 4. 检查指标
    check_metrics
    
    # 5. 测试Agent重启
    if test_agent_restart; then
        log_success "Agent重启测试通过"
    else
        log_warning "Agent重启测试失败"
    fi
    
    # 6. 测试Server重启
    if test_server_restart; then
        log_success "Server重启测试通过"
    else
        log_warning "Server重启测试失败"
    fi
    
    # 7. 显示服务信息
    log_info "=== 服务信息 ==="
    echo "Server: $SERVER_URL"
    echo "Metrics: $SERVER_URL/metrics"
    echo "Health: $SERVER_URL/health"
    
    # 8. 显示容器状态
    log_info "=== 容器状态 ==="
    docker-compose -f $COMPOSE_FILE ps
    
    log_success "🎉 部署测试完成！"
}

# 清理函数
cleanup() {
    log_info "清理测试环境..."
    docker-compose -f $COMPOSE_FILE down -v 2>/dev/null || true
}

# 处理命令行参数
case "${1:-}" in
    "start")
        log_info "启动服务..."
        docker-compose -f $COMPOSE_FILE up -d
        check_service "Server" "$SERVER_URL/health"
        log_success "服务启动完成"
        ;;
    "stop")
        log_info "停止服务..."
        docker-compose -f $COMPOSE_FILE down
        log_success "服务停止完成"
        ;;
    "test")
        trap cleanup EXIT
        main
        ;;
    "cleanup")
        cleanup
        ;;
    *)
        echo "用法: $0 {start|stop|test|cleanup}"
        echo "  start   - 启动服务"
        echo "  stop    - 停止服务"
        echo "  test    - 运行完整测试"
        echo "  cleanup - 清理环境"
        exit 1
        ;;
esac
