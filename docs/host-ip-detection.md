# 主机IP检测功能

## 概述

网络流量监控系统现在支持主机IP检测功能，可以自动识别容器环境并获取主机IP地址。这个功能通过在 `network_interface_info` 指标中添加 `host_ip_address` 标签来实现，帮助用户区分虚拟机和物理机环境。

## 功能特性

### 🔍 自动环境检测
- **容器检测**: 自动识别Docker、Podman、LXC等容器环境
- **虚拟机检测**: 识别VMware、VirtualBox、KVM、QEMU等虚拟化环境
- **主机IP获取**: 获取容器宿主机的真实IP地址

### 📊 指标增强
- **新增标签**: `host_ip_address` - 主机IP地址（仅在容器环境中有值）
- **向后兼容**: 保持原有指标格式不变
- **智能填充**: 非容器环境中该标签为空字符串

## 指标格式

### 更新后的 network_interface_info 指标

```prometheus
# HELP network_interface_info Network interface information with IP address, MAC address and host IP address
# TYPE network_interface_info gauge
network_interface_info{host="container_id",host_ip_address="192.168.1.100",interface="eth0",ip_address="172.17.0.2",mac_address="02:42:ac:11:00:02"} 1
```

### 标签说明

| 标签名 | 描述 | 示例值 |
|--------|------|--------|
| `host` | 容器ID或主机名 | `4a0e1a7a7fc4` |
| `interface` | 网络接口名称 | `eth0` |
| `ip_address` | 接口IP地址 | `172.17.0.2` |
| `mac_address` | MAC地址 | `02:42:ac:11:00:02` |
| `host_ip_address` | 主机IP地址 | `192.168.1.100` |

## 使用场景

### 1. 区分虚拟机和物理机

```promql
# 查询所有容器/虚拟机网卡信息
network_interface_info{host_ip_address!=""}

# 查询物理机网卡信息
network_interface_info{host_ip_address=""}
```

### 2. 按主机IP分组统计

```promql
# 统计每个主机上的容器数量
count by (host_ip_address) (network_interface_info{host_ip_address!=""})

# 查看特定主机上的所有容器
network_interface_info{host_ip_address="192.168.1.100"}
```

### 3. 网络拓扑分析

```promql
# 分析容器网络分布
group by (host_ip_address, ip_address) (network_interface_info)

# 查找跨主机通信
network_domains_accessed_total * on(host) group_left(host_ip_address) network_interface_info
```

## 检测逻辑

### 容器环境检测

系统通过以下方法检测容器环境：

1. **文件检测**
   - 检查 `/.dockerenv` 文件（Docker）
   - 检查 `/proc/1/cgroup` 内容
   - 检查 `/proc/self/mountinfo` 内容

2. **环境变量检测**
   - 检查 `container` 环境变量

3. **支持的容器类型**
   - Docker
   - Podman
   - Containerd
   - LXC

### 虚拟机环境检测

系统通过以下方法检测虚拟机环境：

1. **DMI信息检测**
   - 检查 `/sys/class/dmi/id/product_name`
   - 检查 `/sys/class/dmi/id/sys_vendor`

2. **CPU信息检测**
   - 检查 `/proc/cpuinfo` 中的hypervisor标志

3. **支持的虚拟化类型**
   - VMware
   - VirtualBox
   - KVM/QEMU
   - Xen
   - Hyper-V

### 主机IP获取

系统使用多种方法获取主机IP：

1. **UDP连接法**: 通过连接外部地址获取本地IP
2. **路由表法**: 从 `/proc/net/route` 获取默认路由对应的IP
3. **接口遍历法**: 获取第一个非回环的IPv4地址

## 配置选项

目前主机IP检测功能是自动启用的，无需额外配置。系统会在启动时自动检测环境并获取相关信息。

## 性能影响

- **检测开销**: 主机检测仅在启动时执行一次，运行时开销极小
- **内存使用**: 增加的内存使用量可忽略不计
- **网络开销**: 无额外网络开销

## 故障排除

### 常见问题

1. **host_ip_address 为空**
   - 原因：系统未检测到容器环境
   - 解决：检查容器运行环境，确认是否在容器中运行

2. **检测到错误的主机IP**
   - 原因：网络配置复杂或多网卡环境
   - 解决：检查路由表和网络配置

3. **虚拟机检测失败**
   - 原因：虚拟化平台不在支持列表中
   - 解决：检查 `/sys/class/dmi/id/` 下的文件内容

### 调试方法

1. **启用调试日志**
   ```bash
   # 启动调试模式
   make docker-up-debug
   
   # 查看主机检测日志
   docker logs netmon-server | grep "Host detection"
   ```

2. **手动测试检测功能**
   ```bash
   # 运行测试脚本
   go run test-host-detection.go
   ```

3. **验证指标输出**
   ```bash
   # 检查指标格式
   curl -s http://localhost:8080/metrics | grep network_interface_info
   ```

## 示例输出

### 容器环境中的指标

```prometheus
network_interface_info{host="e8f91bc76c0e",host_ip_address="192.168.1.100",interface="eth0",ip_address="172.26.0.5",mac_address="02:42:ac:1a:00:05"} 1
```

### 物理机环境中的指标

```prometheus
network_interface_info{host="physical-server",host_ip_address="",interface="eth0",ip_address="192.168.1.100",mac_address="aa:bb:cc:dd:ee:ff"} 1
```

## 更新日志

- **v1.0.0**: 初始版本，支持基本的主机IP检测
- **v1.1.0**: 增加虚拟机环境检测
- **v1.2.0**: 优化IP获取逻辑，支持多种检测方法

## 相关文档

- [网络监控指标说明](metrics.md)
- [Docker部署指南](docker-deployment.md)
- [Prometheus集成指南](prometheus-integration.md)
