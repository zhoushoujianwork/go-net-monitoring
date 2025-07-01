# 配置文档

## Agent配置

Agent配置文件位于 `configs/agent.yaml`，包含以下主要配置项：

### Server配置
```yaml
server:
  host: "localhost"    # Server地址
  port: 8080          # Server端口
```

### 监控配置
```yaml
monitor:
  interface: ""                    # 监控的网络接口，空表示自动选择
  protocols: ["tcp", "udp"]        # 监控的协议类型
  report_interval: "30s"           # 上报间隔
  buffer_size: 1000               # 缓冲区大小
  filters:                        # 过滤规则
    ignore_localhost: true        # 忽略本地回环
    ignore_ports: [22, 53]        # 忽略的端口
    ignore_ips: ["127.0.0.1"]     # 忽略的IP地址
    only_domains: []              # 只监控特定域名
```

### 上报配置
```yaml
reporter:
  server_url: "http://localhost:8080/api/v1/metrics"  # Server API地址
  timeout: "10s"                                      # 请求超时时间
  retry_count: 3                                      # 重试次数
  retry_delay: "5s"                                   # 重试延迟
  batch_size: 100                                     # 批处理大小
  enable_tls: false                                   # 是否启用TLS
  tls_cert_path: ""                                   # TLS证书路径
  tls_key_path: ""                                    # TLS密钥路径
```

### 日志配置
```yaml
log:
  level: "info"        # 日志级别: debug, info, warn, error
  format: "json"       # 日志格式: json, text
  output: "stdout"     # 日志输出: stdout, stderr, 或文件路径
```

## Server配置

Server配置文件位于 `configs/server.yaml`，包含以下主要配置项：

### HTTP服务配置
```yaml
http:
  host: "0.0.0.0"        # 监听地址
  port: 8080             # 监听端口
  read_timeout: "30s"    # 读取超时
  write_timeout: "30s"   # 写入超时
  enable_tls: false      # 是否启用TLS
  tls_cert_path: ""      # TLS证书路径
  tls_key_path: ""       # TLS密钥路径
```

### Prometheus指标配置
```yaml
metrics:
  path: "/metrics"       # 指标暴露路径
  enabled: true          # 是否启用指标
  interval: "15s"        # 指标更新间隔
```

### 存储配置
```yaml
storage:
  type: "memory"         # 存储类型: memory, redis
  ttl: "1h"             # 数据保留时间
  max_entries: 10000     # 最大条目数
```

## 环境变量

可以通过环境变量覆盖配置文件中的设置：

- `AGENT_SERVER_HOST`: Agent Server地址
- `AGENT_SERVER_PORT`: Agent Server端口
- `AGENT_LOG_LEVEL`: Agent日志级别
- `SERVER_HTTP_HOST`: Server监听地址
- `SERVER_HTTP_PORT`: Server监听端口
- `SERVER_LOG_LEVEL`: Server日志级别

## 配置验证

启动时会自动验证配置文件的正确性，如果配置有误会输出详细的错误信息。

## 配置热重载

目前不支持配置热重载，修改配置后需要重启服务。

## 最佳实践

1. **生产环境建议**：
   - 启用TLS加密
   - 设置合适的日志级别（info或warn）
   - 配置日志输出到文件
   - 设置合适的缓冲区大小

2. **性能优化**：
   - 根据网络流量调整`buffer_size`
   - 根据网络延迟调整`report_interval`
   - 合理设置过滤规则减少无用数据

3. **安全考虑**：
   - 使用TLS保护数据传输
   - 限制监听地址
   - 设置防火墙规则
