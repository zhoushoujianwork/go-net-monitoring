package agent

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go-net-monitoring/internal/common"
	"go-net-monitoring/internal/config"
	"go-net-monitoring/pkg/ebpf/loader"
	"go-net-monitoring/pkg/reporter"

	"github.com/sirupsen/logrus"
)

// EBPFAgent eBPF网络监控代理
type EBPFAgent struct {
	config     *config.AgentConfig
	logger     *logrus.Logger
	xdpLoader  *loader.XDPLoader
	reporter   *reporter.Reporter
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	startTime  time.Time
	
	// 统计数据
	lastStats  *loader.PacketStats
	metrics    common.NetworkMetrics
	mutex      sync.RWMutex
}

// NewEBPFAgent 创建新的eBPF Agent
func NewEBPFAgent(cfg *config.AgentConfig) (*EBPFAgent, error) {
	// 初始化日志
	logger := logrus.New()
	if err := setupLogger(logger, &cfg.Log); err != nil {
		return nil, fmt.Errorf("初始化日志失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建XDP加载器
	xdpLoader := loader.NewXDPLoader(cfg.Monitor.Interface, logger)

	// 创建Reporter
	rep, err := reporter.NewReporter(&cfg.Reporter, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建Reporter失败: %w", err)
	}

	agent := &EBPFAgent{
		config:    cfg,
		logger:    logger,
		xdpLoader: xdpLoader,
		reporter:  rep,
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
		metrics:   common.NetworkMetrics{
			DomainsAccessed: make(map[string]uint64),
			IPsAccessed:     make(map[string]uint64),
			ProtocolStats:   make(map[string]uint64),
			PortStats:       make(map[int]uint64),
			DomainTraffic:   make(map[string]*common.DomainTrafficStats),
			Interface:       cfg.Monitor.Interface,
		},
	}

	return agent, nil
}

// Start 启动eBPF Agent
func (a *EBPFAgent) Start() error {
	a.logger.Info("启动eBPF网络监控代理")

	// 检查eBPF程序文件
	programPath := a.getEBPFProgramPath()
	
	// 尝试加载eBPF程序
	if err := a.loadEBPFProgram(programPath); err != nil {
		a.logger.WithError(err).Warn("eBPF程序加载失败，启用模拟模式")
		return a.startSimulationMode()
	}

	// 启动eBPF监控
	return a.startEBPFMode()
}

// loadEBPFProgram 加载eBPF程序
func (a *EBPFAgent) loadEBPFProgram(programPath string) error {
	// 加载eBPF程序
	if err := a.xdpLoader.Load(programPath); err != nil {
		return fmt.Errorf("加载eBPF程序失败: %w", err)
	}

	// 附加到网络接口
	if err := a.xdpLoader.Attach(); err != nil {
		return fmt.Errorf("附加XDP程序失败: %w", err)
	}

	a.logger.WithField("program", programPath).Info("eBPF程序加载成功")
	return nil
}

// startEBPFMode 启动eBPF模式
func (a *EBPFAgent) startEBPFMode() error {
	a.logger.Info("启动eBPF监控模式")

	// 启动统计收集
	interval := time.Duration(a.config.Monitor.ReportInterval)
	a.xdpLoader.StartStatsCollection(interval, a.handleEBPFStats)

	// 启动数据上报
	a.wg.Add(1)
	go a.reportLoop()

	return nil
}

// startSimulationMode 启动模拟模式
func (a *EBPFAgent) startSimulationMode() error {
	a.logger.Info("启动模拟监控模式")

	// 启动模拟数据生成
	a.wg.Add(1)
	go a.simulationLoop()

	// 启动数据上报
	a.wg.Add(1)
	go a.reportLoop()

	return nil
}

// handleEBPFStats 处理eBPF统计数据
func (a *EBPFAgent) handleEBPFStats(stats *loader.PacketStats) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// 计算增量
	var deltaStats loader.PacketStats
	if a.lastStats != nil {
		deltaStats = loader.PacketStats{
			TotalPackets: stats.TotalPackets - a.lastStats.TotalPackets,
			TotalBytes:   stats.TotalBytes - a.lastStats.TotalBytes,
			TCPPackets:   stats.TCPPackets - a.lastStats.TCPPackets,
			UDPPackets:   stats.UDPPackets - a.lastStats.UDPPackets,
			OtherPackets: stats.OtherPackets - a.lastStats.OtherPackets,
		}
	} else {
		deltaStats = *stats
	}

	// 更新指标
	a.updateMetrics(&deltaStats)
	a.lastStats = stats

	a.logger.WithFields(logrus.Fields{
		"total_packets": stats.TotalPackets,
		"total_bytes":   stats.TotalBytes,
		"tcp_packets":   stats.TCPPackets,
		"udp_packets":   stats.UDPPackets,
		"other_packets": stats.OtherPackets,
	}).Debug("eBPF统计数据更新")
}

// updateMetrics 更新指标数据
func (a *EBPFAgent) updateMetrics(stats *loader.PacketStats) {
	// 更新协议统计
	a.metrics.ProtocolStats["tcp"] += stats.TCPPackets
	a.metrics.ProtocolStats["udp"] += stats.UDPPackets
	a.metrics.ProtocolStats["other"] += stats.OtherPackets

	// 更新总体统计
	a.metrics.TotalConnections += stats.TotalPackets
	a.metrics.TotalBytesSent += stats.TotalBytes / 2    // 简化：假设发送和接收各占一半
	a.metrics.TotalBytesRecv += stats.TotalBytes / 2
	a.metrics.TotalPacketsSent += stats.TotalPackets / 2
	a.metrics.TotalPacketsRecv += stats.TotalPackets / 2

	// 更新时间戳
	a.metrics.Timestamp = time.Now()
}

// simulationLoop 模拟数据循环
func (a *EBPFAgent) simulationLoop() {
	defer a.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var totalPackets, totalBytes uint64

	for {
		select {
		case <-ticker.C:
			// 生成模拟数据
			totalPackets += 100 + uint64(time.Now().Unix()%50)
			totalBytes += 64000 + uint64(time.Now().Unix()%32000)

			mockStats := &loader.PacketStats{
				TotalPackets: totalPackets,
				TotalBytes:   totalBytes,
				TCPPackets:   totalPackets * 70 / 100,
				UDPPackets:   totalPackets * 20 / 100,
				OtherPackets: totalPackets * 10 / 100,
			}

			a.handleEBPFStats(mockStats)

		case <-a.ctx.Done():
			return
		}
	}
}

// reportLoop 数据上报循环
func (a *EBPFAgent) reportLoop() {
	defer a.wg.Done()

	interval := time.Duration(a.config.Monitor.ReportInterval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.mutex.RLock()
			metrics := a.metrics
			a.mutex.RUnlock()

			if err := a.reporter.Report(metrics); err != nil {
				a.logger.WithError(err).Error("数据上报失败")
			}

		case <-a.ctx.Done():
			return
		}
	}
}

// Stop 停止Agent
func (a *EBPFAgent) Stop() error {
	a.logger.Info("停止eBPF网络监控代理")

	// 取消上下文
	a.cancel()

	// 等待所有goroutine结束
	a.wg.Wait()

	// 清理XDP加载器
	if a.xdpLoader != nil {
		if err := a.xdpLoader.Close(); err != nil {
			a.logger.WithError(err).Error("清理XDP加载器失败")
		}
	}

	a.logger.Info("eBPF网络监控代理已停止")
	return nil
}

// GetMetrics 获取当前指标
func (a *EBPFAgent) GetMetrics() common.NetworkMetrics {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.metrics
}

// getEBPFProgramPath 获取eBPF程序路径
func (a *EBPFAgent) getEBPFProgramPath() string {
	// 优先使用Linux版本
	linuxPath := "bin/bpf/xdp_monitor_linux.o"
	if _, err := os.Stat(linuxPath); err == nil {
		return linuxPath
	}
	
	// 回退到通用版本
	return "bin/bpf/xdp_monitor.o"
}
