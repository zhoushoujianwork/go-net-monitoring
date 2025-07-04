package network

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// HostDetector 主机检测器
type HostDetector struct {
	logger *logrus.Logger
}

// HostInfo 主机信息
type HostInfo struct {
	HostIP        string    `json:"host_ip"`         // 主机IP地址
	Gateway       string    `json:"gateway"`         // 网关地址
	IsContainer   bool      `json:"is_container"`    // 是否在容器中
	IsVM          bool      `json:"is_vm"`           // 是否在虚拟机中
	ContainerType string    `json:"container_type"`  // 容器类型 (docker, podman, etc.)
	VMType        string    `json:"vm_type"`         // 虚拟机类型 (kvm, vmware, etc.)
	DetectedAt    time.Time `json:"detected_at"`     // 检测时间
}

// NewHostDetector 创建主机检测器
func NewHostDetector(logger *logrus.Logger) *HostDetector {
	return &HostDetector{
		logger: logger,
	}
}

// DetectHostInfo 检测主机信息
func (hd *HostDetector) DetectHostInfo() (*HostInfo, error) {
	info := &HostInfo{
		DetectedAt: time.Now(),
	}

	// 检测是否在容器中
	info.IsContainer, info.ContainerType = hd.detectContainer()

	// 检测是否在虚拟机中
	info.IsVM, info.VMType = hd.detectVM()

	// 获取主机IP地址
	hostIP, err := hd.getHostIP()
	if err != nil {
		hd.logger.WithError(err).Warn("Failed to detect host IP")
		info.HostIP = ""
	} else {
		info.HostIP = hostIP
	}

	// 获取网关地址
	gateway, err := hd.getGateway()
	if err != nil {
		hd.logger.WithError(err).Warn("Failed to detect gateway")
		info.Gateway = ""
	} else {
		info.Gateway = gateway
	}

	hd.logger.WithFields(logrus.Fields{
		"host_ip":        info.HostIP,
		"gateway":        info.Gateway,
		"is_container":   info.IsContainer,
		"container_type": info.ContainerType,
		"is_vm":          info.IsVM,
		"vm_type":        info.VMType,
	}).Info("Host detection completed")

	return info, nil
}

// detectContainer 检测是否在容器中运行
func (hd *HostDetector) detectContainer() (bool, string) {
	// 检查 /.dockerenv 文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true, "docker"
	}

	// 检查 /proc/1/cgroup
	if content, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		contentStr := string(content)
		if strings.Contains(contentStr, "docker") {
			return true, "docker"
		}
		if strings.Contains(contentStr, "podman") {
			return true, "podman"
		}
		if strings.Contains(contentStr, "containerd") {
			return true, "containerd"
		}
		if strings.Contains(contentStr, "lxc") {
			return true, "lxc"
		}
	}

	// 检查环境变量
	if os.Getenv("container") != "" {
		return true, os.Getenv("container")
	}

	// 检查 /proc/self/mountinfo
	if content, err := os.ReadFile("/proc/self/mountinfo"); err == nil {
		if strings.Contains(string(content), "docker") {
			return true, "docker"
		}
	}

	return false, ""
}

// detectVM 检测是否在虚拟机中运行
func (hd *HostDetector) detectVM() (bool, string) {
	// 检查 DMI 信息
	if content, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil {
		productName := strings.TrimSpace(string(content))
		switch {
		case strings.Contains(strings.ToLower(productName), "vmware"):
			return true, "vmware"
		case strings.Contains(strings.ToLower(productName), "virtualbox"):
			return true, "virtualbox"
		case strings.Contains(strings.ToLower(productName), "kvm"):
			return true, "kvm"
		case strings.Contains(strings.ToLower(productName), "qemu"):
			return true, "qemu"
		case strings.Contains(strings.ToLower(productName), "xen"):
			return true, "xen"
		}
	}

	// 检查系统供应商
	if content, err := os.ReadFile("/sys/class/dmi/id/sys_vendor"); err == nil {
		vendor := strings.TrimSpace(string(content))
		switch {
		case strings.Contains(strings.ToLower(vendor), "vmware"):
			return true, "vmware"
		case strings.Contains(strings.ToLower(vendor), "innotek"):
			return true, "virtualbox"
		case strings.Contains(strings.ToLower(vendor), "qemu"):
			return true, "qemu"
		case strings.Contains(strings.ToLower(vendor), "microsoft"):
			return true, "hyperv"
		}
	}

	// 检查 /proc/cpuinfo
	if content, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		cpuInfo := strings.ToLower(string(content))
		if strings.Contains(cpuInfo, "hypervisor") {
			return true, "unknown"
		}
	}

	return false, ""
}

// getHostIP 获取主机IP地址
func (hd *HostDetector) getHostIP() (string, error) {
	// 方法1: 通过连接外部地址获取本地IP
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err == nil {
		defer conn.Close()
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		return localAddr.IP.String(), nil
	}

	// 方法2: 通过路由表获取默认网关对应的接口IP
	if ip, err := hd.getIPFromRoute(); err == nil {
		return ip, nil
	}

	// 方法3: 获取第一个非回环的IPv4地址
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("failed to get interfaces: %w", err)
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if ip := ipnet.IP.To4(); ip != nil && !ip.IsLoopback() {
					// 跳过Docker网桥和虚拟接口
					if !hd.isVirtualIP(ip.String()) {
						return ip.String(), nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no suitable host IP found")
}

// getIPFromRoute 从路由表获取IP地址
func (hd *HostDetector) getIPFromRoute() (string, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// 跳过标题行
	if !scanner.Scan() {
		return "", fmt.Errorf("empty route table")
	}

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 8 {
			// 查找默认路由 (Destination = 00000000)
			if fields[1] == "00000000" {
				iface := fields[0]
				// 获取该接口的IP地址
				if ip, err := hd.getInterfaceIP(iface); err == nil {
					return ip, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no default route found")
}

// getInterfaceIP 获取指定接口的IP地址
func (hd *HostDetector) getInterfaceIP(ifaceName string) (string, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return "", err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok {
			if ip := ipnet.IP.To4(); ip != nil && !ip.IsLoopback() {
				return ip.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no IPv4 address found for interface %s", ifaceName)
}

// getGateway 获取默认网关地址
func (hd *HostDetector) getGateway() (string, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// 跳过标题行
	if !scanner.Scan() {
		return "", fmt.Errorf("empty route table")
	}

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 8 {
			// 查找默认路由 (Destination = 00000000)
			if fields[1] == "00000000" {
				// Gateway 在第3个字段，是十六进制格式
				gatewayHex := fields[2]
				if gateway, err := hd.hexToIP(gatewayHex); err == nil {
					return gateway, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no default gateway found")
}

// hexToIP 将十六进制字符串转换为IP地址
func (hd *HostDetector) hexToIP(hexStr string) (string, error) {
	if len(hexStr) != 8 {
		return "", fmt.Errorf("invalid hex string length: %d", len(hexStr))
	}

	// 解析十六进制字符串 (小端序)
	var ip [4]byte
	for i := 0; i < 4; i++ {
		hex := hexStr[i*2 : i*2+2]
		var b byte
		if _, err := fmt.Sscanf(hex, "%02x", &b); err != nil {
			return "", err
		}
		ip[3-i] = b // 小端序转换
	}

	return fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3]), nil
}

// isVirtualIP 判断是否为虚拟IP地址
func (hd *HostDetector) isVirtualIP(ip string) bool {
	// Docker默认网段
	dockerNetworks := []string{
		"172.17.0.0/16", // Docker默认bridge网络
		"172.18.0.0/16", // Docker自定义网络
		"172.19.0.0/16",
		"172.20.0.0/16",
		"172.21.0.0/16",
		"172.22.0.0/16",
		"172.23.0.0/16",
		"172.24.0.0/16",
		"172.25.0.0/16",
		"172.26.0.0/16",
		"172.27.0.0/16",
		"172.28.0.0/16",
		"172.29.0.0/16",
		"172.30.0.0/16",
		"172.31.0.0/16",
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, network := range dockerNetworks {
		_, cidr, err := net.ParseCIDR(network)
		if err != nil {
			continue
		}
		if cidr.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// GetHostIPForInterface 为指定接口获取主机IP地址
func (hd *HostDetector) GetHostIPForInterface(interfaceName string) string {
	hostInfo, err := hd.DetectHostInfo()
	if err != nil {
		hd.logger.WithError(err).Warn("Failed to detect host info")
		return ""
	}

	// 如果不在容器中，返回空字符串
	if !hostInfo.IsContainer {
		return ""
	}

	return hostInfo.HostIP
}
