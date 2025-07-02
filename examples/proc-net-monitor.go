package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// NetworkStats 网络统计信息
type NetworkStats struct {
	Interface     string
	BytesReceived uint64
	BytesSent     uint64
	PacketsReceived uint64
	PacketsSent   uint64
}

// readNetworkStats 从 /proc/net/dev 读取网络统计信息
func readNetworkStats() ([]NetworkStats, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var stats []NetworkStats
	scanner := bufio.NewScanner(file)
	
	// 跳过前两行（标题行）
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// 解析网络接口统计信息
		parts := strings.Fields(line)
		if len(parts) < 17 {
			continue
		}

		interfaceName := strings.TrimSuffix(parts[0], ":")
		
		// 跳过回环接口和虚拟接口
		if interfaceName == "lo" || strings.HasPrefix(interfaceName, "veth") || 
		   strings.HasPrefix(interfaceName, "br-") || strings.HasPrefix(interfaceName, "docker") {
			continue
		}

		bytesReceived, _ := strconv.ParseUint(parts[1], 10, 64)
		packetsReceived, _ := strconv.ParseUint(parts[2], 10, 64)
		bytesSent, _ := strconv.ParseUint(parts[9], 10, 64)
		packetsSent, _ := strconv.ParseUint(parts[10], 10, 64)

		stats = append(stats, NetworkStats{
			Interface:       interfaceName,
			BytesReceived:   bytesReceived,
			BytesSent:       bytesSent,
			PacketsReceived: packetsReceived,
			PacketsSent:     packetsSent,
		})
	}

	return stats, scanner.Err()
}

// readNetworkConnections 从 /proc/net/tcp 和 /proc/net/udp 读取连接信息
func readNetworkConnections() (int, error) {
	tcpCount := 0
	udpCount := 0

	// 读取 TCP 连接
	if file, err := os.Open("/proc/net/tcp"); err == nil {
		scanner := bufio.NewScanner(file)
		scanner.Scan() // 跳过标题行
		for scanner.Scan() {
			tcpCount++
		}
		file.Close()
	}

	// 读取 UDP 连接
	if file, err := os.Open("/proc/net/udp"); err == nil {
		scanner := bufio.NewScanner(file)
		scanner.Scan() // 跳过标题行
		for scanner.Scan() {
			udpCount++
		}
		file.Close()
	}

	return tcpCount + udpCount, nil
}

func main() {
	fmt.Println("=== 基于 /proc/net 的网络监控示例 ===")
	
	for i := 0; i < 5; i++ {
		fmt.Printf("\n--- 第 %d 次采样 ---\n", i+1)
		
		// 读取网络统计
		stats, err := readNetworkStats()
		if err != nil {
			fmt.Printf("读取网络统计失败: %v\n", err)
			continue
		}

		fmt.Println("网络接口统计:")
		for _, stat := range stats {
			fmt.Printf("  %s: 接收 %d 字节 (%d 包), 发送 %d 字节 (%d 包)\n",
				stat.Interface, stat.BytesReceived, stat.PacketsReceived,
				stat.BytesSent, stat.PacketsSent)
		}

		// 读取连接数
		connections, err := readNetworkConnections()
		if err != nil {
			fmt.Printf("读取连接信息失败: %v\n", err)
		} else {
			fmt.Printf("当前连接数: %d\n", connections)
		}

		if i < 4 {
			time.Sleep(2 * time.Second)
		}
	}
}
