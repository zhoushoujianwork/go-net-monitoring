# WebInstall.dev 集成指南

## 🚀 一键安装

### 快速安装

**安装 Agent (网络监控代理):**
```bash
curl -sS https://raw.githubusercontent.com/your-username/go-net-monitoring/main/scripts/quick-install.sh | bash -s agent
```

**安装 Server (数据聚合服务器):**
```bash
curl -sS https://raw.githubusercontent.com/your-username/go-net-monitoring/main/scripts/quick-install.sh | bash -s server
```

**交互式安装:**
```bash
curl -sS https://raw.githubusercontent.com/your-username/go-net-monitoring/main/scripts/quick-install.sh | bash
```

### 通过 webinstall.dev

**安装 Agent:**
```bash
curl -sS https://webinstall.dev/go-net-monitoring-agent | bash
```

**安装 Server:**
```bash
curl -sS https://webinstall.dev/go-net-monitoring-server | bash
```

## 📦 支持的平台

- **Linux**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)

## 🔧 安装位置

- **二进制文件**: `~/.local/bin/`
- **配置文件**: `~/.local/opt/go-net-monitoring/configs/`
- **文档**: `~/.local/opt/go-net-monitoring/`

## 🚀 使用方法

### Agent (网络监控代理)

1. **配置网络接口**:
   ```bash
   # 查看可用网络接口
   ip link show  # Linux
   ifconfig      # macOS
   
   # 编辑配置文件
   nano ~/.local/opt/go-net-monitoring/configs/agent.yaml
   ```

2. **启动 Agent** (需要root权限):
   ```bash
   sudo agent --config ~/.local/opt/go-net-monitoring/configs/agent.yaml
   ```

3. **查看帮助**:
   ```bash
   agent --help
   ```

### Server (数据聚合服务器)

1. **启动 Server**:
   ```bash
   server --config ~/.local/opt/go-net-monitoring/configs/server.yaml
   ```

2. **查看监控指标**:
   ```bash
   curl http://localhost:8080/metrics
   ```

3. **查看帮助**:
   ```bash
   server --help
   ```

## 🔍 验证安装

```bash
# 检查版本
agent --version
server --version

# 检查文件位置
ls -la ~/.local/bin/agent ~/.local/bin/server
ls -la ~/.local/opt/go-net-monitoring/
```

## 🛠️ 故障排除

### 1. 权限问题

如果遇到权限错误：
```bash
# 确保二进制文件有执行权限
chmod +x ~/.local/bin/agent ~/.local/bin/server

# Agent需要root权限
sudo agent --config ~/.local/opt/go-net-monitoring/configs/agent.yaml
```

### 2. 网络接口配置

```bash
# 查看当前网络接口
ip addr show    # Linux
ifconfig -a     # macOS

# 编辑配置文件，修改interface字段
nano ~/.local/opt/go-net-monitoring/configs/agent.yaml
```

### 3. 依赖问题

**libpcap 未安装:**
```bash
# Ubuntu/Debian
sudo apt-get install libpcap-dev

# CentOS/RHEL
sudo yum install libpcap-devel

# macOS
brew install libpcap
```

### 4. 端口占用

如果8080端口被占用：
```bash
# 查看端口使用情况
lsof -i :8080

# 修改server配置文件中的端口
nano ~/.local/opt/go-net-monitoring/configs/server.yaml
```

## 🔄 更新

重新运行安装命令即可更新到最新版本：
```bash
curl -sS https://raw.githubusercontent.com/your-username/go-net-monitoring/main/scripts/quick-install.sh | bash -s agent
```

## 🗑️ 卸载

```bash
# 删除二进制文件
rm -f ~/.local/bin/agent ~/.local/bin/server

# 删除配置和数据
rm -rf ~/.local/opt/go-net-monitoring
```

## 📋 配置示例

### Agent 配置 (agent.yaml)

```yaml
server:
  host: "localhost"
  port: 8080

monitor:
  interface: "eth0"  # 修改为你的网络接口
  protocols:
    - "tcp"
    - "udp"
    - "http"
    - "https"
    - "dns"
  report_interval: "10s"
  buffer_size: 1000
  filters:
    ignore_localhost: true
    ignore_ports:
      - 22    # SSH
      - 123   # NTP
    ignore_ips:
      - "127.0.0.1"
      - "::1"

reporter:
  server_url: "http://localhost:8080/api/v1/metrics"
  timeout: "10s"
  retry_count: 3
  batch_size: 100

log:
  level: "info"
  format: "json"
  output: "stdout"
```

### Server 配置 (server.yaml)

```yaml
server:
  host: "0.0.0.0"
  port: 8080

storage:
  type: "memory"
  retention: "24h"

log:
  level: "info"
  format: "json"
  output: "stdout"
```

## 🌐 集成 Prometheus + Grafana

1. **配置 Prometheus** (`prometheus.yml`):
   ```yaml
   scrape_configs:
     - job_name: 'go-net-monitoring'
       static_configs:
         - targets: ['localhost:8080']
   ```

2. **Grafana 查询示例**:
   ```promql
   # 域名访问Top10
   topk(10, network_domains_accessed_total)
   
   # 域名流量Top10
   topk(10, network_domain_bytes_sent_total)
   
   # 实时连接数
   rate(network_connections_total[5m])
   ```

## 📞 支持

- **文档**: https://github.com/your-username/go-net-monitoring
- **问题反馈**: https://github.com/your-username/go-net-monitoring/issues
- **功能请求**: https://github.com/your-username/go-net-monitoring/discussions
