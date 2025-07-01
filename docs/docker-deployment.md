# Docker 部署指南

## 🐳 Docker 镜像

我们的Docker镜像托管在Docker Hub上：`zhoushoujian/go-net-monitoring`

支持的架构：
- `linux/amd64`
- `linux/arm64`

## 🚀 快速开始

### 1. 运行Server

```bash
docker run -d \
  --name netmon-server \
  -p 8080:8080 \
  -e COMPONENT=server \
  zhoushoujian/go-net-monitoring:latest
```

### 2. 运行Agent

```bash
docker run -d \
  --name netmon-agent \
  --privileged \
  --network host \
  -e COMPONENT=agent \
  -e SERVER_URL=http://localhost:8080/api/v1/metrics \
  zhoushoujian/go-net-monitoring:latest
```

### 3. 验证部署

```bash
# 检查容器状态
docker ps

# 查看Server日志
docker logs netmon-server

# 查看Agent日志
docker logs netmon-agent

# 访问监控指标
curl http://localhost:8080/metrics
```

## 🔧 环境变量配置

### 通用环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `COMPONENT` | `server` | 组件类型 (`server` 或 `agent`) |
| `LOG_LEVEL` | `info` | 日志级别 (`debug`, `info`, `warn`, `error`) |
| `LOG_FORMAT` | `json` | 日志格式 (`json` 或 `text`) |
| `LOG_OUTPUT` | `stdout` | 日志输出 (`stdout` 或文件路径) |

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
| `RETRY_COUNT` | `3` | 重试次数 |
| `REPORTER_TIMEOUT` | `10s` | 上报超时时间 |
| `BATCH_SIZE` | `100` | 批量上报大小 |

## 📋 Docker Compose 部署

### 1. 下载配置文件

```bash
curl -O https://raw.githubusercontent.com/zhoushoujian/go-net-monitoring/main/docker-compose.yml
```

### 2. 启动服务

```bash
# 启动基础服务 (Server + Agent)
docker-compose up -d

# 启动完整监控栈 (包含Prometheus + Grafana)
docker-compose --profile monitoring up -d
```

### 3. 访问服务

- **网络监控指标**: http://localhost:8080/metrics
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin123)

### 4. 停止服务

```bash
docker-compose down
```

## 🔍 故障排除

### 1. Agent权限问题

**问题**: Agent无法监控网络流量
```bash
# 检查容器是否以特权模式运行
docker inspect netmon-agent | grep -i privileged

# 确保使用了正确的参数
docker run -d --privileged --network host ...
```

### 2. 网络接口配置

**问题**: Agent找不到网络接口
```bash
# 查看容器内可用的网络接口
docker exec netmon-agent ip link show

# 设置正确的网络接口
docker run -d -e NETWORK_INTERFACE=ens33 ...
```

### 3. 连接问题

**问题**: Agent无法连接到Server
```bash
# 检查网络连通性
docker exec netmon-agent wget -O- http://server:8080/health

# 检查Server是否正常运行
curl http://localhost:8080/health
```

### 4. 日志调试

```bash
# 启用调试日志
docker run -d -e LOG_LEVEL=debug ...

# 查看详细日志
docker logs -f netmon-agent
```

## 🏗️ 自定义配置

### 1. 挂载配置文件

```bash
# 创建自定义配置
mkdir -p ./config
cat > ./config/agent.yaml << EOF
monitor:
  interface: "eth0"
  protocols: ["tcp", "udp", "http", "https"]
  report_interval: "5s"
  filters:
    ignore_localhost: true
    ignore_ports: [22, 80, 443]
EOF

# 挂载配置文件
docker run -d \
  -v ./config:/app/configs \
  -e CONFIG_FILE=/app/configs/agent.yaml \
  zhoushoujian/go-net-monitoring:latest
```

### 2. 持久化数据

```bash
# 创建数据目录
mkdir -p ./data ./logs

# 挂载数据目录
docker run -d \
  -v ./data:/app/data \
  -v ./logs:/app/logs \
  zhoushoujian/go-net-monitoring:latest
```

## 🔒 安全配置

### 1. 最小权限原则

```bash
# Agent最小权限配置
docker run -d \
  --cap-add=NET_ADMIN \
  --cap-add=NET_RAW \
  --network host \
  zhoushoujian/go-net-monitoring:latest
```

### 2. 网络隔离

```bash
# 创建专用网络
docker network create netmon-network

# 在专用网络中运行
docker run -d \
  --network netmon-network \
  --name netmon-server \
  zhoushoujian/go-net-monitoring:latest
```

## 📊 监控和告警

### 1. 健康检查

```bash
# 检查服务健康状态
curl http://localhost:8080/health

# 响应示例
{
  "status": "healthy",
  "timestamp": 1640995200,
  "version": "1.0.0"
}
```

### 2. Prometheus集成

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'go-net-monitoring'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 10s
```

### 3. Grafana Dashboard

导入预配置的Dashboard：
```bash
curl -O https://raw.githubusercontent.com/zhoushoujian/go-net-monitoring/main/examples/grafana-dashboard.json
```

## 🔄 更新和维护

### 1. 更新镜像

```bash
# 拉取最新镜像
docker pull zhoushoujian/go-net-monitoring:latest

# 重启容器
docker-compose down && docker-compose up -d
```

### 2. 备份配置

```bash
# 备份配置和数据
tar -czf netmon-backup-$(date +%Y%m%d).tar.gz \
  docker-compose.yml \
  data/ \
  logs/ \
  config/
```

### 3. 清理资源

```bash
# 清理停止的容器
docker container prune

# 清理未使用的镜像
docker image prune

# 清理未使用的卷
docker volume prune
```
