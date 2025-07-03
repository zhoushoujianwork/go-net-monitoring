# 网卡信息支持实现指南

## 概述

为了支持多网卡环境下的精确流量监控，需要在所有网络指标中添加 `interface` 标签，以区分不同网卡的流量统计。

## 当前问题

当前指标格式：
```
network_domain_bytes_sent_total{domain="git.patsnap.com",host="zhoushoujianworkspace"} 0
```

期望的指标格式：
```
network_domain_bytes_sent_total{domain="git.patsnap.com",host="zhoushoujianworkspace",interface="eth0"} 5604
network_domain_bytes_sent_total{domain="git.patsnap.com",host="zhoushoujianworkspace",interface="docker0"} 5570
network_domain_bytes_sent_total{domain="git.patsnap.com",host="zhoushoujianworkspace",interface="br-68b956877dd2"} 10578
```

## 检测到的网卡信息

系统中检测到的主要网卡：

| 网卡名称 | 类型 | IP地址 | 状态 | 建议监控 |
|---------|------|--------|------|----------|
| eth0 | 以太网接口 | 192.168.3.233 | UP | ✅ 是 |
| docker0 | Docker网桥 | 172.17.0.1 | UP | ✅ 是 |
| br-68b956877dd2 | Docker Compose网桥 | 172.28.0.1 | UP | ✅ 是 |
| br-49fd65f6b37e | Docker网桥 | 172.26.0.1 | UP | ⚠️ 可选 |
| br-a230bfa5c9d9 | Docker网桥 | 172.10.0.1 | UP | ⚠️ 可选 |
| veth* | 虚拟网卡对 | - | UP | ❌ 否 |
| lo | 回环接口 | 127.0.0.1 | UP | ❌ 否 |

## 实现步骤

### 1. 修改指标定义 (pkg/metrics/metrics.go)

需要在所有网络相关指标中添加 `interface` 标签：

```go
// 修改前
NetworkDomainsAccessedTotal: promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "network_domains_accessed_total",
        Help: "Total number of domains accessed",
    },
    []string{"domain", "host"},
),

// 修改后
NetworkDomainsAccessedTotal: promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "network_domains_accessed_total",
        Help: "Total number of domains accessed",
    },
    []string{"domain", "host", "interface"},
),
```

### 2. 集成网卡检测器 (internal/agent/agent.go)

```go
import "go-net-monitoring/pkg/network"

// 在Agent初始化时添加
detector := network.NewInterfaceDetector(network.InterfaceConfig{
    IncludeLoopback: false,
    IncludeDocker:   true,
    IncludeVirtual:  false,
    AutoDetect:      true,
    Whitelist:       []string{"eth0", "docker0", "br-"},
    Blacklist:       []string{"veth", "lo"},
})

interfaces, err := detector.DetectInterfaces()
```

### 3. 修改数据收集器 (pkg/collector/collector.go)

支持多网卡监控：

```go
// 为每个网卡创建独立的监控协程
for _, iface := range interfaces {
    go c.monitorInterface(iface.Name)
}

func (c *Collector) monitorInterface(interfaceName string) {
    // 在数据包处理时记录网卡信息
    // 更新指标时包含interface标签
    metrics.UpdateDomainMetrics(domain, host, interfaceName, bytesSent, bytesReceived, connections)
}
```

### 4. 更新配置结构 (internal/config/config.go)

```go
type MonitorConfig struct {
    Interface  string   `yaml:"interface"`  // 兼容单网卡
    Interfaces []string `yaml:"interfaces"` // 支持多网卡
    InterfaceConfig InterfaceConfig `yaml:"interface_config"`
    // ... 其他配置
}

type InterfaceConfig struct {
    IncludeLoopback bool     `yaml:"include_loopback"`
    IncludeDocker   bool     `yaml:"include_docker"`
    IncludeVirtual  bool     `yaml:"include_virtual"`
    AutoDetect      bool     `yaml:"auto_detect"`
    Whitelist       []string `yaml:"whitelist"`
    Blacklist       []string `yaml:"blacklist"`
}
```

### 5. 更新Server处理逻辑 (internal/server/server.go)

确保Server能正确处理带网卡信息的指标数据。

## 配置文件

使用 `configs/agent-with-interfaces.yaml` 配置文件：

```yaml
monitor:
  interfaces:
    - "eth0"                    # 主网卡
    - "docker0"                 # Docker网桥
    - "br-68b956877dd2"         # Docker Compose网桥
  
  interface_config:
    include_loopback: false
    include_docker: true
    include_virtual: false
    auto_detect: true
    whitelist: ["eth0", "docker0", "br-"]
    blacklist: ["veth", "lo"]
```

## 期望的指标输出

实现后，指标将包含网卡信息：

```prometheus
# 域名访问统计 (按网卡分组)
network_domains_accessed_total{domain="git.patsnap.com",host="zhoushoujianworkspace",interface="eth0"} 63
network_domains_accessed_total{domain="git.patsnap.com",host="zhoushoujianworkspace",interface="docker0"} 12
network_domains_accessed_total{domain="git.patsnap.com",host="zhoushoujianworkspace",interface="br-68b956877dd2"} 69

# 域名流量统计 (按网卡分组)
network_domain_bytes_sent_total{domain="git.patsnap.com",host="zhoushoujianworkspace",interface="eth0"} 5604
network_domain_bytes_sent_total{domain="git.patsnap.com",host="zhoushoujianworkspace",interface="docker0"} 5570
network_domain_bytes_sent_total{domain="git.patsnap.com",host="zhoushoujianworkspace",interface="br-68b956877dd2"} 10578

# 网卡级别统计
network_interface_uptime_seconds{interface="eth0",host="zhoushoujianworkspace"} 25432
network_interface_speed_mbps{interface="eth0",host="zhoushoujianworkspace"} 1000
```

## Grafana查询示例

实现后可以使用以下查询：

```promql
# 按网卡查看域名访问Top10
topk(10, sum by (domain, interface) (network_domains_accessed_total))

# 特定网卡的流量统计
sum by (domain) (network_domain_bytes_sent_total{interface="eth0"})

# 网卡流量对比
sum by (interface) (rate(network_bytes_sent_total[5m]))

# 容器网络流量 (Docker网桥)
sum by (domain) (network_domain_bytes_sent_total{interface=~"docker0|br-.*"})
```

## 构建和部署

1. 修改代码后重新构建：
```bash
make build
docker build -t zhoushoujian/go-net-monitoring:interface-support .
```

2. 更新docker-compose.yml使用新镜像：
```yaml
services:
  agent:
    image: zhoushoujian/go-net-monitoring:interface-support
    volumes:
      - ./configs/agent-with-interfaces.yaml:/app/configs/agent.yaml:ro
```

3. 重启服务：
```bash
docker-compose down
docker-compose up -d
```

## 验证

实现后验证指标：

```bash
# 检查是否包含interface标签
curl -s http://localhost:8080/metrics | grep "interface=" | head -10

# 检查特定网卡的指标
curl -s http://localhost:8080/metrics | grep 'interface="eth0"' | head -5
```

## 注意事项

1. **性能影响**: 监控多个网卡会增加系统开销，建议只监控必要的网卡
2. **标签基数**: 添加interface标签会增加指标的基数，注意Prometheus存储影响
3. **兼容性**: 保持向后兼容，支持单网卡配置
4. **网卡变化**: 考虑网卡动态添加/删除的情况
5. **权限要求**: 监控网卡可能需要额外的系统权限

## 优先级建议

1. **高优先级**: eth0 (主网卡) - 承载主要的外网流量
2. **中优先级**: docker0, br-* (Docker网桥) - 容器间通信
3. **低优先级**: veth* (虚拟网卡对) - 通常流量较少且数量多
