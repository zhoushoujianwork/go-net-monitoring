# 网络流量监控Agent

一个用于监控主机网络流量的Go语言Agent，支持域名和IP地址访问监控，并通过HTTP接口上报到配套的Server。

## 项目结构

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
├── scripts/            # 部署脚本
├── docs/              # 文档
└── examples/          # 示例配置
```

## 快速开始

### 构建

```bash
# 构建Agent
make build-agent

# 构建Server
make build-server

# 构建全部
make build
```

### 运行

```bash
# 运行Agent
./bin/agent --config configs/agent.yaml

# 运行Server
./bin/server --config configs/server.yaml
```

## 功能特性

- 实时网络流量监控
- 域名和IP地址访问记录
- 支持多种协议（TCP、UDP、HTTP/HTTPS）
- Prometheus指标暴露
- 可配置的上报间隔
- 支持过滤规则
- 高性能数据收集

## 监控指标

- `network_connections_total`: 网络连接总数
- `network_bytes_sent_total`: 发送字节总数
- `network_bytes_received_total`: 接收字节总数
- `network_domains_accessed_total`: 访问域名总数
- `network_ips_accessed_total`: 访问IP总数

## 配置说明

详见 [配置文档](docs/configuration.md)

## 部署指南

详见 [部署文档](docs/deployment.md)
