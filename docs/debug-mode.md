# Debug模式使用指南

## 🎯 概述

Debug模式提供详细的调试信息，帮助开发者和运维人员排查问题。系统已优化debug配置，避免重复设置，简化使用。

## 🔧 配置原则

### 统一配置逻辑
- **DEBUG_MODE=true**: 自动设置日志级别为debug，使用text格式
- **DEBUG_MODE=false**: 使用info级别，使用json格式
- **避免重复**: 不需要同时设置DEBUG_MODE和LOG_LEVEL

## 🚀 使用方法

### 1. Docker Compose方式 (推荐)

#### 启用Debug模式
```bash
# 方式1: 环境变量
DEBUG_MODE=true docker-compose up -d

# 方式2: Make命令 (推荐)
make docker-up-debug

# 方式3: .env文件
echo "DEBUG_MODE=true" > .env
docker-compose up -d
```

#### 关闭Debug模式
```bash
# 方式1: 环境变量
DEBUG_MODE=false docker-compose up -d

# 方式2: Make命令 (推荐)
make docker-up

# 方式3: 删除.env文件
rm .env
docker-compose up -d
```

### 2. 本地开发方式

#### Server Debug模式
```bash
# 命令行参数
./bin/server --debug -c configs/server.yaml

# Make命令
make dev-run-server
```

#### Agent Debug模式
```bash
# 命令行参数 (需要root权限)
sudo ./bin/agent --debug -c configs/agent.yaml

# Make命令
make dev-run-agent
```

## 📊 Debug模式特性

### 1. 日志格式对比

#### 生产模式 (INFO级别, JSON格式)
```json
{"level":"info","msg":"流量方向统计 (总连接: 1000): map[inbound_external:197 outbound_external:246]","time":"2025-07-04T04:37:31Z"}
```

#### Debug模式 (DEBUG级别, TEXT格式)
```
time="2025-07-04T04:39:43Z" level=debug msg="域名访问统计:"
time="2025-07-04T04:39:43Z" level=debug msg="  server-3-167-99-65.iad55.r.cloudfront.net: 18次"
time="2025-07-04T04:39:43Z" level=debug msg="成功上报指标"
```

### 2. 详细信息显示

#### 启动时显示
- ✅ 完整配置文件内容
- ✅ 环境变量设置
- ✅ 网络接口信息
- ✅ 权限检查结果

#### 运行时显示
- ✅ 详细的网络流量统计
- ✅ 域名解析过程
- ✅ 数据上报详情
- ✅ 错误堆栈信息

## 🔍 日志查看

### 实时日志
```bash
# 查看所有服务日志
make docker-logs

# 查看Agent日志
make docker-logs-agent

# 查看Server日志
make docker-logs-server

# 传统方式
docker-compose logs -f
docker-compose logs -f agent
docker-compose logs -f server
```

### 历史日志
```bash
# 查看最近100行
docker logs netmon-agent --tail 100
docker logs netmon-server --tail 100

# 查看特定时间段
docker logs netmon-agent --since "2025-07-04T12:00:00"
```

## 🛠️ 故障排查

### 1. 常见问题

#### Debug模式未生效
```bash
# 检查环境变量
docker exec netmon-agent env | grep DEBUG_MODE

# 检查启动日志
docker logs netmon-agent | head -20

# 重启服务
docker-compose restart agent
```

#### 日志级别不正确
```bash
# 检查配置文件
docker exec netmon-agent cat /app/configs/agent.yaml | grep -A3 log

# 检查运行时配置
docker logs netmon-agent | grep "日志级别"
```

### 2. 性能影响

#### Debug模式影响
- 📈 **日志量增加**: 约5-10倍
- 🐌 **性能下降**: 约10-15%
- 💾 **存储占用**: 显著增加

#### 建议
- ✅ **开发环境**: 推荐使用debug模式
- ⚠️ **测试环境**: 按需使用
- ❌ **生产环境**: 不建议长期使用

## 📋 最佳实践

### 1. 开发阶段
```bash
# 启动debug环境
make docker-up-debug

# 实时查看日志
make docker-logs-agent

# 问题排查后关闭
make docker-down
```

### 2. 问题排查
```bash
# 临时启用debug
DEBUG_MODE=true docker-compose up -d

# 收集日志
docker logs netmon-agent > agent-debug.log
docker logs netmon-server > server-debug.log

# 排查完成后恢复
DEBUG_MODE=false docker-compose up -d
```

### 3. 生产监控
```bash
# 正常运行
make docker-up

# 健康检查
make health

# 查看关键指标
make metrics
```

## 🎯 配置示例

### .env文件示例
```bash
# 开发环境
DEBUG_MODE=true
HOSTNAME=dev-agent

# 生产环境
DEBUG_MODE=false
HOSTNAME=prod-agent-01
```

### docker-compose.override.yml
```yaml
# 开发环境覆盖配置
version: '3.8'
services:
  agent:
    environment:
      - DEBUG_MODE=true
  server:
    environment:
      - DEBUG_MODE=true
```

## 🔄 自动化脚本

### 快速切换脚本
```bash
#!/bin/bash
# toggle-debug.sh

if [ "$1" = "on" ]; then
    echo "启用Debug模式..."
    DEBUG_MODE=true docker-compose up -d
    echo "Debug模式已启用"
elif [ "$1" = "off" ]; then
    echo "关闭Debug模式..."
    DEBUG_MODE=false docker-compose up -d
    echo "Debug模式已关闭"
else
    echo "用法: $0 [on|off]"
fi
```

这个优化后的debug模式配置简化了使用流程，避免了重复设置，提供了清晰的日志输出格式。
