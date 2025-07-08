# eBPF 程序路径配置指南

本文档介绍如何配置 eBPF 程序路径，解决不同环境下的路径兼容性问题。

## 概述

网络监控系统使用 eBPF (Extended Berkeley Packet Filter) 程序进行高性能的网络数据包捕获。为了解决在不同环境（开发、测试、生产）下 eBPF 程序路径不一致的问题，系统提供了灵活的路径配置机制。

## 配置结构

在 `configs/agent.yaml` 中添加 `ebpf` 配置段：

```yaml
ebpf:
  program_path: "/opt/go-net-monitoring/bpf/xdp_monitor.o"  # 主要程序路径
  fallback_paths:                                          # 备用路径列表
    - "bpf/xdp_monitor.o"
    - "bin/bpf/xdp_monitor.o"
    - "bin/bpf/xdp_monitor_linux.o"
    - "/usr/local/bin/bpf/xdp_monitor.o"
  enable_fallback: true                                    # 启用模拟模式回退
```

### 配置参数说明

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `program_path` | string | 否 | `/opt/go-net-monitoring/bpf/xdp_monitor.o` | 主要的 eBPF 程序文件路径 |
| `fallback_paths` | []string | 否 | 见下方默认值 | 备用路径列表，按优先级排序 |
| `enable_fallback` | bool | 否 | `true` | eBPF 加载失败时是否启用模拟模式 |

### 默认备用路径

```yaml
fallback_paths:
  - "bpf/xdp_monitor.o"                    # 开发环境相对路径
  - "bin/bpf/xdp_monitor.o"                # 构建输出目录
  - "bin/bpf/xdp_monitor_linux.o"          # Linux 特定版本
  - "/usr/local/bin/bpf/xdp_monitor.o"     # 系统安装路径
```

## 路径解析机制

系统按以下优先级顺序查找 eBPF 程序：

### 1. 主要路径 (`program_path`)

首先尝试配置文件中指定的主要路径。

**绝对路径示例：**
```yaml
ebpf:
  program_path: "/opt/go-net-monitoring/bpf/xdp_monitor.o"
```

**相对路径示例：**
```yaml
ebpf:
  program_path: "bin/bpf/xdp_monitor.o"
```

### 2. 备用路径 (`fallback_paths`)

如果主要路径不存在，按顺序尝试备用路径列表中的每个路径。

### 3. 默认路径

如果配置的路径都不存在，尝试以下默认路径：
- `bin/bpf/xdp_monitor_linux.o`
- `bin/bpf/xdp_monitor.o`
- `bpf/xdp_monitor.o`

### 4. 相对路径解析

对于相对路径，系统会在以下位置搜索：

1. **当前工作目录**：相对于程序运行时的工作目录
2. **二进制文件目录**：相对于 agent 可执行文件所在目录
3. **项目根目录**：相对于项目根目录（适用于开发环境）

## 使用场景

### 开发环境

```yaml
ebpf:
  program_path: "bpf/xdp_monitor.o"        # 相对于项目根目录
  enable_fallback: true                    # 启用模拟模式，便于调试
```

### 容器环境

```yaml
ebpf:
  program_path: "/opt/go-net-monitoring/bpf/xdp_monitor.o"  # 绝对路径
  enable_fallback: false                   # 生产环境不使用模拟模式
```

### 测试环境

```yaml
ebpf:
  program_path: "bin/bpf/xdp_monitor.o"
  fallback_paths:
    - "bpf/xdp_monitor.o"
    - "/tmp/test-bpf/xdp_monitor.o"
  enable_fallback: true
```

## 错误处理

### eBPF 程序加载失败

当 eBPF 程序加载失败时，系统的处理策略：

1. **记录详细错误日志**：包含尝试的路径和失败原因
2. **尝试备用路径**：按优先级尝试所有配置的备用路径
3. **模拟模式回退**：如果 `enable_fallback: true`，启用模拟模式
4. **完全失败**：如果 `enable_fallback: false`，返回错误并停止启动

### 常见错误和解决方案

#### 1. 文件不存在错误

```
ERROR: eBPF程序文件不存在: /opt/go-net-monitoring/bpf/xdp_monitor.o (no such file or directory)
```

**解决方案：**
- 检查文件路径是否正确
- 确保 eBPF 程序已编译
- 添加备用路径配置

#### 2. 权限错误

```
ERROR: 加载eBPF程序失败: permission denied
```

**解决方案：**
- 确保以特权模式运行（Docker 中使用 `--privileged`）
- 检查文件权限
- 确保内核支持 eBPF

#### 3. 内核兼容性错误

```
ERROR: 附加XDP程序到接口失败: operation not supported
```

**解决方案：**
- 检查内核版本（需要 4.8+）
- 确保网络接口支持 XDP
- 启用模拟模式作为回退

## 日志和调试

### 启用详细日志

```yaml
log:
  level: "debug"    # 启用调试日志
  format: "json"
```

### 关键日志信息

```json
{
  "level": "info",
  "msg": "使用配置文件指定的eBPF程序路径",
  "path": "/opt/go-net-monitoring/bpf/xdp_monitor.o"
}

{
  "level": "debug",
  "msg": "eBPF程序路径解析成功",
  "original_path": "bin/bpf/xdp_monitor.o",
  "resolved_path": "/app/bin/bpf/xdp_monitor.o",
  "location": "二进制文件目录"
}

{
  "level": "info",
  "msg": "eBPF程序加载并附加成功",
  "program_path": "/opt/go-net-monitoring/bpf/xdp_monitor.o",
  "interface": "eth0"
}
```

## 最佳实践

### 1. 环境特定配置

为不同环境创建专门的配置文件：

```bash
configs/
├── agent.yaml                    # 默认配置
├── agent-dev.yaml               # 开发环境
├── agent-prod.yaml              # 生产环境
└── agent-test.yaml              # 测试环境
```

### 2. 容器化部署

在 Dockerfile 中确保 eBPF 程序位于预期路径：

```dockerfile
# 复制 eBPF 程序到标准位置
COPY bin/bpf/ /opt/go-net-monitoring/bpf/

# 设置正确的权限
RUN chmod 644 /opt/go-net-monitoring/bpf/*.o
```

### 3. 健康检查

在启动脚本中添加 eBPF 程序检查：

```bash
#!/bin/bash

# 检查 eBPF 程序文件
EBPF_PROGRAM="/opt/go-net-monitoring/bpf/xdp_monitor.o"
if [[ ! -f "$EBPF_PROGRAM" ]]; then
    echo "警告: eBPF程序文件不存在: $EBPF_PROGRAM"
    echo "将使用模拟模式运行"
fi

# 启动 agent
exec ./agent -config /etc/netmon/agent.yaml
```

### 4. 监控和告警

监控 eBPF 程序加载状态：

```promql
# eBPF 模式运行的 agent 数量
count(network_interface_info{mode="ebpf"})

# 模拟模式运行的 agent 数量  
count(network_interface_info{mode="simulation"})
```

## 测试验证

使用提供的测试脚本验证配置：

```bash
# 运行 eBPF 路径配置测试
./scripts/test-ebpf-path.sh

# 测试特定配置
./bin/agent -config configs/test/agent-absolute-path.yaml -dry-run
```

## 故障排除

### 检查清单

1. **文件存在性**：确认 eBPF 程序文件存在
2. **文件权限**：确认文件可读
3. **路径正确性**：验证绝对路径和相对路径
4. **内核支持**：检查内核版本和 eBPF 支持
5. **网络接口**：确认指定的网络接口存在
6. **特权模式**：确保以足够权限运行

### 常用调试命令

```bash
# 检查 eBPF 程序文件
ls -la /opt/go-net-monitoring/bpf/

# 检查内核 eBPF 支持
zgrep CONFIG_BPF /proc/config.gz

# 检查网络接口
ip link show

# 测试配置文件语法
yq eval '.' configs/agent.yaml
```

## 相关文档

- [容器化部署说明](container-only-deployment.md)
- [Docker Compose 使用指南](docker-compose-usage.md)
- [故障排除指南](troubleshooting.md)
