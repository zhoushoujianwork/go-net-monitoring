# Agent构建指南

## 🎯 概述

由于Agent需要CGO和平台特定的libpcap库，无法进行简单的跨平台构建。本指南说明如何在不同平台上构建Agent。

## 📦 获取Server

Server可以跨平台构建，直接下载对应平台的发布包：

```bash
# 下载对应平台的Server发布包
# go-net-monitoring-server-linux-amd64.tar.gz
# go-net-monitoring-server-darwin-amd64.tar.gz
# go-net-monitoring-server-windows-amd64.zip

# 解压并运行
tar -xzf go-net-monitoring-server-linux-amd64.tar.gz
cd go-net-monitoring-server-linux-amd64
./start-server.sh
```

## 🔧 Agent构建

### Linux平台

#### Ubuntu/Debian
```bash
# 1. 安装依赖
sudo apt-get update
sudo apt-get install git golang-go libpcap-dev

# 2. 克隆源码
git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
cd go-net-monitoring

# 3. 构建Agent
make build-agent
# 或者
CGO_ENABLED=1 go build -o agent ./cmd/agent

# 4. 运行Agent
sudo ./agent --config configs/agent.yaml
```

#### CentOS/RHEL
```bash
# 1. 安装依赖
sudo yum install git golang libpcap-devel
# 或者 (CentOS 8+)
sudo dnf install git golang libpcap-devel

# 2. 克隆源码
git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
cd go-net-monitoring

# 3. 构建Agent
make build-agent

# 4. 运行Agent
sudo ./agent --config configs/agent.yaml
```

### macOS平台

```bash
# 1. 安装依赖
# 安装Homebrew (如果未安装)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# 安装依赖
brew install git go libpcap

# 2. 克隆源码
git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
cd go-net-monitoring

# 3. 构建Agent
make macos-build
# 或者
CGO_ENABLED=1 go build -o agent ./cmd/agent

# 4. 运行Agent
sudo ./agent --config configs/agent-macos.yaml
```

### Windows平台

#### 使用MSYS2/MinGW-w64
```bash
# 1. 安装MSYS2
# 下载并安装: https://www.msys2.org/

# 2. 在MSYS2终端中安装依赖
pacman -S git mingw-w64-x86_64-go mingw-w64-x86_64-libpcap

# 3. 安装Npcap
# 下载并安装: https://npcap.com/

# 4. 克隆源码
git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
cd go-net-monitoring

# 5. 构建Agent
CGO_ENABLED=1 go build -o agent.exe ./cmd/agent

# 6. 以管理员身份运行Agent
# 右键"以管理员身份运行"命令提示符
agent.exe --config configs/agent.yaml
```

#### 使用Visual Studio
```bash
# 1. 安装Visual Studio Community (包含C++工具)
# 2. 安装Go: https://golang.org/dl/
# 3. 安装Git: https://git-scm.com/
# 4. 安装Npcap: https://npcap.com/

# 5. 在Developer Command Prompt中:
git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
cd go-net-monitoring
set CGO_ENABLED=1
go build -o agent.exe ./cmd/agent

# 6. 以管理员身份运行
agent.exe --config configs/agent.yaml
```

## 🐳 Docker方式 (推荐)

如果构建困难，推荐使用Docker方式：

```bash
# 运行Agent (Docker方式)
docker run -d \
  --name netmon-agent \
  --privileged \
  --network host \
  -e COMPONENT=agent \
  -e SERVER_URL=http://your-server:8080/api/v1/metrics \
  zhoushoujian/go-net-monitoring:latest
```

## 📋 构建验证

构建完成后验证：

```bash
# 1. 检查版本
./agent --version

# 2. 检查配置
./agent --config configs/agent.yaml --help

# 3. 测试运行 (需要root/管理员权限)
sudo ./agent --config configs/agent.yaml --debug
```

## 🔍 故障排查

### 常见问题

#### 1. libpcap未找到
```bash
# Linux
sudo apt-get install libpcap-dev  # Ubuntu/Debian
sudo yum install libpcap-devel    # CentOS/RHEL

# macOS
brew install libpcap

# Windows
# 安装Npcap: https://npcap.com/
```

#### 2. CGO编译错误
```bash
# 确保安装了C编译器
# Linux: gcc
sudo apt-get install build-essential  # Ubuntu/Debian
sudo yum groupinstall "Development Tools"  # CentOS/RHEL

# macOS: Xcode Command Line Tools
xcode-select --install

# Windows: Visual Studio或MinGW-w64
```

#### 3. 权限错误
```bash
# Agent需要管理员权限进行网络监控
# Linux/macOS
sudo ./agent --config configs/agent.yaml

# Windows
# 右键"以管理员身份运行"
```

#### 4. 网络接口错误
```bash
# 检查可用网络接口
# Linux/macOS
ip link show
ifconfig

# Windows
ipconfig /all

# 然后修改配置文件中的interface字段
```

## 🎯 推荐部署策略

### 1. 混合部署
- **Server**: 使用预编译发布包
- **Agent**: 在目标节点源码构建

### 2. Docker部署
- **Server**: Docker容器
- **Agent**: Docker容器 (推荐)

### 3. 完全源码部署
- 所有组件都在目标环境源码构建
- 适合高安全要求环境

## 📞 获取帮助

如果遇到构建问题：

1. 查看项目文档: https://github.com/zhoushoujianwork/go-net-monitoring
2. 提交Issue: https://github.com/zhoushoujianwork/go-net-monitoring/issues
3. 使用Docker方式作为替代方案

---

**注意**: Agent需要管理员权限进行网络监控，这是正常的安全要求。
