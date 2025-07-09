package common

import (
	"time"
)

// MetricsReport 指标报告结构
type MetricsReport struct {
	// Agent信息
	AgentID     string    `json:"agent_id"`
	StartupTime time.Time `json:"startup_time"`
	ReportTime  time.Time `json:"report_time"`
	ReportMode  string    `json:"report_mode"` // 固定为 "incremental"

	// 增量数据（自上次上报以来的变化）
	DeltaStats map[string]DomainMetrics `json:"delta_stats"`

	// 系统信息
	SystemInfo SystemMetrics `json:"system_info"`

	// 元数据
	Metadata ReportMetadata `json:"metadata"`
}

// DomainMetrics 域名指标
type DomainMetrics struct {
	Domain          string    `json:"domain"`
	AccessCount     int64     `json:"access_count"`
	BytesSent       int64     `json:"bytes_sent"`
	BytesReceived   int64     `json:"bytes_received"`
	ConnectionCount int64     `json:"connection_count"`
	LastAccessTime  time.Time `json:"last_access_time"`

	// 协议分布
	ProtocolStats map[string]int64 `json:"protocol_stats,omitempty"`

	// 端口分布
	PortStats map[int]int64 `json:"port_stats,omitempty"`
}

// SystemMetrics 系统指标
type SystemMetrics struct {
	Hostname           string `json:"hostname"`
	Interface          string `json:"interface"`
	TotalConnections   int64  `json:"total_connections"`
	TotalBytesSent     int64  `json:"total_bytes_sent"`
	TotalBytesReceived int64  `json:"total_bytes_received"`
	ActiveDomains      int    `json:"active_domains"`
	ReportInterval     string `json:"report_interval"`
	UptimeSeconds      int64  `json:"uptime_seconds"`
}

// ReportMetadata 报告元数据
type ReportMetadata struct {
	Version            string            `json:"version"`
	ConfigHash         string            `json:"config_hash,omitempty"`
	Tags               map[string]string `json:"tags,omitempty"`
	PersistenceEnabled bool              `json:"persistence_enabled"`
	RestartDetected    bool              `json:"restart_detected,omitempty"`
}

// ServerMetricsStorage Server端指标存储
type ServerMetricsStorage struct {
	// 原始数据（Agent上报的）
	RawMetrics map[string]MetricsReport `json:"raw_metrics"`

	// 累计数据（Server计算的全局累计）
	GlobalCumulative map[string]DomainMetrics `json:"global_cumulative"`

	// Agent重启基线
	RestartBaselines map[string]map[string]DomainMetrics `json:"restart_baselines"`

	// Agent状态跟踪
	AgentStates map[string]AgentState `json:"agent_states"`

	// 最后更新时间
	LastUpdated time.Time `json:"last_updated"`
}

// AgentState Agent状态信息
type AgentState struct {
	AgentID         string    `json:"agent_id"`
	LastStartupTime time.Time `json:"last_startup_time"`
	LastReportTime  time.Time `json:"last_report_time"`
	RestartCount    int       `json:"restart_count"`
	IsActive        bool      `json:"is_active"`
	LastHeartbeat   time.Time `json:"last_heartbeat"`
}

// CumulativeMetrics 累计指标计算结果
type CumulativeMetrics struct {
	DomainStats map[string]DomainMetrics `json:"domain_stats"`
	SystemStats SystemMetrics            `json:"system_stats"`
	GeneratedAt time.Time                `json:"generated_at"`
	DataSources []string                 `json:"data_sources"` // 参与计算的Agent列表
}

// MetricsAggregator 指标聚合器接口
type MetricsAggregator interface {
	// 处理Agent上报的指标
	ProcessMetrics(report MetricsReport) error

	// 获取累计指标
	GetCumulativeMetrics() (*CumulativeMetrics, error)

	// 获取指定域名的累计指标
	GetDomainMetrics(domain string) (*DomainMetrics, error)

	// 检测Agent重启
	DetectAgentRestart(agentID string, report MetricsReport) bool

	// 清理过期数据
	CleanupExpiredData(maxAge time.Duration) error
}

// RestartDetectionResult 重启检测结果
type RestartDetectionResult struct {
	IsRestart       bool      `json:"is_restart"`
	PreviousStartup time.Time `json:"previous_startup,omitempty"`
	CurrentStartup  time.Time `json:"current_startup"`
	RestartCount    int       `json:"restart_count"`
}

// MetricsDelta 指标增量计算
type MetricsDelta struct {
	Domain             string        `json:"domain"`
	AccessDelta        int64         `json:"access_delta"`
	BytesSentDelta     int64         `json:"bytes_sent_delta"`
	BytesReceivedDelta int64         `json:"bytes_received_delta"`
	ConnectionDelta    int64         `json:"connection_delta"`
	TimePeriod         time.Duration `json:"time_period"`
}

// CalculateDelta 计算两个指标之间的增量
func CalculateDelta(current, previous DomainMetrics, timePeriod time.Duration) MetricsDelta {
	return MetricsDelta{
		Domain:             current.Domain,
		AccessDelta:        current.AccessCount - previous.AccessCount,
		BytesSentDelta:     current.BytesSent - previous.BytesSent,
		BytesReceivedDelta: current.BytesReceived - previous.BytesReceived,
		ConnectionDelta:    current.ConnectionCount - previous.ConnectionCount,
		TimePeriod:         timePeriod,
	}
}

// MergeMetrics 合并两个域名指标
func MergeMetrics(base, additional DomainMetrics) DomainMetrics {
	merged := base
	merged.AccessCount += additional.AccessCount
	merged.BytesSent += additional.BytesSent
	merged.BytesReceived += additional.BytesReceived
	merged.ConnectionCount += additional.ConnectionCount

	// 使用较新的访问时间
	if additional.LastAccessTime.After(base.LastAccessTime) {
		merged.LastAccessTime = additional.LastAccessTime
	}

	// 合并协议统计
	if merged.ProtocolStats == nil {
		merged.ProtocolStats = make(map[string]int64)
	}
	for protocol, count := range additional.ProtocolStats {
		merged.ProtocolStats[protocol] += count
	}

	// 合并端口统计
	if merged.PortStats == nil {
		merged.PortStats = make(map[int]int64)
	}
	for port, count := range additional.PortStats {
		merged.PortStats[port] += count
	}

	return merged
}
