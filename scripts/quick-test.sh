#!/bin/bash

# 快速测试脚本 - 启动环境并验证功能

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_ENV_SCRIPT="$SCRIPT_DIR/test-env.sh"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# 清理函数
cleanup() {
    log_info "🧹 清理测试环境..."
    $TEST_ENV_SCRIPT stop >/dev/null 2>&1 || true
}

# 设置清理陷阱
trap cleanup EXIT

main() {
    log_info "🚀 开始快速测试..."
    
    # 1. 停止现有服务
    log_info "1️⃣ 清理现有服务..."
    $TEST_ENV_SCRIPT stop >/dev/null 2>&1 || true
    
    # 2. 启动测试环境
    log_info "2️⃣ 启动测试环境..."
    if ! $TEST_ENV_SCRIPT start; then
        log_error "❌ 启动失败"
        exit 1
    fi
    
    # 3. 等待服务稳定
    log_info "3️⃣ 等待服务稳定..."
    sleep 5
    
    # 4. 验证服务状态
    log_info "4️⃣ 验证服务状态..."
    $TEST_ENV_SCRIPT status
    
    # 5. 测试API接口
    log_info "5️⃣ 测试API接口..."
    test_apis
    
    # 6. 等待数据上报
    log_info "6️⃣ 等待数据上报..."
    sleep 15
    
    # 7. 验证指标数据
    log_info "7️⃣ 验证指标数据..."
    test_metrics
    
    log_info "✅ 快速测试完成!"
    log_info "💡 使用 './scripts/test-env.sh logs' 查看详细日志"
    log_info "💡 使用 './scripts/test-env.sh stop' 停止服务"
}

# 测试API接口
test_apis() {
    local server_url="http://localhost:8080"
    
    # 健康检查
    if curl -s "$server_url/health" >/dev/null; then
        log_info "  ✅ 健康检查通过"
    else
        log_error "  ❌ 健康检查失败"
        return 1
    fi
    
    # Prometheus指标
    if curl -s "$server_url/metrics" | head -1 >/dev/null; then
        log_info "  ✅ Prometheus指标可访问"
    else
        log_error "  ❌ Prometheus指标访问失败"
        return 1
    fi
}

# 测试指标数据
test_metrics() {
    local server_url="http://localhost:8080"
    
    # 检查网络指标
    local network_metrics=$(curl -s "$server_url/metrics" | grep "network_" | wc -l)
    if [ "$network_metrics" -gt 0 ]; then
        log_info "  ✅ 发现 $network_metrics 个网络指标"
        
        # 显示部分指标
        echo "  📊 部分指标示例:"
        curl -s "$server_url/metrics" | grep "network_" | head -3 | sed 's/^/    /'
    else
        log_warn "  ⚠️  暂未发现网络指标，可能需要更多时间"
    fi
    
    # 检查Agent指标
    local agent_metrics=$(curl -s "$server_url/metrics" | grep "agent_" | wc -l)
    if [ "$agent_metrics" -gt 0 ]; then
        log_info "  ✅ 发现 $agent_metrics 个Agent指标"
    else
        log_warn "  ⚠️  暂未发现Agent指标"
    fi
}

# 运行主函数
main "$@"
