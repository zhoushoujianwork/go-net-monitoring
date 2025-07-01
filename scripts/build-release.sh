#!/bin/bash

set -e

VERSION=${1:-"v1.0.0"}
BUILD_DIR="dist"
PROJECT_NAME="go-net-monitoring"

echo "🚀 构建 ${PROJECT_NAME} ${VERSION} 发布版本..."

# 清理构建目录
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

# 获取版本信息
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# 构建标志
LDFLAGS="-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT} -s -w"

# 支持的平台
PLATFORMS=(
    "linux/amd64"
    "linux/arm64" 
    "darwin/amd64"
    "darwin/arm64"
)

echo "📦 开始构建多平台二进制文件..."

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    echo "  构建 ${GOOS}/${GOARCH}..."
    
    # 设置CGO环境变量
    export CGO_ENABLED=1
    if [ "$GOOS" != "$(go env GOOS)" ] || [ "$GOARCH" != "$(go env GOARCH)" ]; then
        # 交叉编译时禁用CGO（需要预编译的libpcap）
        export CGO_ENABLED=0
        echo "    注意: 交叉编译时禁用CGO，某些功能可能受限"
    fi
    
    # 构建 agent
    output_name="agent"
    if [ "$GOOS" = "windows" ]; then
        output_name="agent.exe"
    fi
    
    mkdir -p "${BUILD_DIR}/${PROJECT_NAME}-agent-${VERSION}-${GOOS}-${GOARCH}"
    
    env GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=$CGO_ENABLED go build \
        -ldflags "$LDFLAGS" \
        -o "${BUILD_DIR}/${PROJECT_NAME}-agent-${VERSION}-${GOOS}-${GOARCH}/${output_name}" \
        ./cmd/agent
    
    # 构建 server (server不需要libpcap)
    output_name="server"
    if [ "$GOOS" = "windows" ]; then
        output_name="server.exe"
    fi
    
    mkdir -p "${BUILD_DIR}/${PROJECT_NAME}-server-${VERSION}-${GOOS}-${GOARCH}"
    
    env GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build \
        -ldflags "$LDFLAGS" \
        -o "${BUILD_DIR}/${PROJECT_NAME}-server-${VERSION}-${GOOS}-${GOARCH}/${output_name}" \
        ./cmd/server
    
    # 复制配置文件
    cp configs/agent.yaml "${BUILD_DIR}/${PROJECT_NAME}-agent-${VERSION}-${GOOS}-${GOARCH}/configs/"
    cp configs/server.yaml "${BUILD_DIR}/${PROJECT_NAME}-server-${VERSION}-${GOOS}-${GOARCH}/configs/"
    
    # 复制文档
    cp README.md "${BUILD_DIR}/${PROJECT_NAME}-agent-${VERSION}-${GOOS}-${GOARCH}/"
    cp LICENSE "${BUILD_DIR}/${PROJECT_NAME}-agent-${VERSION}-${GOOS}-${GOARCH}/"
    cp README.md "${BUILD_DIR}/${PROJECT_NAME}-server-${VERSION}-${GOOS}-${GOARCH}/"
    cp LICENSE "${BUILD_DIR}/${PROJECT_NAME}-server-${VERSION}-${GOOS}-${GOARCH}/"
    
    # 创建安装说明
    cat > "${BUILD_DIR}/${PROJECT_NAME}-agent-${VERSION}-${GOOS}-${GOARCH}/INSTALL.md" << EOF
# Go Network Monitoring Agent ${VERSION}

## 安装说明

1. 将 agent 二进制文件复制到系统PATH中：
   \`\`\`bash
   sudo cp agent /usr/local/bin/
   sudo chmod +x /usr/local/bin/agent
   \`\`\`

2. 复制配置文件：
   \`\`\`bash
   sudo mkdir -p /etc/go-net-monitoring
   sudo cp configs/agent.yaml /etc/go-net-monitoring/
   \`\`\`

3. 根据需要修改配置文件中的网络接口设置

4. 启动agent（需要root权限）：
   \`\`\`bash
   sudo agent --config /etc/go-net-monitoring/agent.yaml
   \`\`\`

## 系统要求

- Linux/macOS 系统
- Root权限（用于网络监控）
- libpcap库（如果使用CGO版本）

## 安装libpcap

**Ubuntu/Debian:**
\`\`\`bash
sudo apt-get install libpcap-dev
\`\`\`

**CentOS/RHEL:**
\`\`\`bash
sudo yum install libpcap-devel
\`\`\`

**macOS:**
\`\`\`bash
brew install libpcap
\`\`\`
EOF

    cat > "${BUILD_DIR}/${PROJECT_NAME}-server-${VERSION}-${GOOS}-${GOARCH}/INSTALL.md" << EOF
# Go Network Monitoring Server ${VERSION}

## 安装说明

1. 将 server 二进制文件复制到系统PATH中：
   \`\`\`bash
   sudo cp server /usr/local/bin/
   sudo chmod +x /usr/local/bin/server
   \`\`\`

2. 复制配置文件：
   \`\`\`bash
   sudo mkdir -p /etc/go-net-monitoring
   sudo cp configs/server.yaml /etc/go-net-monitoring/
   \`\`\`

3. 启动server：
   \`\`\`bash
   server --config /etc/go-net-monitoring/server.yaml
   \`\`\`

4. 访问Prometheus指标：
   \`\`\`bash
   curl http://localhost:8080/metrics
   \`\`\`

## 系统要求

- Linux/macOS 系统
- 无特殊依赖
EOF
    
    # 创建压缩包
    cd ${BUILD_DIR}
    tar -czf "${PROJECT_NAME}-agent-${VERSION}-${GOOS}-${GOARCH}.tar.gz" "${PROJECT_NAME}-agent-${VERSION}-${GOOS}-${GOARCH}"
    tar -czf "${PROJECT_NAME}-server-${VERSION}-${GOOS}-${GOARCH}.tar.gz" "${PROJECT_NAME}-server-${VERSION}-${GOOS}-${GOARCH}"
    cd ..
    
    echo "  ✅ ${GOOS}/${GOARCH} 构建完成"
done

echo ""
echo "🎉 构建完成！生成的文件："
ls -la ${BUILD_DIR}/*.tar.gz

echo ""
echo "📋 发布清单："
echo "版本: ${VERSION}"
echo "提交: ${GIT_COMMIT}"
echo "构建时间: ${BUILD_TIME}"
echo "平台数量: ${#PLATFORMS[@]}"

echo ""
echo "🚀 下一步："
echo "1. 将 dist/*.tar.gz 上传到 GitHub Releases"
echo "2. 创建 webinstall.dev 安装脚本"
echo "3. 测试安装脚本"
