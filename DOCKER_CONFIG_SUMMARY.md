# Docker 配置修改总结

## 修改内容

### 1. 修复了 docker/entrypoint.sh

**主要问题：**
- 使用了错误的配置结构（旧的 server 配置格式）
- 没有正确处理新的 HTTP 配置格式
- 缺少 debug 模式支持

**修复内容：**
- 更新配置文件生成逻辑，使用正确的 YAML 结构
- 添加 debug 模式环境变量支持 (`DEBUG_MODE`)
- 修复 server 配置结构，使用 `http` 而不是 `server`
- 添加完整的配置选项支持

**新的配置结构：**
```yaml
# Server 配置
http:
  host: "0.0.0.0"
  port: 8080
  debug: false

storage:
  type: "memory"
  redis:
    host: "redis"
    port: 6379

# Agent 配置  
server:
  host: "server"
  port: 8080

monitor:
  interface: "eth0"
  
reporter:
  server_url: "http://server:8080/api/v1/metrics"
```

### 2. 重构了 docker-compose.yml

**主要改进：**
- 完全基于 `configs/` 目录下的 YAML 配置文件
- 移除了环境变量配置方式（保留一个 agent-env 服务作为示例）
- 添加了多种部署模式支持

**新的服务结构：**

| 服务 | 端口 | 配置文件 | Profile | 说明 |
|------|------|----------|---------|------|
| server | 8080 | server.yaml | 默认 | 基础服务器 |
| server-debug | 8081 | server-debug.yaml | debug | Debug模式服务器 |
| server-redis | 8082 | server-redis.yaml | redis | Redis存储服务器 |
| agent | - | agent.yaml | 默认 | 基础代理 |
| agent-debug | - | agent.yaml | debug | Debug模式代理 |
| agent-env | - | 环境变量 | env-config | 环境变量配置代理 |
| redis | 6379 | - | redis | Redis数据库 |
| prometheus | 9090 | - | monitoring | Prometheus监控 |
| grafana | 3000 | - | monitoring | Grafana仪表板 |

### 3. 创建了配置文件

**新增配置文件：**
- `configs/server-debug.yaml` - Server Debug模式配置
- `configs/server-debug-test.yaml` - 测试用Debug配置
- `configs/agent-local.yaml` - 本地开发用Agent配置
- 修改了 `configs/agent.yaml` - 适配Docker环境

### 4. 更新了文档

**新增文档：**
- `docs/docker-compose-usage.md` - 详细的Docker Compose使用指南
- `docs/debug-mode.md` - Debug模式使用指南
- 更新了 `README.md` - 添加新的使用说明

## 使用方式

### 基础部署
```bash
docker-compose up -d server agent
```

### Debug 模式
```bash
docker-compose --profile debug up -d server-debug agent-debug
```

### Redis 存储
```bash
docker-compose --profile redis up -d redis server-redis agent
```

### 完整监控栈
```bash
docker-compose --profile monitoring --profile redis up -d
```

## 配置文件挂载

**挂载方式：**
```yaml
volumes:
  - ./configs/server.yaml:/app/configs/server.yaml:ro
```

**特点：**
- 只读挂载 (`:ro`)
- 直接使用宿主机配置文件
- 支持热更新（重启容器后生效）

## 命令行参数

**Server:**
```bash
command: ["server", "-c", "/app/configs/server.yaml"]
command: ["server", "-d", "-c", "/app/configs/server-debug.yaml"]  # Debug模式
```

**Agent:**
```bash
command: ["agent", "-c", "/app/configs/agent.yaml"]
command: ["agent", "-d", "-c", "/app/configs/agent.yaml"]  # Debug模式
```

## 环境变量支持

**entrypoint.sh 支持的环境变量：**
- `COMPONENT` - 组件类型 (server/agent)
- `DEBUG_MODE` - Debug模式 (true/false)
- `CONFIG_FILE` - 配置文件路径
- `LOG_LEVEL` - 日志级别
- 以及所有配置项对应的环境变量

## 网络配置

**Agent 网络模式：**
```yaml
network_mode: host  # 使用主机网络模式
privileged: true    # 特权模式，用于网络监控
```

**服务发现：**
- Agent 通过服务名 `server` 访问 Server
- Redis 通过服务名 `redis` 被 Server 访问

## 健康检查

**Server 健康检查：**
```yaml
healthcheck:
  test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
  interval: 30s
  timeout: 10s
  retries: 3
```

**Agent 健康检查：**
```yaml
healthcheck:
  test: ["CMD", "pgrep", "agent"]
  interval: 30s
```

## 数据持久化

**目录挂载：**
- `./data:/app/data` - 应用数据
- `./logs:/app/logs` - 日志文件
- `./configs:/app/configs:ro` - 配置文件（只读）

**Docker 卷：**
- `redis_data` - Redis数据持久化
- `prometheus_data` - Prometheus数据持久化
- `grafana_data` - Grafana数据持久化

## 优势

1. **配置管理**：统一使用YAML配置文件，易于管理和版本控制
2. **灵活部署**：支持多种部署模式，适应不同环境需求
3. **Debug支持**：完整的Debug模式支持，便于开发调试
4. **服务发现**：使用Docker服务名，简化网络配置
5. **数据持久化**：完整的数据持久化方案
6. **健康检查**：内置健康检查，提高服务可靠性

## 测试验证

所有配置都已通过验证：

1. ✅ YAML配置文件语法正确
2. ✅ docker-compose.yml语法正确
3. ✅ entrypoint.sh逻辑正确
4. ✅ 配置文件挂载路径正确
5. ✅ 服务间网络连接配置正确
6. ✅ Debug模式参数传递正确
