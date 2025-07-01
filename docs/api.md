# API文档

## 概述

网络监控Server提供RESTful API接口，用于接收Agent上报的数据和查询监控信息。

## 基础信息

- **Base URL**: `http://localhost:8080`
- **Content-Type**: `application/json`
- **认证**: 暂不支持（后续版本会添加）

## API接口

### 1. 指标上报

**POST** `/api/v1/metrics`

Agent向Server上报网络监控指标。

#### 请求头
```
Content-Type: application/json
X-Agent-ID: agent-unique-id
X-Hostname: hostname
```

#### 请求体
```json
{
  "agent_id": "hostname-1234567890",
  "hostname": "web-server-01",
  "timestamp": "2024-01-01T12:00:00Z",
  "metrics": {
    "timestamp": "2024-01-01T12:00:00Z",
    "host_id": "host-001",
    "hostname": "web-server-01",
    "total_connections": 150,
    "total_bytes_sent": 1048576,
    "total_bytes_received": 2097152,
    "total_packets_sent": 1000,
    "total_packets_received": 1500,
    "domains_accessed": {
      "google.com": 10,
      "github.com": 5,
      "stackoverflow.com": 3
    },
    "ips_accessed": {
      "8.8.8.8": 15,
      "1.1.1.1": 8,
      "192.168.1.1": 20
    },
    "protocol_stats": {
      "tcp": 120,
      "udp": 30,
      "http": 80,
      "https": 70
    },
    "port_stats": {
      "80": 50,
      "443": 70,
      "53": 30
    },
    "top_processes": [
      {
        "process_name": "nginx",
        "pid": 1234,
        "connections": 50,
        "bytes_sent": 524288,
        "bytes_received": 1048576
      }
    ]
  }
}
```

#### 响应
```json
{
  "success": true,
  "message": "Metrics received successfully",
  "timestamp": "2024-01-01T12:00:01Z"
}
```

#### 状态码
- `200 OK`: 成功接收指标
- `400 Bad Request`: 请求格式错误
- `500 Internal Server Error`: 服务器内部错误

### 2. 心跳上报

**POST** `/api/v1/heartbeat`

Agent向Server发送心跳，表明Agent在线状态。

#### 请求头
```
Content-Type: application/json
X-Agent-ID: agent-unique-id
```

#### 请求体
```json
{
  "id": "hostname-1234567890",
  "hostname": "web-server-01",
  "ip": "192.168.1.100",
  "version": "1.0.0",
  "start_time": "2024-01-01T10:00:00Z",
  "last_seen": "2024-01-01T12:00:00Z",
  "status": "online"
}
```

#### 响应
```json
{
  "success": true,
  "timestamp": "2024-01-01T12:00:01Z"
}
```

### 3. 查询Agent列表

**GET** `/api/v1/agents`

查询所有已注册的Agent信息。

#### 响应
```json
{
  "agents": [
    {
      "id": "hostname-1234567890",
      "hostname": "web-server-01",
      "ip": "192.168.1.100",
      "version": "1.0.0",
      "start_time": "2024-01-01T10:00:00Z",
      "last_seen": "2024-01-01T12:00:00Z",
      "status": "online"
    },
    {
      "id": "hostname-0987654321",
      "hostname": "db-server-01",
      "ip": "192.168.1.101",
      "version": "1.0.0",
      "start_time": "2024-01-01T09:30:00Z",
      "last_seen": "2024-01-01T11:58:00Z",
      "status": "offline"
    }
  ],
  "count": 2
}
```

### 4. 查询服务状态

**GET** `/api/v1/status`

查询Server运行状态。

#### 响应
```json
{
  "status": "running",
  "timestamp": "2024-01-01T12:00:00Z",
  "agent_count": 5,
  "version": "1.0.0"
}
```

### 5. 健康检查

**GET** `/health`

服务健康检查接口。

#### 响应
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 6. Prometheus指标

**GET** `/metrics`

获取Prometheus格式的监控指标。

#### 响应
```
# HELP network_connections_total Total number of network connections
# TYPE network_connections_total counter
network_connections_total{protocol="tcp",direction="outbound",host="web-server-01"} 120
network_connections_total{protocol="udp",direction="outbound",host="web-server-01"} 30

# HELP network_bytes_sent_total Total bytes sent over network
# TYPE network_bytes_sent_total counter
network_bytes_sent_total{protocol="tcp",destination="8.8.8.8",host="web-server-01"} 1048576

# HELP network_domains_accessed_total Total number of domains accessed
# TYPE network_domains_accessed_total counter
network_domains_accessed_total{domain="google.com",host="web-server-01"} 10
network_domains_accessed_total{domain="github.com",host="web-server-01"} 5
```

## 错误处理

### 错误响应格式
```json
{
  "success": false,
  "message": "Error description",
  "error_code": "ERROR_CODE",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 常见错误码
- `INVALID_REQUEST`: 请求格式错误
- `MISSING_AGENT_ID`: 缺少Agent ID
- `STORAGE_ERROR`: 存储错误
- `INTERNAL_ERROR`: 内部服务器错误

## 限流和配额

目前版本暂不支持限流，后续版本会添加：
- 每个Agent的上报频率限制
- API调用次数限制
- 数据存储配额限制

## 认证和授权

当前版本不支持认证，所有API接口都是公开的。生产环境建议：
- 使用防火墙限制访问来源
- 通过反向代理添加认证
- 使用VPN或内网部署

## SDK和客户端

### Go客户端示例
```go
package main

import (
    "bytes"
    "encoding/json"
    "net/http"
    "time"
)

type Client struct {
    baseURL string
    client  *http.Client
}

func NewClient(baseURL string) *Client {
    return &Client{
        baseURL: baseURL,
        client:  &http.Client{Timeout: 10 * time.Second},
    }
}

func (c *Client) ReportMetrics(metrics NetworkMetrics) error {
    data, _ := json.Marshal(metrics)
    resp, err := c.client.Post(
        c.baseURL+"/api/v1/metrics",
        "application/json",
        bytes.NewBuffer(data),
    )
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    return nil
}
```

### curl示例
```bash
# 上报指标
curl -X POST http://localhost:8080/api/v1/metrics \
  -H "Content-Type: application/json" \
  -H "X-Agent-ID: test-agent" \
  -d '{"agent_id":"test-agent","hostname":"test-host","timestamp":"2024-01-01T12:00:00Z","metrics":{}}'

# 查询Agent列表
curl http://localhost:8080/api/v1/agents

# 健康检查
curl http://localhost:8080/health
```

## 版本兼容性

- **v1.0.0**: 初始版本
- **v1.1.0**: 添加认证支持（计划中）
- **v1.2.0**: 添加限流支持（计划中）

API版本通过URL路径中的版本号进行管理，如 `/api/v1/`。
