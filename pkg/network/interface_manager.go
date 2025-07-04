package network

import (
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// InterfaceManager 网络接口管理器
type InterfaceManager struct {
	interfaces map[string]*InterfaceInfo
	mutex      sync.RWMutex
	logger     *logrus.Logger
	hostname   string
}

// NewInterfaceManager 创建网络接口管理器
func NewInterfaceManager(logger *logrus.Logger) *InterfaceManager {
	hostname, _ := os.Hostname()

	return &InterfaceManager{
		interfaces: make(map[string]*InterfaceInfo),
		logger:     logger,
		hostname:   hostname,
	}
}

// RefreshInterfaces 刷新网络接口信息
func (im *InterfaceManager) RefreshInterfaces() error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces: %w", err)
	}

	// 清空旧的接口信息
	im.interfaces = make(map[string]*InterfaceInfo)

	for _, iface := range interfaces {
		// 跳过回环接口和未启用的接口
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		info := &InterfaceInfo{
			Name:         iface.Name,
			DisplayName:  iface.Name,
			HardwareAddr: iface.HardwareAddr.String(),
			MTU:          iface.MTU,
			IsUp:         iface.Flags&net.FlagUp != 0,
			IsLoopback:   iface.Flags&net.FlagLoopback != 0,
			IsVirtual:    im.isVirtualInterface(iface.Name),
			LastSeen:     time.Now(),
		}

		// 获取IP地址
		addrs, err := iface.Addrs()
		if err == nil {
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					// 只添加IPv4地址，过滤掉IPv6
					if ipnet.IP.To4() != nil {
						info.IPAddresses = append(info.IPAddresses, ipnet.IP.String())
					}
				}
			}
		}

		// 只保存有IP地址的接口
		if len(info.IPAddresses) > 0 {
			im.interfaces[info.Name] = info
			im.logger.WithFields(logrus.Fields{
				"interface": info.Name,
				"ips":       info.IPAddresses,
				"mac":       info.HardwareAddr,
			}).Debug("Detected network interface")
		}
	}

	return nil
}

// GetActiveInterface 获取主要的活跃网络接口
func (im *InterfaceManager) GetActiveInterface() *InterfaceInfo {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	// 优先级顺序：eth0 > en0 > wlan0 > 其他
	priorities := []string{"eth0", "en0", "wlan0", "ens", "enp"}

	// 首先尝试按优先级查找
	for _, priority := range priorities {
		for name, info := range im.interfaces {
			if strings.HasPrefix(name, priority) && len(info.IPAddresses) > 0 {
				return info
			}
		}
	}

	// 如果没有找到优先级接口，返回第一个有IP的接口
	for _, info := range im.interfaces {
		if len(info.IPAddresses) > 0 {
			return info
		}
	}

	return nil
}

// GetInterfaceByName 根据名称获取网络接口
func (im *InterfaceManager) GetInterfaceByName(name string) *InterfaceInfo {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	if info, exists := im.interfaces[name]; exists {
		return info
	}
	return nil
}

// GetAllInterfaces 获取所有网络接口
func (im *InterfaceManager) GetAllInterfaces() map[string]*InterfaceInfo {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	// 返回副本
	result := make(map[string]*InterfaceInfo)
	for k, v := range im.interfaces {
		result[k] = v
	}
	return result
}

// GetInterfaceForPacket 根据数据包信息推断网络接口
func (im *InterfaceManager) GetInterfaceForPacket(srcIP, dstIP string) string {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	// 如果源IP匹配某个接口的IP，返回该接口
	for name, info := range im.interfaces {
		for _, ip := range info.IPAddresses {
			if ip == srcIP {
				return name
			}
		}
	}

	// 如果没有匹配，返回主要接口
	if activeInterface := im.GetActiveInterface(); activeInterface != nil {
		return activeInterface.Name
	}

	return "unknown"
}

// GetPrimaryIP 获取主要网络接口的IP地址
func (im *InterfaceManager) GetPrimaryIP() string {
	if activeInterface := im.GetActiveInterface(); activeInterface != nil {
		if len(activeInterface.IPAddresses) > 0 {
			return activeInterface.IPAddresses[0]
		}
	}
	return "unknown"
}

// GetPrimaryMAC 获取主要网络接口的MAC地址
func (im *InterfaceManager) GetPrimaryMAC() string {
	if activeInterface := im.GetActiveInterface(); activeInterface != nil {
		return activeInterface.HardwareAddr
	}
	return "unknown"
}

// GetPrimaryInterfaceName 获取主要网络接口名称
func (im *InterfaceManager) GetPrimaryInterfaceName() string {
	if activeInterface := im.GetActiveInterface(); activeInterface != nil {
		return activeInterface.Name
	}
	return "unknown"
}

// isVirtualInterface 判断是否为虚拟网卡
func (im *InterfaceManager) isVirtualInterface(name string) bool {
	virtualPrefixes := []string{
		"veth",    // Docker veth pairs
		"tap",     // TAP interfaces
		"tun",     // TUN interfaces
		"virbr",   // libvirt bridges
		"vmnet",   // VMware interfaces
		"vboxnet", // VirtualBox interfaces
		"docker",  // Docker interfaces
		"br-",     // Bridge interfaces
	}

	for _, prefix := range virtualPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

// UpdateMetrics 更新网络接口指标
func (im *InterfaceManager) UpdateMetrics(metricsUpdater func(interfaceName, ipAddress, macAddress, hostname string)) {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	for _, info := range im.interfaces {
		for _, ip := range info.IPAddresses {
			metricsUpdater(info.Name, ip, info.HardwareAddr, im.hostname)
		}
	}
}

// GetInterfaceStats 获取接口统计信息
func (im *InterfaceManager) GetInterfaceStats() map[string]map[string]interface{} {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	stats := make(map[string]map[string]interface{})

	for name, info := range im.interfaces {
		stats[name] = map[string]interface{}{
			"name":         info.Name,
			"ip_addresses": info.IPAddresses,
			"mac_address":  info.HardwareAddr,
			"mtu":          info.MTU,
			"is_up":        info.IsUp,
			"is_virtual":   info.IsVirtual,
			"last_seen":    info.LastSeen,
		}
	}

	return stats
}

// String 返回接口管理器的字符串表示
func (im *InterfaceManager) String() string {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	var interfaces []string
	for name, info := range im.interfaces {
		interfaces = append(interfaces, fmt.Sprintf("%s(%v)", name, info.IPAddresses))
	}

	return fmt.Sprintf("InterfaceManager{interfaces: [%s]}", strings.Join(interfaces, ", "))
}
