# eBPF 路径配置快速使用指南

## 🚀 快速开始

### 1. 检查当前配置

```bash
# 验证配置文件
go run scripts/validate-ebpf-config.go configs/agent.yaml

# 或使用测试脚本
./scripts/test-ebpf-path.sh
```

### 2. 基本配置

在 `configs/agent.yaml` 中添加或修改 eBPF 配置：

```yaml
ebpf:
  program_path: "/opt/go-net-monitoring/bpf/xdp_monitor.o"  # 主要路径
  fallback_paths:                                          # 备用路径
    - "bpf/xdp_monitor.o"
    - "bin/bpf/xdp_monitor.o"
    - "bin/bpf/xdp_monitor_linux.o"
  enable_fallback: true                                    # 启用模拟模式
```

### 3. 环境特定配置

#### 开发环境
```yaml
ebpf:
  program_path: "bin/bpf/xdp_monitor.o"    # 相对路径
  enable_fallback: true                    # 启用模拟模式
```

#### 生产环境
```yaml
ebpf:
  program_path: "/opt/go-net-monitoring/bpf/xdp_monitor.o"  # 绝对路径
  enable_fallback: false                   # 禁用模拟模式
```

#### 容器环境
```yaml
ebpf:
  program_path: "/opt/go-net-monitoring/bpf/xdp_monitor.o"
  fallback_paths:
    - "/app/bin/bpf/xdp_monitor.o"
  enable_fallback: true
```

## 🔧 常用命令

### 验证配置
```bash
# 验证默认配置
go run scripts/validate-ebpf-config.go configs/agent.yaml

# 验证自定义配置
go run scripts/validate-ebpf-config.go /path/to/your/config.yaml
```

### 测试路径解析
```bash
# 运行完整测试
./scripts/test-ebpf-path.sh

# 检查 eBPF 程序文件
ls -la bin/bpf/
ls -la bpf/programs/
```

### 构建和部署
```bash
# 构建 Docker 镜像（包含 eBPF 程序）
make docker-build

# 启动服务
make docker-up

# 查看日志（检查 eBPF 加载状态）
make docker-logs-agent
```

## 🐛 故障排除

### 常见错误

#### 1. 文件不存在
```
ERROR: eBPF程序文件不存在: /opt/go-net-monitoring/bpf/xdp_monitor.o
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
- 使用 `--privileged` 模式运行 Docker
- 检查文件权限：`chmod 644 /path/to/xdp_monitor.o`

#### 3. 内核不支持
```
ERROR: 附加XDP程序失败: operation not supported
```

**解决方案：**
- 检查内核版本：`uname -r`（需要 4.8+）
- 启用模拟模式：`enable_fallback: true`

### 调试步骤

1. **检查配置**
   ```bash
   go run scripts/validate-ebpf-config.go configs/agent.yaml
   ```

2. **检查文件**
   ```bash
   find . -name "*.o" -type f
   ls -la bin/bpf/ bpf/programs/
   ```

3. **启用调试日志**
   ```yaml
   log:
     level: "debug"
   ```

4. **测试模拟模式**
   ```yaml
   ebpf:
     enable_fallback: true
   ```

## 📋 配置检查清单

- [ ] eBPF 程序文件存在
- [ ] 文件路径配置正确
- [ ] 文件权限设置正确
- [ ] 内核版本支持 eBPF
- [ ] 容器以特权模式运行
- [ ] 网络接口配置正确
- [ ] 备用路径配置合理
- [ ] 回退模式设置适当

## 🎯 最佳实践

1. **使用绝对路径**（生产环境）
2. **配置备用路径**（提高可靠性）
3. **启用详细日志**（便于调试）
4. **定期验证配置**（使用验证脚本）
5. **监控 eBPF 状态**（通过日志和指标）

## 📚 相关文档

- [详细配置指南](ebpf-path-configuration.md)
- [实现总结](ebpf-path-implementation.md)
- [容器化部署](container-only-deployment.md)
