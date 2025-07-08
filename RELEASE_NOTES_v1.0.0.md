# Go 网络监控系统 v1.0.0 发布说明

🎉 **正式版本发布！** 经过全面重构和优化，Go 网络监控系统 v1.0.0 正式发布！

## 🚀 版本亮点

### 🔥 全新 eBPF 架构
- **高性能监控**: 基于 eBPF/XDP 的内核级网络数据包捕获
- **零拷贝技术**: 最小化性能开销，支持高并发网络监控
- **智能路径配置**: 自动检测和回退的 eBPF 程序加载机制

### 🐳 完善的容器化支持
- **一键部署**: `./run.sh` 脚本快速启动完整监控栈
- **Docker Compose**: 生产级编排配置，支持多种部署模式
- **多平台支持**: Linux/macOS/Windows 容器环境兼容

### 📊 强大的监控能力
- **实时流量监控**: TCP/UDP/HTTP/HTTPS 协议全覆盖
- **域名智能解析**: 自动 DNS 解析和访问统计
- **多维度统计**: 按域名、协议、网卡的详细流量分析
- **网卡信息检测**: 自动识别容器环境并获取主机 IP

### 📈 专业级可视化
- **Prometheus 集成**: 完整的指标导出和时序数据存储
- **Grafana Dashboard**: 多设备监控和灵活数据分析面板
- **实时告警**: 支持自定义告警规则和通知

## 📋 主要功能

### 网络监控指标
- `network_domains_accessed_total` - 域名访问次数统计
- `network_domain_bytes_sent/received_total` - 按域名的流量统计
- `network_connections_total` - 网络连接总数
- `network_interface_info` - 网卡信息（IP、MAC、主机IP）
- `network_protocol_stats_total` - 协议分布统计

### 部署方式
```bash
# 快速启动
./run.sh

# Docker Compose
docker-compose up -d
docker-compose --profile monitoring up -d

# 独立容器
docker run -d --name netmon-agent --privileged --network host \
  -e COMPONENT=agent -e SERVER_URL=http://server:8080/api/v1/metrics \
  zhoushoujian/go-net-monitoring:latest
```

### 访问地址
- **Server API**: http://localhost:8080
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin123)

## 🔧 技术规格

### 系统要求
- **Docker**: ≥ 20.10
- **内存**: ≥ 512MB
- **操作系统**: Linux/macOS/Windows
- **权限**: 需要特权模式进行网络监控

### 性能指标
- **CPU 使用率**: < 5% (正常负载)
- **内存占用**: < 100MB (Agent + Server)
- **网络延迟**: < 1ms 额外开销
- **数据上报**: 10秒间隔 (可配置)

## 📚 文档资源

### 部署指南
- [快速开始](docs/ebpf-quick-start.md)
- [Docker 部署](docs/docker-deployment.md)
- [容器 Agent 部署](docs/html/container-agent-deployment.html)
- [Kubernetes 部署](docs/kubernetes-deployment.md)

### 配置文档
- [eBPF 路径配置](docs/ebpf-path-configuration.md)
- [配置文件说明](docs/configuration.md)
- [环境变量配置](docs/docker-compose-usage.md)

### 监控文档
- [Dashboard 展示](docs/dashboards.md)
- [指标说明](docs/api.md)
- [故障排查](docs/html/container-agent-deployment.html#troubleshooting)

## 🛠️ 开发者信息

### 构建信息
- **Go 版本**: 1.21+
- **构建时间**: 2025-07-08
- **Git 提交**: 5d9d63a
- **镜像大小**: ~45MB (优化后)

### 项目结构
```
go-net-monitoring/
├── cmd/agent-ebpf/     # eBPF Agent 主程序
├── cmd/server/         # Server 主程序
├── pkg/ebpf/          # eBPF 相关包
├── bpf/programs/      # eBPF 程序源码
├── docs/html/         # HTML 文档
├── docker/            # Docker 相关文件
└── scripts/           # 构建和测试脚本
```

## 🔄 升级说明

### 从旧版本升级
如果你使用的是基于 libpcap 的旧版本，请注意：

1. **架构变更**: 新版本完全基于 eBPF，性能更优
2. **配置变更**: 配置文件格式有所调整
3. **部署方式**: 推荐使用容器化部署
4. **指标变更**: 新增了网卡信息等指标

### 迁移步骤
```bash
# 1. 停止旧版本
docker-compose down

# 2. 备份配置和数据
cp -r configs/ configs.backup/
cp -r data/ data.backup/

# 3. 更新到新版本
git pull origin main
git checkout v1.0.0

# 4. 启动新版本
./run.sh
```

## 🐛 已知问题

1. **macOS 限制**: 在 macOS 上 eBPF 功能受限，会自动回退到模拟模式
2. **权限要求**: Agent 需要特权模式，请确保在可信环境中运行
3. **网络接口**: 某些虚拟网络接口可能无法正确监控

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request！

### 开发环境
```bash
# 克隆项目
git clone https://github.com/zhoushoujianwork/go-net-monitoring.git
cd go-net-monitoring

# 构建和测试
make build-linux
./scripts/test-integration.sh
```

### 反馈渠道
- **GitHub Issues**: 报告 Bug 和功能请求
- **GitHub Discussions**: 技术讨论和使用交流
- **Pull Requests**: 代码贡献

## 📞 支持

如果遇到问题：

1. 查看 [故障排查文档](docs/html/container-agent-deployment.html#troubleshooting)
2. 搜索 [GitHub Issues](https://github.com/zhoushoujianwork/go-net-monitoring/issues)
3. 创建新的 [Issue](https://github.com/zhoushoujianwork/go-net-monitoring/issues/new)

---

🎉 **感谢使用 Go 网络监控系统！** 

⭐ 如果这个项目对你有帮助，请给个 Star 支持一下！

**发布时间**: 2025-07-08  
**版本**: v1.0.0  
**维护者**: zhoushoujianwork
