package agent

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go-net-monitoring/internal/common"
	"go-net-monitoring/internal/config"
	"go-net-monitoring/pkg/network"
	"go-net-monitoring/pkg/reporter"

	"github.com/sirupsen/logrus"
)

// Agent 网络监控代理 (传统版本 - 已弃用)
// 注意: 此版本已被eBPF版本替代，仅保留用于兼容性
type Agent struct {
	config    *config.AgentConfig
	logger    *logrus.Logger
	reporter         *reporter.Reporter
	interfaceManager *network.InterfaceManager
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	startTime        time.Time
}

// NewAgent 创建新的Agent (传统版本 - 已弃用)
func NewAgent(cfg *config.AgentConfig) (*Agent, error) {
	// 初始化日志
	logger := logrus.New()
	if err := setupLogger(logger, &cfg.Log); err != nil {
		return nil, fmt.Errorf("初始化日志失败: %w", err)
	}

	logger.Warn("传统Agent已弃用，建议使用eBPF版本: cmd/agent-ebpf")

	ctx, cancel := context.WithCancel(context.Background())

	// 创建Reporter
	rep, err := reporter.NewReporter(&cfg.Reporter, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建Reporter失败: %w", err)
	}

	// 创建网络接口管理器
	interfaceManager := network.NewInterfaceManager(logger)

	agent := &Agent{
		config:           cfg,
		logger:           logger,
		reporter:         rep,
		interfaceManager: interfaceManager,
		ctx:              ctx,
		cancel:           cancel,
		startTime:        time.Now(),
	}

	return agent, nil
}

// Start 启动Agent (传统版本 - 已弃用)
func (a *Agent) Start() error {
	a.logger.Warn("启动传统网络监控代理 (已弃用)")
	a.logger.Info("建议使用eBPF版本获得更好的性能")

	// 启动模拟模式
	return a.startSimulationMode()
}

// startSimulationMode 启动模拟模式
func (a *Agent) startSimulationMode() error {
	a.logger.Info("启动模拟监控模式")

	// 启动模拟数据生成
	a.wg.Add(1)
	go a.simulationLoop()

	// 启动数据上报
	a.wg.Add(1)
	go a.reportLoop()

	return nil
}

// simulationLoop 模拟数据循环
func (a *Agent) simulationLoop() {
	defer a.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 生成模拟指标
			metrics := a.generateMockMetrics()
			
			// 上报数据
			if err := a.reporter.Report(metrics); err != nil {
				a.logger.WithError(err).Error("数据上报失败")
			}

		case <-a.ctx.Done():
			return
		}
	}
}

// reportLoop 数据上报循环
func (a *Agent) reportLoop() {
	defer a.wg.Done()

	// 这个版本中reportLoop由simulationLoop处理
	<-a.ctx.Done()
}

// generateMockMetrics 生成模拟指标
func (a *Agent) generateMockMetrics() common.NetworkMetrics {
	now := time.Now()
	
	return common.NetworkMetrics{
		Timestamp:        now,
		HostID:           a.getHostID(),
		Hostname:         a.getHostname(),
		Interface:        a.config.Monitor.Interface,
		TotalConnections: uint64(100 + now.Unix()%50),
		TotalBytesSent:   uint64(64000 + now.Unix()%32000),
		TotalBytesRecv:   uint64(48000 + now.Unix()%24000),
		DomainsAccessed: map[string]uint64{
			"example.com": uint64(10 + now.Unix()%5),
			"api.test.com": uint64(5 + now.Unix()%3),
		},
		IPsAccessed: map[string]uint64{
			"192.168.1.1": uint64(20 + now.Unix()%10),
		},
		ProtocolStats: map[string]uint64{
			"tcp":  uint64(70 + now.Unix()%20),
			"udp":  uint64(20 + now.Unix()%10),
			"http": uint64(30 + now.Unix()%15),
		},
		DomainTraffic: map[string]*common.DomainTrafficStats{
			"example.com": {
				Domain:        "example.com",
				BytesSent:     uint64(32000 + now.Unix()%16000),
				BytesReceived: uint64(24000 + now.Unix()%12000),
				Connections:   uint64(5 + now.Unix()%3),
				LastAccess:    now,
			},
		},
	}
}

// Stop 停止Agent
func (a *Agent) Stop() error {
	a.logger.Info("停止传统网络监控代理")

	// 取消上下文
	a.cancel()

	// 等待所有goroutine结束
	a.wg.Wait()

	a.logger.Info("传统网络监控代理已停止")
	return nil
}

// GetMetrics 获取当前指标
func (a *Agent) GetMetrics() common.NetworkMetrics {
	return a.generateMockMetrics()
}

// getHostID 获取主机ID
func (a *Agent) getHostID() string {
	hostname, _ := os.Hostname()
	return hostname
}

// getHostname 获取主机名
func (a *Agent) getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

// setupLogger 设置日志
func setupLogger(logger *logrus.Logger, cfg *config.LogConfig) error {
	// 设置日志级别
	level, err := logrus.ParseLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("无效的日志级别: %s", cfg.Level)
	}
	logger.SetLevel(level)

	// 设置日志格式
	if cfg.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	// 设置输出
	if cfg.Output == "stdout" {
		logger.SetOutput(os.Stdout)
	} else if cfg.Output != "" {
		file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("无法打开日志文件: %w", err)
		}
		logger.SetOutput(file)
	}

	return nil
}
