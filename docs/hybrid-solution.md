# 混合方案使用指南

## 概述

混合方案结合了Agent端持久化和Server端智能累计的优势，解决了Agent重启导致的累计统计数据丢失问题。

## 核心特性

### 1. Agent端持久化
- **状态保存**: Agent定期将累计统计保存到本地文件
- **重启恢复**: Agent重启后自动加载历史状态
- **增量上报**: 支持增量和累计两种上报模式
- **备份机制**: 支持多个备份文件轮转

### 2. Server端智能累计
- **重启检测**: 自动检测Agent重启事件
- **基线跟踪**: 保存Agent重启前的累计基线
- **全局累计**: 计算跨Agent重启的真实累计值
- **数据清理**: 自动清理过期Agent数据

### 3. 数据一致性保证
- **双重检测**: 通过启动时间和指标值变化检测重启
- **基线合并**: 将重启前后的数据正确合并
- **负值保护**: 防止累计值出现负数
- **并发安全**: 使用读写锁保证数据一致性

## 部署方式

### 1. Docker Compose部署 (推荐)

```bash
# 启动混合方案
docker-compose -f docker-compose-hybrid.yml up -d

# 查看服务状态
docker-compose -f docker-compose-hybrid.yml ps

# 查看Agent持久化状态
docker exec netmon-agent-hybrid ls -la /var/lib/netmon/

# 查看日志
docker-compose -f docker-compose-hybrid.yml logs -f agent-hybrid
docker-compose -f docker-compose-hybrid.yml logs -f server-hybrid
```

### 2. 手动部署

**启动Redis:**
```bash
docker run -d --name redis -p 6379:6379 redis:7-alpine
```

**启动Server:**
```bash
./bin/server --config configs/server-hybrid.yaml
```

**启动Agent:**
```bash
# 创建状态目录
sudo mkdir -p /var/lib/netmon
sudo chown $USER:$USER /var/lib/netmon

# 启动Agent
sudo ./bin/agent --config configs/agent-hybrid.yaml
```

## 配置说明

### Agent配置 (agent-hybrid.yaml)

```yaml
# 持久化配置
persistence:
  enabled: true                              # 启用持久化
  state_file: "/var/lib/netmon/agent-state.json"  # 状态文件路径
  save_interval: "30s"                       # 保存间隔
  backup_count: 3                           # 备份文件数量

# 上报配置
reporter:
  mode: "incremental"                        # 增量模式
  include_totals: true                       # 包含累计数据
  agent_id: "${HOSTNAME}"                    # Agent唯一标识
```

### Server配置 (server-hybrid.yaml)

```yaml
storage:
  type: "redis"                             # Redis存储
  cumulative_mode: true                     # 累计模式
  baseline_tracking: true                   # 基线跟踪
  agent_restart_detection: true             # 重启检测
```

## 监控指标

### 原始指标 (可能重置)
- `network_domains_accessed_raw_total`
- `network_domain_bytes_sent_raw_total`
- `network_domain_bytes_received_raw_total`

### 累计指标 (永不重置)
- `network_domains_accessed_cumulative_total`
- `network_domain_bytes_sent_cumulative_total`
- `network_domain_bytes_received_cumulative_total`

### Agent状态指标
- `network_agent_restarts_total`
- `network_agent_uptime_seconds`
- `network_agent_last_report_timestamp`

## Grafana查询示例

### 1. 累计域名访问量
```promql
# 使用累计指标
topk(10, network_domains_accessed_cumulative_total)

# 或处理重启的原始指标
increase(network_domains_accessed_raw_total[5m]) + 
  (resets(network_domains_accessed_raw_total[5m]) * 
   (network_domains_accessed_raw_total offset 5m))
```

### 2. 域名流量趋势
```promql
# 发送流量趋势
rate(network_domain_bytes_sent_cumulative_total[5m])

# 接收流量趋势
rate(network_domain_bytes_received_cumulative_total[5m])
```

### 3. Agent重启监控
```promql
# Agent重启次数
network_agent_restarts_total

# Agent运行时间
network_agent_uptime_seconds
```

## 故障排查

### 1. Agent持久化问题

**检查状态文件:**
```bash
# 查看状态文件
cat /var/lib/netmon/agent-state.json | jq .

# 检查文件权限
ls -la /var/lib/netmon/
```

**常见问题:**
- 权限不足: `sudo chown -R $USER:$USER /var/lib/netmon`
- 磁盘空间不足: `df -h /var/lib/netmon`
- JSON格式错误: 删除状态文件重新开始

### 2. Server累计问题

**检查Redis连接:**
```bash
# 连接Redis
redis-cli -h localhost -p 6379

# 查看存储的键
KEYS netmon:*

# 查看Agent状态
HGETALL netmon:agents
```

**检查累计数据:**
```bash
# 查看累计指标API
curl http://localhost:8080/api/v1/cumulative

# 查看特定域名
curl http://localhost:8080/api/v1/domains/example.com
```

### 3. 重启检测问题

**查看日志:**
```bash
# Agent日志
docker logs netmon-agent-hybrid | grep restart

# Server日志
docker logs netmon-server-hybrid | grep restart
```

**手动触发重启检测:**
```bash
# 重启Agent容器
docker restart netmon-agent-hybrid

# 观察Server日志中的重启检测
docker logs -f netmon-server-hybrid
```

## 性能优化

### 1. Agent端优化

```yaml
# 调整保存间隔
persistence:
  save_interval: "60s"  # 减少磁盘IO

# 调整上报间隔
monitor:
  report_interval: "30s"  # 减少网络请求

# 调整缓冲区大小
monitor:
  buffer_size: 5000  # 增大缓冲区
```

### 2. Server端优化

```yaml
# Redis连接池
storage:
  redis:
    pool_size: 20  # 增大连接池

# 清理间隔
advanced:
  cleanup_interval: "2h"  # 减少清理频率
  agent_timeout: "10m"    # 增加超时时间
```

### 3. Prometheus优化

```yaml
# 抓取间隔
scrape_configs:
  - job_name: 'network-monitoring-hybrid'
    scrape_interval: 30s  # 减少抓取频率
```

## 数据迁移

### 从内存模式迁移到混合方案

1. **停止现有服务**
2. **备份现有数据** (如果需要)
3. **更新配置文件**
4. **启动Redis服务**
5. **启动混合方案服务**

### 从单机部署迁移到容器部署

1. **导出Agent状态**
2. **准备Docker环境**
3. **挂载状态目录**
4. **启动容器服务**

## 最佳实践

### 1. 生产环境部署
- 使用Redis集群提高可用性
- 配置Prometheus告警规则
- 定期备份Agent状态文件
- 监控磁盘空间使用

### 2. 开发环境调试
- 启用debug模式查看详细日志
- 使用较短的保存和上报间隔
- 监控内存和CPU使用情况

### 3. 监控配置
- 设置Agent重启告警
- 监控累计数据一致性
- 配置数据保留策略

## 常见问题

**Q: Agent重启后数据会丢失吗？**
A: 不会。混合方案会自动恢复Agent重启前的累计数据。

**Q: Server重启会影响累计数据吗？**
A: 不会。数据存储在Redis中，Server重启后会自动恢复。

**Q: 如何验证累计数据的正确性？**
A: 可以通过API查看累计指标，并与Prometheus中的数据对比。

**Q: 支持多个Agent同时监控吗？**
A: 支持。每个Agent有独立的状态文件和唯一标识。

**Q: 如何处理Agent ID冲突？**
A: 使用环境变量或配置文件设置唯一的Agent ID。
