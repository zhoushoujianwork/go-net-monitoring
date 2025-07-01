# 网络流量监控系统

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.19+-blue.svg)](https://golang.org)

一个用Go语言开发的高性能网络流量监控系统，支持实时监控主机网络流量，包括域名访问统计、流量分析和Prometheus指标导出。

## ✨ 主要特性

- 🚀 **实时网络监控** - 基于BPF的高性能数据包捕获
- 🌐 **域名解析** - 自动解析IP地址到域名，支持DNS缓存
- 📊 **流量统计** - 按域名统计访问次数、发送/接收字节数、连接数
- 🎯 **智能过滤** - 支持端口、IP、协议等多维度过滤
- 📈 **Prometheus集成** - 内置Prometheus指标导出
- 🔧 **灵活配置** - 支持YAML配置文件，可自定义监控规则
- 🏗️ **分布式架构** - Agent/Server架构，支持多节点部署

## 🏗️ 系统架构

```
┌─────────────┐    HTTP API    ┌─────────────┐    Prometheus    ┌─────────────┐
│   Agent     │ ──────────────► │   Server    │ ──────────────► │  Grafana    │
│             │                │             │                │             │
│ - 数据采集   │                │ - 数据聚合   │                │ - 数据可视化 │
│ - DNS解析   │                │ - 指标导出   │                │ - 告警监控   │
│ - 流量过滤   │                │ - API服务   │                │             │
└─────────────┘                └─────────────┘                └─────────────┘
```

## 📊 监控指标

### 域名相关指标
- `network_domains_accessed_total` - 域名访问次数统计
- `network_domain_bytes_sent_total` - 按域名统计发送字节数
- `network_domain_bytes_received_total` - 按域名统计接收字节数
- `network_domain_connections_total` - 按域名统计连接数

### 网络基础指标
- `network_connections_total` - 网络连接总数
- `network_bytes_sent_total` - 发送字节总数
- `network_bytes_received_total` - 接收字节总数
- `network_protocol_stats` - 协议统计
- `network_ips_accessed_total` - IP访问统计

## 🚀 快速开始

### 一键安装 (推荐)

**安装 Agent (网络监控代理):**
```bash
curl -sS https://raw.githubusercontent.com/your-username/go-net-monitoring/main/scripts/quick-install.sh | bash -s agent
```

**安装 Server (数据聚合服务器):**
```bash
curl -sS https://raw.githubusercontent.com/your-username/go-net-monitoring/main/scripts/quick-install.sh | bash -s server
```

**交互式安装 (选择组件):**
```bash
curl -sS https://raw.githubusercontent.com/your-username/go-net-monitoring/main/scripts/quick-install.sh | bash
```

### 通过 webinstall.dev 安装

**安装 Agent:**
```bash
curl -sS https://webinstall.dev/go-net-monitoring-agent | bash
```

**安装 Server:**
```bash
curl -sS https://webinstall.dev/go-net-monitoring-server | bash
```

### 环境要求

- Go 1.19+
- Linux/macOS (需要root权限进行网络监控)
- libpcap开发库

### 安装依赖

**Ubuntu/Debian:**
```bash
sudo apt-get update
sudo apt-get install libpcap-dev
```

**CentOS/RHEL:**
```bash
sudo yum install libpcap-devel
```

**macOS:**
```bash
brew install libpcap
```

### 编译安装

```bash
# 克隆项目
git clone https://github.com/your-username/go-net-monitoring.git
cd go-net-monitoring

# 编译
make build

# 或者分别编译
make build-agent  # 编译Agent
make build-server # 编译Server
```

### 配置文件

**Agent配置 (configs/agent.yaml):**
```yaml
server:
  host: "localhost"
  port: 8080

monitor:
  interface: "en0"  # 网络接口
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

**Server配置 (configs/server.yaml):**
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

### 运行

```bash
# 启动Server
./bin/server --config configs/server.yaml

# 启动Agent (需要root权限)
sudo ./bin/agent --config configs/agent.yaml
```

### 查看指标

```bash
# 查看Prometheus指标
curl http://localhost:8080/metrics

# 查看域名访问统计
curl http://localhost:8080/metrics | grep network_domains_accessed_total

# 查看域名流量统计
curl http://localhost:8080/metrics | grep network_domain_bytes
```

## 📈 Grafana集成

1. 添加Prometheus数据源：`http://localhost:8080`
2. 导入示例Dashboard配置
3. 创建自定义面板监控域名流量

### 示例查询

```promql
# 域名访问Top10
topk(10, network_domains_accessed_total)

# 域名流量Top10
topk(10, network_domain_bytes_sent_total)

# 实时连接数
rate(network_connections_total[5m])
```

## 🔧 高级配置

### 过滤规则

```yaml
filters:
  ignore_localhost: true
  ignore_ports:
    - 22    # SSH
    - 80    # HTTP
    - 443   # HTTPS
  ignore_ips:
    - "127.0.0.1"
    - "192.168.1.1"
  only_domains:
    - "example.com"
    - "api.example.com"
```

### 性能调优

```yaml
monitor:
  buffer_size: 10000      # 增大缓冲区
  report_interval: "30s"  # 调整上报间隔
  
reporter:
  batch_size: 1000        # 批量上报大小
  timeout: "30s"          # 超时时间
```

## 🛠️ 开发

### 项目结构

```
go-net-monitoring/
├── cmd/
│   ├── agent/          # Agent主程序
│   └── server/         # Server主程序
├── internal/
│   ├── agent/          # Agent核心逻辑
│   ├── server/         # Server核心逻辑
│   ├── common/         # 公共组件
│   └── config/         # 配置管理
├── pkg/
│   ├── collector/      # 网络流量收集器
│   ├── reporter/       # 数据上报器
│   └── metrics/        # Prometheus指标
├── configs/            # 配置文件
└── docs/              # 文档
```

### 构建命令

```bash
make build          # 构建所有组件
make build-agent    # 构建Agent
make build-server   # 构建Server
make clean          # 清理构建文件
make test           # 运行测试
```

## 🤝 贡献

欢迎提交Issue和Pull Request！

1. Fork项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开Pull Request

## 📝 许可证

本项目采用MIT许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [gopacket](https://github.com/google/gopacket) - 网络数据包处理
- [Prometheus](https://prometheus.io/) - 监控指标系统
- [logrus](https://github.com/sirupsen/logrus) - 结构化日志
- [cobra](https://github.com/spf13/cobra) - CLI框架

## 📞 支持

如果你遇到问题或有建议，请：

1. 查看 [文档](docs/)
2. 搜索 [Issues](https://github.com/your-username/go-net-monitoring/issues)
3. 创建新的 [Issue](https://github.com/your-username/go-net-monitoring/issues/new)

---

⭐ 如果这个项目对你有帮助，请给个Star支持一下！
