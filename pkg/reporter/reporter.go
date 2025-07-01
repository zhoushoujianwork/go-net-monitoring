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
	config     *config.ReporterConfig
	logger     *logrus.Logger
	client     *http.Client
	queue      chan common.NetworkMetrics
	batch      []common.NetworkMetrics
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex
	agentID    string
	hostname   string
	stats      *ReporterStats
}

// ReporterStats 上报统计
type ReporterStats struct {
	mu              sync.RWMutex
	TotalReports    uint64
	SuccessReports  uint64
	FailedReports   uint64
	RetryCount      uint64
	LastReportTime  time.Time
	LastError       string
	QueueSize       int
	BatchSize       int
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
	r.wg.Add(1)
	go r.batchProcessor()
	
	// 启动上报协程
	r.wg.Add(1)
	go r.reportProcessor()
	
	return nil
}

// Stop 停止上报器
func (r *Reporter) Stop() error {
	r.logger.Info("停止数据上报器")
	
	r.cancel()
	close(r.queue)
	
	r.wg.Wait()
	
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
	
	ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次批处理
	defer ticker.Stop()
	
	for {
		select {
		case <-r.ctx.Done():
			// 处理剩余的批数据
			if len(r.batch) > 0 {
				r.sendBatch()
			}
			return
			
		case metrics := <-r.queue:
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
	
	// 创建上报请求
	request := common.ReportRequest{
		AgentID:   r.agentID,
		Hostname:  r.hostname,
		Timestamp: time.Now(),
		Metrics:   batch[0], // 简化处理，实际应该支持批量
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
