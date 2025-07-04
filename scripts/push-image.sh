#!/bin/bash

set -e

# 简化的镜像推送脚本

# 配置
DOCKER_REGISTRY="zhoushoujian"
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
    echo "║                Docker 镜像推送脚本                           ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
}

# 检查本地镜像
check_local_image() {
    log_info "检查本地镜像..."
    
    if docker images | grep -q "$IMAGE_NAME"; then
        log_success "找到本地镜像: $IMAGE_NAME"
        docker images | grep "$IMAGE_NAME" | head -3
    else
        log_error "本地镜像不存在: $IMAGE_NAME"
        log_info "请先构建镜像: make docker-build-local"
        exit 1
    fi
}

# 标记镜像
tag_image() {
    log_info "标记镜像..."
    
    local source_image="$IMAGE_NAME:latest"
    local target_image="$DOCKER_REGISTRY/$IMAGE_NAME:$VERSION"
    local latest_image="$DOCKER_REGISTRY/$IMAGE_NAME:latest"
    
    # 标记版本镜像
    if docker tag "$source_image" "$target_image"; then
        log_success "镜像标记成功: $target_image"
    else
        log_error "镜像标记失败: $target_image"
        exit 1
    fi
    
    # 如果是latest或版本号，也标记为latest
    if [ "$VERSION" = "latest" ] || [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        if docker tag "$source_image" "$latest_image"; then
            log_success "镜像标记成功: $latest_image"
        else
            log_warn "latest标记失败: $latest_image"
        fi
    fi
}

# 推送镜像
push_image() {
    log_info "推送镜像到Docker Hub..."
    
    local target_image="$DOCKER_REGISTRY/$IMAGE_NAME:$VERSION"
    local latest_image="$DOCKER_REGISTRY/$IMAGE_NAME:latest"
    
    # 检查Docker Hub登录状态
    if ! docker info | grep -q "Username:"; then
        log_warn "未登录Docker Hub，请先登录:"
        log_info "docker login"
        
        # 尝试自动登录提示
        echo -n "是否现在登录? (y/N): "
        read -r response
        if [[ "$response" =~ ^[Yy]$ ]]; then
            docker login
        else
            log_error "需要登录Docker Hub才能推送镜像"
            exit 1
        fi
    fi
    
    # 推送版本镜像
    log_info "推送镜像: $target_image"
    if docker push "$target_image"; then
        log_success "镜像推送成功: $target_image"
    else
        log_error "镜像推送失败: $target_image"
        exit 1
    fi
    
    # 推送latest镜像
    if [ "$VERSION" = "latest" ] || [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        log_info "推送镜像: $latest_image"
        if docker push "$latest_image"; then
            log_success "镜像推送成功: $latest_image"
        else
            log_warn "latest镜像推送失败: $latest_image"
        fi
    fi
}

# 显示使用说明
show_usage() {
    echo "使用方法: $0 [VERSION]"
    echo ""
    echo "参数:"
    echo "  VERSION     镜像版本 (默认: latest)"
    echo ""
    echo "示例:"
    echo "  $0                    # 推送latest版本"
    echo "  $0 v1.0.0            # 推送v1.0.0版本"
    echo ""
    echo "前置条件:"
    echo "  1. 本地已构建镜像: make docker-build-local"
    echo "  2. 已登录Docker Hub: docker login"
}

# 主函数
main() {
    show_banner
    
    # 检查参数
    if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
        show_usage
        exit 0
    fi
    
    log_info "推送版本: $VERSION"
    log_info "目标仓库: $DOCKER_REGISTRY/$IMAGE_NAME"
    
    # 执行推送流程
    check_local_image
    tag_image
    push_image
    
    echo ""
    log_success "镜像推送完成！"
    echo ""
    echo "📋 推送结果:"
    echo "  🚀 镜像: $DOCKER_REGISTRY/$IMAGE_NAME:$VERSION"
    if [ "$VERSION" = "latest" ] || [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "  🚀 镜像: $DOCKER_REGISTRY/$IMAGE_NAME:latest"
    fi
    echo ""
    echo "🔍 验证推送:"
    echo "  docker pull $DOCKER_REGISTRY/$IMAGE_NAME:$VERSION"
    echo ""
}

# 运行主函数
main "$@"
