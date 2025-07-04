#!/bin/bash

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

# 检查Docker是否运行
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker未运行或无权限访问"
        exit 1
    fi
}

# 清理旧的构建缓存
cleanup_cache() {
    log_info "清理Docker构建缓存..."
    docker builder prune -f >/dev/null 2>&1 || true
    log_success "缓存清理完成"
}

# 构建镜像
build_image() {
    log_info "开始构建优化镜像..."
    
    # 获取构建信息
    VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    
    log_info "构建信息:"
    log_info "  版本: $VERSION"
    log_info "  构建时间: $BUILD_TIME"
    log_info "  Git提交: $GIT_COMMIT"
    
    # 构建镜像
    docker build \
        --build-arg VERSION="$VERSION" \
        --build-arg BUILD_TIME="$BUILD_TIME" \
        --build-arg GIT_COMMIT="$GIT_COMMIT" \
        --tag go-net-monitoring:latest \
        --tag go-net-monitoring:$VERSION \
        . || {
        log_error "镜像构建失败"
        exit 1
    }
    
    log_success "镜像构建完成"
}

# 显示镜像信息
show_image_info() {
    log_info "镜像信息:"
    docker images go-net-monitoring:latest --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
}

# 运行构建后测试
test_image() {
    log_info "测试镜像..."
    
    # 测试server组件
    if docker run --rm -e COMPONENT=server go-net-monitoring:latest --version >/dev/null 2>&1; then
        log_success "Server组件测试通过"
    else
        log_warn "Server组件测试失败"
    fi
    
    # 测试agent组件
    if docker run --rm -e COMPONENT=agent go-net-monitoring:latest --version >/dev/null 2>&1; then
        log_success "Agent组件测试通过"
    else
        log_warn "Agent组件测试失败"
    fi
}

# 主函数
main() {
    log_info "开始优化构建流程..."
    
    check_docker
    
    # 解析参数
    CLEAN_CACHE=false
    RUN_TESTS=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --clean-cache)
                CLEAN_CACHE=true
                shift
                ;;
            --test)
                RUN_TESTS=true
                shift
                ;;
            --help)
                echo "用法: $0 [选项]"
                echo "选项:"
                echo "  --clean-cache  清理构建缓存"
                echo "  --test         运行构建后测试"
                echo "  --help         显示帮助信息"
                exit 0
                ;;
            *)
                log_error "未知参数: $1"
                exit 1
                ;;
        esac
    done
    
    if [ "$CLEAN_CACHE" = true ]; then
        cleanup_cache
    fi
    
    build_image
    show_image_info
    
    if [ "$RUN_TESTS" = true ]; then
        test_image
    fi
    
    log_success "优化构建完成！"
    log_info "使用方法:"
    log_info "  docker-compose up -d"
    log_info "  或者"
    log_info "  DEBUG_MODE=true LOG_LEVEL=debug docker-compose up -d"
}

# 运行主函数
main "$@"
