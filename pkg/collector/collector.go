package collector

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"go-net-monitoring/internal/common"
	"go-net-monitoring/internal/config"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/sirupsen/logrus"
)

// Collector 网络流量收集器
type Collector struct {
	config      *config.MonitorConfig
	logger      *logrus.Logger
	handle      *pcap.Handle
	packetChan  chan gopacket.Packet
	eventChan   chan common.NetworkEvent
	metrics     *NetworkMetrics
	dnsCache    *DNSCache
	connTracker *ConnectionTracker
	localIPs    map[string]bool // 本机IP地址缓存
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mu          sync.RWMutex
}

// NetworkMetrics 网络指标统计
type NetworkMetrics struct {
	mu               sync.RWMutex
	TotalConnections uint64
	TotalBytesSent   uint64
	TotalBytesRecv   uint64
	TotalPacketsSent uint64
	TotalPacketsRecv uint64
	DomainsAccessed  map[string]uint64
	IPsAccessed      map[string]uint64
	ProtocolStats    map[string]uint64
	PortStats        map[int]uint64
	DomainTraffic    map[string]*common.DomainTrafficStats
	ProcessStats     map[int]*common.ProcessStats
	LastUpdate       time.Time
}

// NewTestCollector 创建测试收集器（用于测试环境）
func NewTestCollector(cfg *config.MonitorConfig, logger *logrus.Logger) (*Collector, error) {
	logger.Info("创建测试收集器")
	
	ctx, cancel := context.WithCancel(context.Background())
	
	collector := &Collector{
		config:      cfg,
		logger:      logger,
		packetChan:  make(chan gopacket.Packet, cfg.BufferSize),
		eventChan:   make(chan common.NetworkEvent, cfg.BufferSize),
		metrics:     NewNetworkMetrics(),
		dnsCache:    NewDNSCache(),
		connTracker: NewConnectionTracker(),
		localIPs:    make(map[string]bool),
		ctx:         ctx,
		cancel:      cancel,
	}
	
	// 测试模式下不需要真实的网络接口，handle保持为nil
	logger.Info("测试收集器创建成功")
	return collector, nil
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
		localIPs:    make(map[string]bool),
		ctx:         ctx,
		cancel:      cancel,
	}

	// 检测本机IP地址
	if err := collector.detectLocalIPs(); err != nil {
		logger.WithError(err).Warn("检测本机IP失败，使用默认配置")
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
	// 在测试或开发环境中，跳过实际的网络接口初始化
	if os.Getenv("TEST_MODE") == "true" || os.Getenv("DEV_MODE") == "true" {
		c.logger.Info("运行在测试/开发模式，跳过网络接口初始化")
		return nil
	}

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
	// 使用30秒超时而不是BlockForever，避免程序卡死
	handle, err := pcap.OpenLive(device, 1600, true, 30*time.Second)
	if err != nil {
		return fmt.Errorf("打开网络接口 %s 失败: %w", device, err)
	}

	// 临时移除BPF过滤器，捕获所有流量进行调试
	c.logger.Info("暂时不设置BPF过滤器，将捕获所有流量")

	c.handle = handle
	c.logger.Infof("成功初始化网络接口: %s", device)

	return nil
}

// Start 启动收集器
func (c *Collector) Start() error {
	c.logger.Info("启动网络流量收集器")

	// 启动数据包处理协程
	c.logger.Debug("启动数据包处理协程...")
	c.wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.logger.Errorf("数据包处理协程panic: %v", r)
			}
			c.wg.Done()
		}()
		c.packetProcessor()
	}()

	// 启动事件处理协程
	c.logger.Debug("启动事件处理协程...")
	c.wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.logger.Errorf("事件处理协程panic: %v", r)
			}
			c.wg.Done()
		}()
		c.eventProcessor()
	}()

	// 启动连接跟踪协程
	c.logger.Debug("启动连接跟踪协程...")
	c.wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.logger.Errorf("连接跟踪协程panic: %v", r)
			}
			c.wg.Done()
		}()
		c.connectionTracker()
	}()

	// 启动数据包捕获
	c.logger.Debug("启动数据包捕获协程...")
	c.wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				c.logger.Errorf("数据包捕获协程panic: %v", r)
			}
			c.wg.Done()
		}()
		c.capturePackets()
	}()

	c.logger.Debug("收集器所有协程启动完成")
	return nil
}

// Stop 停止收集器
func (c *Collector) Stop() error {
	c.logger.Info("停止网络流量收集器")

	// 取消上下文
	c.cancel()

	// 关闭网络句柄
	if c.handle != nil {
		c.handle.Close()
		c.logger.Debug("已关闭网络句柄")
	}

	// 关闭通道
	close(c.packetChan)
	close(c.eventChan)

	// 等待所有goroutine结束，但设置超时
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		c.logger.Debug("所有收集器goroutine已结束")
	case <-time.After(3 * time.Second):
		c.logger.Warn("等待收集器goroutine结束超时")
	}

	return nil
}

// capturePackets 捕获数据包
func (c *Collector) capturePackets() {
	defer c.wg.Done()

	// 测试模式下handle为nil，不进行真实的数据包捕获
	if c.handle == nil {
		c.logger.Info("测试模式：跳过数据包捕获")
		// 在测试模式下，我们只是等待停止信号
		<-c.ctx.Done()
		c.logger.Debug("测试模式：收到停止信号，退出数据包捕获")
		return
	}

	packetSource := gopacket.NewPacketSource(c.handle, c.handle.LinkType())

	// 设置非阻塞模式
	packetChan := packetSource.Packets()

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("收到停止信号，退出数据包捕获")
			return
		case packet, ok := <-packetChan:
			if !ok {
				c.logger.Debug("数据包通道已关闭")
				return
			}

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
		case <-time.After(1 * time.Second):
			// 定期检查是否需要停止，避免长时间阻塞
			continue
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
			c.logger.Debugf("处理网络事件: Domain=%s, DestIP=%s, Protocol=%s", 
				event.Domain, event.DestIP, event.Protocol)
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
	event := &common.NetworkEvent{
		Timestamp: time.Now(),
	}

	// 解析IP层
	var srcIP, dstIP string
	if ipv4Layer := packet.Layer(layers.LayerTypeIPv4); ipv4Layer != nil {
		ipv4, _ := ipv4Layer.(*layers.IPv4)
		srcIP = ipv4.SrcIP.String()
		dstIP = ipv4.DstIP.String()
		event.Protocol = "ipv4"
	} else if ipv6Layer := packet.Layer(layers.LayerTypeIPv6); ipv6Layer != nil {
		ipv6, _ := ipv6Layer.(*layers.IPv6)
		srcIP = ipv6.SrcIP.String()
		dstIP = ipv6.DstIP.String()
		event.Protocol = "ipv6"
	}

	if srcIP == "" || dstIP == "" {
		return nil
	}

	event.SourceIP = srcIP
	event.DestIP = dstIP

	// 解析传输层
	c.parseTransportLayer(packet, event)

	// 特殊处理DNS流量 - DNS流量总是处理，不管方向
	if event.DestPort == 53 || event.SourcePort == 53 {
		c.parseApplicationLayer(packet, event)
		c.logger.Debugf("处理DNS流量: %s:%d -> %s:%d", srcIP, event.SourcePort, dstIP, event.DestPort)
		return event
	}

	// 判断流量方向
	event.Direction = c.getTrafficDirection(srcIP, dstIP)

	// 处理出站流量（包括内网和外网）
	if strings.HasPrefix(event.Direction, "outbound") {
		// 解析应用层
		c.parseApplicationLayer(packet, event)

		// 尝试解析域名
		c.resolveDomain(event)

		// 获取数据包大小 - 使用整个数据包的长度
		var packetSize uint64
		if packet.Metadata() != nil && packet.Metadata().Length > 0 {
			packetSize = uint64(packet.Metadata().Length)
		} else {
			// 备用方案：使用数据包数据长度
			packetSize = uint64(len(packet.Data()))
			if packetSize == 0 {
				// 如果没有数据，至少计算一个最小包大小（以太网头+IP头+TCP头）
				packetSize = 64 // 最小以太网帧大小
			}
		}
		
		// 根据方向设置字节数
		if strings.HasPrefix(event.Direction, "inbound") {
			event.BytesRecv = packetSize
			event.PacketsRecv = 1
			event.BytesSent = 0
			event.PacketsSent = 0
		} else if strings.HasPrefix(event.Direction, "outbound") {
			event.BytesSent = packetSize
			event.PacketsSent = 1
			event.BytesRecv = 0
			event.PacketsRecv = 0
		} else {
			// 本地流量，同时计算发送和接收
			event.BytesSent = packetSize / 2
			event.BytesRecv = packetSize / 2
			event.PacketsSent = 1
			event.PacketsRecv = 1
		}

		// 为内网流量添加特殊标记
		if event.Direction == "outbound_internal" {
			c.logger.Debugf("解析到内网出站数据包: %s -> %s (%s:%d) domain=%s",
				event.SourceIP, event.DestIP, event.Protocol, event.DestPort, event.Domain)
		} else {
			c.logger.Debugf("解析到外网出站数据包: %s -> %s (%s:%d) domain=%s",
				event.SourceIP, event.DestIP, event.Protocol, event.DestPort, event.Domain)
		}

		return event
	}

	// 其他流量暂时不处理
	return nil
}

// getTrafficDirection 判断流量方向
func (c *Collector) getTrafficDirection(srcIP, dstIP string) string {
	srcIsLocal := c.isLocalIP(srcIP)
	dstIsLocal := c.isLocalIP(dstIP)

	// 如果源IP是本机IP，目标IP不是本机IP，则为出站流量
	if srcIsLocal && !dstIsLocal {
		// 进一步区分是否为内网流量
		if c.isPrivateIP(dstIP) {
			return "outbound_internal" // 出站内网流量
		}
		return "outbound_external" // 出站外网流量
	}

	// 如果源IP不是本机IP，目标IP是本机IP，则为入站流量
	if !srcIsLocal && dstIsLocal {
		if c.isPrivateIP(srcIP) {
			return "inbound_internal" // 入站内网流量
		}
		return "inbound_external" // 入站外网流量
	}

	// 两个都是本机IP或都不是本机IP
	return "local"
}

// isPrivateIP 判断是否为私有IP地址
func (c *Collector) isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// 检查是否为私有IP范围
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

// detectLocalIPs 检测本机所有IP地址
func (c *Collector) detectLocalIPs() error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("获取网络接口失败: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	for _, iface := range interfaces {
		// 跳过非活跃接口
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil && !ip.IsLoopback() {
				ipStr := ip.String()
				c.localIPs[ipStr] = true
				c.logger.Debugf("检测到本机IP: %s (接口: %s)", ipStr, iface.Name)
			}
		}
	}

	c.logger.Infof("检测到 %d 个本机IP地址", len(c.localIPs))
	return nil
}

// getLocalIPs 获取本机所有IP地址
func (c *Collector) getLocalIPs() []string {
	var ips []string
	
	interfaces, err := net.Interfaces()
	if err != nil {
		c.logger.WithError(err).Error("获取网络接口失败")
		return ips
	}
	
	for _, iface := range interfaces {
		// 跳过回环接口和未启用的接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				// 只添加IPv4地址
				if ipnet.IP.To4() != nil {
					ips = append(ips, ipnet.IP.String())
				}
			}
		}
	}
	
	c.logger.Infof("检测到本机IP地址: %v", ips)
	return ips
}

// isLocalIP 判断是否为本机IP
func (c *Collector) isLocalIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}

	// 检查是否为回环地址
	if ip.IsLoopback() {
		return true
	}

	// 检查是否在本机IP列表中
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.localIPs[ipStr]
}

// parseTransportLayer 解析传输层
func (c *Collector) parseTransportLayer(packet gopacket.Packet, event *common.NetworkEvent) {
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		event.Protocol = "tcp"
		event.SourcePort = int(tcp.SrcPort)
		event.DestPort = int(tcp.DstPort)

		// TCP状态
		if tcp.SYN {
			event.Status = "syn"
		} else if tcp.FIN {
			event.Status = "fin"
		} else if tcp.RST {
			event.Status = "rst"
		} else {
			event.Status = "established"
		}

	} else if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		event.Protocol = "udp"
		event.SourcePort = int(udp.SrcPort)
		event.DestPort = int(udp.DstPort)
		event.Status = "active"
	}
}

// parseApplicationLayer 解析应用层
func (c *Collector) parseApplicationLayer(packet gopacket.Packet, event *common.NetworkEvent) {
	// 解析DNS
	if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
		c.parseDNS(dnsLayer, event)
		return
	}

	// 解析HTTP
	if event.Protocol == "tcp" && (event.DestPort == 80 || event.SourcePort == 80) {
		c.parseHTTP(packet, event)
		return
	}

	// 标记HTTPS
	if event.Protocol == "tcp" && (event.DestPort == 443 || event.SourcePort == 443) {
		event.Protocol = "https"
	}
}

// parseDNS 解析DNS查询
func (c *Collector) parseDNS(dnsLayer gopacket.Layer, event *common.NetworkEvent) {
	dns, _ := dnsLayer.(*layers.DNS)
	event.Protocol = "dns"

	c.logger.Debugf("DNS包: QR=%t, Questions=%d, Answers=%d", dns.QR, len(dns.Questions), len(dns.Answers))

	// 处理DNS查询
	if !dns.QR && len(dns.Questions) > 0 {
		for _, question := range dns.Questions {
			domain := strings.TrimSuffix(string(question.Name), ".")
			if domain != "" && !strings.HasSuffix(domain, ".local") {
				event.Domain = domain
				c.logger.Infof("🔍 DNS查询: %s", domain)
			}
		}
	}

	// 处理DNS响应 - 这里是关键！
	if dns.QR && len(dns.Answers) > 0 && len(dns.Questions) > 0 {
		// 获取查询的原始域名
		originalDomain := strings.TrimSuffix(string(dns.Questions[0].Name), ".")

		c.logger.Infof("🎯 DNS响应: %s", originalDomain)

		// 记录所有解析出的IP地址
		for _, answer := range dns.Answers {
			if answer.Type == layers.DNSTypeA && answer.IP != nil {
				ip := answer.IP.String()
				c.dnsCache.SetIPDomainMapping(ip, originalDomain)
				c.logger.Infof("📝 DNS映射: %s -> %s", ip, originalDomain)
			} else if answer.Type == layers.DNSTypeAAAA && answer.IP != nil {
				ip := answer.IP.String()
				c.dnsCache.SetIPDomainMapping(ip, originalDomain)
				c.logger.Infof("📝 DNS映射(IPv6): %s -> %s", ip, originalDomain)
			}
		}

		// 设置事件的域名为原始域名
		if originalDomain != "" && !strings.HasSuffix(originalDomain, ".local") {
			event.Domain = originalDomain
		}
	}
}

// parseHTTP 解析HTTP请求
func (c *Collector) parseHTTP(packet gopacket.Packet, event *common.NetworkEvent) {
	event.Protocol = "http"

	// 获取应用层数据
	if appLayer := packet.ApplicationLayer(); appLayer != nil {
		payload := string(appLayer.Payload())

		// 简单的HTTP请求解析
		lines := strings.Split(payload, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)

			// 解析Host头
			if strings.HasPrefix(strings.ToLower(line), "host:") {
				host := strings.TrimSpace(line[5:])
				if host != "" {
					event.Domain = host
					c.logger.Debugf("发现HTTP Host: %s", host)
				}
				break
			}
		}
	}
}

// resolveDomain 解析域名
func (c *Collector) resolveDomain(event *common.NetworkEvent) {
	// 如果已经有域名，直接返回
	if event.Domain != "" {
		return
	}

	// 首先检查DNS缓存中是否有原始域名
	if originalDomain, exists := c.dnsCache.GetOriginalDomain(event.DestIP); exists {
		event.Domain = originalDomain
		c.logger.Infof("✅ 从DNS缓存获取域名: %s -> %s", event.DestIP, originalDomain)
		return
	}

	// 为内网IP提供特殊的域名标识（只有在没有DNS缓存时才使用）
	if c.isPrivateIP(event.DestIP) {
		event.Domain = c.generateInternalDomainName(event.DestIP, event.DestPort)
		c.logger.Debugf("生成内网域名: %s -> %s", event.DestIP, event.Domain)
		return
	}

	// 如果缓存中没有，尝试反向DNS解析（异步）
	go func() {
		if names, err := net.LookupAddr(event.DestIP); err == nil && len(names) > 0 {
			domain := strings.TrimSuffix(names[0], ".")
			c.logger.Debugf("反向DNS解析 %s -> %s", event.DestIP, domain)

			// 只有在DNS缓存中没有更好的域名时才使用反向DNS结果
			if _, exists := c.dnsCache.GetOriginalDomain(event.DestIP); !exists {
				// 创建一个新的事件来记录域名信息
				domainEvent := *event
				domainEvent.Domain = domain

				// 直接更新指标
				c.updateMetrics(&domainEvent)
			}
		}
	}()
}

// generateInternalDomainName 为内网IP生成有意义的域名标识
func (c *Collector) generateInternalDomainName(ip string, port int) string {
	// 根据端口推断服务类型
	serviceMap := map[int]string{
		22:    "ssh",
		80:    "http",
		443:   "https",
		3306:  "mysql",
		5432:  "postgresql",
		6379:  "redis",
		27017: "mongodb",
		9200:  "elasticsearch",
		8080:  "http-alt",
		8443:  "https-alt",
		3389:  "rdp",
		5900:  "vnc",
	}

	service := "unknown"
	if s, exists := serviceMap[port]; exists {
		service = s
	}

	// 生成描述性域名
	return fmt.Sprintf("%s.%s.internal", service, strings.ReplaceAll(ip, ".", "-"))
}

// shouldProcess 判断是否应该处理该事件
func (c *Collector) shouldProcess(event *common.NetworkEvent) bool {
	// 应用过滤规则
	if c.config.Filters.IgnoreLocalhost {
		if event.SourceIP == "127.0.0.1" || event.DestIP == "127.0.0.1" {
			c.logger.Debugf("过滤localhost流量: %s -> %s", event.SourceIP, event.DestIP)
			return false
		}
	}

	// 检查忽略端口
	for _, port := range c.config.Filters.IgnorePorts {
		if event.SourcePort == port || event.DestPort == port {
			c.logger.Debugf("过滤端口 %d: %s:%d -> %s:%d", port, event.SourceIP, event.SourcePort, event.DestIP, event.DestPort)
			return false
		}
	}

	// 检查忽略IP
	for _, ip := range c.config.Filters.IgnoreIPs {
		if event.SourceIP == ip || event.DestIP == ip {
			c.logger.Debugf("过滤IP %s: %s -> %s", ip, event.SourceIP, event.DestIP)
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
		c.logger.Debugf("更新域名统计: %s (总计: %d)", event.Domain, c.metrics.DomainsAccessed[event.Domain])
		
		// 更新域名流量统计
		if c.metrics.DomainTraffic[event.Domain] == nil {
			c.metrics.DomainTraffic[event.Domain] = &common.DomainTrafficStats{
				Domain: event.Domain,
			}
		}
		
		domainStats := c.metrics.DomainTraffic[event.Domain]
		domainStats.BytesSent += event.BytesSent
		domainStats.BytesReceived += event.BytesRecv
		domainStats.PacketsSent += event.PacketsSent
		domainStats.PacketsRecv += event.PacketsRecv
		domainStats.Connections++
		domainStats.LastAccess = time.Now()
		
		c.logger.Debugf("更新域名流量: %s (发送: %d bytes, 接收: %d bytes, 连接: %d)", 
			event.Domain, domainStats.BytesSent, domainStats.BytesReceived, domainStats.Connections)
	}

	// 更新IP访问统计
	if event.DestIP != "" {
		c.metrics.IPsAccessed[event.DestIP]++
		c.logger.Debugf("更新IP统计: %s (总计: %d)", event.DestIP, c.metrics.IPsAccessed[event.DestIP])
	}

	// 更新协议统计
	if event.Protocol != "" {
		c.metrics.ProtocolStats[event.Protocol]++
		c.logger.Debugf("更新协议统计: %s (总计: %d)", event.Protocol, c.metrics.ProtocolStats[event.Protocol])
	}

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
		DomainTraffic:    copyDomainTrafficMap(c.metrics.DomainTraffic),
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
		DomainTraffic:   make(map[string]*common.DomainTrafficStats),
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

func copyDomainTrafficMap(src map[string]*common.DomainTrafficStats) map[string]*common.DomainTrafficStats {
	dst := make(map[string]*common.DomainTrafficStats)
	for k, v := range src {
		if v != nil {
			dst[k] = &common.DomainTrafficStats{
				Domain:        v.Domain,
				BytesSent:     v.BytesSent,
				BytesReceived: v.BytesReceived,
				PacketsSent:   v.PacketsSent,
				PacketsRecv:   v.PacketsRecv,
				Connections:   v.Connections,
				LastAccess:    v.LastAccess,
			}
		}
	}
	return dst
}
