package common

import (
	"net"
	"time"
)

// NetworkEvent 网络事件结构
type NetworkEvent struct {
	Timestamp    time.Time `json:"timestamp"`
	Protocol     string    `json:"protocol"`     // tcp, udp, http, https
	Direction    string    `json:"direction"`    // inbound, outbound
	SourceIP     string    `json:"source_ip"`
	SourcePort   int       `json:"source_port"`
	DestIP       string    `json:"dest_ip"`
	DestPort     int       `json:"dest_port"`
	Domain       string    `json:"domain,omitempty"`       // 域名（如果有）
	BytesSent    uint64    `json:"bytes_sent"`
	BytesRecv    uint64    `json:"bytes_received"`
	PacketsSent  uint64    `json:"packets_sent"`
	PacketsRecv  uint64    `json:"packets_received"`
	Duration     time.Duration `json:"duration"`
	Status       string    `json:"status"`       // established, closed, etc.
	ProcessName  string    `json:"process_name,omitempty"`
	ProcessPID   int       `json:"process_pid,omitempty"`
}

// NetworkMetrics 网络指标汇总
type NetworkMetrics struct {
	Timestamp         time.Time            `json:"timestamp"`
	HostID            string               `json:"host_id"`
	Hostname          string               `json:"hostname"`
	TotalConnections  uint64               `json:"total_connections"`
	TotalBytesSent    uint64               `json:"total_bytes_sent"`
	TotalBytesRecv    uint64               `json:"total_bytes_received"`
	TotalPacketsSent  uint64               `json:"total_packets_sent"`
	TotalPacketsRecv  uint64               `json:"total_packets_received"`
	DomainsAccessed   map[string]uint64    `json:"domains_accessed"`   // domain -> count
	IPsAccessed       map[string]uint64    `json:"ips_accessed"`       // ip -> count
	ProtocolStats     map[string]uint64    `json:"protocol_stats"`     // protocol -> count
	PortStats         map[int]uint64       `json:"port_stats"`         // port -> count
	TopProcesses      []ProcessStats       `json:"top_processes"`
	Events            []NetworkEvent       `json:"events,omitempty"`   // 详细事件（可选）
}

// ProcessStats 进程统计
type ProcessStats struct {
	ProcessName   string `json:"process_name"`
	PID           int    `json:"pid"`
	Connections   uint64 `json:"connections"`
	BytesSent     uint64 `json:"bytes_sent"`
	BytesReceived uint64 `json:"bytes_received"`
}

// ConnectionInfo 连接信息
type ConnectionInfo struct {
	LocalAddr  net.Addr      `json:"local_addr"`
	RemoteAddr net.Addr      `json:"remote_addr"`
	Protocol   string        `json:"protocol"`
	State      string        `json:"state"`
	PID        int           `json:"pid"`
	Process    string        `json:"process"`
	StartTime  time.Time     `json:"start_time"`
	BytesSent  uint64        `json:"bytes_sent"`
	BytesRecv  uint64        `json:"bytes_received"`
}

// DNSQuery DNS查询记录
type DNSQuery struct {
	Timestamp time.Time `json:"timestamp"`
	Domain    string    `json:"domain"`
	QueryType string    `json:"query_type"` // A, AAAA, CNAME, etc.
	Response  []string  `json:"response"`   // 解析结果
	Duration  time.Duration `json:"duration"`
	Status    string    `json:"status"`     // success, failed, timeout
}

// HTTPRequest HTTP请求记录
type HTTPRequest struct {
	Timestamp    time.Time `json:"timestamp"`
	Method       string    `json:"method"`
	URL          string    `json:"url"`
	Host         string    `json:"host"`
	UserAgent    string    `json:"user_agent,omitempty"`
	StatusCode   int       `json:"status_code"`
	ResponseSize uint64    `json:"response_size"`
	Duration     time.Duration `json:"duration"`
	RemoteIP     string    `json:"remote_ip"`
}

// ReportRequest 上报请求结构
type ReportRequest struct {
	AgentID   string         `json:"agent_id"`
	Hostname  string         `json:"hostname"`
	Timestamp time.Time      `json:"timestamp"`
	Metrics   NetworkMetrics `json:"metrics"`
}

// ReportResponse 上报响应结构
type ReportResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// AgentInfo Agent信息
type AgentInfo struct {
	ID        string    `json:"id"`
	Hostname  string    `json:"hostname"`
	IP        string    `json:"ip"`
	Version   string    `json:"version"`
	StartTime time.Time `json:"start_time"`
	LastSeen  time.Time `json:"last_seen"`
	Status    string    `json:"status"` // online, offline
}

// AlertRule 告警规则
type AlertRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Metric      string            `json:"metric"`      // 监控指标
	Operator    string            `json:"operator"`    // >, <, >=, <=, ==, !=
	Threshold   float64           `json:"threshold"`   // 阈值
	Duration    time.Duration     `json:"duration"`    // 持续时间
	Labels      map[string]string `json:"labels"`      // 标签匹配
	Enabled     bool              `json:"enabled"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// Alert 告警事件
type Alert struct {
	ID          string            `json:"id"`
	RuleID      string            `json:"rule_id"`
	RuleName    string            `json:"rule_name"`
	Level       string            `json:"level"`       // info, warning, critical
	Message     string            `json:"message"`
	Labels      map[string]string `json:"labels"`
	Value       float64           `json:"value"`
	Threshold   float64           `json:"threshold"`
	Status      string            `json:"status"`      // firing, resolved
	StartsAt    time.Time         `json:"starts_at"`
	EndsAt      *time.Time        `json:"ends_at,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}
