# Go Network Monitoring

[![Docker Pulls](https://img.shields.io/docker/pulls/zhoushoujian/go-net-monitoring)](https://hub.docker.com/r/zhoushoujian/go-net-monitoring)
[![Docker Image Size](https://img.shields.io/docker/image-size/zhoushoujian/go-net-monitoring/latest)](https://hub.docker.com/r/zhoushoujian/go-net-monitoring)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

一个用Go语言开发的高性能网络流量监控系统，支持实时监控主机网络流量，包括域名访问统计、流量分析和Prometheus指标导出。

## 🚀 快速开始

### 运行Server (数据聚合服务器)

```bash
docker run -d \
  --name netmon-server \
  -p 8080:8080 \
  -e COMPONENT=server \
  zhoushoujian/go-net-monitoring:latest
```

### 运行Agent (网络监控代理)

```bash
docker run -d \
  --name netmon-agent \
  --privileged \
  --network host \
  -e COMPONENT=agent \
  -e SERVER_URL=http://localhost:8080/api/v1/metrics \
  zhoushoujian/go-net-monitoring:latest
```

### 使用Docker Compose

```bash
curl -O https://raw.githubusercontent.com/zhoushoujian/go-net-monitoring/main/docker-compose.yml
docker-compose up -d
```

## 📊 监控指标

访问 `http://localhost:8080/metrics` 查看Prometheus指标：

- `network_domains_accessed_total` - 域名访问次数统计
- `network_domain_bytes_sent_total` - 按域名统计发送字节数
- `network_domain_bytes_received_total` - 按域名统计接收字节数
- `network_domain_connections_total` - 按域名统计连接数
- `network_connections_total` - 网络连接总数
- `network_protocol_stats` - 协议统计

## 🔧 环境变量

### 通用环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `COMPONENT` | `server` | 组件类型 (`server` 或 `agent`) |
| `LOG_LEVEL` | `info` | 日志级别 (`debug`, `info`, `warn`, `error`) |
| `CONFIG_FILE` | 自动生成 | 配置文件路径 |

### Server环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `SERVER_HOST` | `0.0.0.0` | 服务器监听地址 |
| `SERVER_PORT` | `8080` | 服务器监听端口 |
| `STORAGE_TYPE` | `memory` | 存储类型 |
| `STORAGE_RETENTION` | `24h` | 数据保留时间 |

### Agent环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `NETWORK_INTERFACE` | `eth0` | 监控的网络接口 |
| `SERVER_URL` | `http://localhost:8080/api/v1/metrics` | Server API地址 |
| `REPORT_INTERVAL` | `10s` | 上报间隔 |
| `BUFFER_SIZE` | `1000` | 缓冲区大小 |
| `IGNORE_LOCALHOST` | `true` | 是否忽略本地流量 |

## ☸️ Kubernetes部署

### 部署Server (Deployment)

```bash
kubectl apply -f https://raw.githubusercontent.com/zhoushoujian/go-net-monitoring/main/k8s/namespace.yaml
kubectl apply -f https://raw.githubusercontent.com/zhoushoujian/go-net-monitoring/main/k8s/server-deployment.yaml
```

### 部署Agent (DaemonSet)

```bash
kubectl apply -f https://raw.githubusercontent.com/zhoushoujian/go-net-monitoring/main/k8s/agent-daemonset.yaml
```

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

## 📈 与Prometheus集成

### Prometheus配置

```yaml
scrape_configs:
  - job_name: 'go-net-monitoring'
    static_configs:
      - targets: ['localhost:8080']
```

### Grafana查询示例

```promql
# 域名访问Top10
topk(10, network_domains_accessed_total)

# 域名流量Top10
topk(10, network_domain_bytes_sent_total)

# 实时连接数
rate(network_connections_total[5m])
```

## 🔒 安全注意事项

- Agent需要特权模式 (`--privileged`) 进行网络监控
- 建议在生产环境中使用最小权限原则
- 定期更新镜像以获取安全补丁

## 🏷️ 支持的标签

- `latest` - 最新稳定版本
- `v1.x.x` - 特定版本
- `main` - 开发版本

## 🏗️ 支持的架构

- `linux/amd64`
- `linux/arm64`

## 📖 文档

- [GitHub仓库](https://github.com/zhoushoujian/go-net-monitoring)
- [配置文档](https://github.com/zhoushoujian/go-net-monitoring/blob/main/docs/configuration.md)
- [部署指南](https://github.com/zhoushoujian/go-net-monitoring/blob/main/docs/deployment.md)

## 🐛 问题反馈

如果遇到问题，请在 [GitHub Issues](https://github.com/zhoushoujian/go-net-monitoring/issues) 中反馈。

## 📄 许可证

本项目采用 [MIT许可证](https://github.com/zhoushoujian/go-net-monitoring/blob/main/LICENSE)。
