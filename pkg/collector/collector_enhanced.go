package collector

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go-net-monitoring/internal/common"
	"go-net-monitoring/internal/config"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/sirupsen/logrus"
)

// EnhancedCollector 增强的网络流量收集器
type EnhancedCollector struct {
	config      *config.MonitorConfig
	logger      *logrus.Logger
	handle      *pcap.Handle
	parser      *PacketParser
	packetChan  chan gopacket.Packet
	eventChan   chan common.NetworkEvent
	metrics     *NetworkMetrics
	connTracker *ConnectionTracker
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup

	// 统计信息
	stats struct {
		mu               sync.RWMutex
		totalPackets     uint64
		processedPackets uint64
		droppedPackets   uint64
		outboundPackets  uint64
		uniqueIPs        map[string]bool
		uniqueDomains    map[string]bool
		lastStatsTime    time.Time
	}
}

// NewEnhancedCollector 创建增强的收集器
func NewEnhancedCollector(cfg *config.MonitorConfig, logger *logrus.Logger) (*EnhancedCollector, error) {
	ctx, cancel := context.WithCancel(context.Background())

	dnsCache := NewDNSCache()
	parser := NewPacketParser(dnsCache)

	collector := &EnhancedCollector{
		config:      cfg,
		logger:      logger,
		parser:      parser,
		packetChan:  make(chan gopacket.Packet, cfg.BufferSize),
		eventChan:   make(chan common.NetworkEvent, cfg.BufferSize),
		metrics:     NewNetworkMetrics(),
		connTracker: NewConnectionTracker(),
		ctx:         ctx,
		cancel:      cancel,
	}

	// 初始化统计信息
	collector.stats.uniqueIPs = make(map[string]bool)
	collector.stats.uniqueDomains = make(map[string]bool)
	collector.stats.lastStatsTime = time.Now()

	// 初始化网络接口
	if err := collector.initInterface(); err != nil {
		cancel()
		return nil, fmt.Errorf("初始化网络接口失败: %w", err)
	}

	return collector, nil
}

// initInterface 初始化网络接口
func (c *EnhancedCollector) initInterface() error {
	var device string

	if c.config.Interface == "" {
		// 自动选择默认网络接口
		devices, err := pcap.FindAllDevs()
		if err != nil {
			return fmt.Errorf("查找网络设备失败: %w", err)
		}

		for _, dev := range devices {
			if len(dev.Addresses) > 0 && dev.Name != "lo" && dev.Name != "any" {
				device = dev.Name
				break
			}
		}

		if device == "" {
			return fmt.Errorf("未找到可用的网络接口")
		}
	} else {
		device = c.config.Interface
	}

	// 打开网络接口进行抓包
	handle, err := pcap.OpenLive(device, 1600, true, pcap.BlockForever)
	if err != nil {
		return fmt.Errorf("打开网络接口 %s 失败: %w", device, err)
	}

	// 设置BPF过滤器（只捕获出站流量）
	filter := c.buildBPFFilter()
	if filter != "" {
		if err := handle.SetBPFFilter(filter); err != nil {
			c.logger.WithError(err).Warn("设置BPF过滤器失败，将捕获所有流量")
		} else {
			c.logger.Infof("设置BPF过滤器: %s", filter)
		}
	}

	c.handle = handle
	c.logger.Infof("成功初始化网络接口: %s", device)

	return nil
}

// buildBPFFilter 构建BPF过滤器
func (c *EnhancedCollector) buildBPFFilter() string {
	var filters []string

	// 根据配置的协议构建过滤器
	for _, protocol := range c.config.Protocols {
		switch protocol {
		case "tcp":
			filters = append(filters, "tcp")
		case "udp":
			filters = append(filters, "udp")
		case "http":
			filters = append(filters, "tcp port 80")
		case "https":
			filters = append(filters, "tcp port 443")
		case "dns":
			filters = append(filters, "udp port 53 or tcp port 53")
		}
	}

	if len(filters) == 0 {
		return ""
	}

	// 组合过滤器
	filter := "(" + filters[0]
	for i := 1; i < len(filters); i++ {
		filter += " or " + filters[i]
	}
	filter += ")"

	return filter
}

// Start 启动收集器
func (c *EnhancedCollector) Start() error {
	c.logger.Info("启动增强网络流量收集器")

	// 启动数据包捕获协程
	c.wg.Add(1)
	go c.capturePackets()

	// 启动数据包处理协程
	c.wg.Add(1)
	go c.packetProcessor()

	// 启动事件处理协程
	c.wg.Add(1)
	go c.eventProcessor()

	// 启动连接跟踪协程
	c.wg.Add(1)
	go c.connectionTracker()

	// 启动统计协程
	c.wg.Add(1)
	go c.statsReporter()

	return nil
}

// Stop 停止收集器
func (c *EnhancedCollector) Stop() error {
	c.logger.Info("停止增强网络流量收集器")

	c.cancel()

	if c.handle != nil {
		c.handle.Close()
	}

	close(c.packetChan)
	close(c.eventChan)

	c.wg.Wait()

	return nil
}

// capturePackets 捕获数据包
func (c *EnhancedCollector) capturePackets() {
	defer c.wg.Done()

	packetSource := gopacket.NewPacketSource(c.handle, c.handle.LinkType())

	for {
		select {
		case <-c.ctx.Done():
			return
		case packet := <-packetSource.Packets():
			if packet == nil {
				continue
			}

			// 更新统计
			c.updatePacketStats(1, 0)

			select {
			case c.packetChan <- packet:
			case <-c.ctx.Done():
				return
			default:
				// 缓冲区满，丢弃包
				c.updatePacketStats(0, 1)
				c.logger.Warn("数据包缓冲区满，丢弃数据包")
			}
		}
	}
}

// packetProcessor 处理数据包
func (c *EnhancedCollector) packetProcessor() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case packet := <-c.packetChan:
			if packet == nil {
				continue
			}

			// 解析数据包
			event := c.parser.ParsePacket(packet)
			if event != nil && c.shouldProcess(event) {
				// 更新统计
				c.updateEventStats(event)

				select {
				case c.eventChan <- *event:
					c.updatePacketStats(0, 0) // 标记为已处理
				case <-c.ctx.Done():
					return
				default:
					c.logger.Warn("事件缓冲区满，丢弃事件")
				}
			}
		}
	}
}

// eventProcessor 处理网络事件
func (c *EnhancedCollector) eventProcessor() {
	defer c.wg.Done()

	for {
		select {
		case <-c.ctx.Done():
			return
		case event := <-c.eventChan:
			c.updateMetrics(&event)
		}
	}
}

// connectionTracker 连接跟踪
func (c *EnhancedCollector) connectionTracker() {
	defer c.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			// 清理过期连接
			c.connTracker.Cleanup(5 * time.Minute)
		}
	}
}

// statsReporter 统计报告
func (c *EnhancedCollector) statsReporter() {
	defer c.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.logStats()
		}
	}
}

// shouldProcess 判断是否应该处理该事件
func (c *EnhancedCollector) shouldProcess(event *common.NetworkEvent) bool {
	// 只处理出站流量
	if event.Direction != "outbound" {
		return false
	}

	// 应用过滤规则
	if c.config.Filters.IgnoreLocalhost {
		if event.SourceIP == "127.0.0.1" || event.DestIP == "127.0.0.1" {
			return false
		}
	}

	// 检查忽略端口
	for _, port := range c.config.Filters.IgnorePorts {
		if event.SourcePort == port || event.DestPort == port {
			return false
		}
	}

	// 检查忽略IP
	for _, ip := range c.config.Filters.IgnoreIPs {
		if event.SourceIP == ip || event.DestIP == ip {
			return false
		}
	}

	// 检查只监控特定域名
	if len(c.config.Filters.OnlyDomains) > 0 && event.Domain != "" {
		found := false
		for _, domain := range c.config.Filters.OnlyDomains {
			if event.Domain == domain {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// updateMetrics 更新指标
func (c *EnhancedCollector) updateMetrics(event *common.NetworkEvent) {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()

	c.metrics.TotalConnections++
	c.metrics.TotalBytesSent += event.BytesSent
	c.metrics.TotalBytesRecv += event.BytesRecv
	c.metrics.TotalPacketsSent += event.PacketsSent
	c.metrics.TotalPacketsRecv += event.PacketsRecv

	// 更新域名访问统计
	if event.Domain != "" {
		c.metrics.DomainsAccessed[event.Domain]++
	}

	// 更新IP访问统计
	if event.DestIP != "" {
		c.metrics.IPsAccessed[event.DestIP]++
	}

	// 更新协议统计
	c.metrics.ProtocolStats[event.Protocol]++

	// 更新端口统计
	if event.DestPort > 0 {
		c.metrics.PortStats[event.DestPort]++
	}

	c.metrics.LastUpdate = time.Now()
}

// updatePacketStats 更新数据包统计
func (c *EnhancedCollector) updatePacketStats(total, dropped uint64) {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()

	c.stats.totalPackets += total
	c.stats.droppedPackets += dropped
	if total > 0 && dropped == 0 {
		c.stats.processedPackets++
	}
}

// updateEventStats 更新事件统计
func (c *EnhancedCollector) updateEventStats(event *common.NetworkEvent) {
	c.stats.mu.Lock()
	defer c.stats.mu.Unlock()

	if event.Direction == "outbound" {
		c.stats.outboundPackets++
	}

	if event.DestIP != "" {
		c.stats.uniqueIPs[event.DestIP] = true
	}

	if event.Domain != "" {
		c.stats.uniqueDomains[event.Domain] = true
	}
}

// logStats 记录统计信息
func (c *EnhancedCollector) logStats() {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	c.logger.WithFields(map[string]interface{}{
		"total_packets":     c.stats.totalPackets,
		"processed_packets": c.stats.processedPackets,
		"dropped_packets":   c.stats.droppedPackets,
		"outbound_packets":  c.stats.outboundPackets,
		"unique_ips":        len(c.stats.uniqueIPs),
		"unique_domains":    len(c.stats.uniqueDomains),
	}).Info("收集器统计信息")
}

// GetMetrics 获取当前指标
func (c *EnhancedCollector) GetMetrics() common.NetworkMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()

	hostname, _ := os.Hostname()

	return common.NetworkMetrics{
		Timestamp:        time.Now(),
		Hostname:         hostname,
		TotalConnections: c.metrics.TotalConnections,
		TotalBytesSent:   c.metrics.TotalBytesSent,
		TotalBytesRecv:   c.metrics.TotalBytesRecv,
		TotalPacketsSent: c.metrics.TotalPacketsSent,
		TotalPacketsRecv: c.metrics.TotalPacketsRecv,
		DomainsAccessed:  copyMap(c.metrics.DomainsAccessed),
		IPsAccessed:      copyMap(c.metrics.IPsAccessed),
		ProtocolStats:    copyMap(c.metrics.ProtocolStats),
		PortStats:        copyIntMap(c.metrics.PortStats),
	}
}

// GetStats 获取收集器统计信息
func (c *EnhancedCollector) GetStats() map[string]interface{} {
	c.stats.mu.RLock()
	defer c.stats.mu.RUnlock()

	return map[string]interface{}{
		"total_packets":     c.stats.totalPackets,
		"processed_packets": c.stats.processedPackets,
		"dropped_packets":   c.stats.droppedPackets,
		"outbound_packets":  c.stats.outboundPackets,
		"unique_ips":        len(c.stats.uniqueIPs),
		"unique_domains":    len(c.stats.uniqueDomains),
		"drop_rate":         float64(c.stats.droppedPackets) / float64(c.stats.totalPackets) * 100,
	}
}
