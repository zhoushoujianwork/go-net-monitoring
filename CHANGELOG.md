# 更新日志

## [1.1.1] - 2025-07-02

### 🔧 Docker 部署简化
- **简化部署模式**: 移除多余的部署配置，只保留 Redis 存储（推荐）和内存存储（备选）
- **默认 Redis 部署**: `docker-compose up -d` 直接启动 Redis + Server + Agent
- **内存存储备选**: `docker-compose --profile memory up -d server-memory agent` 启动内存存储模式
- **统一配置管理**: 简化配置文件结构，减少维护复杂度

### 📝 文档更新
- **简化部署指南**: 更新 README.md 和 Docker Compose 使用指南
- **清晰的端口分配**: Server (Redis: 8080), Server (Memory: 8081)
- **新增部署测试脚本**: `test-docker-deployment.sh` 用于验证部署

### 🐛 配置修复
- **修复 Redis 主机地址**: server-redis.yaml 中使用正确的 Docker 服务名 `redis`
- **优化服务依赖**: 确保服务启动顺序正确

### 🚀 简化后的部署方式

**默认部署 (Redis 存储):**
```bash
docker-compose up -d
```

**内存存储模式:**
```bash
docker-compose --profile memory up -d server-memory agent
```

**完整监控栈:**
```bash
docker-compose --profile monitoring up -d
```

---

## [1.1.0] - 2025-07-02

### ✨ 新增功能
- **Debug 模式支持**: 添加了完整的 debug 模式，支持详细日志输出和路由信息打印
- **Docker Compose 重构**: 完全重构了 Docker Compose 配置，基于 YAML 配置文件
- **多部署模式**: 支持基础部署、Debug 模式、Redis 存储等多种部署方式
- **配置文件管理**: 统一使用 `configs/` 目录下的 YAML 配置文件
- **网络监控测试**: 添加了 `/proc/net` 监控示例和测试脚本

### 🔧 技术改进
- **Docker 配置优化**: 修复了 `docker/entrypoint.sh` 的配置生成逻辑
- **服务发现**: 使用 Docker 服务名简化网络配置
- **健康检查**: 为所有服务添加了健康检查机制
- **数据持久化**: 完整的数据持久化方案，支持 Redis、Prometheus、Grafana 数据持久化

### 📝 文档更新
- **Docker Compose 使用指南**: 新增详细的 Docker Compose 使用文档
- **Debug 模式文档**: 添加 Debug 模式使用指南
- **配置文件说明**: 完善了各种配置文件的说明和示例
- **Docker 配置总结**: 添加了 Docker 配置修改的详细总结

### 🐛 问题修复
- 修复了 Docker 容器中配置文件结构不匹配的问题
- 修复了环境变量配置生成的逻辑错误
- 修复了服务间网络连接配置问题
- 优化了 `.gitignore` 配置，排除不必要的文件

### 🚀 部署方式

**基础部署:**
```bash
docker-compose up -d server agent
```

**Debug 模式:**
```bash
docker-compose --profile debug up -d server-debug agent-debug
```

**Redis 存储:**
```bash
docker-compose --profile redis up -d redis server-redis agent
```

**完整监控栈:**
```bash
docker-compose --profile monitoring --profile redis up -d
```

### 📊 服务端口分配
- Server (默认): http://localhost:8080
- Server (Debug): http://localhost:8081  
- Server (Redis): http://localhost:8082
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin123)

---

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
