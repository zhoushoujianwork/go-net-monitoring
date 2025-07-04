#!/bin/bash

set -e

# 简化的本地构建脚本 - 避免网络问题

# 配置
IMAGE_NAME="go-net-monitoring"
VERSION=${1:-"latest"}

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

# 显示横幅
show_banner() {
    echo ""
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                本地Docker镜像构建脚本                         ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
}

# 检查Docker
check_docker() {
    log_info "检查Docker环境..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker未安装"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker服务未运行"
        exit 1
    fi
    
    log_success "Docker环境检查通过"
}

# 构建镜像
build_image() {
    log_info "开始构建Docker镜像..."
    
    # 获取构建信息
    BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    
    log_info "版本: $VERSION"
    log_info "构建时间: $BUILD_TIME"
    log_info "Git提交: $GIT_COMMIT"
    
    # 构建参数
    local build_args=(
        --build-arg "VERSION=$VERSION"
        --build-arg "BUILD_TIME=$BUILD_TIME"
        --build-arg "GIT_COMMIT=$GIT_COMMIT"
        --tag "$IMAGE_NAME:$VERSION"
        --tag "$IMAGE_NAME:latest"
        --progress=plain
    )
    
    # 执行构建
    log_info "执行构建命令: docker build ${build_args[*]} ."
    
    if docker build "${build_args[@]}" .; then
        log_success "镜像构建成功: $IMAGE_NAME:$VERSION"
        log_success "镜像构建成功: $IMAGE_NAME:latest"
    else
        log_error "镜像构建失败"
        exit 1
    fi
}

# 测试镜像
test_image() {
    log_info "测试Docker镜像..."
    
    # 测试server
    log_info "测试server组件..."
    if timeout 30 docker run --rm -e COMPONENT=server "$IMAGE_NAME:$VERSION" --version 2>/dev/null; then
        log_success "Server组件测试通过"
    else
        log_warn "Server组件测试失败或超时"
    fi
    
    # 测试agent
    log_info "测试agent组件..."
    if timeout 30 docker run --rm -e COMPONENT=agent "$IMAGE_NAME:$VERSION" agent --version 2>/dev/null; then
        log_success "Agent组件测试通过"
    else
        log_warn "Agent组件测试失败或超时"
    fi
}

# 显示镜像信息
show_image_info() {
    log_info "镜像信息:"
    docker images | grep "$IMAGE_NAME" | head -5
}

# 主函数
main() {
    show_banner
    
    # 检查环境
    check_docker
    
    # 构建镜像
    build_image
    
    # 测试镜像
    test_image
    
    # 显示镜像信息
    show_image_info
    
    echo ""
    log_success "本地Docker镜像构建完成！"
    echo ""
    echo "📋 使用说明:"
    echo ""
    echo "🚀 运行Server:"
    echo "  docker run -d -p 8080:8080 -e COMPONENT=server $IMAGE_NAME:$VERSION"
    echo ""
    echo "🔍 运行Agent:"
    echo "  docker run -d --privileged --network host -e COMPONENT=agent $IMAGE_NAME:$VERSION"
    echo ""
    echo "📊 使用Docker Compose:"
    echo "  docker-compose up -d"
    echo ""
}

# 运行主函数
main "$@"
