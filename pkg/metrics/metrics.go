package metrics

import (
	"go-net-monitoring/internal/common"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics Prometheus指标集合
type Metrics struct {
	// 网络连接指标
	NetworkConnectionsTotal *prometheus.CounterVec
	NetworkBytesSentTotal   *prometheus.CounterVec
	NetworkBytesRecvTotal   *prometheus.CounterVec
	NetworkPacketsSentTotal *prometheus.CounterVec
	NetworkPacketsRecvTotal *prometheus.CounterVec

	// 域名和IP访问指标
	NetworkDomainsAccessedTotal *prometheus.CounterVec
	NetworkIPsAccessedTotal     *prometheus.CounterVec

	// 按域名的流量指标
	NetworkDomainBytesSentTotal     *prometheus.CounterVec
	NetworkDomainBytesReceivedTotal *prometheus.CounterVec
	NetworkDomainConnectionsTotal   *prometheus.CounterVec

	// 协议统计指标
	NetworkProtocolStats *prometheus.CounterVec

	// 连接状态指标
	NetworkActiveConnections  *prometheus.GaugeVec
	NetworkConnectionDuration *prometheus.HistogramVec

	// Agent状态指标
	AgentUptime         prometheus.Gauge
	AgentLastReportTime prometheus.Gauge
	AgentReportTotal    *prometheus.CounterVec
	AgentReportErrors   *prometheus.CounterVec

	// 性能指标
	PacketProcessingDuration *prometheus.HistogramVec
	EventQueueSize           prometheus.Gauge
	DNSCacheSize             prometheus.Gauge
	ConnectionTrackerSize    prometheus.Gauge

	// 网卡信息指标 (新增)
	NetworkInterfaceInfo *prometheus.GaugeVec
}

// NewMetrics 创建新的指标集合
func NewMetrics() *Metrics {
	return &Metrics{
		// 网络连接指标 (添加interface标签)
		NetworkConnectionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_connections_total",
				Help: "Total number of network connections",
			},
			[]string{"protocol", "direction", "host", "interface"},
		),

		NetworkBytesSentTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_bytes_sent_total",
				Help: "Total bytes sent over network",
			},
			[]string{"protocol", "destination", "host", "interface"},
		),

		NetworkBytesRecvTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_bytes_received_total",
				Help: "Total bytes received over network",
			},
			[]string{"protocol", "source", "host", "interface"},
		),

		NetworkPacketsSentTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_packets_sent_total",
				Help: "Total packets sent over network",
			},
			[]string{"protocol", "destination", "host", "interface"},
		),

		NetworkPacketsRecvTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_packets_received_total",
				Help: "Total packets received over network",
			},
			[]string{"protocol", "source", "host", "interface"},
		),

		// 域名和IP访问指标 (添加interface标签)
		NetworkDomainsAccessedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_domains_accessed_total",
				Help: "Total number of domains accessed",
			},
			[]string{"domain", "host", "interface"},
		),

		NetworkIPsAccessedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_ips_accessed_total",
				Help: "Total number of IP addresses accessed",
			},
			[]string{"ip", "host", "interface"},
		),

		// 按域名的流量指标 (添加interface标签)
		NetworkDomainBytesSentTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_domain_bytes_sent_total",
				Help: "Total bytes sent to each domain",
			},
			[]string{"domain", "host", "interface"},
		),

		NetworkDomainBytesReceivedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_domain_bytes_received_total",
				Help: "Total bytes received from each domain",
			},
			[]string{"domain", "host", "interface"},
		),

		NetworkDomainConnectionsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_domain_connections_total",
				Help: "Total connections to each domain",
			},
			[]string{"domain", "host", "interface"},
		),

		// 协议统计指标 (添加interface标签)
		NetworkProtocolStats: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_protocol_stats_total",
				Help: "Network protocol statistics",
			},
			[]string{"protocol", "host", "interface"},
		),

		// 连接状态指标
		NetworkActiveConnections: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_active_connections",
				Help: "Number of active network connections",
			},
			[]string{"protocol", "state", "host"},
		),

		NetworkConnectionDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "network_connection_duration_seconds",
				Help:    "Duration of network connections in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"protocol", "host"},
		),

		// Agent状态指标
		AgentUptime: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "agent_uptime_seconds",
				Help: "Agent uptime in seconds",
			},
		),

		AgentLastReportTime: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "agent_last_report_timestamp",
				Help: "Timestamp of last successful report",
			},
		),

		AgentReportTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_reports_total",
				Help: "Total number of reports sent by agent",
			},
			[]string{"status", "host"},
		),

		AgentReportErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_report_errors_total",
				Help: "Total number of report errors",
			},
			[]string{"error_type", "host"},
		),

		// 性能指标
		PacketProcessingDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "packet_processing_duration_seconds",
				Help:    "Time spent processing packets",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
			},
			[]string{"stage", "host"},
		),

		EventQueueSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "event_queue_size",
				Help: "Current size of event queue",
			},
		),

		DNSCacheSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "dns_cache_size",
				Help: "Current size of DNS cache",
			},
		),

		ConnectionTrackerSize: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "connection_tracker_size",
				Help: "Current size of connection tracker",
			},
		),

		// 网卡信息指标 (新增)
		NetworkInterfaceInfo: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_interface_info",
				Help: "Network interface information with IP address and MAC address",
			},
			[]string{"interface", "ip_address", "mac_address", "host"},
		),
	}
}

// UpdateNetworkMetrics 更新网络指标 (支持interface参数)
func (m *Metrics) UpdateNetworkMetrics(metrics common.NetworkMetrics) {
	hostname := metrics.Hostname
	// 默认使用 "unknown" 作为interface，保持向后兼容
	interfaceName := "unknown"
	if metrics.Interface != "" {
		interfaceName = metrics.Interface
	}

	// 更新网络指标 - 使用正确的标签数量 (添加interface标签)
	m.NetworkConnectionsTotal.WithLabelValues("total", "all", hostname, interfaceName).Add(float64(metrics.TotalConnections))
	m.NetworkBytesSentTotal.WithLabelValues("total", "all", hostname, interfaceName).Add(float64(metrics.TotalBytesSent))
	m.NetworkBytesRecvTotal.WithLabelValues("total", "all", hostname, interfaceName).Add(float64(metrics.TotalBytesRecv))
	m.NetworkPacketsSentTotal.WithLabelValues("total", "all", hostname, interfaceName).Add(float64(metrics.TotalPacketsSent))
	m.NetworkPacketsRecvTotal.WithLabelValues("total", "all", hostname, interfaceName).Add(float64(metrics.TotalPacketsRecv))

	// 更新域名访问统计 (添加interface标签)
	for domain, count := range metrics.DomainsAccessed {
		m.NetworkDomainsAccessedTotal.WithLabelValues(domain, hostname, interfaceName).Add(float64(count))
	}

	// 更新域名流量统计 (添加interface标签)
	for domain, stats := range metrics.DomainTraffic {
		if stats != nil {
			m.NetworkDomainBytesSentTotal.WithLabelValues(domain, hostname, interfaceName).Add(float64(stats.BytesSent))
			m.NetworkDomainBytesReceivedTotal.WithLabelValues(domain, hostname, interfaceName).Add(float64(stats.BytesReceived))
			m.NetworkDomainConnectionsTotal.WithLabelValues(domain, hostname, interfaceName).Add(float64(stats.Connections))
		}
	}

	// 更新IP访问统计 (添加interface标签)
	for ip, count := range metrics.IPsAccessed {
		m.NetworkIPsAccessedTotal.WithLabelValues(ip, hostname, interfaceName).Add(float64(count))
	}

	// 更新协议统计 (添加interface标签)
	for protocol, count := range metrics.ProtocolStats {
		m.NetworkProtocolStats.WithLabelValues(protocol, hostname, interfaceName).Add(float64(count))
	}
}

// UpdateInterfaceInfo 更新网卡信息指标 (新增方法)
func (m *Metrics) UpdateInterfaceInfo(interfaceName, ipAddress, macAddress, hostname string) {
	// 设置网卡信息指标，值为1表示该网卡存在
	m.NetworkInterfaceInfo.WithLabelValues(interfaceName, ipAddress, macAddress, hostname).Set(1)
}

// ClearInterfaceInfo 清除网卡信息指标 (新增方法)
func (m *Metrics) ClearInterfaceInfo() {
	m.NetworkInterfaceInfo.Reset()
}

// UpdateNetworkEvent 更新网络事件指标 (支持interface参数)
func (m *Metrics) UpdateNetworkEvent(event common.NetworkEvent) {
	hostname := event.ProcessName // 或者从其他地方获取hostname
	interfaceName := "unknown"
	if event.Interface != "" {
		interfaceName = event.Interface
	}

	// 更新连接计数 (添加interface标签)
	m.NetworkConnectionsTotal.WithLabelValues(
		event.Protocol,
		event.Direction,
		hostname,
		interfaceName,
	).Inc()

	// 更新字节统计 (添加interface标签)
	if event.BytesSent > 0 {
		m.NetworkBytesSentTotal.WithLabelValues(
			event.Protocol,
			event.DestIP,
			hostname,
			interfaceName,
		).Add(float64(event.BytesSent))
	}

	if event.BytesRecv > 0 {
		m.NetworkBytesRecvTotal.WithLabelValues(
			event.Protocol,
			event.SourceIP,
			hostname,
			interfaceName,
		).Add(float64(event.BytesRecv))
	}

	// 更新包统计 (添加interface标签)
	if event.PacketsSent > 0 {
		m.NetworkPacketsSentTotal.WithLabelValues(
			event.Protocol,
			event.DestIP,
			hostname,
			interfaceName,
		).Add(float64(event.PacketsSent))
	}

	if event.PacketsRecv > 0 {
		m.NetworkPacketsRecvTotal.WithLabelValues(
			event.Protocol,
			event.SourceIP,
			hostname,
			interfaceName,
		).Add(float64(event.PacketsRecv))
	}

	// 更新连接持续时间 (添加interface标签)
	if event.Duration > 0 {
		m.NetworkConnectionDuration.WithLabelValues(
			event.Protocol,
			hostname,
			interfaceName,
		).Observe(event.Duration.Seconds())
	}
}

// UpdateAgentStats 更新Agent统计
func (m *Metrics) UpdateAgentStats(hostname string, uptime float64, lastReportTime float64) {
	m.AgentUptime.Set(uptime)
	m.AgentLastReportTime.Set(lastReportTime)
}

// UpdateReportStats 更新上报统计
func (m *Metrics) UpdateReportStats(hostname, status string) {
	m.AgentReportTotal.WithLabelValues(status, hostname).Inc()
}

// UpdateReportError 更新上报错误统计
func (m *Metrics) UpdateReportError(hostname, errorType string) {
	m.AgentReportErrors.WithLabelValues(errorType, hostname).Inc()
}

// UpdatePerformanceMetrics 更新性能指标
func (m *Metrics) UpdatePerformanceMetrics(hostname string, processingDuration float64, queueSize, cacheSize, trackerSize int) {
	m.PacketProcessingDuration.WithLabelValues("total", hostname).Observe(processingDuration)
	m.EventQueueSize.Set(float64(queueSize))
	m.DNSCacheSize.Set(float64(cacheSize))
	m.ConnectionTrackerSize.Set(float64(trackerSize))
}

// UpdateActiveConnections 更新活跃连接数
func (m *Metrics) UpdateActiveConnections(hostname, protocol, state string, count int) {
	m.NetworkActiveConnections.WithLabelValues(protocol, state, hostname).Set(float64(count))
}
