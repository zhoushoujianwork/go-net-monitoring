package loader

import (
	"fmt"
	"net"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
	"github.com/sirupsen/logrus"
)

// PacketStats 对应 eBPF 程序中的统计结构
type PacketStats struct {
	TotalPackets uint64
	TotalBytes   uint64
	TCPPackets   uint64
	UDPPackets   uint64
	OtherPackets uint64
}

// XDPLoader XDP程序加载器
type XDPLoader struct {
	spec     *ebpf.CollectionSpec
	coll     *ebpf.Collection
	link     link.Link
	iface    string
	statsMap *ebpf.Map
	logger   *logrus.Logger
}

// NewXDPLoader 创建新的XDP加载器
func NewXDPLoader(ifaceName string, logger *logrus.Logger) *XDPLoader {
	return &XDPLoader{
		iface:  ifaceName,
		logger: logger,
	}
}

// Load 加载eBPF程序
func (x *XDPLoader) Load(programPath string) error {
	// 移除内存限制
	if err := rlimit.RemoveMemlock(); err != nil {
		return fmt.Errorf("failed to remove memlock: %w", err)
	}

	// 加载eBPF程序规范
	spec, err := ebpf.LoadCollectionSpec(programPath)
	if err != nil {
		return fmt.Errorf("failed to load collection spec: %w", err)
	}
	x.spec = spec

	// 创建eBPF集合
	coll, err := ebpf.NewCollection(spec)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	x.coll = coll

	// 获取统计Map
	x.statsMap = coll.Maps["packet_stats_map"]
	if x.statsMap == nil {
		return fmt.Errorf("packet_stats_map not found")
	}

	x.logger.WithFields(logrus.Fields{
		"interface": x.iface,
		"program":   programPath,
	}).Info("eBPF program loaded successfully")

	return nil
}

// Attach 将XDP程序附加到网络接口
func (x *XDPLoader) Attach() error {
	// 获取网络接口
	iface, err := net.InterfaceByName(x.iface)
	if err != nil {
		return fmt.Errorf("failed to get interface %s: %w", x.iface, err)
	}

	// 附加XDP程序
	l, err := link.AttachXDP(link.XDPOptions{
		Program:   x.coll.Programs["xdp_packet_monitor"],
		Interface: iface.Index,
	})
	if err != nil {
		return fmt.Errorf("failed to attach XDP program: %w", err)
	}
	x.link = l

	x.logger.WithField("interface", x.iface).Info("XDP program attached successfully")
	return nil
}

// GetStats 获取包统计信息
func (x *XDPLoader) GetStats() (*PacketStats, error) {
	if x.statsMap == nil {
		return nil, fmt.Errorf("stats map not initialized")
	}

	var key uint32 = 0
	var values []PacketStats

	// 读取per-CPU统计信息
	if err := x.statsMap.Lookup(&key, &values); err != nil {
		return nil, fmt.Errorf("failed to lookup stats: %w", err)
	}

	// 聚合所有CPU的统计信息
	var total PacketStats
	for _, stats := range values {
		total.TotalPackets += stats.TotalPackets
		total.TotalBytes += stats.TotalBytes
		total.TCPPackets += stats.TCPPackets
		total.UDPPackets += stats.UDPPackets
		total.OtherPackets += stats.OtherPackets
	}

	return &total, nil
}

// StartStatsCollection 开始统计信息收集
func (x *XDPLoader) StartStatsCollection(interval time.Duration, callback func(*PacketStats)) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			stats, err := x.GetStats()
			if err != nil {
				x.logger.WithError(err).Error("Failed to get stats")
				continue
			}
			callback(stats)
		}
	}()
}

// Close 清理资源
func (x *XDPLoader) Close() error {
	if x.link != nil {
		if err := x.link.Close(); err != nil {
			x.logger.WithError(err).Error("Failed to close XDP link")
		}
	}

	if x.coll != nil {
		x.coll.Close() // 修复：不返回错误值
	}

	x.logger.Info("XDP loader closed successfully")
	return nil
}
