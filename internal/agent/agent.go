package agent

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go-net-monitoring/internal/common"
	"go-net-monitoring/internal/config"
	"go-net-monitoring/pkg/collector"
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
	reporter  *reporter.Reporter
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	startTime time.Time
}

// NewAgent 创建新的Agent
func NewAgent(cfg *config.AgentConfig) (*Agent, error) {
	// 初始化日志
	logger := logrus.New()
	if err := setupLogger(logger, &cfg.Log); err != nil {
		return nil, fmt.Errorf("初始化日志失败: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// 创建收集器
	var coll interface {
		Start() error
		Stop() error
		GetMetrics() common.NetworkMetrics
	}
	
	// 根据环境选择收集器类型
	if os.Getenv("TEST_MODE") == "true" {
		testCollector, err := collector.NewTestCollector(&cfg.Monitor, logger)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("创建测试收集器失败: %w", err)
		}
		coll = testCollector
	} else {
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
		config:    cfg,
		logger:    logger,
		collector: coll,
		reporter:  reporter,
		ctx:       ctx,
		cancel:    cancel,
		startTime: time.Now(),
	}

	return agent, nil
}

// Start 启动Agent
func (a *Agent) Start() error {
	a.logger.Info("启动网络监控Agent")

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

	// 获取当前指标
	metrics := a.collector.GetMetrics()
	a.logger.Debugf("获取到指标数据: 连接数=%d, 发送字节=%d, IP数量=%d, 域名数量=%d", 
		metrics.TotalConnections, metrics.TotalBytesSent, 
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
		"status":          "running",
		"uptime":          uptime.String(),
		"start_time":      a.startTime.Format(time.RFC3339),
		"reporter_stats":  reporterStats,
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
