#!/bin/bash

###########################################
# 测试安装脚本
###########################################

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

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 测试本地安装脚本
test_local_install() {
    local component=$1
    log_info "测试本地安装脚本: $component"
    
    # 创建临时目录
    local test_dir=$(mktemp -d)
    export WEBI_PKG_DIR="$test_dir/.local/opt/go-net-monitoring"
    export WEBI_PKG_BIN="$test_dir/.local/bin"
    
    # 运行安装脚本
    if bash scripts/install.sh "$component"; then
        log_success "本地安装测试通过: $component"
        
        # 检查文件是否存在
        if [ -f "$WEBI_PKG_BIN/$component" ]; then
            log_success "二进制文件存在: $WEBI_PKG_BIN/$component"
        else
            log_error "二进制文件不存在: $WEBI_PKG_BIN/$component"
        fi
        
        # 检查配置文件
        if [ -f "$WEBI_PKG_DIR/configs/${component}.yaml" ]; then
            log_success "配置文件存在: $WEBI_PKG_DIR/configs/${component}.yaml"
        else
            log_error "配置文件不存在: $WEBI_PKG_DIR/configs/${component}.yaml"
        fi
    else
        log_error "本地安装测试失败: $component"
    fi
    
    # 清理
    rm -rf "$test_dir"
}

# 测试构建脚本
test_build_script() {
    log_info "测试构建脚本..."
    
    if [ -f "scripts/build-release.sh" ]; then
        chmod +x scripts/build-release.sh
        if ./scripts/build-release.sh "v1.0.0-test"; then
            log_success "构建脚本测试通过"
            
            # 检查生成的文件
            if [ -d "dist" ] && [ "$(ls -A dist/*.tar.gz 2>/dev/null | wc -l)" -gt 0 ]; then
                log_success "构建产物生成成功"
                ls -la dist/*.tar.gz
            else
                log_error "构建产物生成失败"
            fi
        else
            log_error "构建脚本测试失败"
        fi
    else
        log_error "构建脚本不存在"
    fi
}

# 测试快速安装脚本语法
test_quick_install_syntax() {
    log_info "测试快速安装脚本语法..."
    
    if bash -n scripts/quick-install.sh; then
        log_success "快速安装脚本语法检查通过"
    else
        log_error "快速安装脚本语法错误"
    fi
}

# 主函数
main() {
    echo "🧪 开始测试安装脚本..."
    echo ""
    
    # 检查必要文件
    local required_files=(
        "scripts/install.sh"
        "scripts/quick-install.sh"
        "scripts/build-release.sh"
    )
    
    for file in "${required_files[@]}"; do
        if [ ! -f "$file" ]; then
            log_error "必要文件不存在: $file"
            exit 1
        fi
    done
    
    # 运行测试
    test_quick_install_syntax
    test_build_script
    
    # 注意：本地安装测试需要实际的发布文件，这里跳过
    # test_local_install "agent"
    # test_local_install "server"
    
    echo ""
    log_success "所有测试完成！"
    echo ""
    echo "📋 下一步："
    echo "1. 推送代码到 GitHub"
    echo "2. 创建 Release 标签"
    echo "3. 测试实际安装"
    echo "4. 提交到 webinstall.dev"
}

# 运行测试
main "$@"
