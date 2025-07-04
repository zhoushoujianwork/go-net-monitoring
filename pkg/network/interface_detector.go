package network

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// InterfaceInfo 网卡信息
type InterfaceInfo struct {
	Name         string    `json:"name"`
	DisplayName  string    `json:"display_name"`
	HardwareAddr string    `json:"hardware_addr"`
	IPAddresses  []string  `json:"ip_addresses"`
	IsUp         bool      `json:"is_up"`
	IsLoopback   bool      `json:"is_loopback"`
	IsVirtual    bool      `json:"is_virtual"`
	MTU          int       `json:"mtu"`
	Speed        int64     `json:"speed_mbps"` // Mbps
	LastSeen     time.Time `json:"last_seen"`
}

// InterfaceDetector 网卡检测器
type InterfaceDetector struct {
	interfaces map[string]*InterfaceInfo
	config     InterfaceConfig
}

// InterfaceConfig 网卡配置
type InterfaceConfig struct {
	IncludeLoopback bool     `yaml:"include_loopback"`
	IncludeDocker   bool     `yaml:"include_docker"`
	IncludeVirtual  bool     `yaml:"include_virtual"`
	AutoDetect      bool     `yaml:"auto_detect"`
	Whitelist       []string `yaml:"whitelist"`
	Blacklist       []string `yaml:"blacklist"`
}

// NewInterfaceDetector 创建网卡检测器
func NewInterfaceDetector(config InterfaceConfig) *InterfaceDetector {
	return &InterfaceDetector{
		interfaces: make(map[string]*InterfaceInfo),
		config:     config,
	}
}

// DetectInterfaces 检测系统网卡
func (d *InterfaceDetector) DetectInterfaces() ([]*InterfaceInfo, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var result []*InterfaceInfo

	for _, iface := range interfaces {
		info := &InterfaceInfo{
			Name:         iface.Name,
			DisplayName:  iface.Name,
			HardwareAddr: iface.HardwareAddr.String(),
			MTU:          iface.MTU,
			IsUp:         iface.Flags&net.FlagUp != 0,
			IsLoopback:   iface.Flags&net.FlagLoopback != 0,
			IsVirtual:    d.isVirtualInterface(iface.Name),
			LastSeen:     time.Now(),
		}

		// 获取IP地址
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok {
					info.IPAddresses = append(info.IPAddresses, ipnet.IP.String())
				}
			}
		}

		// 应用过滤规则
		if d.shouldIncludeInterface(info) {
			result = append(result, info)
			d.interfaces[info.Name] = info
		}
	}

	return result, nil
}

// shouldIncludeInterface 判断是否应该包含该网卡
func (d *InterfaceDetector) shouldIncludeInterface(info *InterfaceInfo) bool {
	// 检查黑名单
	for _, blacklisted := range d.config.Blacklist {
		if strings.Contains(info.Name, blacklisted) {
			return false
		}
	}

	// 检查白名单
	if len(d.config.Whitelist) > 0 {
		found := false
		for _, whitelisted := range d.config.Whitelist {
			if strings.Contains(info.Name, whitelisted) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查回环接口
	if info.IsLoopback && !d.config.IncludeLoopback {
		return false
	}

	// 检查虚拟接口
	if info.IsVirtual && !d.config.IncludeVirtual {
		return false
	}

	// 检查Docker接口
	if d.isDockerInterface(info.Name) && !d.config.IncludeDocker {
		return false
	}

	// 自动检测模式下，只包含活跃的接口
	if d.config.AutoDetect && !info.IsUp {
		return false
	}

	return true
}

// isVirtualInterface 判断是否为虚拟网卡
func (d *InterfaceDetector) isVirtualInterface(name string) bool {
	virtualPrefixes := []string{
		"veth",    // Docker veth pairs
		"tap",     // TAP interfaces
		"tun",     // TUN interfaces
		"virbr",   // libvirt bridges
		"vmnet",   // VMware interfaces
		"vboxnet", // VirtualBox interfaces
	}

	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

// isDockerInterface 判断是否为Docker网卡
func (d *InterfaceDetector) isDockerInterface(name string) bool {
	dockerPrefixes := []string{
		"docker",
		"br-",
		"veth",
	}

	for _, prefix := range dockerPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

// GetActiveInterfaces 获取活跃的网卡列表
func (d *InterfaceDetector) GetActiveInterfaces() []string {
	var active []string
	for name, info := range d.interfaces {
		if info.IsUp {
			active = append(active, name)
		}
	}
	return active
}

// GetInterfaceInfo 获取指定网卡信息
func (d *InterfaceDetector) GetInterfaceInfo(name string) (*InterfaceInfo, bool) {
	info, exists := d.interfaces[name]
	return info, exists
}

// UpdateInterfaceStats 更新网卡统计信息
func (d *InterfaceDetector) UpdateInterfaceStats(name string) error {
	if info, exists := d.interfaces[name]; exists {
		info.LastSeen = time.Now()
		// 这里可以添加更多的统计信息更新逻辑
		// 比如从 /proc/net/dev 读取流量统计
		return nil
	}
	return fmt.Errorf("interface %s not found", name)
}

// GetInterfaceTrafficStats 获取网卡流量统计 (从系统读取)
func (d *InterfaceDetector) GetInterfaceTrafficStats(name string) (map[string]int64, error) {
	// 这里应该从 /proc/net/dev 或其他系统接口读取实际的流量统计
	// 为了演示，返回模拟数据
	stats := map[string]int64{
		"rx_bytes":   0,
		"tx_bytes":   0,
		"rx_packets": 0,
		"tx_packets": 0,
		"rx_errors":  0,
		"tx_errors":  0,
	}

	return stats, nil
}

// String 返回网卡信息的字符串表示
func (info *InterfaceInfo) String() string {
	return fmt.Sprintf("Interface{Name: %s, IPs: %v, Up: %t, Loopback: %t, Virtual: %t}",
		info.Name, info.IPAddresses, info.IsUp, info.IsLoopback, info.IsVirtual)
}
