package agent

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"go-net-monitoring/internal/common"
	"go-net-monitoring/internal/config"
	"go-net-monitoring/pkg/collector"
	"go-net-monitoring/pkg/network"
	"go-net-monitoring/pkg/reporter"

	"github.com/sirupsen/logrus"
)

// Agent 网络监控代理
type Agent struct {
	config    *config.AgentConfig
	logger    *logrus.Logger
	collector interface {
		Start() error
		Stop() error
		GetMetrics() common.NetworkMetrics
	}
	reporter         *reporter.Reporter
	interfaceManager *network.InterfaceManager // 新增：网络接口管理器
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	startTime        time.Time
}

// NewAgent 创建新的Agent
func NewAgent(cfg *config.AgentConfig) (*Agent, error) {
	// 初始化日志
	logger := logrus.New()
	if err := setupLogger(logger, &cfg.Log); err != nil {
		return nil, fmt.Errorf("初始化日志失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建网络接口管理器
	interfaceManager := network.NewInterfaceManager(logger)

	// 刷新网络接口信息
	if err := interfaceManager.RefreshInterfaces(); err != nil {
		logger.WithError(err).Warn("刷新网络接口信息失败，将使用默认值")
	} else {
		// 记录检测到的网络接口
		logger.WithField("interfaces", interfaceManager.String()).Info("检测到网络接口")

		// 如果配置中的接口为空或为默认值，尝试自动检测
		if cfg.Monitor.Interface == "" || cfg.Monitor.Interface == "eth0" {
			if primaryInterface := interfaceManager.GetPrimaryInterfaceName(); primaryInterface != "unknown" {
				cfg.Monitor.Interface = primaryInterface
				logger.WithField("interface", primaryInterface).Info("自动检测到主要网络接口")
			}
		}
	}

	// 创建收集器
	var coll interface {
		Start() error
		Stop() error
		GetMetrics() common.NetworkMetrics
	}

	// 根据环境选择收集器类型
	// 在开发环境或没有sudo权限时使用测试收集器
	if os.Getenv("TEST_MODE") == "true" || os.Getenv("DEV_MODE") == "true" || os.Geteuid() != 0 {
		logger.Info("使用测试收集器（非root权限或测试模式）")
		testCollector, err := collector.NewTestCollector(&cfg.Monitor, logger)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("创建测试收集器失败: %w", err)
		}
		coll = testCollector
	} else {
		logger.Info("使用真实收集器（root权限）")
		realCollector, err := collector.NewCollector(&cfg.Monitor, logger)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("创建收集器失败: %w", err)
		}
		coll = realCollector
	}

	// 创建上报器
	reporter, err := reporter.NewReporter(&cfg.Reporter, logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建上报器失败: %w", err)
	}

	agent := &Agent{
		config:           cfg,
		logger:           logger,
		collector:        coll,
		reporter:         reporter,
		interfaceManager: interfaceManager, // 新增
		ctx:              ctx,
		cancel:           cancel,
		startTime:        time.Now(),
	}

	return agent, nil
}

// Start 启动Agent
func (a *Agent) Start() error {
	a.logger.Info("启动网络监控Agent")

	// 显示网络接口信息
	if err := a.printNetworkInterfaces(); err != nil {
		a.logger.WithError(err).Warn("显示网络接口信息失败")
	}

	// 启动收集器
	a.logger.Debug("正在启动收集器...")
	if err := a.collector.Start(); err != nil {
		return fmt.Errorf("启动收集器失败: %w", err)
	}
	a.logger.Debug("收集器启动成功")

	// 启动上报器
	a.logger.Debug("正在启动上报器...")
	if err := a.reporter.Start(); err != nil {
		return fmt.Errorf("启动上报器失败: %w", err)
	}
	a.logger.Debug("上报器启动成功")

	// 启动指标上报协程
	a.logger.Debug("启动指标上报协程...")
	a.wg.Add(1)
	go a.metricsReporter()

	// 启动健康检查协程
	a.logger.Debug("启动健康检查协程...")
	a.wg.Add(1)
	go a.healthChecker()

	a.logger.Info("网络监控Agent启动成功")
	return nil
}

// Stop 停止Agent
func (a *Agent) Stop() error {
	a.logger.Info("停止网络监控Agent")

	// 取消上下文，通知所有goroutine停止
	a.cancel()

	// 创建一个带超时的上下文用于停止操作
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer stopCancel()

	// 在goroutine中执行停止操作
	stopDone := make(chan error, 2)

	// 停止收集器
	go func() {
		if err := a.collector.Stop(); err != nil {
			a.logger.WithError(err).Error("停止收集器失败")
			stopDone <- err
		} else {
			stopDone <- nil
		}
	}()

	// 停止上报器
	go func() {
		if err := a.reporter.Stop(); err != nil {
			a.logger.WithError(err).Error("停止上报器失败")
			stopDone <- err
		} else {
			stopDone <- nil
		}
	}()

	// 等待停止完成或超时
	stoppedCount := 0
	for stoppedCount < 2 {
		select {
		case <-stopDone:
			stoppedCount++
		case <-stopCtx.Done():
			a.logger.Warn("停止操作超时，强制退出")
			return fmt.Errorf("停止操作超时")
		}
	}

	// 等待所有goroutine结束，但设置超时
	waitDone := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		a.logger.Info("网络监控Agent已停止")
	case <-time.After(3 * time.Second):
		a.logger.Warn("等待goroutine结束超时，强制退出")
	}

	return nil
}

// Run 运行Agent（阻塞直到收到停止信号）
func (a *Agent) Run() error {
	// 启动Agent
	if err := a.Start(); err != nil {
		return err
	}

	a.logger.Info("Agent运行中，按Ctrl+C停止...")

	// 等待停止信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 简单等待信号
	sig := <-sigChan
	a.logger.Infof("收到信号 %v，开始停止Agent", sig)

	// 停止Agent
	return a.Stop()
}

// metricsReporter 指标上报协程
func (a *Agent) metricsReporter() {
	defer func() {
		if r := recover(); r != nil {
			a.logger.Errorf("指标上报协程panic: %v", r)
		}
		a.wg.Done()
	}()

	// 验证上报间隔
	interval := a.config.Monitor.ReportInterval
	if interval <= 0 {
		a.logger.Warn("上报间隔无效，使用默认值30秒")
		interval = 30 * time.Second
	}

	a.logger.Debugf("指标上报间隔: %v", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			a.logger.Debug("指标上报协程收到停止信号")
			return
		case <-ticker.C:
			a.reportMetrics()
		}
	}
}

// reportMetrics 上报指标
func (a *Agent) reportMetrics() {
	defer func() {
		if r := recover(); r != nil {
			a.logger.Errorf("上报指标时panic: %v", r)
		}
	}()

	// 刷新网络接口信息
	if err := a.interfaceManager.RefreshInterfaces(); err != nil {
		a.logger.WithError(err).Debug("刷新网络接口信息失败")
	}

	// 获取当前指标
	metrics := a.collector.GetMetrics()

	// 设置正确的接口信息
	if metrics.Interface == "" || metrics.Interface == "unknown" {
		// 尝试从配置获取接口名
		if a.config.Monitor.Interface != "" {
			metrics.Interface = a.config.Monitor.Interface
		} else {
			// 使用主要接口
			metrics.Interface = a.interfaceManager.GetPrimaryInterfaceName()
		}
	}

	a.logger.Debugf("获取到指标数据: 接口=%s, 连接数=%d, 发送字节=%d, IP数量=%d, 域名数量=%d",
		metrics.Interface, metrics.TotalConnections, metrics.TotalBytesSent,
		len(metrics.IPsAccessed), len(metrics.DomainsAccessed))

	// 打印一些具体的IP和域名统计
	if len(metrics.IPsAccessed) > 0 {
		a.logger.Debug("IP访问统计:")
		count := 0
		for ip, visits := range metrics.IPsAccessed {
			a.logger.Debugf("  %s: %d次", ip, visits)
			count++
			if count >= 5 { // 只打印前5个
				break
			}
		}
	}

	if len(metrics.DomainsAccessed) > 0 {
		a.logger.Debug("域名访问统计:")
		count := 0
		for domain, visits := range metrics.DomainsAccessed {
			a.logger.Debugf("  %s: %d次", domain, visits)
			count++
			if count >= 5 { // 只打印前5个
				break
			}
		}
	}

	// 上报指标
	if err := a.reporter.Report(metrics); err != nil {
		a.logger.WithError(err).Error("上报指标失败")
	} else {
		a.logger.Debug("成功上报指标")
	}
}

// healthChecker 健康检查协程
func (a *Agent) healthChecker() {
	defer func() {
		if r := recover(); r != nil {
			a.logger.Errorf("健康检查协程panic: %v", r)
		}
		a.wg.Done()
	}()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			a.logger.Debug("健康检查协程收到停止信号")
			return
		case <-ticker.C:
			a.performHealthCheck()
		}
	}
}

// performHealthCheck 执行健康检查
func (a *Agent) performHealthCheck() {
	// 检查收集器状态
	// 检查上报器状态
	// 检查系统资源使用情况
	// 记录健康状态日志

	uptime := time.Since(a.startTime)
	reporterStats := a.reporter.GetStats()

	a.logger.WithFields(logrus.Fields{
		"uptime":          uptime.String(),
		"total_reports":   reporterStats.TotalReports,
		"success_reports": reporterStats.SuccessReports,
		"failed_reports":  reporterStats.FailedReports,
		"queue_size":      reporterStats.QueueSize,
		"last_report":     reporterStats.LastReportTime.Format(time.RFC3339),
	}).Info("Agent健康检查")
}

// GetStatus 获取Agent状态
func (a *Agent) GetStatus() map[string]interface{} {
	uptime := time.Since(a.startTime)
	reporterStats := a.reporter.GetStats()

	return map[string]interface{}{
		"status":         "running",
		"uptime":         uptime.String(),
		"start_time":     a.startTime.Format(time.RFC3339),
		"reporter_stats": reporterStats,
		"config": map[string]interface{}{
			"server_url":      a.config.Reporter.ServerURL,
			"report_interval": a.config.Monitor.ReportInterval.String(),
			"protocols":       a.config.Monitor.Protocols,
		},
	}
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
	switch cfg.Format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	default:
		return fmt.Errorf("不支持的日志格式: %s", cfg.Format)
	}

	// 设置日志输出
	switch cfg.Output {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "stderr":
		logger.SetOutput(os.Stderr)
	default:
		// 输出到文件
		file, err := os.OpenFile(cfg.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("打开日志文件失败: %w", err)
		}
		logger.SetOutput(file)
	}

	return nil
}

// printNetworkInterfaces 打印网络接口信息
func (a *Agent) printNetworkInterfaces() error {
	a.logger.Info("=== 网络接口信息 ===")

	// 获取所有网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	a.logger.Infof("发现 %d 个网络接口:", len(interfaces))

	var targetInterface *net.Interface
	for _, iface := range interfaces {
		status := "DOWN"
		if iface.Flags&net.FlagUp != 0 {
			status = "UP"
		}

		a.logger.Infof("  %s: %s (%s) - %s", iface.Name, iface.HardwareAddr, status, iface.Flags)

		// 获取IP地址
		addrs, err := iface.Addrs()
		if err == nil && len(addrs) > 0 {
			for _, addr := range addrs {
				a.logger.Infof("    IP: %s", addr.String())
			}
		}

		// 检查是否是目标接口
		if iface.Name == a.config.Monitor.Interface {
			targetInterface = &iface
		}
	}

	a.logger.Infof("目标监控接口: %s", a.config.Monitor.Interface)
	if targetInterface != nil {
		a.logger.Infof("✅ 接口 %s 存在且可用", targetInterface.Name)
		a.logger.Infof("   MAC地址: %s", targetInterface.HardwareAddr)
		a.logger.Infof("   状态: %s", targetInterface.Flags)

		// 获取接口统计信息
		if stats, err := a.getInterfaceStats(a.config.Monitor.Interface); err == nil {
			a.logger.Infof("   当前统计: 接收 %d 字节, 发送 %d 字节", stats.BytesReceived, stats.BytesSent)
		}
	} else {
		a.logger.Warnf("❌ 接口 %s 不存在", a.config.Monitor.Interface)
		a.logger.Info("可用接口列表:")
		for _, iface := range interfaces {
			if iface.Flags&net.FlagUp != 0 && iface.Name != "lo" {
				a.logger.Infof("  - %s", iface.Name)
			}
		}
	}

	return nil
}

// NetworkStats 网络统计信息
type NetworkStats struct {
	Interface       string
	BytesReceived   uint64
	BytesSent       uint64
	PacketsReceived uint64
	PacketsSent     uint64
	Timestamp       time.Time
}

// getInterfaceStats 获取接口统计信息
func (a *Agent) getInterfaceStats(interfaceName string) (*NetworkStats, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// 跳过前两行（标题行）
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 17 {
			continue
		}

		ifaceName := strings.TrimSuffix(parts[0], ":")
		if ifaceName != interfaceName {
			continue
		}

		bytesReceived, _ := strconv.ParseUint(parts[1], 10, 64)
		packetsReceived, _ := strconv.ParseUint(parts[2], 10, 64)
		bytesSent, _ := strconv.ParseUint(parts[9], 10, 64)
		packetsSent, _ := strconv.ParseUint(parts[10], 10, 64)

		return &NetworkStats{
			Interface:       ifaceName,
			BytesReceived:   bytesReceived,
			BytesSent:       bytesSent,
			PacketsReceived: packetsReceived,
			PacketsSent:     packetsSent,
			Timestamp:       time.Now(),
		}, nil
	}

	return nil, fmt.Errorf("接口 %s 未找到", interfaceName)
}
