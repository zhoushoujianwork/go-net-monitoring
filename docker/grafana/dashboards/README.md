# Grafana Dashboard 配置

本目录包含了网络流量监控系统的 Grafana Dashboard 配置文件，支持多设备监控和网卡选择功能。

## 📊 看板列表

### 1. 网络监控 - 总体概览 (`network-overview.json`)

**用途**: 提供所有监控设备的全局视图和汇总统计

**主要功能**:
- 🏠 **全局概览**: 监控设备总数、全网连接速率、发送/接收速率
- 📈 **流量趋势**: 按设备显示网络流量和连接速率趋势
- 🖥️ **设备状态**: 实时显示所有设备的状态、IP、MAC地址和性能指标
- 🌐 **热门域名**: 全网域名访问次数和流量排行榜

**Dashboard UID**: `network-overview`

### 2. 网络监控 - 详细分析 (`network-detailed.json`)

**用途**: 支持选择特定设备和网卡进行深入分析

**主要功能**:
- 🎯 **设备选择**: 支持多选主机和网卡进行过滤
- 📊 **详细统计**: 概览统计、设备信息、流量趋势
- 🔍 **域名分析**: 域名访问次数、流量分布、详细统计表
- 📋 **数据表格**: 完整的域名访问统计，包含访问次数、发送/接收字节数

**Dashboard UID**: `network-detailed`

**变量控制器**:
- `$host` - 主机选择器 (支持多选)
- `$interface` - 网卡选择器 (支持多选，根据主机动态更新)

## 🆕 多设备支持特性

- 🏠 **多 Agent 监控**: 支持同时监控多个设备/主机
- 🔧 **网卡选择**: 支持选择特定网卡进行分析
- 📊 **动态过滤**: 灵活的主机和网卡过滤器
- 🔄 **实时更新**: 自动发现新设备和网卡

## 🚀 自动导入

这些 Dashboard 会在 Docker Compose 启动时自动导入到 Grafana 中：

```bash
# 启动包含 Grafana 的完整监控栈
docker-compose --profile monitoring up -d

# 或使用简化配置
docker-compose -f docker-compose-simple.yml --profile monitoring up -d
```

## 📋 配置文件说明

### dashboard.yml
Dashboard 提供者配置文件，告诉 Grafana 从哪里加载 Dashboard：

```yaml
apiVersion: 1
providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
```

### 数据源配置
数据源配置位于 `../datasources/prometheus.yml`：

```yaml
apiVersion: 1
datasources:
  - name: prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: true
    uid: prometheus
```

**重要说明**: 看板文件中直接使用 `"uid": "prometheus"` 引用数据源，无需使用变量。

## 🔍 访问 Dashboard

启动服务后，可以通过以下地址访问：

- **Grafana 主页**: http://localhost:3000 (admin/admin123)
- **总体概览**: http://localhost:3000/d/network-overview
- **详细分析**: http://localhost:3000/d/network-detailed

## 📈 关键指标

### 网络基础指标
- `network_bytes_sent_total` - 发送字节总数
- `network_bytes_received_total` - 接收字节总数
- `network_connections_total` - 连接总数
- `network_interface_info` - 网卡信息 (包含 IP 和 MAC 地址)

### 域名相关指标
- `network_domains_accessed_total` - 域名访问次数
- `network_domain_bytes_sent_total` - 域名发送字节数
- `network_domain_bytes_received_total` - 域名接收字节数

## 🎨 自定义 Dashboard

如果需要修改 Dashboard：

1. 在 Grafana UI 中编辑 Dashboard
2. 导出 JSON 配置
3. 替换对应的 JSON 文件
4. 重启 Grafana 容器使更改生效

```bash
# 重启 Grafana 容器
docker-compose restart grafana
```

## 🔧 故障排查

### Dashboard 未显示数据

1. **检查数据源配置**:
   - 确保 Prometheus 数据源配置正确
   - URL: `http://prometheus:9090` (Docker 环境)

2. **检查指标数据**:
   ```bash
   # 检查指标是否存在
   curl http://localhost:8080/metrics | grep network_
   ```

3. **检查变量查询**:
   - 在 Dashboard 设置中检查变量查询是否正确
   - 确保 `network_interface_info` 指标存在

### 变量选择器为空

```bash
# 检查标签值
curl -G http://localhost:9090/api/v1/label/host/values
curl -G http://localhost:9090/api/v1/label/interface/values
```

## 📚 相关文档

- [主要文档](../../../README.md)
- [Docker Compose 使用指南](../../../docs/docker-compose-usage.md)
- [Dashboard 展示文档](../../../docs/dashboards.md)

---

如有问题或建议，请提交 [Issue](https://github.com/zhoushoujianwork/go-net-monitoring/issues)。
