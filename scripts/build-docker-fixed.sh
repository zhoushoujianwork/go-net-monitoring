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
    echo "║                Docker 镜像构建脚本 (网络优化版)                ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo ""
}

# 检查网络连接
check_network() {
    log_info "检查网络连接..."
    
    # 检查Docker Hub连接
    if curl -I --connect-timeout 10 https://registry-1.docker.io/v2/ >/dev/null 2>&1; then
        log_success "Docker Hub连接正常"
        return 0
    else
        log_warn "Docker Hub连接失败，尝试使用镜像源"
        return 1
    fi
}

# 配置Docker镜像源
configure_docker_mirror() {
    log_info "配置Docker镜像源..."
    
    # 创建或更新daemon.json
    local daemon_config="/etc/docker/daemon.json"
    local temp_config="/tmp/daemon.json"
    
    # 备份现有配置
    if [ -f "$daemon_config" ]; then
        cp "$daemon_config" "${daemon_config}.backup"
        log_info "已备份现有Docker配置"
    fi
    
    # 创建新配置
    cat > "$temp_config" << 'EOF'
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com"
  ],
  "insecure-registries": [],
  "experimental": false,
  "features": {
    "buildkit": true
  }
}
EOF
    
    # 应用配置
    if sudo cp "$temp_config" "$daemon_config" 2>/dev/null; then
        log_success "Docker镜像源配置完成"
        
        # 重启Docker服务
        log_info "重启Docker服务..."
        if sudo systemctl restart docker 2>/dev/null; then
            log_success "Docker服务重启成功"
            sleep 5  # 等待服务完全启动
        else
            log_warn "无法重启Docker服务，请手动重启"
        fi
    else
        log_warn "无法配置Docker镜像源，权限不足"
    fi
    
    rm -f "$temp_config"
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

# 预拉取基础镜像
pull_base_images() {
    log_info "预拉取基础镜像..."
    
    local images=(
        "golang:1.21-alpine"
        "alpine:3.19"
    )
    
    for image in "${images[@]}"; do
        log_info "拉取镜像: $image"
        
        # 尝试拉取镜像，设置超时
        if timeout 300 docker pull "$image"; then
            log_success "镜像拉取成功: $image"
        else
            log_error "镜像拉取失败: $image"
            
            # 尝试使用不同的标签
            if [[ "$image" == *":3.19" ]]; then
                log_info "尝试使用alpine:latest替代"
                if timeout 300 docker pull alpine:latest; then
                    docker tag alpine:latest alpine:3.19
                    log_success "使用alpine:latest作为替代"
                else
                    log_error "无法拉取Alpine镜像"
                    return 1
                fi
            elif [[ "$image" == *":1.21-alpine" ]]; then
                log_info "尝试使用golang:alpine替代"
                if timeout 300 docker pull golang:alpine; then
                    docker tag golang:alpine golang:1.21-alpine
                    log_success "使用golang:alpine作为替代"
                else
                    log_error "无法拉取Golang镜像"
                    return 1
                fi
            fi
        fi
    done
    
    log_success "基础镜像准备完成"
}

# 准备构建环境
prepare_build() {
    log_info "准备构建环境..."
    
    # 创建buildx builder
    if ! docker buildx ls | grep -q "netmon-builder"; then
        log_info "创建buildx builder..."
        docker buildx create --name netmon-builder --use --driver-opt network=host
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
        --progress=plain
        --network=host
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
        # 多平台构建时使用--load会有问题，改为本地构建单平台
        if [ "$PLATFORMS" = "linux/amd64,linux/arm64" ]; then
            log_warn "多平台构建不支持--load，将只构建amd64平台用于本地测试"
            build_args=(
                --platform "linux/amd64"
                --build-arg "VERSION=$VERSION"
                --build-arg "BUILD_TIME=$BUILD_TIME"
                --build-arg "GIT_COMMIT=$GIT_COMMIT"
                --tag "$full_image_name"
                --load
                --progress=plain
                --network=host
            )
        else
            build_args+=(--load)
        fi
        log_warn "仅本地构建，不推送到仓库"
    fi
    
    # 执行构建
    log_info "执行构建命令: docker buildx build ${build_args[*]} ."
    
    if docker buildx build "${build_args[@]}" .; then
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
    
    # 检查镜像是否存在
    if ! docker images | grep -q "$IMAGE_NAME"; then
        log_warn "本地镜像不存在，跳过测试"
        return
    fi
    
    # 测试server
    log_info "测试server组件..."
    if timeout 30 docker run --rm -e COMPONENT=server "$test_image" --version 2>/dev/null; then
        log_success "Server组件测试通过"
    else
        log_warn "Server组件测试失败或超时"
    fi
    
    # 测试agent（不需要特权模式，只测试版本）
    log_info "测试agent组件..."
    if timeout 30 docker run --rm -e COMPONENT=agent "$test_image" agent --version 2>/dev/null; then
        log_success "Agent组件测试通过"
    else
        log_warn "Agent组件测试失败或超时"
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
    echo "  --fix-network 修复网络问题"
    echo "  --help      显示帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                    # 构建latest版本"
    echo "  $0 v1.0.0            # 构建v1.0.0版本"
    echo "  $0 v1.0.0 --push     # 构建并推送v1.0.0版本"
    echo "  $0 --fix-network     # 修复网络问题后构建"
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
    FIX_NETWORK=false
    
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
            --fix-network)
                FIX_NETWORK=true
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
    
    # 网络检查和修复
    if ! check_network || [ "$FIX_NETWORK" = "true" ]; then
        configure_docker_mirror
    fi
    
    # 检查环境
    check_docker
    
    # 预拉取基础镜像
    pull_base_images
    
    # 准备构建环境
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
