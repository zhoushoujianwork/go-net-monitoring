package reporter

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"go-net-monitoring/internal/common"
	"go-net-monitoring/internal/config"

	"github.com/sirupsen/logrus"
)

// Reporter 数据上报器
type Reporter struct {
	config   *config.ReporterConfig
	logger   *logrus.Logger
	client   *http.Client
	queue    chan common.NetworkMetrics
	batch    []common.NetworkMetrics
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.Mutex
	agentID  string
	hostname string
	stats    *ReporterStats
}

// ReporterStats 上报统计
type ReporterStats struct {
	mu             sync.RWMutex
	TotalReports   uint64
	SuccessReports uint64
	FailedReports  uint64
	RetryCount     uint64
	LastReportTime time.Time
	LastError      string
	QueueSize      int
	BatchSize      int
}

// NewReporter 创建新的上报器
func NewReporter(cfg *config.ReporterConfig, logger *logrus.Logger) (*Reporter, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: cfg.Timeout,
	}

	// 配置TLS
	if cfg.EnableTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false, // 生产环境应该验证证书
		}

		if cfg.TLSCertPath != "" && cfg.TLSKeyPath != "" {
			cert, err := tls.LoadX509KeyPair(cfg.TLSCertPath, cfg.TLSKeyPath)
			if err != nil {
				cancel()
				return nil, fmt.Errorf("加载TLS证书失败: %w", err)
			}
			tlsConfig.Certificates = []tls.Certificate{cert}
		}

		client.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	hostname, _ := os.Hostname()
	agentID := generateAgentID(hostname)

	reporter := &Reporter{
		config:   cfg,
		logger:   logger,
		client:   client,
		queue:    make(chan common.NetworkMetrics, cfg.BatchSize*2),
		batch:    make([]common.NetworkMetrics, 0, cfg.BatchSize),
		ctx:      ctx,
		cancel:   cancel,
		agentID:  agentID,
		hostname: hostname,
		stats:    &ReporterStats{},
	}

	return reporter, nil
}

// Start 启动上报器
func (r *Reporter) Start() error {
	r.logger.Info("启动数据上报器")

	// 启动批处理协程
	r.logger.Debug("启动批处理协程...")
	r.wg.Add(1)
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				r.logger.Errorf("批处理协程panic: %v", rec)
			}
			r.wg.Done()
		}()
		r.batchProcessor()
	}()

	// 启动上报协程
	r.logger.Debug("启动上报协程...")
	r.wg.Add(1)
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				r.logger.Errorf("上报协程panic: %v", rec)
			}
			r.wg.Done()
		}()
		r.reportProcessor()
	}()

	r.logger.Debug("上报器所有协程启动完成")
	return nil
}

// Stop 停止上报器
func (r *Reporter) Stop() error {
	r.logger.Info("停止数据上报器")

	// 取消上下文
	r.cancel()

	// 关闭队列
	close(r.queue)

	// 等待所有goroutine结束，但设置超时
	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		r.logger.Debug("所有上报器goroutine已结束")
	case <-time.After(3 * time.Second):
		r.logger.Warn("等待上报器goroutine结束超时")
	}

	return nil
}

// Report 上报网络指标
func (r *Reporter) Report(metrics common.NetworkMetrics) error {
	select {
	case r.queue <- metrics:
		r.updateQueueSize(len(r.queue))
		return nil
	case <-r.ctx.Done():
		return fmt.Errorf("上报器已停止")
	default:
		r.logger.Warn("上报队列已满，丢弃数据")
		return fmt.Errorf("上报队列已满")
	}
}

// batchProcessor 批处理协程
func (r *Reporter) batchProcessor() {
	defer r.wg.Done()

	ticker := time.NewTicker(5 * time.Second) // 减少检查间隔
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			// 处理剩余的批数据
			if len(r.batch) > 0 {
				r.logger.Debug("处理剩余的批数据")
				r.sendBatch()
			}
			return

		case metrics, ok := <-r.queue:
			if !ok {
				// 队列已关闭，处理剩余数据后退出
				if len(r.batch) > 0 {
					r.sendBatch()
				}
				return
			}

			r.mu.Lock()
			r.batch = append(r.batch, metrics)
			shouldSend := len(r.batch) >= r.config.BatchSize
			r.mu.Unlock()

			if shouldSend {
				r.sendBatch()
			}

		case <-ticker.C:
			// 定时发送批数据（即使未满）
			r.mu.Lock()
			shouldSend := len(r.batch) > 0
			r.mu.Unlock()

			if shouldSend {
				r.sendBatch()
			}
		}
	}
}

// reportProcessor 上报处理协程
func (r *Reporter) reportProcessor() {
	defer r.wg.Done()

	// 这里可以添加额外的上报逻辑
	// 比如健康检查、状态上报等

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.sendHeartbeat()
		}
	}
}

// sendBatch 发送批数据
func (r *Reporter) sendBatch() {
	r.mu.Lock()
	if len(r.batch) == 0 {
		r.mu.Unlock()
		return
	}

	batch := make([]common.NetworkMetrics, len(r.batch))
	copy(batch, r.batch)
	r.batch = r.batch[:0] // 清空批数据
	r.mu.Unlock()

	r.updateBatchSize(len(batch))
	r.logger.Debugf("发送批数据，包含 %d 个指标", len(batch))

	// 合并所有指标数据
	mergedMetrics := r.mergeMetrics(batch)

	// 创建上报请求
	request := common.ReportRequest{
		AgentID:   r.agentID,
		Hostname:  r.hostname,
		Timestamp: time.Now(),
		Metrics:   mergedMetrics,
	}

	// 发送请求
	if err := r.sendRequest(request); err != nil {
		r.logger.WithError(err).Error("发送批数据失败")
		r.updateStats(false, err.Error())

		// 重试逻辑
		r.retryRequest(request)
	} else {
		r.logger.Debug("成功发送批数据")
		r.updateStats(true, "")
	}
}

// mergeMetrics 合并多个指标数据
func (r *Reporter) mergeMetrics(batch []common.NetworkMetrics) common.NetworkMetrics {
	if len(batch) == 0 {
		return common.NetworkMetrics{}
	}

	if len(batch) == 1 {
		return batch[0]
	}

	// 合并所有指标
	merged := common.NetworkMetrics{
		Timestamp:       time.Now(),
		HostID:          r.agentID,
		Hostname:        r.hostname,
		DomainsAccessed: make(map[string]uint64),
		IPsAccessed:     make(map[string]uint64),
		ProtocolStats:   make(map[string]uint64),
		PortStats:       make(map[int]uint64),
		DomainTraffic:   make(map[string]*common.DomainTrafficStats),
	}

	for _, metrics := range batch {
		// 合并基础统计
		merged.TotalConnections += metrics.TotalConnections
		merged.TotalBytesSent += metrics.TotalBytesSent
		merged.TotalBytesRecv += metrics.TotalBytesRecv
		merged.TotalPacketsSent += metrics.TotalPacketsSent
		merged.TotalPacketsRecv += metrics.TotalPacketsRecv

		// 合并域名统计
		for domain, count := range metrics.DomainsAccessed {
			merged.DomainsAccessed[domain] += count
		}

		// 合并IP统计
		for ip, count := range metrics.IPsAccessed {
			merged.IPsAccessed[ip] += count
		}

		// 合并协议统计
		for protocol, count := range metrics.ProtocolStats {
			merged.ProtocolStats[protocol] += count
		}

		// 合并端口统计
		for port, count := range metrics.PortStats {
			merged.PortStats[port] += count
		}

		// 合并域名流量统计
		for domain, stats := range metrics.DomainTraffic {
			if stats == nil {
				continue
			}

			if merged.DomainTraffic[domain] == nil {
				merged.DomainTraffic[domain] = &common.DomainTrafficStats{
					Domain: domain,
				}
			}

			mergedStats := merged.DomainTraffic[domain]
			mergedStats.BytesSent += stats.BytesSent
			mergedStats.BytesReceived += stats.BytesReceived
			mergedStats.PacketsSent += stats.PacketsSent
			mergedStats.PacketsRecv += stats.PacketsRecv
			mergedStats.Connections += stats.Connections

			// 使用最新的访问时间
			if stats.LastAccess.After(mergedStats.LastAccess) {
				mergedStats.LastAccess = stats.LastAccess
			}
		}
	}

	r.logger.Debugf("合并后的指标: 域名=%d, IP=%d, 协议=%d, 域名流量=%d",
		len(merged.DomainsAccessed), len(merged.IPsAccessed), len(merged.ProtocolStats), len(merged.DomainTraffic))

	return merged
}

// sendRequest 发送HTTP请求
func (r *Reporter) sendRequest(request common.ReportRequest) error {
	data, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("序列化请求数据失败: %w", err)
	}

	req, err := http.NewRequestWithContext(r.ctx, "POST", r.config.ServerURL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("network-monitor-agent/%s", "1.0.0"))
	req.Header.Set("X-Agent-ID", r.agentID)
	req.Header.Set("X-Hostname", r.hostname)

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var response common.ReportResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("服务器返回错误: %s", response.Message)
	}

	return nil
}

// retryRequest 重试请求
func (r *Reporter) retryRequest(request common.ReportRequest) {
	for i := 0; i < r.config.RetryCount; i++ {
		r.logger.Infof("重试发送请求，第 %d 次", i+1)

		time.Sleep(r.config.RetryDelay)

		if err := r.sendRequest(request); err != nil {
			r.logger.WithError(err).Warnf("第 %d 次重试失败", i+1)
			r.stats.mu.Lock()
			r.stats.RetryCount++
			r.stats.mu.Unlock()
			continue
		}

		r.logger.Info("重试成功")
		r.updateStats(true, "")
		return
	}

	r.logger.Error("重试次数用尽，放弃发送")
	r.updateStats(false, "重试次数用尽")
}

// sendHeartbeat 发送心跳
func (r *Reporter) sendHeartbeat() {
	agentInfo := common.AgentInfo{
		ID:        r.agentID,
		Hostname:  r.hostname,
		Version:   "1.0.0",
		StartTime: time.Now(), // 应该记录实际启动时间
		LastSeen:  time.Now(),
		Status:    "online",
	}

	data, err := json.Marshal(agentInfo)
	if err != nil {
		r.logger.WithError(err).Error("序列化心跳数据失败")
		return
	}

	heartbeatURL := r.config.ServerURL + "/heartbeat"
	req, err := http.NewRequestWithContext(r.ctx, "POST", heartbeatURL, bytes.NewBuffer(data))
	if err != nil {
		r.logger.WithError(err).Error("创建心跳请求失败")
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-ID", r.agentID)

	resp, err := r.client.Do(req)
	if err != nil {
		r.logger.WithError(err).Debug("发送心跳失败")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		r.logger.Debug("心跳发送成功")
	}
}

// updateStats 更新统计信息
func (r *Reporter) updateStats(success bool, errorMsg string) {
	r.stats.mu.Lock()
	defer r.stats.mu.Unlock()

	r.stats.TotalReports++
	r.stats.LastReportTime = time.Now()

	if success {
		r.stats.SuccessReports++
		r.stats.LastError = ""
	} else {
		r.stats.FailedReports++
		r.stats.LastError = errorMsg
	}
}

// updateQueueSize 更新队列大小
func (r *Reporter) updateQueueSize(size int) {
	r.stats.mu.Lock()
	r.stats.QueueSize = size
	r.stats.mu.Unlock()
}

// updateBatchSize 更新批大小
func (r *Reporter) updateBatchSize(size int) {
	r.stats.mu.Lock()
	r.stats.BatchSize = size
	r.stats.mu.Unlock()
}

// GetStats 获取统计信息
func (r *Reporter) GetStats() ReporterStats {
	r.stats.mu.RLock()
	defer r.stats.mu.RUnlock()

	return ReporterStats{
		TotalReports:   r.stats.TotalReports,
		SuccessReports: r.stats.SuccessReports,
		FailedReports:  r.stats.FailedReports,
		RetryCount:     r.stats.RetryCount,
		LastReportTime: r.stats.LastReportTime,
		LastError:      r.stats.LastError,
		QueueSize:      r.stats.QueueSize,
		BatchSize:      r.stats.BatchSize,
	}
}

// generateAgentID 生成Agent ID
func generateAgentID(hostname string) string {
	return fmt.Sprintf("%s-%d", hostname, time.Now().Unix())
}
