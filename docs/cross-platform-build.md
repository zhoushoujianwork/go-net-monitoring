# 跨平台构建说明

## 🎯 构建策略

### Server (无CGO依赖)
- ✅ **完全跨平台构建**
- ✅ **无需额外依赖**
- ✅ **开箱即用**

### Agent (CGO依赖)
- ✅ **Windows**: 纯Go实现，可交叉编译
- ⚠️ **Unix/Linux/macOS**: 需要libpcap开发库

## 🔧 技术原理

### 为什么Windows Agent可以交叉编译？

**gopacket库的平台差异：**

1. **Unix/Linux/macOS** (`pcap_unix.go`)
   ```go
   /*
   #cgo linux LDFLAGS: -lpcap
   #cgo darwin LDFLAGS: -lpcap
   #include <pcap.h>
   */
   import "C"
   ```
   - 使用CGO调用libpcap C库
   - 编译时必须有`pcap.h`和`libpcap.so`

2. **Windows** (`pcap_windows.go`)
   ```go
   // 纯Go实现，使用Windows API
   // 无CGO依赖
   ```
   - 使用纯Go实现或Windows API
   - 可以交叉编译
   - 运行时需要WinPcap/Npcap

### 依赖对比

| 平台 | 编译时依赖 | 运行时依赖 | 交叉编译 |
|------|------------|------------|----------|
| Windows | 无 | WinPcap/Npcap | ✅ 可以 |
| Linux | libpcap-dev | libpcap0.8 | ❌ 需要交叉编译环境 |
| macOS | libpcap | libpcap | ❌ 需要交叉编译环境 |

## 📦 构建结果

### 当前构建产物

```bash
make build-release
```

**生成文件：**
```
bin/
├── server-linux-amd64          # ✅ 跨平台构建
├── server-linux-arm64          # ✅ 跨平台构建  
├── server-darwin-amd64          # ✅ 跨平台构建
├── server-darwin-arm64          # ✅ 跨平台构建
├── server-windows-amd64.exe     # ✅ 跨平台构建
├── agent-windows-amd64.exe      # ✅ Windows特殊实现
├── agent-linux-*.build-required # ⚠️ 需要目标平台构建
└── agent-darwin-*.build-required# ⚠️ 需要目标平台构建

dist/
├── go-net-monitoring-server-*.tar.gz    # Server发布包
└── go-net-monitoring-full-windows-*.zip # Windows完整包
```

## 🚀 分发策略

### 1. Server分发 (推荐)
```bash
# 所有平台都有预编译Server
tar -xzf go-net-monitoring-server-linux-amd64.tar.gz
./start-server.sh
```

### 2. Windows完整包
```bash
# Windows有完整的Agent+Server包
unzip go-net-monitoring-full-windows-amd64.zip
start-server.bat  # 启动Server
start-agent.bat   # 启动Agent (需要管理员权限)
```

### 3. Unix/Linux/macOS Agent构建
```bash
# 在目标平台构建Agent
git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
cd go-net-monitoring

# 安装依赖
sudo apt-get install libpcap-dev  # Ubuntu/Debian
brew install libpcap              # macOS

# 构建Agent
make build-agent
sudo ./agent --config configs/agent.yaml
```

## 🔄 替代方案

### 1. Docker方式 (推荐)
```bash
# Agent使用Docker，避免构建问题
docker run -d \
  --name netmon-agent \
  --privileged \
  --network host \
  -e COMPONENT=agent \
  -e SERVER_URL=http://your-server:8080/api/v1/metrics \
  zhoushoujian/go-net-monitoring:latest
```

### 2. CI/CD构建
```yaml
# GitHub Actions示例
- name: Build Linux Agent
  runs-on: ubuntu-latest
  steps:
    - run: sudo apt-get install libpcap-dev
    - run: make build-agent

- name: Build macOS Agent  
  runs-on: macos-latest
  steps:
    - run: brew install libpcap
    - run: make build-agent
```

### 3. 构建服务器
- 在各个平台设置构建环境
- 定期构建并分发Agent二进制文件

## 📋 最佳实践

### 开发者
```bash
# 构建所有平台发布包
make build-release

# 分发策略:
# 1. Server发布包 -> 所有平台
# 2. Windows完整包 -> Windows用户  
# 3. 构建指南 -> Unix/Linux/macOS用户
```

### 用户
```bash
# 1. 下载对应平台Server包
# 2. 启动Server
# 3. 根据平台选择Agent部署方式:
#    - Windows: 使用完整包
#    - 其他: 源码构建或Docker
```

## 🔍 故障排查

### 编译错误
```bash
# 错误: pcap.h: No such file or directory
# 解决: 安装libpcap开发库
sudo apt-get install libpcap-dev  # Ubuntu/Debian
sudo yum install libpcap-devel    # CentOS/RHEL
brew install libpcap              # macOS
```

### 运行时错误
```bash
# 错误: libpcap.so.0.8: cannot open shared object file
# 解决: 安装libpcap运行时库
sudo apt-get install libpcap0.8  # Ubuntu/Debian
sudo yum install libpcap          # CentOS/RHEL
```

### Windows运行错误
```bash
# 错误: 无法找到网络适配器
# 解决: 安装Npcap
# 下载: https://npcap.com/
```

## 📊 总结

**当前策略是最实用的解决方案：**

✅ **优点:**
- Server完全跨平台，无依赖
- Windows Agent可预编译
- 其他平台提供详细构建指南
- Docker作为通用替代方案

⚠️ **限制:**
- Unix/Linux/macOS Agent需要目标平台构建
- 这是CGO和libpcap的技术限制，无法避免

🎯 **结论:**
这种混合策略平衡了技术限制和用户体验，是目前最实用的跨平台分发方案。
