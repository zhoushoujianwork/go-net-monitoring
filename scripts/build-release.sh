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

# 获取版本信息
get_version_info() {
    VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
    BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    GIT_COMMIT=${GIT_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
    
    log_info "版本信息: $VERSION"
    log_info "构建时间: $BUILD_TIME"
    log_info "Git提交: $GIT_COMMIT"
}

# 构建Server (跨平台)
build_servers() {
    log_info "构建Server (所有平台)..."
    
    local platforms=(
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
    )
    
    local ldflags="-s -w -X main.version=$VERSION -X main.buildTime=$BUILD_TIME -X main.gitCommit=$GIT_COMMIT"
    
    for platform in "${platforms[@]}"; do
        IFS='/' read -r goos goarch <<< "$platform"
        local output="bin/server-$goos-$goarch"
        
        if [ "$goos" = "windows" ]; then
            output="$output.exe"
        fi
        
        log_info "构建Server ($goos/$goarch)..."
        CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build \
            -ldflags "$ldflags" \
            -o "$output" \
            ./cmd/server
        
        log_success "Server构建完成: $output"
    done
}

# 构建所有平台的Agent
build_agents() {
    log_info "构建Agent (所有平台)..."
    
    local platforms=(
        "linux/amd64"
        "linux/arm64"
        "darwin/amd64"
        "darwin/arm64"
        "windows/amd64"
    )
    
    local ldflags="-s -w -X main.version=$VERSION -X main.buildTime=$BUILD_TIME -X main.gitCommit=$GIT_COMMIT"
    
    for platform in "${platforms[@]}"; do
        IFS='/' read -r goos goarch <<< "$platform"
        local output="bin/agent-$goos-$goarch"
        
        if [ "$goos" = "windows" ]; then
            output="$output.exe"
        fi
        
        log_info "尝试构建Agent ($goos/$goarch)..."
        
        if [ "$goos" = "windows" ]; then
            # Windows使用纯Go实现，可以交叉编译
            if CGO_ENABLED=1 GOOS=$goos GOARCH=$goarch go build \
                -ldflags "$ldflags" \
                -o "$output" \
                ./cmd/agent 2>/dev/null; then
                log_success "Agent构建成功: $output (Windows使用纯Go实现)"
            else
                log_warn "Agent构建失败 ($goos/$goarch)"
                echo "# Agent需要在$goos/$goarch平台上构建" > "$output.build-required"
            fi
        else
            # Unix/Linux/macOS需要libpcap开发库
            log_info "Unix/Linux/macOS平台需要libpcap开发库进行编译"
            if CGO_ENABLED=1 GOOS=$goos GOARCH=$goarch go build \
                -ldflags "$ldflags" \
                -o "$output" \
                ./cmd/agent 2>/dev/null; then
                log_success "Agent构建成功: $output"
            else
                log_warn "Agent构建失败 ($goos/$goarch) - 需要libpcap开发库"
                echo "# Agent需要在$goos/$goarch平台上构建，需要libpcap开发库" > "$output.build-required"
            fi
        fi
    done
    
    log_info "Agent构建说明:"
    log_info "  ✅ Windows: 使用纯Go实现，可交叉编译"
    log_info "  ⚠️  Unix/Linux/macOS: 需要libpcap开发库，建议目标平台构建"
}

# 创建发布包
create_release_packages() {
    log_info "创建发布包..."
    
    mkdir -p dist
    
    # Server发布包 (跨平台)
    local platforms=(
        "linux-amd64"
        "linux-arm64"
        "darwin-amd64"
        "darwin-arm64"
        "windows-amd64"
    )
    
    for platform in "${platforms[@]}"; do
        # 检查是否有对应的Agent
        local has_agent=false
        if [[ $platform == *"windows"* ]]; then
            if [ -f "bin/agent-$platform.exe" ]; then
                has_agent=true
            fi
        else
            if [ -f "bin/agent-$platform" ]; then
                has_agent=true
            fi
        fi
        
        if [ "$has_agent" = true ]; then
            create_full_package "$platform"
        else
            create_server_package "$platform"
        fi
    done
}

# 创建Server发布包
create_server_package() {
    local platform=$1
    local temp_dir="temp-server-$platform"
    
    log_info "创建Server发布包 ($platform)..."
    
    mkdir -p "$temp_dir"
    
    # 复制Server二进制
    if [[ $platform == *"windows"* ]]; then
        cp "bin/server-$platform.exe" "$temp_dir/"
    else
        cp "bin/server-$platform" "$temp_dir/"
    fi
    
    # 复制配置文件
    cp configs/server.yaml "$temp_dir/"
    
    # 创建README
    create_server_readme "$platform" "$temp_dir"
    
    # 创建启动脚本
    create_server_startup_script "$platform" "$temp_dir"
    
    # 打包
    cd "$temp_dir"
    if [[ $platform == *"windows"* ]]; then
        zip -r "../dist/go-net-monitoring-server-$platform.zip" .
    else
        tar -czf "../dist/go-net-monitoring-server-$platform.tar.gz" .
    fi
    cd ..
    
    rm -rf "$temp_dir"
    log_success "Server发布包创建完成: dist/go-net-monitoring-server-$platform.*"
}

# 创建完整发布包 (包含Agent)
create_full_package() {
    local platform=$1
    local temp_dir="temp-full-$platform"
    
    log_info "创建完整发布包 ($platform)..."
    
    mkdir -p "$temp_dir"
    
    # 复制二进制文件
    if [[ $platform == *"windows"* ]]; then
        cp "bin/server-$platform.exe" "$temp_dir/"
        cp "bin/agent-$platform.exe" "$temp_dir/"
    else
        cp "bin/server-$platform" "$temp_dir/"
        cp "bin/agent-$platform" "$temp_dir/"
    fi
    
    # 复制配置文件
    cp -r configs "$temp_dir/"
    
    # 创建README
    create_full_readme "$platform" "$temp_dir"
    
    # 创建启动脚本
    create_full_startup_scripts "$platform" "$temp_dir"
    
    # 打包
    cd "$temp_dir"
    if [[ $platform == *"windows"* ]]; then
        zip -r "../dist/go-net-monitoring-full-$platform.zip" .
    else
        tar -czf "../dist/go-net-monitoring-full-$platform.tar.gz" .
    fi
    cd ..
    
    rm -rf "$temp_dir"
    log_success "完整发布包创建完成: dist/go-net-monitoring-full-$platform.*"
}

# 创建Server README
create_server_readme() {
    local platform=$1
    local temp_dir=$2
    
    IFS='-' read -r goos goarch <<< "$platform"
    
    cat > "$temp_dir/README.md" << EOF
# Go Network Monitoring Server - $goos/$goarch

## 概述

这是Go Network Monitoring的Server组件，负责：
- 接收Agent上报的网络监控数据
- 提供Prometheus指标接口
- 提供REST API接口
- 数据聚合和存储

## 快速开始

### 1. 运行Server

EOF

    if [ "$goos" = "windows" ]; then
        cat >> "$temp_dir/README.md" << EOF
\`\`\`cmd
# 使用默认配置
server-$platform.exe --config server.yaml

# 启用debug模式
server-$platform.exe --config server.yaml --debug
\`\`\`
EOF
    else
        cat >> "$temp_dir/README.md" << EOF
\`\`\`bash
# 使用默认配置
./server-$platform --config server.yaml

# 启用debug模式
./server-$platform --config server.yaml --debug
\`\`\`
EOF
    fi

    cat >> "$temp_dir/README.md" << EOF

### 2. 验证运行

\`\`\`bash
# 检查健康状态
curl http://localhost:8080/health

# 查看Prometheus指标
curl http://localhost:8080/metrics
\`\`\`

### 3. Agent部署

Server运行后，需要在各个监控节点部署Agent。

#### 获取Agent:
1. **源码构建** (推荐):
   \`\`\`bash
   git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
   cd go-net-monitoring
   
   # 安装依赖
EOF

    if [ "$goos" = "linux" ]; then
        cat >> "$temp_dir/README.md" << EOF
   sudo apt-get install libpcap-dev  # Ubuntu/Debian
   sudo yum install libpcap-devel    # CentOS/RHEL
EOF
    elif [ "$goos" = "darwin" ]; then
        cat >> "$temp_dir/README.md" << EOF
   brew install libpcap
EOF
    elif [ "$goos" = "windows" ]; then
        cat >> "$temp_dir/README.md" << EOF
   # 安装Npcap: https://npcap.com/
EOF
    fi

    cat >> "$temp_dir/README.md" << EOF
   
   # 构建Agent
   make build-agent
   \`\`\`

2. **Docker方式**:
   \`\`\`bash
   docker run -d \\
     --name netmon-agent \\
     --privileged \\
     --network host \\
     -e COMPONENT=agent \\
     -e SERVER_URL=http://your-server:8080/api/v1/metrics \\
     zhoushoujian/go-net-monitoring:latest
   \`\`\`

## 配置说明

编辑 \`server.yaml\` 配置文件：

\`\`\`yaml
server:
  host: "0.0.0.0"
  port: 8080

storage:
  type: "memory"  # 或 "redis"
  
log:
  level: "info"   # debug, info, warn, error
  format: "json"  # json, text
\`\`\`

## 监控集成

### Prometheus
Server提供Prometheus指标接口: \`http://localhost:8080/metrics\`

### Grafana
导入项目提供的Dashboard配置文件进行可视化监控。

---
构建信息: $platform - $(date)
版本: $VERSION
EOF
}

# 创建完整README
create_full_readme() {
    local platform=$1
    local temp_dir=$2
    
    IFS='-' read -r goos goarch <<< "$platform"
    
    cat > "$temp_dir/README.md" << EOF
# Go Network Monitoring - $goos/$goarch

## 概述

完整的网络监控解决方案，包含：
- **Server**: 数据聚合和指标导出
- **Agent**: 网络流量监控和数据采集

## 快速开始

### 1. 安装依赖

EOF

    if [ "$goos" = "linux" ]; then
        cat >> "$temp_dir/README.md" << EOF
\`\`\`bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install libpcap0.8

# CentOS/RHEL  
sudo yum install libpcap
\`\`\`
EOF
    elif [ "$goos" = "darwin" ]; then
        cat >> "$temp_dir/README.md" << EOF
\`\`\`bash
# 使用Homebrew
brew install libpcap
\`\`\`
EOF
    elif [ "$goos" = "windows" ]; then
        cat >> "$temp_dir/README.md" << EOF
1. 下载并安装 Npcap: https://npcap.com/
2. 重启系统确保驱动加载
EOF
    fi

    cat >> "$temp_dir/README.md" << EOF

### 2. 启动服务

#### 方式1: 使用启动脚本
EOF

    if [ "$goos" = "windows" ]; then
        cat >> "$temp_dir/README.md" << EOF
\`\`\`cmd
# 启动Server
start-server.bat

# 启动Agent (需要管理员权限)
start-agent.bat
\`\`\`
EOF
    else
        cat >> "$temp_dir/README.md" << EOF
\`\`\`bash
# 启动Server
./start-server.sh

# 启动Agent (需要root权限)
./start-agent.sh
\`\`\`
EOF
    fi

    cat >> "$temp_dir/README.md" << EOF

#### 方式2: 直接运行
EOF

    if [ "$goos" = "windows" ]; then
        cat >> "$temp_dir/README.md" << EOF
\`\`\`cmd
# Server
server-$platform.exe --config configs/server.yaml

# Agent (管理员权限)
agent-$platform.exe --config configs/agent.yaml
\`\`\`
EOF
    else
        cat >> "$temp_dir/README.md" << EOF
\`\`\`bash
# Server
./server-$platform --config configs/server.yaml

# Agent (root权限)
sudo ./agent-$platform --config configs/agent.yaml
\`\`\`
EOF
    fi

    cat >> "$temp_dir/README.md" << EOF

### 3. 验证运行

\`\`\`bash
# 检查Server
curl http://localhost:8080/health

# 查看监控指标
curl http://localhost:8080/metrics
\`\`\`

## 配置文件

- \`configs/server.yaml\` - Server配置
- \`configs/agent.yaml\` - Agent配置
EOF

    if [ "$goos" = "darwin" ]; then
        cat >> "$temp_dir/README.md" << EOF
- \`configs/agent-macos.yaml\` - macOS优化配置
EOF
    fi

    cat >> "$temp_dir/README.md" << EOF

## 故障排查

1. **Agent权限错误**: 确保以管理员/root权限运行
2. **网络接口错误**: 检查配置文件中的interface设置
3. **依赖缺失**: 确保已安装libpcap运行时库

---
构建信息: $platform - $(date)
版本: $VERSION
EOF
}

# 创建启动脚本
create_server_startup_script() {
    local platform=$1
    local temp_dir=$2
    
    IFS='-' read -r goos goarch <<< "$platform"
    
    if [ "$goos" = "windows" ]; then
        cat > "$temp_dir/start-server.bat" << EOF
@echo off
echo Starting Go Network Monitoring Server...
server-$platform.exe --config server.yaml
pause
EOF
    else
        cat > "$temp_dir/start-server.sh" << EOF
#!/bin/bash
echo "Starting Go Network Monitoring Server..."
./server-$platform --config server.yaml
EOF
        chmod +x "$temp_dir/start-server.sh"
    fi
}

create_full_startup_scripts() {
    local platform=$1
    local temp_dir=$2
    
    IFS='-' read -r goos goarch <<< "$platform"
    
    if [ "$goos" = "windows" ]; then
        cat > "$temp_dir/start-server.bat" << EOF
@echo off
echo Starting Go Network Monitoring Server...
server-$platform.exe --config configs/server.yaml
pause
EOF
        
        cat > "$temp_dir/start-agent.bat" << EOF
@echo off
echo Starting Go Network Monitoring Agent...
echo Note: This requires Administrator privileges
agent-$platform.exe --config configs/agent.yaml
pause
EOF
    else
        cat > "$temp_dir/start-server.sh" << EOF
#!/bin/bash
echo "Starting Go Network Monitoring Server..."
./server-$platform --config configs/server.yaml
EOF
        chmod +x "$temp_dir/start-server.sh"
        
        cat > "$temp_dir/start-agent.sh" << EOF
#!/bin/bash
echo "Starting Go Network Monitoring Agent..."
echo "Note: This requires root privileges"
sudo ./agent-$platform --config configs/agent.yaml
EOF
        chmod +x "$temp_dir/start-agent.sh"
    fi
}

# 显示构建结果
show_results() {
    log_info "构建结果:"
    
    echo "二进制文件:"
    ls -lh bin/ 2>/dev/null | grep -E "(agent|server)" | awk '{print "  " $9 " (" $5 ")"}'
    
    echo ""
    echo "发布包:"
    ls -lh dist/ 2>/dev/null | grep -E "\.(tar\.gz|zip)$" | awk '{print "  " $9 " (" $5 ")"}'
    
    echo ""
    log_info "分发说明:"
    echo "  📦 Server发布包: 可在任何平台运行，无需额外依赖"
    echo "  🔧 完整发布包: 包含预编译的Agent，需要运行时依赖"
    echo "  🏗️  Agent构建: 其他平台需要在目标系统上源码构建"
    echo ""
    echo "  推荐分发策略:"
    echo "  1. 分发Server发布包到各个平台"
    echo "  2. 在目标节点上源码构建Agent"
    echo "  3. 或使用Docker方式部署Agent"
}

# 主函数
main() {
    log_info "开始构建发布包..."
    
    get_version_info
    
    # 清理旧的构建产物
    rm -rf bin dist
    mkdir -p bin dist
    
    # 构建Server (所有平台)
    build_servers
    
    # 构建Agent (所有平台，尽力而为)
    build_agents
    
    # 创建发布包
    create_release_packages
    
    show_results
    log_success "发布包构建完成！"
}

# 运行主函数
main "$@"
