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

// Collector 网络流量收集器
type Collector struct {
	config     *config.MonitorConfig
	logger     *logrus.Logger
	handle     *pcap.Handle
	packetChan chan gopacket.Packet
	eventChan  chan common.NetworkEvent
	metrics    *NetworkMetrics
	dnsCache   *DNSCache
	connTracker *ConnectionTracker
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.RWMutex
}

// NetworkMetrics 网络指标统计
type NetworkMetrics struct {
	mu                sync.RWMutex
	TotalConnections  uint64
	TotalBytesSent    uint64
	TotalBytesRecv    uint64
	TotalPacketsSent  uint64
	TotalPacketsRecv  uint64
	DomainsAccessed   map[string]uint64
	IPsAccessed       map[string]uint64
	ProtocolStats     map[string]uint64
	PortStats         map[int]uint64
	ProcessStats      map[int]*common.ProcessStats
	LastUpdate        time.Time
}

// NewCollector 创建新的收集器
func NewCollector(cfg *config.MonitorConfig, logger *logrus.Logger) (*Collector, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	collector := &Collector{
		config:      cfg,
		logger:      logger,
		packetChan:  make(chan gopacket.Packet, cfg.BufferSize),
		eventChan:   make(chan common.NetworkEvent, cfg.BufferSize),
		metrics:     NewNetworkMetrics(),
		dnsCache:    NewDNSCache(),
		connTracker: NewConnectionTracker(),
		ctx:         ctx,
		cancel:      cancel,
	}

	// 初始化网络接口
	if err := collector.initInterface(); err != nil {
		cancel()
		return nil, fmt.Errorf("初始化网络接口失败: %w", err)
	}

	return collector, nil
}

// initInterface 初始化网络接口
func (c *Collector) initInterface() error {
	var device string
	
	if c.config.Interface == "" {
		// 自动选择默认网络接口
		devices, err := pcap.FindAllDevs()
		if err != nil {
			return fmt.Errorf("查找网络设备失败: %w", err)
		}
		
		for _, dev := range devices {
			if len(dev.Addresses) > 0 && dev.Name != "lo" {
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

	c.handle = handle
	c.logger.Infof("成功初始化网络接口: %s", device)
	
	return nil
}

// Start 启动收集器
func (c *Collector) Start() error {
	c.logger.Info("启动网络流量收集器")

	// 启动数据包处理协程
	c.wg.Add(1)
	go c.packetProcessor()

	// 启动事件处理协程
	c.wg.Add(1)
	go c.eventProcessor()

	// 启动连接跟踪协程
	c.wg.Add(1)
	go c.connectionTracker()

	// 启动数据包捕获
	c.wg.Add(1)
	go c.capturePackets()

	return nil
}

// Stop 停止收集器
func (c *Collector) Stop() error {
	c.logger.Info("停止网络流量收集器")
	
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
func (c *Collector) capturePackets() {
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
			
			select {
			case c.packetChan <- packet:
			case <-c.ctx.Done():
				return
			default:
				// 缓冲区满，丢弃包
				c.logger.Warn("数据包缓冲区满，丢弃数据包")
			}
		}
	}
}

// packetProcessor 处理数据包
func (c *Collector) packetProcessor() {
	defer c.wg.Done()
	
	for {
		select {
		case <-c.ctx.Done():
			return
		case packet := <-c.packetChan:
			if packet == nil {
				continue
			}
			
			event := c.parsePacket(packet)
			if event != nil && c.shouldProcess(event) {
				select {
				case c.eventChan <- *event:
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
func (c *Collector) eventProcessor() {
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
func (c *Collector) connectionTracker() {
	defer c.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.trackConnections()
		}
	}
}

// parsePacket 解析数据包
func (c *Collector) parsePacket(packet gopacket.Packet) *common.NetworkEvent {
	// 这里实现数据包解析逻辑
	// 提取IP层、传输层信息
	// 识别协议类型、源目地址端口等
	
	event := &common.NetworkEvent{
		Timestamp: time.Now(),
	}
	
	// TODO: 实现具体的数据包解析逻辑
	// 1. 解析以太网帧
	// 2. 解析IP层（IPv4/IPv6）
	// 3. 解析传输层（TCP/UDP）
	// 4. 解析应用层（HTTP/HTTPS/DNS等）
	// 5. 进行DNS反向解析获取域名
	
	return event
}

// shouldProcess 判断是否应该处理该事件
func (c *Collector) shouldProcess(event *common.NetworkEvent) bool {
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
func (c *Collector) updateMetrics(event *common.NetworkEvent) {
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

// trackConnections 跟踪活跃连接
func (c *Collector) trackConnections() {
	// TODO: 实现连接跟踪逻辑
	// 1. 读取 /proc/net/tcp, /proc/net/udp 等文件
	// 2. 获取进程信息
	// 3. 更新连接状态
}

// GetMetrics 获取当前指标
func (c *Collector) GetMetrics() common.NetworkMetrics {
	c.metrics.mu.RLock()
	defer c.metrics.mu.RUnlock()
	
	hostname, _ := os.Hostname()
	
	return common.NetworkMetrics{
		Timestamp:         time.Now(),
		Hostname:          hostname,
		TotalConnections:  c.metrics.TotalConnections,
		TotalBytesSent:    c.metrics.TotalBytesSent,
		TotalBytesRecv:    c.metrics.TotalBytesRecv,
		TotalPacketsSent:  c.metrics.TotalPacketsSent,
		TotalPacketsRecv:  c.metrics.TotalPacketsRecv,
		DomainsAccessed:   copyMap(c.metrics.DomainsAccessed),
		IPsAccessed:       copyMap(c.metrics.IPsAccessed),
		ProtocolStats:     copyMap(c.metrics.ProtocolStats),
		PortStats:         copyIntMap(c.metrics.PortStats),
	}
}

// GetEventChannel 获取事件通道
func (c *Collector) GetEventChannel() <-chan common.NetworkEvent {
	return c.eventChan
}

// NewNetworkMetrics 创建新的网络指标
func NewNetworkMetrics() *NetworkMetrics {
	return &NetworkMetrics{
		DomainsAccessed: make(map[string]uint64),
		IPsAccessed:     make(map[string]uint64),
		ProtocolStats:   make(map[string]uint64),
		PortStats:       make(map[int]uint64),
		ProcessStats:    make(map[int]*common.ProcessStats),
		LastUpdate:      time.Now(),
	}
}

// 辅助函数
func copyMap(src map[string]uint64) map[string]uint64 {
	dst := make(map[string]uint64)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyIntMap(src map[int]uint64) map[int]uint64 {
	dst := make(map[int]uint64)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
