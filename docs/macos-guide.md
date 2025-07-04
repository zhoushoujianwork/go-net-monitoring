# macOS使用指南

## 🍎 概述

本指南详细说明如何在macOS系统上构建、配置和运行go-net-monitoring项目，包括Intel和Apple Silicon (M1/M2)处理器的支持。

## 🔧 环境准备

### 1. 系统要求

- **macOS版本**: 10.15+ (推荐macOS 12+)
- **处理器**: Intel x64 或 Apple Silicon (M1/M2)
- **Go版本**: 1.19+
- **权限**: Agent需要管理员权限进行网络监控

### 2. 依赖安装

#### 自动安装 (推荐)
```bash
# 一键设置macOS环境
make macos-setup
```

#### 手动安装
```bash
# 1. 安装Homebrew (如果未安装)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# 2. 安装依赖
brew install libpcap go

# 3. 验证安装
go version
brew list libpcap
```

## 🏗️ 构建

### 1. 自动构建 (推荐)

```bash
# 自动检测架构并构建
make macos-build

# 或者构建当前平台
make build-current
```

### 2. 手动构建

#### Apple Silicon (M1/M2)
```bash
make build-darwin-arm64
```

#### Intel处理器
```bash
make build-darwin-amd64
```

#### 构建所有macOS版本
```bash
make build-darwin
```

### 3. 验证构建

```bash
# 检查构建产物
ls -la bin/

# 验证二进制文件
./bin/agent --version
./bin/server --version
```

## 🚀 运行

### 1. 运行Server

```bash
# 方式1: Make命令 (推荐)
make macos-run-server

# 方式2: 直接运行
./bin/server --config configs/server.yaml --debug
```

### 2. 运行Agent

#### 使用Make命令 (推荐)
```bash
# 自动处理sudo权限
make macos-run-agent
```

#### 手动运行
```bash
# 需要sudo权限进行网络监控
sudo ./bin/agent --config configs/agent-macos.yaml --debug
```

### 3. 使用专用配置

```bash
# 使用macOS优化配置
sudo ./bin/agent --config configs/agent-macos.yaml --debug
```

## ⚙️ 配置

### 1. macOS特定配置

Agent在macOS上的特殊配置 (`configs/agent-macos.yaml`):

```yaml
monitor:
  interface: "en0"                 # macOS主网络接口
  filters:
    ignore_ports:
      - 5353                      # mDNS (Bonjour)
    ignore_ips:
      - "169.254.0.0/16"          # Link-local
      - "224.0.0.0/4"             # Multicast

macos:
  auto_detect_interface: true      # 自动检测网络接口
  preferred_interfaces:
    - "en0"                       # WiFi/以太网
    - "en1"                       # 备用接口
    - "utun0"                     # VPN接口
```

### 2. 网络接口检测

```bash
# 查看网络接口
ifconfig | grep "^[a-z]"

# 常见接口:
# en0  - 主WiFi/以太网
# en1  - 备用网络接口
# lo0  - 回环接口
# utun0 - VPN接口
```

### 3. 权限配置

```bash
# 检查当前用户权限
whoami

# Agent需要root权限访问网络接口
sudo ./bin/agent --config configs/agent-macos.yaml
```

## 🔍 故障排查

### 1. 常见问题

#### libpcap未找到
```bash
# 错误: fatal error: pcap.h: No such file or directory
# 解决:
brew install libpcap

# 如果仍有问题，设置环境变量:
export CGO_CFLAGS="-I$(brew --prefix libpcap)/include"
export CGO_LDFLAGS="-L$(brew --prefix libpcap)/lib"
```

#### 权限被拒绝
```bash
# 错误: permission denied
# 解决: 使用sudo运行Agent
sudo ./bin/agent --config configs/agent-macos.yaml
```

#### 网络接口未找到
```bash
# 错误: interface not found
# 解决: 检查并修改配置文件中的interface设置
ifconfig | grep "^en"
# 然后修改configs/agent-macos.yaml中的interface字段
```

### 2. 调试技巧

#### 启用详细日志
```bash
# 使用debug模式
sudo ./bin/agent --config configs/agent-macos.yaml --debug
```

#### 检查网络流量
```bash
# 使用tcpdump验证网络监控
sudo tcpdump -i en0 -c 10

# 检查Agent是否正常捕获
sudo ./bin/agent --config configs/agent-macos.yaml --debug | head -20
```

#### 验证Server连接
```bash
# 检查Server是否运行
curl http://localhost:8080/health

# 检查指标端点
curl http://localhost:8080/metrics
```

## 📊 性能优化

### 1. macOS特定优化

```yaml
# configs/agent-macos.yaml
macos:
  use_bpf: true                   # 使用BPF提高性能
  capture_timeout: "1s"           # 优化捕获超时
  
monitor:
  buffer_size: 2000               # 增大缓冲区 (macOS内存充足)
  report_interval: "15s"          # 适当增加上报间隔
```

### 2. 系统资源监控

```bash
# 监控Agent资源使用
top -pid $(pgrep agent)

# 监控网络使用
nettop -p agent
```

## 🔄 开发工作流

### 1. 日常开发

```bash
# 1. 设置环境
make macos-setup

# 2. 构建
make macos-build

# 3. 运行测试
make macos-run-server    # 终端1
make macos-run-agent     # 终端2

# 4. 验证
curl http://localhost:8080/metrics
```

### 2. 调试流程

```bash
# 1. 启用debug模式
sudo ./bin/agent --config configs/agent-macos.yaml --debug

# 2. 查看详细日志
# 观察网络接口检测、包捕获、数据处理等过程

# 3. 验证数据上报
curl http://localhost:8080/api/v1/metrics
```

## 📦 分发

### 1. 构建发布版本

```bash
# 构建所有macOS版本
make build-darwin

# 检查构建产物
ls -la dist/
# go-net-monitoring-darwin-amd64.tar.gz
# go-net-monitoring-darwin-arm64.tar.gz
```

### 2. 安装到系统

```bash
# 复制到系统路径
sudo cp bin/agent /usr/local/bin/go-net-monitoring-agent
sudo cp bin/server /usr/local/bin/go-net-monitoring-server

# 创建配置目录
sudo mkdir -p /usr/local/etc/go-net-monitoring
sudo cp configs/agent-macos.yaml /usr/local/etc/go-net-monitoring/
sudo cp configs/server.yaml /usr/local/etc/go-net-monitoring/
```

## 🎯 最佳实践

### 1. 开发环境
- 使用`make macos-setup`一键设置环境
- 使用专用的`agent-macos.yaml`配置
- 启用debug模式进行开发调试

### 2. 生产环境
- 使用发布版本的二进制文件
- 配置适当的日志级别 (info)
- 设置系统服务自动启动

### 3. 安全考虑
- Agent需要root权限，注意安全风险
- 定期更新依赖和系统
- 监控异常网络活动

## 📋 快速参考

### 常用命令
```bash
# 环境设置
make macos-setup

# 构建
make macos-build

# 运行
make macos-run-server
make macos-run-agent

# 清理
make clean
```

### 配置文件
- `configs/agent-macos.yaml` - macOS专用Agent配置
- `configs/server.yaml` - Server配置

### 日志位置
- Agent: stdout (可重定向到文件)
- Server: stdout (可重定向到文件)

这个指南涵盖了在macOS上使用go-net-monitoring的所有方面，从环境准备到生产部署。
