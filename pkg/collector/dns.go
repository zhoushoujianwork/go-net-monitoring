package collector

import (
	"net"
	"sync"
	"time"
)

// DNSCache DNS缓存
type DNSCache struct {
	cache map[string]*DNSEntry
	mu    sync.RWMutex
	ttl   time.Duration
}

// DNSEntry DNS缓存条目
type DNSEntry struct {
	Domain    string
	IPs       []string
	Timestamp time.Time
}

// NewDNSCache 创建DNS缓存
func NewDNSCache() *DNSCache {
	return &DNSCache{
		cache: make(map[string]*DNSEntry),
		ttl:   5 * time.Minute, // 默认5分钟TTL
	}
}

// Lookup 查找域名对应的IP
func (d *DNSCache) Lookup(ip string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// 检查缓存
	if entry, exists := d.cache[ip]; exists {
		if time.Since(entry.Timestamp) < d.ttl {
			return entry.Domain
		}
		// 缓存过期，删除
		delete(d.cache, ip)
	}
	
	return ""
}

// ReverseLookup 反向DNS查询
func (d *DNSCache) ReverseLookup(ip string) string {
	// 先检查缓存
	if domain := d.Lookup(ip); domain != "" {
		return domain
	}
	
	// 执行反向DNS查询
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	
	domain := names[0]
	
	// 更新缓存
	d.mu.Lock()
	d.cache[ip] = &DNSEntry{
		Domain:    domain,
		IPs:       []string{ip},
		Timestamp: time.Now(),
	}
	d.mu.Unlock()
	
	return domain
}

// Add 添加DNS记录到缓存
func (d *DNSCache) Add(domain string, ips []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	entry := &DNSEntry{
		Domain:    domain,
		IPs:       ips,
		Timestamp: time.Now(),
	}
	
	// 为每个IP添加缓存条目
	for _, ip := range ips {
		d.cache[ip] = entry
	}
}

// Cleanup 清理过期缓存
func (d *DNSCache) Cleanup() {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	now := time.Now()
	for ip, entry := range d.cache {
		if now.Sub(entry.Timestamp) > d.ttl {
			delete(d.cache, ip)
		}
	}
}

// ConnectionTracker 连接跟踪器
type ConnectionTracker struct {
	connections map[string]*ConnectionState
	mu          sync.RWMutex
}

// ConnectionState 连接状态
type ConnectionState struct {
	LocalAddr     string
	RemoteAddr    string
	Protocol      string
	State         string
	PID           int
	ProcessName   string
	StartTime     time.Time
	LastSeen      time.Time
	BytesSent     uint64
	BytesReceived uint64
	PacketsSent   uint64
	PacketsRecv   uint64
}

// NewConnectionTracker 创建连接跟踪器
func NewConnectionTracker() *ConnectionTracker {
	return &ConnectionTracker{
		connections: make(map[string]*ConnectionState),
	}
}

// Track 跟踪连接
func (ct *ConnectionTracker) Track(localAddr, remoteAddr, protocol string) {
	key := ct.makeKey(localAddr, remoteAddr, protocol)
	
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	if conn, exists := ct.connections[key]; exists {
		conn.LastSeen = time.Now()
	} else {
		ct.connections[key] = &ConnectionState{
			LocalAddr:   localAddr,
			RemoteAddr:  remoteAddr,
			Protocol:    protocol,
			StartTime:   time.Now(),
			LastSeen:    time.Now(),
		}
	}
}

// UpdateBytes 更新字节统计
func (ct *ConnectionTracker) UpdateBytes(localAddr, remoteAddr, protocol string, bytesSent, bytesRecv uint64) {
	key := ct.makeKey(localAddr, remoteAddr, protocol)
	
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	if conn, exists := ct.connections[key]; exists {
		conn.BytesSent += bytesSent
		conn.BytesReceived += bytesRecv
		conn.LastSeen = time.Now()
	}
}

// UpdatePackets 更新包统计
func (ct *ConnectionTracker) UpdatePackets(localAddr, remoteAddr, protocol string, packetsSent, packetsRecv uint64) {
	key := ct.makeKey(localAddr, remoteAddr, protocol)
	
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	if conn, exists := ct.connections[key]; exists {
		conn.PacketsSent += packetsSent
		conn.PacketsRecv += packetsRecv
		conn.LastSeen = time.Now()
	}
}

// GetConnections 获取所有连接
func (ct *ConnectionTracker) GetConnections() map[string]*ConnectionState {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	
	result := make(map[string]*ConnectionState)
	for k, v := range ct.connections {
		// 深拷贝
		result[k] = &ConnectionState{
			LocalAddr:     v.LocalAddr,
			RemoteAddr:    v.RemoteAddr,
			Protocol:      v.Protocol,
			State:         v.State,
			PID:           v.PID,
			ProcessName:   v.ProcessName,
			StartTime:     v.StartTime,
			LastSeen:      v.LastSeen,
			BytesSent:     v.BytesSent,
			BytesReceived: v.BytesReceived,
			PacketsSent:   v.PacketsSent,
			PacketsRecv:   v.PacketsRecv,
		}
	}
	
	return result
}

// Cleanup 清理过期连接
func (ct *ConnectionTracker) Cleanup(timeout time.Duration) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	
	now := time.Now()
	for key, conn := range ct.connections {
		if now.Sub(conn.LastSeen) > timeout {
			delete(ct.connections, key)
		}
	}
}

// makeKey 生成连接键
func (ct *ConnectionTracker) makeKey(localAddr, remoteAddr, protocol string) string {
	return protocol + ":" + localAddr + "->" + remoteAddr
}

// SetIPDomainMapping 设置IP到域名的映射
func (d *DNSCache) SetIPDomainMapping(ip, domain string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// 为这个IP创建或更新DNS条目
	entry, exists := d.cache[ip]
	if !exists {
		entry = &DNSEntry{
			Domain:    domain,
			IPs:       []string{ip},
			Timestamp: time.Now(),
		}
		d.cache[ip] = entry
	} else {
		// 更新域名和时间戳
		entry.Domain = domain
		entry.Timestamp = time.Now()
	}
}

// GetOriginalDomain 获取IP对应的原始域名
func (d *DNSCache) GetOriginalDomain(ip string) (string, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	entry, exists := d.cache[ip]
	if !exists {
		return "", false
	}
	
	// 检查是否过期
	if time.Since(entry.Timestamp) > d.ttl {
		return "", false
	}
	
	return entry.Domain, true
}
