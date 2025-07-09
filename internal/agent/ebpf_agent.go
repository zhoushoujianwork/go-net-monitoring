package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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
	config        *config.AgentConfig
	logger        *logrus.Logger
	xdpLoader     *loader.XDPLoader
	reporter      *reporter.Reporter
	onlineManager *OnlineManager  // 新增：上线管理器
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	startTime     time.Time
	
	// 统计数据
	lastStats  *loader.PacketStats
	metrics    common.NetworkMetrics
	mutex      sync.RWMutex
}

// NewEBPFAgent 创建新的eBPF Agent
func NewEBPFAgent(cfg *config.AgentConfig) (*EBPFAgent, error) {
	// 初始化日志
	logger := logrus.New()
	if err := setupEBPFLogger(logger, &cfg.Log); err != nil {
		return nil, fmt.Errorf("初始化日志失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建上线管理器
	onlineManager, err := NewOnlineManager(&cfg.Agent, &cfg.Reporter, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建上线管理器失败: %w", err)
	}

	// 自动检测网卡（如果启用）
	monitorInterface := cfg.Monitor.Interface
	if cfg.Agent.AutoDetectInterface {
		recommendedInterface := onlineManager.GetRecommendedInterface()
		if recommendedInterface != "" {
			logger.WithFields(logrus.Fields{
				"original":    monitorInterface,
				"recommended": recommendedInterface,
			}).Info("自动检测到推荐的监控网卡")
			monitorInterface = recommendedInterface
			cfg.Monitor.Interface = monitorInterface // 更新配置
		}
	}

	// 创建XDP加载器
	xdpLoader := loader.NewXDPLoader(monitorInterface, logger)

	// 创建Reporter
	rep, err := reporter.NewReporter(&cfg.Reporter, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建Reporter失败: %w", err)
	}

	agent := &EBPFAgent{
		config:        cfg,
		logger:        logger,
		xdpLoader:     xdpLoader,
		reporter:      rep,
		onlineManager: onlineManager,
		ctx:           ctx,
		cancel:        cancel,
		startTime:     time.Now(),
		metrics:       common.NetworkMetrics{
			DomainsAccessed: make(map[string]uint64),
			IPsAccessed:     make(map[string]uint64),
			ProtocolStats:   make(map[string]uint64),
			PortStats:       make(map[int]uint64),
			DomainTraffic:   make(map[string]*common.DomainTrafficStats),
			Interface:       monitorInterface,
		},
	}

	return agent, nil
}

// Start 启动eBPF Agent
func (a *EBPFAgent) Start() error {
	a.logger.Info("启动eBPF网络监控代理")

	// 启动上线管理器
	if err := a.onlineManager.Start(); err != nil {
		a.logger.WithError(err).Error("启动上线管理器失败")
		// 不返回错误，允许Agent继续运行
	}

	// 启动Reporter
	if err := a.reporter.Start(); err != nil {
		return fmt.Errorf("启动Reporter失败: %w", err)
	}

	// 获取eBPF程序路径
	programPath := a.getEBPFProgramPath()
	a.logger.WithField("program_path", programPath).Info("准备加载eBPF程序")
	
	// 尝试加载eBPF程序
	if err := a.loadEBPFProgram(programPath); err != nil {
		a.logger.WithError(err).Warn("eBPF程序加载失败")
		
		// 检查是否启用回退模式
		if a.config.EBPF.EnableFallback {
			a.logger.Info("启用模拟模式作为回退方案")
			return a.startSimulationMode()
		} else {
			return fmt.Errorf("eBPF程序加载失败且未启用回退模式: %w", err)
		}
	}

	// 启动eBPF监控
	return a.startEBPFMode()
}

// loadEBPFProgram 加载eBPF程序
func (a *EBPFAgent) loadEBPFProgram(programPath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(programPath); err != nil {
		return fmt.Errorf("eBPF程序文件不存在: %s (%v)", programPath, err)
	}

	a.logger.WithField("program_path", programPath).Info("开始加载eBPF程序")

	// 加载eBPF程序
	if err := a.xdpLoader.Load(programPath); err != nil {
		return fmt.Errorf("加载eBPF程序失败 [%s]: %w", programPath, err)
	}

	// 附加到网络接口
	if err := a.xdpLoader.Attach(); err != nil {
		return fmt.Errorf("附加XDP程序到接口 [%s] 失败: %w", a.config.Monitor.Interface, err)
	}

	a.logger.WithFields(logrus.Fields{
		"program_path": programPath,
		"interface":    a.config.Monitor.Interface,
	}).Info("eBPF程序加载并附加成功")
	
	return nil
}

// startEBPFMode 启动eBPF模式
func (a *EBPFAgent) startEBPFMode() error {
	a.logger.Info("启动eBPF监控模式")

	// 启动统计收集
	interval := a.config.Monitor.ReportInterval
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

	// 更新时间戳和主机信息
	a.metrics.Timestamp = time.Now()
	a.metrics.HostID = a.getHostID()
	a.metrics.Hostname = a.getHostname()
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

	interval := a.config.Monitor.ReportInterval
	a.logger.WithField("interval", interval).Debug("启动数据上报循环")
	
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.logger.Debug("触发数据上报")
			
			a.mutex.RLock()
			metrics := a.metrics
			a.mutex.RUnlock()

			if err := a.reporter.Report(metrics); err != nil {
				a.logger.WithError(err).Error("数据上报失败")
			} else {
				a.logger.Debug("数据上报成功")
			}

		case <-a.ctx.Done():
			a.logger.Debug("上报循环退出")
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

	// 停止上线管理器
	if a.onlineManager != nil {
		if err := a.onlineManager.Stop(); err != nil {
			a.logger.WithError(err).Error("停止上线管理器失败")
		}
	}

	// 停止Reporter
	if a.reporter != nil {
		if err := a.reporter.Stop(); err != nil {
			a.logger.WithError(err).Error("停止Reporter失败")
		}
	}

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

// getEBPFProgramPath 获取eBPF程序路径，支持智能路径解析
func (a *EBPFAgent) getEBPFProgramPath() string {
	// 1. 优先使用配置文件中指定的路径
	programPath := a.config.EBPF.ProgramPath
	if programPath != "" {
		if resolvedPath, err := a.resolveEBPFPath(programPath); err == nil {
			a.logger.WithField("path", resolvedPath).Info("使用配置文件指定的eBPF程序路径")
			return resolvedPath
		}
		a.logger.WithField("path", programPath).Warn("配置文件指定的eBPF程序路径不存在")
	}

	// 2. 尝试备用路径列表
	for _, fallbackPath := range a.config.EBPF.FallbackPaths {
		if resolvedPath, err := a.resolveEBPFPath(fallbackPath); err == nil {
			a.logger.WithField("path", resolvedPath).Info("使用备用eBPF程序路径")
			return resolvedPath
		}
	}

	// 3. 使用默认路径（兼容旧版本）
	defaultPaths := []string{
		"bin/bpf/xdp_monitor_linux.o",
		"bin/bpf/xdp_monitor.o",
		"bpf/xdp_monitor.o",
	}

	for _, defaultPath := range defaultPaths {
		if resolvedPath, err := a.resolveEBPFPath(defaultPath); err == nil {
			a.logger.WithField("path", resolvedPath).Info("使用默认eBPF程序路径")
			return resolvedPath
		}
	}

	// 4. 如果所有路径都失败，返回第一个配置路径用于错误提示
	if programPath != "" {
		return programPath
	}
	return "bpf/xdp_monitor.o"
}

// resolveEBPFPath 解析eBPF程序路径，支持相对路径和绝对路径
func (a *EBPFAgent) resolveEBPFPath(programPath string) (string, error) {
	// 如果是绝对路径，直接检查
	if filepath.IsAbs(programPath) {
		if _, err := os.Stat(programPath); err != nil {
			return "", fmt.Errorf("绝对路径文件不存在: %s", programPath)
		}
		return programPath, nil
	}

	// 相对路径处理：尝试多个位置
	searchPaths := []string{
		// 1. 相对于当前工作目录
		programPath,
		// 2. 相对于二进制文件目录
		"",
		// 3. 相对于项目根目录
		"",
	}

	// 获取二进制文件目录
	if execPath, err := os.Executable(); err == nil {
		binDir := filepath.Dir(execPath)
		searchPaths[1] = filepath.Join(binDir, programPath)
		
		// 尝试项目根目录（假设二进制在 bin/ 或 cmd/ 子目录中）
		parentDir := filepath.Dir(binDir)
		searchPaths[2] = filepath.Join(parentDir, programPath)
	}

	// 按顺序尝试每个路径
	for i, searchPath := range searchPaths {
		if searchPath == "" {
			continue
		}
		
		if _, err := os.Stat(searchPath); err == nil {
			location := []string{"当前工作目录", "二进制文件目录", "项目根目录"}[i]
			a.logger.WithFields(logrus.Fields{
				"original_path": programPath,
				"resolved_path": searchPath,
				"location":      location,
			}).Debug("eBPF程序路径解析成功")
			return searchPath, nil
		}
	}

	return "", fmt.Errorf("在所有搜索路径中都未找到文件: %s", programPath)
}

// getHostID 获取主机ID
func (a *EBPFAgent) getHostID() string {
	hostname, _ := os.Hostname()
	return hostname
}

// getHostname 获取主机名
func (a *EBPFAgent) getHostname() string {
	hostname, _ := os.Hostname()
	return hostname
}

// setupEBPFLogger 设置eBPF Agent日志
func setupEBPFLogger(logger *logrus.Logger, cfg *config.LogConfig) error {
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
