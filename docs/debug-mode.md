# Debug 模式使用指南

本项目支持 debug 模式，可以帮助开发者和运维人员调试和排查问题。

## 功能特性

### Server Debug 模式

当启用 debug 模式时，Server 会：

1. **Gin 框架运行在 Debug 模式**
   - 显示详细的路由注册信息
   - 打印每个 HTTP 请求的详细日志
   - 显示中间件执行信息

2. **打印所有注册的路由**
   - 启动时显示所有 API 端点
   - 包括 HTTP 方法和路径信息

3. **详细的调试日志**
   - 记录接收到的指标数据详情
   - 显示 Agent 连接和断开信息
   - 输出性能统计信息

### Agent Debug 模式

当启用 debug 模式时，Agent 会：

1. **显示详细的启动信息**
   - 显示使用的网络监控工具
   - 输出配置文件路径和内容

2. **详细的监控日志**
   - 记录网络事件捕获过程
   - 显示数据上报详情

## 使用方法

### 1. 命令行参数

**启用 Server Debug 模式：**
```bash
# 使用 -d 或 --debug 标志
./bin/server -d
./bin/server --debug

# 结合配置文件使用
./bin/server -d -c configs/server-debug.yaml
```

**启用 Agent Debug 模式：**
```bash
# 使用 -d 或 --debug 标志
sudo ./bin/agent -d
sudo ./bin/agent --debug

# 结合配置文件使用
sudo ./bin/agent -d -c configs/agent.yaml
```

### 2. 配置文件

**Server 配置文件中启用 debug：**
```yaml
# configs/server-debug.yaml
http:
  host: "0.0.0.0"
  port: 8080
  debug: true        # 启用debug模式

log:
  level: "debug"     # 设置日志级别为debug
  format: "text"     # 使用text格式，更易读
  output: "stdout"
```

**普通配置文件：**
```yaml
# configs/server.yaml
http:
  host: "0.0.0.0"
  port: 8080
  debug: false       # 关闭debug模式（默认）

log:
  level: "info"      # 生产环境使用info级别
  format: "json"     # 生产环境使用json格式
  output: "stdout"
```

## Debug 模式输出示例

### Server Debug 模式输出

```bash
$ ./bin/server -d
Debug模式已启用
time="2025-07-02T14:46:35+08:00" level=info msg="Gin运行在Debug模式"
[GIN-debug] [WARNING] Running in "debug" mode. Switch to "release" mode in production.
[GIN-debug] POST   /api/v1/metrics           --> handler (3 handlers)
[GIN-debug] POST   /api/v1/heartbeat         --> handler (3 handlers)
[GIN-debug] GET    /api/v1/agents            --> handler (3 handlers)
[GIN-debug] GET    /api/v1/agents/:id        --> handler (3 handlers)
[GIN-debug] DELETE /api/v1/agents/:id        --> handler (3 handlers)
[GIN-debug] GET    /api/v1/stats             --> handler (3 handlers)
[GIN-debug] GET    /api/v1/status            --> handler (3 handlers)
[GIN-debug] GET    /metrics                  --> handler (3 handlers)
[GIN-debug] GET    /health                   --> handler (3 handlers)
[GIN-debug] GET    /ready                    --> handler (3 handlers)
[GIN-debug] GET    /                         --> handler (3 handlers)
time="2025-07-02T14:46:35+08:00" level=info msg="=== 注册的路由 ==="
time="2025-07-02T14:46:35+08:00" level=info msg="路由已注册" method=POST path=/api/v1/metrics
time="2025-07-02T14:46:35+08:00" level=info msg="路由已注册" method=POST path=/api/v1/heartbeat
...
time="2025-07-02T14:46:35+08:00" level=info msg="=== 路由注册完成 ==="
```

### Release 模式输出

```bash
$ ./bin/server
{"level":"info","msg":"Gin运行在Release模式","time":"2025-07-02T14:46:56+08:00"}
{"debug":false,"host":"0.0.0.0","level":"info","msg":"Server初始化完成","port":8080,"time":"2025-07-02T14:46:56+08:00"}
```

## 使用场景

### 开发调试

在开发过程中使用 debug 模式：

```bash
# 启动 debug 模式的 server
./bin/server -d -c configs/server-debug.yaml

# 启动 debug 模式的 agent
sudo ./bin/agent -d -c configs/agent.yaml
```

### 问题排查

当遇到问题时，启用 debug 模式获取详细信息：

1. **路由问题**：查看所有注册的路由是否正确
2. **请求问题**：查看 HTTP 请求的详细日志
3. **数据流问题**：查看 Agent 上报和 Server 接收的数据详情

### 性能分析

Debug 模式下可以看到：
- HTTP 请求处理时间
- 数据处理详情
- 内存和连接使用情况

## 注意事项

1. **生产环境不建议使用 debug 模式**
   - 会产生大量日志，影响性能
   - 可能暴露敏感信息

2. **日志级别配合使用**
   - debug 模式建议配合 `log.level: "debug"`
   - 生产环境使用 `log.level: "info"` 或更高级别

3. **命令行参数优先级**
   - 命令行 `-d` 参数会覆盖配置文件中的 `debug` 设置

4. **Gin 框架特性**
   - Debug 模式下 Gin 会自动打印路由信息
   - Release 模式下 Gin 不会打印额外信息，性能更好

## Docker 使用

在 Docker 中使用 debug 模式：

```bash
# Server debug 模式
docker run -d \
  --name netmon-server-debug \
  -p 8080:8080 \
  -e COMPONENT=server \
  zhoushoujian/go-net-monitoring:latest \
  server -d

# Agent debug 模式
docker run -d \
  --name netmon-agent-debug \
  --privileged \
  --network host \
  -e COMPONENT=agent \
  zhoushoujian/go-net-monitoring:latest \
  agent -d
```

## 相关命令

```bash
# 查看帮助信息
./bin/server --help
./bin/agent --help

# 查看版本信息
./bin/server --version
./bin/agent --version

# 使用不同配置文件
./bin/server -c configs/server-debug.yaml
./bin/agent -c configs/agent-debug.yaml
```
