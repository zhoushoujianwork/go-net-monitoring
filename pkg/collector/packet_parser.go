package collector

import (
	"net"
	"strings"
	"time"

	"go-net-monitoring/internal/common"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// PacketParser 数据包解析器
type PacketParser struct {
	dnsCache    *DNSCache
	localIPs    map[string]bool
	localNets   []*net.IPNet
}

// NewPacketParser 创建数据包解析器
func NewPacketParser(dnsCache *DNSCache) *PacketParser {
	parser := &PacketParser{
		dnsCache: dnsCache,
		localIPs: make(map[string]bool),
	}
	
	// 初始化本地IP地址
	parser.initLocalIPs()
	
	return parser
}

// initLocalIPs 初始化本地IP地址列表
func (p *PacketParser) initLocalIPs() {
	// 获取本地网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return
	}
	
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
				// 记录本地网络段
				if v.IP.IsPrivate() {
					p.localNets = append(p.localNets, v)
				}
			case *net.IPAddr:
				ip = v.IP
			}
			
			if ip != nil {
				p.localIPs[ip.String()] = true
			}
		}
	}
	
	// 添加常见的本地地址
	p.localIPs["127.0.0.1"] = true
	p.localIPs["::1"] = true
}

// ParsePacket 解析数据包
func (p *PacketParser) ParsePacket(packet gopacket.Packet) *common.NetworkEvent {
	event := &common.NetworkEvent{
		Timestamp: time.Now(),
	}
	
	// 解析以太网层
	if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
		// 可以获取MAC地址信息
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
	
	// 判断流量方向
	event.Direction = p.getTrafficDirection(srcIP, dstIP)
	
	// 只处理出站流量
	if event.Direction != "outbound" {
		return nil
	}
	
	// 解析传输层
	p.parseTransportLayer(packet, event)
	
	// 解析应用层
	p.parseApplicationLayer(packet, event)
	
	// 获取域名信息
	p.resolveDomain(event)
	
	// 获取数据包大小
	event.BytesSent = uint64(len(packet.Data()))
	event.PacketsSent = 1
	
	return event
}

// getTrafficDirection 判断流量方向
func (p *PacketParser) getTrafficDirection(srcIP, dstIP string) string {
	// 如果源IP是本地IP，目标IP是外部IP，则为出站流量
	if p.isLocalIP(srcIP) && !p.isLocalIP(dstIP) {
		return "outbound"
	}
	
	// 如果源IP是外部IP，目标IP是本地IP，则为入站流量
	if !p.isLocalIP(srcIP) && p.isLocalIP(dstIP) {
		return "inbound"
	}
	
	return "local"
}

// isLocalIP 判断是否为本地IP
func (p *PacketParser) isLocalIP(ipStr string) bool {
	// 检查是否在本地IP列表中
	if p.localIPs[ipStr] {
		return true
	}
	
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	
	// 检查是否为私有IP
	if ip.IsLoopback() || ip.IsPrivate() {
		return true
	}
	
	// 检查是否在本地网络段中
	for _, localNet := range p.localNets {
		if localNet.Contains(ip) {
			return true
		}
	}
	
	return false
}

// parseTransportLayer 解析传输层
func (p *PacketParser) parseTransportLayer(packet gopacket.Packet, event *common.NetworkEvent) {
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
func (p *PacketParser) parseApplicationLayer(packet gopacket.Packet, event *common.NetworkEvent) {
	// 解析DNS
	if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
		p.parseDNS(dnsLayer, event)
		return
	}
	
	// 解析HTTP
	if event.Protocol == "tcp" && (event.DestPort == 80 || event.SourcePort == 80) {
		p.parseHTTP(packet, event)
		return
	}
	
	// 解析HTTPS (只能获取SNI)
	if event.Protocol == "tcp" && (event.DestPort == 443 || event.SourcePort == 443) {
		event.Protocol = "https"
		// TODO: 解析TLS SNI获取域名
	}
}

// parseDNS 解析DNS查询
func (p *PacketParser) parseDNS(dnsLayer gopacket.Layer, event *common.NetworkEvent) {
	dns, _ := dnsLayer.(*layers.DNS)
	event.Protocol = "dns"
	
	// 处理DNS查询
	for _, question := range dns.Questions {
		domain := string(question.Name)
		if domain != "" {
			event.Domain = domain
			// 记录DNS查询
			p.dnsCache.Add(domain, []string{event.DestIP})
		}
	}
	
	// 处理DNS响应
	for _, answer := range dns.Answers {
		if answer.Type == layers.DNSTypeA || answer.Type == layers.DNSTypeAAAA {
			domain := string(answer.Name)
			ip := answer.IP.String()
			if domain != "" && ip != "" {
				p.dnsCache.Add(domain, []string{ip})
			}
		}
	}
}

// parseHTTP 解析HTTP请求
func (p *PacketParser) parseHTTP(packet gopacket.Packet, event *common.NetworkEvent) {
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
					// 将域名和IP关联
					p.dnsCache.Add(host, []string{event.DestIP})
				}
				break
			}
		}
	}
}

// resolveDomain 解析域名
func (p *PacketParser) resolveDomain(event *common.NetworkEvent) {
	// 如果已经有域名，直接返回
	if event.Domain != "" {
		return
	}
	
	// 从DNS缓存中查找
	if domain := p.dnsCache.Lookup(event.DestIP); domain != "" {
		event.Domain = domain
		return
	}
	
	// 异步进行反向DNS解析（避免阻塞）
	go func() {
		if domain := p.dnsCache.ReverseLookup(event.DestIP); domain != "" {
			// 这里可以通过channel通知更新事件
			// 或者在后续的数据包中使用缓存的结果
		}
	}()
}
