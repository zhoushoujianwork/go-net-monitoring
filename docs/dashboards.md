# Grafana Dashboard 展示文档

本文档详细介绍网络流量监控系统的 Grafana Dashboard 功能和使用方法。

## 🎯 Dashboard 概览

系统提供了两个专业级的 Grafana Dashboard，全面支持多设备监控和灵活的数据分析：

| Dashboard | 文件名 | 用途 | 特点 |
|-----------|--------|------|------|
| 网络监控 - 总体概览 | `network-overview-complete.json` | 全局监控视图 | 汇总统计、设备状态、热门域名 |
| 网络监控 - 详细分析 | `network-detailed.json` | 深入分析工具 | 设备选择、网卡过滤、详细统计 |

## 📊 Dashboard 详细介绍

### 1. 网络监控 - 总体概览

#### 🎯 设计目标
为运维人员提供一个全局的网络监控视图，快速了解整体网络状况和识别异常。

#### 📋 面板布局

**全局概览区域**
- **监控设备总数**: 显示当前在线的设备数量
- **全网连接速率**: 所有设备的连接建立速率汇总
- **全网发送速率**: 所有设备的数据发送速率汇总  
- **全网接收速率**: 所有设备的数据接收速率汇总

**网络流量趋势区域**
- **网络流量速率 - 按设备**: 时间序列图，显示每个设备的发送/接收速率
- **网络连接速率 - 按设备**: 时间序列图，显示每个设备的连接建立速率

**设备状态监控区域**
- **设备实时状态表**: 表格显示所有设备的详细信息
  - 主机名、网卡、IP地址、MAC地址
  - 实时连接速率、发送速率、接收速率
  - 在线状态指示

**热门域名访问区域**
- **全网域名访问次数 Top10**: 饼图显示访问最频繁的域名
- **全网域名发送流量 Top10**: 饼图显示流量最大的域名

#### 🔍 使用场景

```bash
# 场景1: 快速健康检查
# 查看总体概览面板，确认：
# - 所有设备都在线
# - 流量速率在正常范围内
# - 没有异常的连接峰值

# 场景2: 异常流量识别
# 通过流量趋势图发现：
# - 某个设备流量异常增长
# - 特定时间段的流量峰值
# - 设备间流量分布不均

# 场景3: 热门域名分析
# 通过域名排行榜了解：
# - 最常访问的外部服务
# - 流量消耗最大的域名
# - 可能的异常访问行为
```

### 2. 网络监控 - 详细分析

#### 🎯 设计目标
为技术人员提供深入的分析工具，支持针对特定设备和网卡进行详细的网络行为分析。

#### 🔧 变量控制器

**主机选择器 (`$host`)**
- 数据源: `label_values(network_interface_info, host)`
- 支持多选和 "All" 选项
- 动态更新可用主机列表

**网卡选择器 (`$interface`)**  
- 数据源: `label_values(network_interface_info{host=~"$host"}, interface)`
- 根据选择的主机动态过滤网卡
- 支持多选和 "All" 选项

#### 📋 面板布局

**概览统计区域**
- **在线设备数**: 根据过滤条件显示设备数量
- **当前连接速率**: 过滤后设备的连接速率汇总
- **当前发送速率**: 过滤后设备的发送速率汇总
- **当前接收速率**: 过滤后设备的接收速率汇总

**设备信息区域**
- **设备网卡信息表**: 显示过滤后设备的详细信息
  - 主机名、网卡、IP地址、MAC地址、在线状态

**网络流量趋势区域**
- **网络流量速率**: 时间序列图，按主机-网卡显示流量趋势
- **网络连接速率**: 时间序列图，按主机-网卡显示连接趋势

**域名访问分析区域**
- **域名访问次数 Top10**: 饼图，过滤后设备的域名访问排行
- **域名发送流量 Top10**: 饼图，过滤后设备的域名流量排行
- **域名访问详细统计表**: 完整的域名统计数据
  - 域名、主机、网卡、访问次数、发送/接收字节数

#### 🔍 使用场景

```bash
# 场景1: 特定设备问题排查
# 1. 选择问题设备: host = "problematic-server"
# 2. 查看该设备的流量趋势
# 3. 分析域名访问模式
# 4. 识别异常的网络行为

# 场景2: 网卡性能分析
# 1. 选择特定网卡: interface = "eth0"
# 2. 对比不同主机上同一网卡的性能
# 3. 识别网卡瓶颈或配置问题

# 场景3: 域名访问行为分析
# 1. 选择目标设备组
# 2. 查看域名访问详细统计表
# 3. 分析访问模式和流量分布
# 4. 识别可疑的外部连接
```

## 🚀 快速开始

### 1. 环境准备

确保以下服务正在运行：
- Network Monitoring Agent (数据采集)
- Network Monitoring Server (数据聚合)
- Prometheus (指标存储)
- Grafana (数据可视化)

```bash
# 启动完整监控栈
make docker-up-monitoring

# 或使用 docker-compose
docker-compose --profile monitoring up -d
```

### 2. 导入 Dashboard

#### 方法一: 自动导入脚本 (推荐)

```bash
cd grafana
./import-dashboards.sh
```

#### 方法二: 手动导入

1. 访问 Grafana: http://localhost:3000
2. 登录 (admin/admin123)
3. 点击 "+" → "Import"
4. 上传 JSON 文件或粘贴内容
5. 配置数据源为 "prometheus"
6. 点击 "Import"

### 3. 配置数据源

确保 Prometheus 数据源配置正确：

```yaml
Name: prometheus
Type: Prometheus
URL: http://prometheus:9090  # Docker 环境
# 或 http://localhost:8080   # 直接访问 Server
Access: Server (default)
```

### 4. 验证数据

```bash
# 检查指标数据
curl http://localhost:8080/metrics | grep network_

# 检查设备信息
curl http://localhost:8080/metrics | grep network_interface_info
```

## 📈 关键指标解读

### 网络基础指标

| 指标 | 含义 | 单位 | 用途 |
|------|------|------|------|
| `network_bytes_sent_total` | 累计发送字节数 | bytes | 流量统计、趋势分析 |
| `network_bytes_received_total` | 累计接收字节数 | bytes | 流量统计、趋势分析 |
| `network_connections_total` | 累计连接数 | count | 连接活跃度分析 |
| `network_interface_info` | 网卡信息 | - | 设备识别、状态监控 |

### 域名相关指标

| 指标 | 含义 | 单位 | 用途 |
|------|------|------|------|
| `network_domains_accessed_total` | 域名访问次数 | count | 访问频率分析 |
| `network_domain_bytes_sent_total` | 域名发送字节数 | bytes | 域名流量分析 |
| `network_domain_bytes_received_total` | 域名接收字节数 | bytes | 域名流量分析 |

### 速率计算

所有速率指标都使用 `rate()` 函数计算：

```promql
# 发送速率 (bytes/second)
rate(network_bytes_sent_total[5m])

# 连接速率 (connections/second)  
rate(network_connections_total[5m])

# 域名访问速率 (accesses/second)
rate(network_domains_accessed_total[5m])
```

## 🎨 自定义和扩展

### 添加新的监控面板

1. **选择可视化类型**
   - Time series: 时间序列数据
   - Stat: 单值统计
   - Table: 表格数据
   - Pie chart: 饼图分布

2. **编写 Prometheus 查询**
   ```promql
   # 示例: 协议分布
   sum by (protocol) (network_connections_total)
   
   # 示例: 设备流量排行
   topk(5, sum by (host) (rate(network_bytes_sent_total[5m])))
   ```

3. **配置面板选项**
   - 设置单位 (bytes, bps, cps)
   - 配置阈值和颜色
   - 添加图例和标签

### 创建告警规则

```promql
# 高流量告警 (>100MB/s)
rate(network_bytes_sent_total[5m]) > 100*1024*1024

# 高连接数告警 (>1000 conn/s)
rate(network_connections_total[5m]) > 1000

# 设备离线告警
up{job="network-monitoring"} == 0
```

### 优化查询性能

1. **使用合适的时间范围**
   ```promql
   # 短期趋势: 5m
   rate(network_bytes_sent_total[5m])
   
   # 长期趋势: 1h
   rate(network_bytes_sent_total[1h])
   ```

2. **限制结果数量**
   ```promql
   # 只显示 Top 10
   topk(10, sum by (domain) (network_domains_accessed_total))
   ```

3. **使用标签过滤**
   ```promql
   # 过滤特定主机
   network_bytes_sent_total{host="target-host"}
   ```

## 🔧 故障排查

### 常见问题及解决方案

#### 1. Dashboard 显示 "No data"

**可能原因**:
- Prometheus 数据源配置错误
- Agent 未正常上报数据
- 时间范围设置问题

**解决方法**:
```bash
# 检查数据源连接
curl http://localhost:9090/api/v1/query?query=up

# 检查 Agent 数据上报
curl http://localhost:8080/metrics | head -20

# 检查指标是否存在
curl http://localhost:8080/metrics | grep network_interface_info
```

#### 2. 变量选择器为空

**可能原因**:
- 指标中缺少对应标签
- 变量查询语句错误
- 数据源配置问题

**解决方法**:
```bash
# 检查标签值
curl -G http://localhost:9090/api/v1/label/host/values

# 测试变量查询
curl -G http://localhost:9090/api/v1/query \
  --data-urlencode 'query=label_values(network_interface_info, host)'
```

#### 3. 图表显示异常

**可能原因**:
- Prometheus 查询语法错误
- 指标名称或标签错误
- 数据类型不匹配

**解决方法**:
1. 在 Grafana Query Inspector 中检查查询
2. 在 Prometheus UI 中测试查询语句
3. 检查指标的数据类型和标签

### 调试工具

```bash
# Grafana 查询调试
# 在面板编辑模式下点击 "Query Inspector"

# Prometheus 查询测试
# 访问 http://localhost:9090/graph

# 指标数据检查
curl -s http://localhost:8080/metrics | grep -E "network_|agent_"

# 标签值查询
curl -G http://localhost:9090/api/v1/label/host/values
curl -G http://localhost:9090/api/v1/label/interface/values
```

## 📚 参考资源

- [Grafana Dashboard 最佳实践](https://grafana.com/docs/grafana/latest/best-practices/)
- [Prometheus 查询语言指南](https://prometheus.io/docs/prometheus/latest/querying/)
- [网络监控指标说明](../README.md#监控指标)
- [Docker Compose 部署指南](docker-compose-usage.md)

---

如有问题或建议，请提交 [Issue](https://github.com/zhoushoujianwork/go-net-monitoring/issues)。
