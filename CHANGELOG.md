# 更新日志

## [1.0.0] - 2025-07-01

### ✨ 新增功能
- **按域名流量统计**: 实现了按域名统计发送/接收字节数、连接数的功能
- **域名访问统计**: 统计每个域名的访问次数
- **实时网络监控**: 基于BPF的高性能数据包捕获
- **DNS解析**: 自动解析IP地址到域名，支持DNS缓存
- **Prometheus集成**: 内置Prometheus指标导出
- **分布式架构**: Agent/Server架构，支持多节点部署

### 📊 监控指标
- `network_domains_accessed_total` - 域名访问次数统计
- `network_domain_bytes_sent_total` - 按域名统计发送字节数
- `network_domain_bytes_received_total` - 按域名统计接收字节数
- `network_domain_connections_total` - 按域名统计连接数
- `network_connections_total` - 网络连接总数
- `network_bytes_sent_total` - 发送字节总数
- `network_bytes_received_total` - 接收字节总数
- `network_protocol_stats` - 协议统计
- `network_ips_accessed_total` - IP访问统计

### 🔧 技术特性
- 智能过滤规则 (端口、IP、协议等多维度过滤)
- 可配置的监控规则和上报间隔
- 高性能数据收集和处理
- 结构化日志和完善的错误处理
- 资源清理和优雅关闭

### 🏗️ 系统架构
```
Agent (数据采集) -> Server (数据聚合) -> Prometheus (指标导出) -> Grafana (可视化)
```

### 📝 文档
- 完整的README文档
- 配置文件示例和说明
- 快速开始指南
- Grafana集成说明
- MIT开源许可证

### 🛠️ 开发工具
- Makefile构建脚本
- 调试和测试脚本
- Docker支持 (计划中)
- CI/CD配置 (计划中)

### 🎯 使用场景
- 网络流量监控和分析
- 域名访问行为分析
- 网络安全监控
- 性能优化和故障排查
- 合规性监控

---

## 下一步计划

### v1.1.0 (计划中)
- [ ] Docker容器化支持
- [ ] 数据持久化存储
- [ ] 告警规则配置
- [ ] Web管理界面
- [ ] 更多协议支持

### v1.2.0 (计划中)
- [ ] 集群部署支持
- [ ] 数据导出功能
- [ ] 性能优化
- [ ] 更多可视化图表
