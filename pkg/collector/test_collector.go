package collector

import (
	"context"
	"math/rand"
	"os"
	"sync"
	"time"

	"go-net-monitoring/internal/common"
	"go-net-monitoring/internal/config"

	"github.com/sirupsen/logrus"
)

// TestCollector 测试用的收集器（模拟数据）
type TestCollector struct {
	config    *config.MonitorConfig
	logger    *logrus.Logger
	metrics   *NetworkMetrics
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// NewTestCollector 创建测试收集器
func NewTestCollector(cfg *config.MonitorConfig, logger *logrus.Logger) (*TestCollector, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	collector := &TestCollector{
		config:  cfg,
		logger:  logger,
		metrics: NewNetworkMetrics(),
		ctx:     ctx,
		cancel:  cancel,
	}
	
	return collector, nil
}

// Start 启动测试收集器
func (c *TestCollector) Start() error {
	c.logger.Info("启动测试网络流量收集器")
	
	// 启动模拟数据生成协程
	c.wg.Add(1)
	go c.generateTestData()
	
	return nil
}

// Stop 停止测试收集器
func (c *TestCollector) Stop() error {
	c.logger.Info("停止测试网络流量收集器")
	
	c.cancel()
	c.wg.Wait()
	
	return nil
}

// generateTestData 生成测试数据
func (c *TestCollector) generateTestData() {
	defer c.wg.Done()
	
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	// 模拟的域名和IP
	testDomains := []string{
		"google.com", "github.com", "stackoverflow.com", 
		"baidu.com", "taobao.com", "qq.com",
	}
	
	testIPs := []string{
		"8.8.8.8", "1.1.1.1", "114.114.114.114",
		"223.5.5.5", "180.76.76.76", "119.29.29.29",
	}
	
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.generateRandomEvent(testDomains, testIPs)
		}
	}
}

// generateRandomEvent 生成随机事件
func (c *TestCollector) generateRandomEvent(domains, ips []string) {
	c.metrics.mu.Lock()
	defer c.metrics.mu.Unlock()
	
	// 随机选择域名和IP
	domain := domains[rand.Intn(len(domains))]
	ip := ips[rand.Intn(len(ips))]
	
	// 更新统计
	c.metrics.TotalConnections++
	c.metrics.TotalBytesSent += uint64(rand.Intn(10000) + 1000)
	c.metrics.TotalPacketsSent += uint64(rand.Intn(100) + 10)
	
	// 更新域名和IP统计
	c.metrics.DomainsAccessed[domain]++
	c.metrics.IPsAccessed[ip]++
	
	// 更新协议统计
	protocols := []string{"tcp", "udp", "http", "https"}
	protocol := protocols[rand.Intn(len(protocols))]
	c.metrics.ProtocolStats[protocol]++
	
	// 更新端口统计
	ports := []int{80, 443, 53, 22, 3306, 6379}
	port := ports[rand.Intn(len(ports))]
	c.metrics.PortStats[port]++
	
	c.metrics.LastUpdate = time.Now()
	
	c.logger.WithFields(map[string]interface{}{
		"domain":   domain,
		"ip":       ip,
		"protocol": protocol,
		"port":     port,
	}).Debug("生成测试网络事件")
}

// GetMetrics 获取当前指标
func (c *TestCollector) GetMetrics() common.NetworkMetrics {
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

// GetEventChannel 获取事件通道（测试版本返回nil）
func (c *TestCollector) GetEventChannel() <-chan common.NetworkEvent {
	return nil
}
