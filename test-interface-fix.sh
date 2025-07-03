#!/bin/bash

# 测试网络接口和IP地址指标修复
set -e

echo "=== 测试网络接口和IP地址指标修复 ==="

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
    local max_attempts=30
    local attempt=1
    
    log_info "等待服务就绪..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s "$SERVER_URL/health" > /dev/null 2>&1; then
            log_success "服务就绪"
            return 0
        fi
        
        log_info "尝试 $attempt/$max_attempts - 等待服务启动..."
        sleep 2
        ((attempt++))
    done
    
    log_error "服务启动超时"
    return 1
}

# 生成测试流量
generate_traffic() {
    log_info "生成测试流量..."
    
    # 生成一些HTTP请求
    for i in {1..3}; do
        curl -s http://httpbin.org/get > /dev/null 2>&1 || true
        curl -s https://api.github.com > /dev/null 2>&1 || true
        sleep 1
    done
    
    log_success "测试流量生成完成"
}

# 检查接口指标
check_interface_metrics() {
    log_info "检查网络接口指标..."
    
    local metrics=$(curl -s "$SERVER_URL/metrics" 2>/dev/null || echo "")
    
    if [ -z "$metrics" ]; then
        log_error "无法获取指标数据"
        return 1
    fi
    
    # 检查interface标签是否不再是"unknown"
    local unknown_interfaces=$(echo "$metrics" | grep 'interface="unknown"' | wc -l)
    local total_interface_metrics=$(echo "$metrics" | grep 'interface=' | wc -l)
    
    log_info "指标统计:"
    echo "  总接口指标数: $total_interface_metrics"
    echo "  unknown接口数: $unknown_interfaces"
    
    # 显示接口指标示例
    log_info "接口指标示例:"
    echo "$metrics" | grep 'network_domain_connections_total.*interface=' | head -5
    
    # 检查网卡信息指标
    local interface_info_metrics=$(echo "$metrics" | grep 'network_interface_info' | wc -l)
    log_info "网卡信息指标数量: $interface_info_metrics"
    
    if [ $interface_info_metrics -gt 0 ]; then
        log_success "✅ 网卡信息指标存在"
        log_info "网卡信息指标示例:"
        echo "$metrics" | grep 'network_interface_info' | head -3
    else
        log_warning "⚠️  网卡信息指标不存在"
    fi
    
    # 检查是否有真实的接口名称
    local real_interfaces=$(echo "$metrics" | grep -E 'interface="(eth[0-9]+|en[0-9]+|wlan[0-9]+|ens[0-9]+|enp[0-9]+)"' | wc -l)
    
    if [ $real_interfaces -gt 0 ]; then
        log_success "✅ 检测到真实网络接口名称"
        log_info "真实接口指标示例:"
        echo "$metrics" | grep -E 'interface="(eth[0-9]+|en[0-9]+|wlan[0-9]+|ens[0-9]+|enp[0-9]+)"' | head -3
    else
        log_warning "⚠️  未检测到真实网络接口名称"
    fi
    
    # 检查IP地址信息
    local ip_addresses=$(echo "$metrics" | grep 'network_interface_info' | grep -oE 'ip_address="[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+"' | sort -u)
    
    if [ -n "$ip_addresses" ]; then
        log_success "✅ 检测到网卡IP地址"
        log_info "检测到的IP地址:"
        echo "$ip_addresses" | sed 's/ip_address="//g' | sed 's/"//g' | while read ip; do
            echo "  - $ip"
        done
    else
        log_warning "⚠️  未检测到网卡IP地址"
    fi
    
    return 0
}

# 检查容器内的网络接口
check_container_interfaces() {
    log_info "检查容器内的网络接口..."
    
    # 检查Agent容器的网络接口
    if docker exec netmon-agent ip addr show 2>/dev/null; then
        log_success "Agent容器网络接口信息获取成功"
    else
        log_warning "无法获取Agent容器网络接口信息"
    fi
    
    # 检查Agent容器的路由表
    log_info "Agent容器路由信息:"
    docker exec netmon-agent ip route show 2>/dev/null || log_warning "无法获取路由信息"
}

# 主测试流程
main() {
    log_info "开始测试网络接口和IP地址指标修复..."
    
    # 1. 启动服务
    log_info "启动服务..."
    docker-compose -f $COMPOSE_FILE down -v 2>/dev/null || true
    docker-compose -f $COMPOSE_FILE up -d
    
    # 2. 等待服务就绪
    if ! check_service; then
        log_error "服务启动失败"
        docker-compose -f $COMPOSE_FILE logs
        exit 1
    fi
    
    # 3. 检查容器网络接口
    check_container_interfaces
    
    # 4. 生成测试流量
    generate_traffic
    sleep 15  # 等待数据上报
    
    # 5. 检查接口指标
    if check_interface_metrics; then
        log_success "接口指标检查完成"
    else
        log_error "接口指标检查失败"
    fi
    
    # 6. 显示详细的指标信息
    log_info "=== 详细指标信息 ==="
    
    local metrics=$(curl -s "$SERVER_URL/metrics" 2>/dev/null || echo "")
    
    # 域名连接指标
    log_info "域名连接指标 (前10个):"
    echo "$metrics" | grep 'network_domain_connections_total' | head -10
    
    # 网卡信息指标
    log_info "网卡信息指标:"
    echo "$metrics" | grep 'network_interface_info'
    
    # 接口分布统计
    log_info "接口分布统计:"
    echo "$metrics" | grep 'interface=' | grep -oE 'interface="[^"]*"' | sort | uniq -c | sort -nr
    
    log_success "🎉 网络接口和IP地址指标测试完成！"
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
        check_service
        log_success "服务启动完成"
        ;;
    "check")
        log_info "检查指标..."
        check_interface_metrics
        ;;
    "test")
        trap cleanup EXIT
        main
        ;;
    "cleanup")
        cleanup
        ;;
    *)
        echo "用法: $0 {start|check|test|cleanup}"
        echo "  start   - 启动服务"
        echo "  check   - 检查指标"
        echo "  test    - 运行完整测试"
        echo "  cleanup - 清理环境"
        exit 1
        ;;
esac
