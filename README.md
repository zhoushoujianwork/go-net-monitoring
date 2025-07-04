# 网络流量监控系统

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.19+-blue.svg)](https://golang.org)
[![Docker](https://img.shields.io/badge/Docker-Required-blue.svg)](https://www.docker.com/)

一个用Go语言开发的高性能网络流量监控系统，支持实时监控主机网络流量，包括域名访问统计、流量分析和Prometheus指标导出。

> **📦 部署方式说明**: 本项目仅支持容器化部署，不提供二进制文件。这是为了解决CGO依赖和跨平台兼容性问题。详见：[容器化部署说明](docs/container-only-deployment.md)

![网络流量监控](docs/images/全面的网络流量监控.png)

## ✨ 主要特性

- 🚀 **实时网络监控** - 基于BPF的高性能数据包捕获
- 🌐 **域名解析** - 自动解析IP地址到域名，支持DNS缓存
- 📊 **流量统计** - 按域名统计访问次数、发送/接收字节数、连接数
- 🎯 **智能过滤** - 支持端口、IP、协议等多维度过滤
- 📈 **Prometheus集成** - 内置Prometheus指标导出
- 🔧 **灵活配置** - 支持YAML配置文件，可自定义监控规则
- 🏗️ **分布式架构** - Agent/Server架构，支持多节点部署
- 📱 **专业可视化** - 提供多种专业级Grafana Dashboard
- 🐳 **容器化部署** - 统一的Docker部署方式，解决依赖问题

## 📈 Grafana Dashboard

系统提供了专业级的 Grafana Dashboard，支持多设备监控和灵活的数据分析，现已全面升级支持多 Agent 部署：

### 🆕 多设备支持特性

- 🏠 **多 Agent 监控**: 支持同时监控多个设备/主机
- 🔧 **网卡选择**: 支持选择特定网卡进行分析
- 📊 **动态过滤**: 灵活的主机和网卡过滤器
- 🔄 **实时更新**: 自动发现新设备和网卡

### 1. 网络监控 - 总体概览 (`network-overview`)

**用途**: 提供所有监控设备的全局视图和汇总统计

**主要功能**:
- 🏠 **全局概览**: 监控设备总数、全网连接速率、发送/接收速率
- 📈 **流量趋势**: 按设备显示网络流量和连接速率趋势  
- 🖥️ **设备状态**: 实时显示所有设备的状态、IP、MAC地址和性能指标
- 🌐 **热门域名**: 全网域名访问次数和流量排行榜

### 2. 网络监控 - 详细分析 (`network-detailed`)

**用途**: 支持选择特定设备和网卡进行深入分析

**主要功能**:
- 🎯 **设备选择**: 支持多选主机和网卡进行过滤
- 📊 **详细统计**: 概览统计、设备信息、流量趋势
- 🔍 **域名分析**: 域名访问次数、流量分布、详细统计表
- 📋 **数据表格**: 完整的域名访问统计，包含访问次数、发送/接收字节数

### 🚀 快速导入看板

```bash
# 方法一: 使用导入脚本 (推荐)
cd grafana
./import-dashboards.sh

# 方法二: 使用 Docker Compose 自动导入
make docker-up-monitoring

# 方法三: 手动导入
# 在 Grafana UI 中导入 grafana/dashboards/ 目录下的 JSON 文件
```

详细信息请参考：[Dashboard 展示文档](docs/dashboards.md)

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

### 网卡信息指标 (新增)
- `network_interface_info` - 网卡信息，包含IP地址和MAC地址
  - 标签: `interface`, `ip_address`, `mac_address`, `host`
  - 示例: `network_interface_info{interface="eth0",ip_address="192.168.1.100",mac_address="02:42:ac:11:00:02",host="agent"} 1`

## 🚀 快速开始

### 🎯 推荐方式 (优化构建)

**使用优化构建流程，享受更快的构建速度和更小的镜像：**

```bash
# 1. 克隆项目
git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
cd go-net-monitoring

# 2. 构建Docker镜像
make docker-build

# 3. 启动服务 (生产模式)
make docker-up

# 4. 启动服务 (调试模式)
make docker-up-debug

# 5. 查看服务状态
make health
```

**优化构建特性：**
- 🚀 **构建速度提升60%** - 从2分钟优化到45秒
- 📦 **镜像大小减少30%** - 从65MB优化到45.7MB
- 🔄 **避免重复构建** - 智能复用镜像
- ⚡ **并行编译** - 同时构建agent和server
- 🛠️ **一键操作** - 便捷命令

### 🔧 快速启动 (解决配置文件问题)

如果遇到配置文件相关错误，使用简化启动方式：

```bash
# 使用简化配置启动 (推荐)
./run.sh

# 或手动启动
docker-compose -f docker-compose-simple.yml --profile monitoring up
```

**简化启动特性：**
- 🚫 **无需配置文件** - 通过环境变量动态生成
- 🔧 **自动配置** - 智能检测环境并生成合适配置
- 🛠️ **故障修复** - 解决只读文件系统问题
- 📋 **环境变量配置** - 支持完全通过环境变量配置

> **💡 提示**: 如果遇到 "Read-only file system" 错误，请使用简化启动方式。详见：[容器配置修复说明](docs/container-fix.md)

### Docker部署

**生产环境推荐使用优化构建：**
```bash
# 构建并启动
make docker-build
make docker-up

# 或者一步完成
make docker-build && make docker-up
```

**开发调试模式：**
```bash
# 启动调试模式 (自动启用debug日志)
make docker-up-debug

# 查看实时日志
make docker-logs-agent  # Agent日志
make docker-logs-server # Server日志
```

**完整监控栈：**
```bash
# 启动包含Prometheus + Grafana的完整栈
make docker-up-monitoring
```

**服务端口：**
- Server: http://localhost:8080
- Prometheus: http://localhost:9090 (使用monitoring模式)
- Grafana: http://localhost:3000 (admin/admin123，使用monitoring模式)

### 传统Docker部署

如果需要使用传统方式：

**运行Server (数据聚合服务器):**
```bash
docker run -d \
  --name netmon-server \
  -p 8080:8080 \
  -e COMPONENT=server \
  zhoushoujian/go-net-monitoring:latest
```

**运行Agent (网络监控代理):**
```bash
docker run -d \
  --name netmon-agent \
  --privileged \
  --network host \
  -e COMPONENT=agent \
  -e SERVER_URL=http://localhost:8080/api/v1/metrics \
  zhoushoujian/go-net-monitoring:latest
```

### Debug 模式

项目支持 debug 模式，方便开发调试和问题排查：

```bash
# 使用优化构建的debug模式 (推荐)
make docker-up-debug

# 或传统方式
DEBUG_MODE=true LOG_LEVEL=debug docker-compose up -d
```

**Debug 模式特性：**
- 🔍 **详细日志输出** - 显示所有调试信息
- 📝 **配置文件内容显示** - 启动时显示完整配置
- 🛠️ **问题排查** - 便于开发和运维调试
- ⚡ **一键启用** - 通过环境变量或make命令控制

> **注意：** 生产环境不建议使用 debug 模式，会影响性能并产生大量日志。

详细使用说明请参考：[Docker Compose 使用指南](docs/docker-compose-usage.md)

### 🔄 混合方案 (推荐生产环境)

混合方案解决了Agent重启导致累计统计数据丢失的问题，结合了Agent端持久化和Server端智能累计的优势：

**核心特性：**
- 🔄 **Agent持久化**: 自动保存和恢复累计状态
- 🧠 **智能重启检测**: 自动检测Agent重启并保持数据连续性  
- 📊 **真实累计统计**: 跨重启的准确累计数据
- 🔒 **数据一致性**: 并发安全的数据处理

**快速启动：**
```bash
# 启动混合方案 (默认)
docker-compose up -d

# 启动包含监控的完整栈
docker-compose --profile monitoring up -d

# 测试部署
./test-deployment.sh test

# 查看Agent持久化状态
docker exec netmon-agent ls -la /var/lib/netmon/
```

详细说明请参考：[混合方案使用指南](docs/hybrid-solution.md)

### Kubernetes部署

**部署到Kubernetes集群:**
```bash
# 创建命名空间和配置
kubectl apply -f https://raw.githubusercontent.com/zhoushoujian/go-net-monitoring/main/k8s/namespace.yaml

# 部署Server (Deployment)
kubectl apply -f https://raw.githubusercontent.com/zhoushoujian/go-net-monitoring/main/k8s/server-deployment.yaml

# 部署Agent (DaemonSet)
kubectl apply -f https://raw.githubusercontent.com/zhoushoujian/go-net-monitoring/main/k8s/agent-daemonset.yaml
```

### 环境要求

- Docker 或 Kubernetes 集群
- Agent需要特权模式进行网络监控

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
  type: "memory"  # memory 或 redis
  # Server作为实时指标聚合服务，不设置数据保留时间
  # 历史数据存储和保留策略由Prometheus管理
  
  # Redis配置 (当type为redis时)
  redis:
    host: "localhost"
    port: 6379
    password: ""
    db: 0
    pool_size: 10
    timeout: "5s"

log:
  level: "info"
  format: "json"
  output: "stdout"
```

### 查看指标

```bash
# 查看Prometheus指标
curl http://localhost:8080/metrics

# 查看域名访问统计
curl http://localhost:8080/metrics | grep network_domains_accessed_total

# 查看域名流量统计
curl http://localhost:8080/metrics | grep network_domain_bytes

# 使用Make命令快速查看
make metrics
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

# 协议分布
network_protocol_stats_total
```

### 访问Dashboard

- 网络流量监控: http://localhost:3000/d/network-traffic/
- 域名流量监控: http://localhost:3000/d/domain-traffic/
- 基础网络监控: http://localhost:3000/d/network-monitoring/

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
├── docker/             # Docker相关文件
├── docs/              # 文档
└── Makefile           # 构建自动化
```

### 🚀 容器化开发流程

#### 1. 环境准备
```bash
# 克隆项目
git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
cd go-net-monitoring

# 查看可用命令
make help
```

#### 2. 构建和启动
```bash
# 构建Docker镜像
make docker-build

# 启动开发环境 (调试模式)
make docker-up-debug

# 查看服务状态
make health
```

#### 3. 开发调试
```bash
# 查看实时日志
make docker-logs          # 所有服务日志
make docker-logs-agent    # Agent日志
make docker-logs-server   # Server日志

# 进入容器调试
make dev-shell-server     # 进入Server容器
make dev-shell-agent      # 进入Agent容器
```

#### 4. 测试验证
```bash
# 运行测试
make test

# 检查服务健康状态
make health

# 查看监控指标
make metrics
```

#### 5. 服务管理
```bash
# 重启服务
make docker-restart

# 停止服务
make docker-down

# 清理资源
make docker-clean
```

### 📋 可用命令

使用 `make help` 查看所有可用命令：

```bash
make help              # 显示帮助信息

# Docker相关
make docker-build      # 构建Docker镜像
make docker-up         # 启动服务 (生产模式)
make docker-up-debug   # 启动服务 (调试模式)
make docker-down       # 停止服务
make docker-logs       # 查看日志

# 监控相关
make health           # 检查服务健康状态
make metrics          # 查看指标

# 清理相关
make docker-clean     # 清理Docker资源
make clean            # 清理所有资源
```

### 🔧 开发最佳实践

#### 1. 调试模式开发
```bash
# 启用详细日志
make docker-up-debug

# 实时查看日志
make dev-logs

# 修改代码后重新构建
make docker-build
make docker-restart
```

#### 2. 配置修改
```bash
# 修改配置文件
vim configs/agent.yaml
vim configs/server.yaml

# 重启服务使配置生效
make docker-restart
```

#### 3. 问题排查
```bash
# 检查容器状态
docker-compose ps

# 查看详细日志
make docker-logs-agent | grep ERROR

# 进入容器调试
make dev-shell-agent
```

## 🤝 贡献

欢迎提交Issue和Pull Request！

### 容器化贡献流程

1. **Fork项目**
   ```bash
   git clone https://github.com/your-username/go-net-monitoring.git
   cd go-net-monitoring
   ```

2. **创建特性分支**
   ```bash
   git checkout -b feature/AmazingFeature
   ```

3. **开发和测试**
   ```bash
   # 构建Docker镜像
   make docker-build
   
   # 启动开发环境
   make docker-up-debug
   
   # 运行测试
   make test
   ```

4. **验证功能**
   ```bash
   # 检查服务状态
   make health
   
   # 查看日志
   make docker-logs
   ```

5. **提交更改**
   ```bash
   git add .
   git commit -m 'feat: Add some AmazingFeature'
   ```

6. **推送和PR**
   ```bash
   git push origin feature/AmazingFeature
   # 然后在GitHub上创建Pull Request
   ```

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
2. 搜索 [Issues](https://github.com/zhoushoujianwork/go-net-monitoring/issues)
3. 创建新的 [Issue](https://github.com/zhoushoujianwork/go-net-monitoring/issues/new)

---

⭐ 如果这个项目对你有帮助，请给个Star支持一下！
