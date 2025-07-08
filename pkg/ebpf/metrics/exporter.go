package metrics

import (
	"go-net-monitoring/internal/common"
	"go-net-monitoring/pkg/ebpf/loader"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// EBPFMetricsExporter eBPF指标导出器
type EBPFMetricsExporter struct {
	logger *logrus.Logger
	
	// eBPF特定指标
	ebpfPacketsTotal    *prometheus.CounterVec
	ebpfBytesTotal      *prometheus.CounterVec
	ebpfProtocolStats   *prometheus.CounterVec
	ebpfProgramInfo     *prometheus.GaugeVec
	
	// 兼容现有指标
	networkDomainsTotal     *prometheus.CounterVec
	networkBytesTotal       *prometheus.CounterVec
	networkConnectionsTotal *prometheus.CounterVec
	networkInterfaceInfo    *prometheus.GaugeVec
}

// NewEBPFMetricsExporter 创建新的eBPF指标导出器
func NewEBPFMetricsExporter(logger *logrus.Logger) *EBPFMetricsExporter {
	exporter := &EBPFMetricsExporter{
		logger: logger,
		
		// eBPF特定指标
		ebpfPacketsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ebpf_network_packets_total",
				Help: "Total number of network packets processed by eBPF program",
			},
			[]string{"protocol", "interface", "host"},
		),
		
		ebpfBytesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ebpf_network_bytes_total",
				Help: "Total number of network bytes processed by eBPF program",
			},
			[]string{"protocol", "interface", "host"},
		),
		
		ebpfProtocolStats: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "ebpf_protocol_stats_total",
				Help: "Protocol statistics from eBPF program",
			},
			[]string{"protocol", "interface", "host"},
		),
		
		ebpfProgramInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "ebpf_program_info",
				Help: "Information about loaded eBPF program",
			},
			[]string{"program_name", "program_type", "interface", "host"},
		),
		
		// 兼容现有指标
		networkDomainsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_domains_accessed_total",
				Help: "Total number of domain accesses",
			},
			[]string{"domain", "host"},
		),
		
		networkBytesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_domain_bytes_total",
				Help: "Total bytes by domain",
			},
			[]string{"domain", "direction", "host"},
		),
		
		networkConnectionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "network_connections_total",
				Help: "Total network connections",
			},
			[]string{"protocol", "host"},
		),
		
		networkInterfaceInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_interface_info",
				Help: "Network interface information",
			},
			[]string{"interface", "ip_address", "mac_address", "host", "host_ip_address"},
		),
	}
	
	// 注册指标
	prometheus.MustRegister(
		exporter.ebpfPacketsTotal,
		exporter.ebpfBytesTotal,
		exporter.ebpfProtocolStats,
		exporter.ebpfProgramInfo,
		exporter.networkDomainsTotal,
		exporter.networkBytesTotal,
		exporter.networkConnectionsTotal,
		exporter.networkInterfaceInfo,
	)
	
	return exporter
}

// UpdateEBPFStats 更新eBPF统计指标
func (e *EBPFMetricsExporter) UpdateEBPFStats(stats *loader.PacketStats, interfaceName, hostName string) {
	labels := prometheus.Labels{
		"interface": interfaceName,
		"host":      hostName,
	}
	
	// 更新包统计
	e.ebpfPacketsTotal.With(prometheus.Labels{
		"protocol":  "tcp",
		"interface": interfaceName,
		"host":      hostName,
	}).Add(float64(stats.TCPPackets))
	
	e.ebpfPacketsTotal.With(prometheus.Labels{
		"protocol":  "udp",
		"interface": interfaceName,
		"host":      hostName,
	}).Add(float64(stats.UDPPackets))
	
	e.ebpfPacketsTotal.With(prometheus.Labels{
		"protocol":  "other",
		"interface": interfaceName,
		"host":      hostName,
	}).Add(float64(stats.OtherPackets))
	
	// 更新字节统计
	e.ebpfBytesTotal.With(prometheus.Labels{
		"protocol":  "total",
		"interface": interfaceName,
		"host":      hostName,
	}).Add(float64(stats.TotalBytes))
	
	// 更新协议统计
	e.ebpfProtocolStats.With(prometheus.Labels{
		"protocol":  "tcp",
		"interface": interfaceName,
		"host":      hostName,
	}).Add(float64(stats.TCPPackets))
	
	e.ebpfProtocolStats.With(prometheus.Labels{
		"protocol":  "udp",
		"interface": interfaceName,
		"host":      hostName,
	}).Add(float64(stats.UDPPackets))
	
	e.logger.WithFields(logrus.Fields{
		"tcp_packets":   stats.TCPPackets,
		"udp_packets":   stats.UDPPackets,
		"total_bytes":   stats.TotalBytes,
		"interface":     interfaceName,
		"host":          hostName,
	}).Debug("eBPF指标已更新")
}

// UpdateProgramInfo 更新eBPF程序信息
func (e *EBPFMetricsExporter) UpdateProgramInfo(programName, programType, interfaceName, hostName string) {
	e.ebpfProgramInfo.With(prometheus.Labels{
		"program_name": programName,
		"program_type": programType,
		"interface":    interfaceName,
		"host":         hostName,
	}).Set(1)
	
	e.logger.WithFields(logrus.Fields{
		"program_name": programName,
		"program_type": programType,
		"interface":    interfaceName,
	}).Info("eBPF程序信息已更新")
}

// UpdateNetworkMetrics 更新网络指标（兼容现有系统）
func (e *EBPFMetricsExporter) UpdateNetworkMetrics(metrics common.NetworkMetrics, hostName string) {
	// 更新域名统计
	for domain, stats := range metrics.Domains {
		e.networkDomainsTotal.With(prometheus.Labels{
			"domain": domain,
			"host":   hostName,
		}).Add(float64(stats.AccessCount))
		
		e.networkBytesTotal.With(prometheus.Labels{
			"domain":    domain,
			"direction": "sent",
			"host":      hostName,
		}).Add(float64(stats.BytesSent))
		
		e.networkBytesTotal.With(prometheus.Labels{
			"domain":    domain,
			"direction": "received",
			"host":      hostName,
		}).Add(float64(stats.BytesReceived))
	}
	
	// 更新连接统计
	for protocol, count := range metrics.Protocols {
		e.networkConnectionsTotal.With(prometheus.Labels{
			"protocol": protocol,
			"host":     hostName,
		}).Add(float64(count))
	}
	
	// 更新接口信息
	for _, iface := range metrics.Interfaces {
		e.networkInterfaceInfo.With(prometheus.Labels{
			"interface":       iface.Name,
			"ip_address":      iface.IPAddress,
			"mac_address":     iface.MACAddress,
			"host":            hostName,
			"host_ip_address": iface.HostIPAddress,
		}).Set(1)
	}
}

// Reset 重置所有指标
func (e *EBPFMetricsExporter) Reset() {
	e.ebpfPacketsTotal.Reset()
	e.ebpfBytesTotal.Reset()
	e.ebpfProtocolStats.Reset()
	e.ebpfProgramInfo.Reset()
	e.networkDomainsTotal.Reset()
	e.networkBytesTotal.Reset()
	e.networkConnectionsTotal.Reset()
	e.networkInterfaceInfo.Reset()
	
	e.logger.Info("所有eBPF指标已重置")
}
