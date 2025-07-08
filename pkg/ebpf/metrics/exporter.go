package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// EBPFMetricsExporter eBPF指标导出器
type EBPFMetricsExporter struct {
	logger *logrus.Logger

	// eBPF特定指标
	ebpfPacketsTotal  *prometheus.CounterVec
	ebpfBytesTotal    *prometheus.CounterVec
	ebpfProtocolStats *prometheus.CounterVec
	ebpfProgramInfo   *prometheus.GaugeVec

	// 兼容现有指标
	networkDomainsTotal     *prometheus.CounterVec
	networkBytesTotal       *prometheus.CounterVec
	networkConnectionsTotal *prometheus.CounterVec
	networkInterfaceInfo    *prometheus.GaugeVec
}
