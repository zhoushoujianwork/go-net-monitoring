# Docker Compose 使用指南

本文档介绍如何使用 Docker Compose 部署网络监控系统，现在提供简化的部署方式，主要使用 Redis 存储，同时支持内存存储作为备选。

## 配置文件结构

```
configs/
├── server.yaml              # Server内存存储配置
├── server-redis.yaml        # Server Redis存储配置
└── agent.yaml               # Agent配置
```

## 部署方式

### 1. 默认部署 (Redis 存储，推荐)

启动完整的 Redis + Server + Agent 服务：

```bash
# 启动默认服务 (Redis 存储)
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f
```

**包含的服务：**
- Redis: 数据存储
- Server: 网络监控服务器 (端口 8080)
- Agent: 网络监控代理

**使用的配置文件：**
- Server: `configs/server-redis.yaml`
- Agent: `configs/agent.yaml`

### 2. 内存存储模式 (备选方案)

如果不想使用 Redis，可以使用内存存储模式：

```bash
# 启动内存存储模式
docker-compose --profile memory up -d server-memory agent

# 查看日志
docker-compose --profile memory logs -f server-memory agent
```

**特点：**
- Server 运行在端口 8081
- 使用内存存储，重启后数据丢失
- 适合测试和开发环境

### 3. 完整监控栈

包含 Prometheus 和 Grafana 的完整监控解决方案：

```bash
# 启动完整监控栈
docker-compose --profile monitoring up -d

# 访问服务
# - Server: http://localhost:8080
# - Prometheus: http://localhost:9090  
# - Grafana: http://localhost:3000 (admin/admin123)
```

## 服务端口分配

| 服务 | 端口 | 访问地址 | 说明 |
|------|------|----------|------|
| Server (Redis) | 8080 | http://localhost:8080 | 默认服务器 |
| Server (Memory) | 8081 | http://localhost:8081 | 内存存储服务器 |
| Redis | 6379 | localhost:6379 | Redis 数据库 |
| Prometheus | 9090 | http://localhost:9090 | 监控系统 |
| Grafana | 3000 | http://localhost:3000 | 仪表板 |

## 常用命令

### 服务管理

```bash
# 启动服务
docker-compose up -d

# 停止服务
docker-compose down

# 重启服务
docker-compose restart

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f [service_name]
```

### 数据管理

```bash
# 清理所有数据 (包括 Redis 数据)
docker-compose down -v

# 只清理容器，保留数据
docker-compose down

# 重建服务
docker-compose up -d --force-recreate
```

### 配置更新

```bash
# 修改配置文件后重启服务
docker-compose restart server

# 重新加载配置
docker-compose up -d --force-recreate server
```

## 配置文件说明

### Server 配置

**Redis 存储配置 (`configs/server-redis.yaml`):**
```yaml
http:
  host: "0.0.0.0"
  port: 8080

storage:
  type: "redis"
  redis:
    host: "redis"    # Docker 服务名
    port: 6379
    db: 0

log:
  level: "info"
  format: "json"
```

**内存存储配置 (`configs/server.yaml`):**
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
```

### Agent 配置

**Agent 配置 (`configs/agent.yaml`):**
```yaml
server:
  host: "server"     # Docker 服务名
  port: 8080

monitor:
  interface: "eth0"
  protocols: ["tcp", "udp", "http", "https"]
  report_interval: "10s"

reporter:
  server_url: "http://server:8080/api/v1/metrics"
  timeout: "10s"
  batch_size: 100
```

## 网络配置

### Agent 网络模式

Agent 使用主机网络模式以监控主机网络流量：

```yaml
agent:
  network_mode: host    # 主机网络模式
  privileged: true      # 特权模式
```

### 服务发现

服务间通过 Docker 服务名进行通信：
- Agent → Server: `http://server:8080`
- Server → Redis: `redis:6379`

## 数据持久化

### 数据卷

```yaml
volumes:
  redis_data:         # Redis 数据持久化
  prometheus_data:    # Prometheus 数据持久化  
  grafana_data:       # Grafana 数据持久化
```

### 目录挂载

```yaml
volumes:
  - ./configs:/app/configs:ro    # 配置文件 (只读)
  - ./data:/app/data            # 应用数据
  - ./logs:/app/logs            # 日志文件
```

## 健康检查

### Server 健康检查

```bash
# 检查 Server 健康状态
curl http://localhost:8080/health

# 检查指标端点
curl http://localhost:8080/metrics
```

### Agent 健康检查

```bash
# 检查 Agent 进程
docker-compose exec agent pgrep agent

# 查看 Agent 日志
docker-compose logs agent
```

## 故障排查

### 常见问题

1. **Agent 无法连接 Server**
   ```bash
   # 检查网络连接
   docker-compose exec agent ping server
   
   # 检查 Server 是否启动
   docker-compose ps server
   ```

2. **Redis 连接失败**
   ```bash
   # 检查 Redis 服务
   docker-compose ps redis
   
   # 测试 Redis 连接
   docker-compose exec redis redis-cli ping
   ```

3. **配置文件错误**
   ```bash
   # 验证配置文件语法
   docker-compose config
   
   # 查看服务日志
   docker-compose logs server
   ```

### 日志查看

```bash
# 查看所有服务日志
docker-compose logs

# 查看特定服务日志
docker-compose logs server
docker-compose logs agent
docker-compose logs redis

# 实时跟踪日志
docker-compose logs -f --tail=100
```

## 性能优化

### Redis 优化

```yaml
redis:
  command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
```

### Agent 优化

```yaml
# 在 agent.yaml 中调整
monitor:
  buffer_size: 10000      # 增大缓冲区
  report_interval: "30s"  # 调整上报间隔

reporter:
  batch_size: 1000        # 批量上报大小
  timeout: "30s"          # 超时时间
```

## 监控集成

### Prometheus 配置

Server 自动暴露 Prometheus 指标：
- 指标端点: `http://localhost:8080/metrics`
- 自动发现: 通过 `docker/prometheus.yml` 配置

### Grafana 仪表板

1. 访问 Grafana: http://localhost:3000
2. 登录: admin/admin123
3. 添加 Prometheus 数据源: `http://prometheus:9090`
4. 导入预配置的仪表板

## 生产环境建议

1. **使用 Redis 存储**: 提供数据持久化和更好的性能
2. **配置资源限制**: 在 docker-compose.yml 中添加资源限制
3. **启用监控**: 使用 Prometheus + Grafana 进行监控
4. **定期备份**: 备份 Redis 数据和配置文件
5. **日志轮转**: 配置日志轮转避免磁盘空间不足

---

## 快速开始

```bash
# 1. 克隆项目
git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
cd go-net-monitoring

# 2. 启动服务 (Redis 存储)
docker-compose up -d

# 3. 检查服务状态
docker-compose ps

# 4. 查看指标
curl http://localhost:8080/metrics

# 5. 启动完整监控栈 (可选)
docker-compose --profile monitoring up -d

# 6. 访问 Grafana (可选)
# http://localhost:3000 (admin/admin123)
```

### 3. Redis 存储部署

启动使用 Redis 存储的完整系统：

```bash
# 启动Redis存储模式
docker-compose --profile redis up -d redis server-redis agent

# 查看服务状态
docker-compose --profile redis ps
```

**特性：**
- Server 运行在端口 8082
- 数据持久化到 Redis
- 支持多个 Server 实例共享数据

### 4. 完整监控栈部署

启动包含 Prometheus 和 Grafana 的完整监控系统：

```bash
# 启动完整监控栈
docker-compose --profile monitoring --profile redis up -d

# 访问服务
# Grafana: http://localhost:3000 (admin/admin123)
# Prometheus: http://localhost:9090
# Server: http://localhost:8082
```

### 5. 环境变量配置模式

使用环境变量快速配置（不依赖配置文件）：

```bash
# 启动环境变量配置模式
docker-compose --profile env-config up -d server agent-env
```

## 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| server | 8080 | 默认 Server |
| server-debug | 8081 | Debug 模式 Server |
| server-redis | 8082 | Redis 存储 Server |
| redis | 6379 | Redis 数据库 |
| prometheus | 9090 | Prometheus 监控 |
| grafana | 3000 | Grafana 仪表板 |

## 配置文件详解

### Server 配置 (configs/server.yaml)

```yaml
http:
  host: "0.0.0.0"
  port: 8080
  debug: false       # 生产模式

storage:
  type: "memory"     # 内存存储
  max_entries: 50000

log:
  level: "info"
  format: "json"
```

### Server Debug 配置 (configs/server-debug.yaml)

```yaml
http:
  host: "0.0.0.0"
  port: 8080
  debug: true        # Debug模式

log:
  level: "debug"     # Debug级别日志
  format: "text"     # 文本格式，更易读
```

### Server Redis 配置 (configs/server-redis.yaml)

```yaml
storage:
  type: "redis"      # Redis存储
  redis:
    host: "redis"    # Docker服务名
    port: 6379
    db: 0
```

### Agent 配置 (configs/agent.yaml)

```yaml
server:
  host: "server"     # Docker服务名
  port: 8080

monitor:
  interface: "eth0"  # Docker容器网络接口

reporter:
  server_url: "http://server:8080/api/v1/metrics"
```

## 常用命令

### 启动服务

```bash
# 基础服务
docker-compose up -d server agent

# Debug模式
docker-compose --profile debug up -d server-debug agent-debug

# Redis存储
docker-compose --profile redis up -d redis server-redis agent

# 完整监控栈
docker-compose --profile monitoring --profile redis up -d
```

### 查看日志

```bash
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f server
docker-compose logs -f agent

# 查看Debug模式日志
docker-compose --profile debug logs -f server-debug
```

### 服务管理

```bash
# 查看服务状态
docker-compose ps

# 停止服务
docker-compose down

# 停止并删除数据卷
docker-compose down -v

# 重启服务
docker-compose restart server
```

### 健康检查

```bash
# 检查Server健康状态
curl http://localhost:8080/health

# 检查Debug模式Server
curl http://localhost:8081/health

# 检查Redis模式Server
curl http://localhost:8082/health
```

## 数据持久化

### 目录挂载

```
./data/     # 应用数据目录
./logs/     # 日志文件目录
./configs/  # 配置文件目录 (只读挂载)
```

### Docker 卷

```
redis_data      # Redis数据持久化
prometheus_data # Prometheus数据持久化
grafana_data    # Grafana数据持久化
```

## 网络配置

### 默认网络

- 网络名称: `netmon`
- 驱动: `bridge`
- Agent 使用 `host` 网络模式以访问主机网络接口

### 服务发现

在 Docker 网络中，服务可以通过服务名相互访问：

- `server` -> Server 服务
- `redis` -> Redis 服务
- `prometheus` -> Prometheus 服务
- `grafana` -> Grafana 服务

## 故障排查

### 1. 配置文件问题

```bash
# 检查配置文件是否存在
ls -la configs/

# 验证配置文件语法
docker run --rm -v $(pwd)/configs:/configs alpine/yaml:latest yamllint /configs/server.yaml
```

### 2. 网络连接问题

```bash
# 检查容器网络
docker network ls
docker network inspect go-net-monitoring_netmon

# 测试服务连接
docker-compose exec agent ping server
```

### 3. 权限问题

```bash
# 检查Agent权限
docker-compose logs agent | grep -i permission

# 确保Agent以特权模式运行
docker-compose exec agent id
```

### 4. 端口冲突

```bash
# 检查端口占用
netstat -tlnp | grep :8080
lsof -i :8080

# 修改端口配置
# 编辑 docker-compose.yml 中的端口映射
```

## 最佳实践

1. **生产环境**：使用 Redis 存储模式
2. **开发调试**：使用 Debug 模式
3. **监控告警**：部署完整监控栈
4. **配置管理**：将配置文件纳入版本控制
5. **日志管理**：定期清理日志文件
6. **安全考虑**：修改默认密码，启用 TLS

## 示例场景

### 开发环境

```bash
# 启动debug模式进行开发
docker-compose --profile debug up -d server-debug agent-debug

# 查看详细日志
docker-compose --profile debug logs -f
```

### 测试环境

```bash
# 启动基础服务进行测试
docker-compose up -d server agent

# 运行健康检查
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/stats
```

### 生产环境

```bash
# 启动生产环境完整栈
docker-compose --profile monitoring --profile redis up -d

# 配置监控告警
# 访问 Grafana 配置仪表板和告警规则
```
