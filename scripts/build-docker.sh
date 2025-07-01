#!/bin/bash

set -e

# 配置
DOCKER_REGISTRY="zhoushoujian"
IMAGE_NAME="go-net-monitoring"
VERSION=${1:-"latest"}
PLATFORMS="linux/amd64,linux/arm64"

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
    echo "║                Docker 镜像构建脚本                            ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
}

# 检查Docker环境
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
    
    # 检查buildx
    if ! docker buildx version &> /dev/null; then
        log_error "Docker buildx未安装"
        exit 1
    fi
    
    log_success "Docker环境检查通过"
}

# 准备构建环境
prepare_build() {
    log_info "准备构建环境..."
    
    # 创建buildx builder
    if ! docker buildx ls | grep -q "netmon-builder"; then
        log_info "创建buildx builder..."
        docker buildx create --name netmon-builder --use
    else
        log_info "使用现有buildx builder..."
        docker buildx use netmon-builder
    fi
    
    # 启动builder
    docker buildx inspect --bootstrap
    
    log_success "构建环境准备完成"
}

# 获取构建信息
get_build_info() {
    log_info "获取构建信息..."
    
    BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    
    log_info "版本: $VERSION"
    log_info "构建时间: $BUILD_TIME"
    log_info "Git提交: $GIT_COMMIT"
    log_info "平台: $PLATFORMS"
}

# 构建镜像
build_image() {
    log_info "开始构建Docker镜像..."
    
    local full_image_name="${DOCKER_REGISTRY}/${IMAGE_NAME}:${VERSION}"
    local latest_image_name="${DOCKER_REGISTRY}/${IMAGE_NAME}:latest"
    
    # 构建参数
    local build_args=(
        --platform "$PLATFORMS"
        --build-arg "VERSION=$VERSION"
        --build-arg "BUILD_TIME=$BUILD_TIME"
        --build-arg "GIT_COMMIT=$GIT_COMMIT"
        --tag "$full_image_name"
    )
    
    # 如果是latest版本，添加latest标签
    if [ "$VERSION" = "latest" ] || [[ "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        build_args+=(--tag "$latest_image_name")
    fi
    
    # 如果设置了推送标志，添加推送参数
    if [ "$PUSH" = "true" ]; then
        build_args+=(--push)
        log_info "将推送到Docker Hub"
    else
        build_args+=(--load)
        log_warn "仅本地构建，不推送到仓库"
    fi
    
    # 执行构建
    docker buildx build "${build_args[@]}" .
    
    if [ $? -eq 0 ]; then
        log_success "镜像构建成功: $full_image_name"
        if [ "$VERSION" != "latest" ]; then
            log_success "镜像构建成功: $latest_image_name"
        fi
    else
        log_error "镜像构建失败"
        exit 1
    fi
}

# 测试镜像
test_image() {
    if [ "$PUSH" = "true" ]; then
        log_info "跳过本地测试（镜像已推送）"
        return
    fi
    
    log_info "测试Docker镜像..."
    
    local test_image="${DOCKER_REGISTRY}/${IMAGE_NAME}:${VERSION}"
    
    # 测试server
    log_info "测试server组件..."
    if docker run --rm -e COMPONENT=server "$test_image" --version; then
        log_success "Server组件测试通过"
    else
        log_error "Server组件测试失败"
        exit 1
    fi
    
    # 测试agent（不需要特权模式，只测试版本）
    log_info "测试agent组件..."
    if docker run --rm -e COMPONENT=agent "$test_image" agent --version; then
        log_success "Agent组件测试通过"
    else
        log_error "Agent组件测试失败"
        exit 1
    fi
}

# 显示使用说明
show_usage() {
    echo "使用方法: $0 [VERSION] [OPTIONS]"
    echo ""
    echo "参数:"
    echo "  VERSION     镜像版本 (默认: latest)"
    echo ""
    echo "选项:"
    echo "  --push      构建后推送到Docker Hub"
    echo "  --no-test   跳过镜像测试"
    echo "  --help      显示帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                    # 构建latest版本"
    echo "  $0 v1.0.0            # 构建v1.0.0版本"
    echo "  $0 v1.0.0 --push     # 构建并推送v1.0.0版本"
    echo ""
    echo "环境变量:"
    echo "  DOCKER_REGISTRY      Docker仓库 (默认: zhoushoujian)"
    echo "  IMAGE_NAME           镜像名称 (默认: go-net-monitoring)"
    echo "  PLATFORMS            构建平台 (默认: linux/amd64,linux/arm64)"
}

# 解析命令行参数
parse_args() {
    PUSH=false
    NO_TEST=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --push)
                PUSH=true
                shift
                ;;
            --no-test)
                NO_TEST=true
                shift
                ;;
            --help)
                show_usage
                exit 0
                ;;
            -*)
                log_error "未知选项: $1"
                show_usage
                exit 1
                ;;
            *)
                if [ -z "$VERSION_SET" ]; then
                    VERSION="$1"
                    VERSION_SET=true
                else
                    log_error "多余的参数: $1"
                    show_usage
                    exit 1
                fi
                shift
                ;;
        esac
    done
}

# 主函数
main() {
    show_banner
    
    # 解析参数
    parse_args "$@"
    
    # 检查环境
    check_docker
    prepare_build
    get_build_info
    
    # 构建镜像
    build_image
    
    # 测试镜像
    if [ "$NO_TEST" != "true" ]; then
        test_image
    fi
    
    echo ""
    log_success "Docker镜像构建完成！"
    echo ""
    echo "📋 使用说明:"
    echo ""
    echo "🚀 运行Server:"
    echo "  docker run -d -p 8080:8080 -e COMPONENT=server ${DOCKER_REGISTRY}/${IMAGE_NAME}:${VERSION}"
    echo ""
    echo "🔍 运行Agent:"
    echo "  docker run -d --privileged --network host -e COMPONENT=agent ${DOCKER_REGISTRY}/${IMAGE_NAME}:${VERSION}"
    echo ""
    echo "📊 使用Docker Compose:"
    echo "  docker-compose up -d"
    echo ""
    echo "☸️  部署到Kubernetes:"
    echo "  kubectl apply -f k8s/"
    echo ""
}

# 运行主函数
main "$@"
