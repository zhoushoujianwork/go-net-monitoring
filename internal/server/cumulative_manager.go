package server

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go-net-monitoring/internal/common"

	"github.com/sirupsen/logrus"
)

// CumulativeManager 累计指标管理器
type CumulativeManager struct {
	storage              *common.ServerMetricsStorage
	mutex                sync.RWMutex
	logger               *logrus.Logger
	agentTimeoutDuration time.Duration
	cleanupInterval      time.Duration
	stopChan             chan struct{}

	// 配置选项
	enableRestartDetection bool
	enableBaselineTracking bool
}

// NewCumulativeManager 创建累计指标管理器
func NewCumulativeManager(logger *logrus.Logger, enableRestartDetection, enableBaselineTracking bool) *CumulativeManager {
	cm := &CumulativeManager{
		storage: &common.ServerMetricsStorage{
			RawMetrics:       make(map[string]common.MetricsReport),
			GlobalCumulative: make(map[string]common.DomainMetrics),
			RestartBaselines: make(map[string]map[string]common.DomainMetrics),
			AgentStates:      make(map[string]common.AgentState),
			LastUpdated:      time.Now(),
		},
		logger:                 logger,
		agentTimeoutDuration:   time.Minute * 5, // 5分钟超时
		cleanupInterval:        time.Hour,       // 1小时清理一次
		stopChan:               make(chan struct{}),
		enableRestartDetection: enableRestartDetection,
		enableBaselineTracking: enableBaselineTracking,
	}

	// 启动清理协程
	go cm.periodicCleanup()

	return cm
}

// ProcessMetrics 处理Agent上报的指标
func (cm *CumulativeManager) ProcessMetrics(report common.MetricsReport) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	agentID := report.AgentID

	// 检测Agent重启
	restartResult := cm.detectAgentRestart(agentID, report)
	if restartResult.IsRestart && cm.enableBaselineTracking {
		cm.handleAgentRestart(agentID, report)
	}

	// 更新Agent状态
	cm.updateAgentState(agentID, report, restartResult.IsRestart)

	// 存储原始指标
	cm.storage.RawMetrics[agentID] = report

	// 计算并更新累计指标
	cm.updateCumulativeMetrics(agentID, report, restartResult.IsRestart)

	cm.storage.LastUpdated = time.Now()

	cm.logger.WithFields(logrus.Fields{
		"agent_id":         agentID,
		"report_mode":      report.ReportMode,
		"restart_detected": restartResult.IsRestart,
		"domains_count":    len(report.DeltaStats),
	}).Debug("Processed metrics report")

	return nil
}

// detectAgentRestart 检测Agent重启
func (cm *CumulativeManager) detectAgentRestart(agentID string, report common.MetricsReport) common.RestartDetectionResult {
	result := common.RestartDetectionResult{
		CurrentStartup: report.StartupTime,
	}

	if !cm.enableRestartDetection {
		return result
	}

	// 检查是否有历史状态
	agentState, exists := cm.storage.AgentStates[agentID]
	if !exists {
		// 首次上报
		return result
	}

	// 比较启动时间
	if report.StartupTime.After(agentState.LastStartupTime) {
		result.IsRestart = true
		result.PreviousStartup = agentState.LastStartupTime
		result.RestartCount = agentState.RestartCount + 1

		cm.logger.WithFields(logrus.Fields{
			"agent_id":         agentID,
			"previous_startup": result.PreviousStartup,
			"current_startup":  result.CurrentStartup,
			"restart_count":    result.RestartCount,
		}).Info("Agent restart detected")
	}

	// 检查指标是否重置（备用检测方法）
	if !result.IsRestart && len(report.DeltaStats) > 0 {
		lastReport, hasLastReport := cm.storage.RawMetrics[agentID]
		if hasLastReport && len(lastReport.DeltaStats) > 0 {
			// 检查是否有指标值减少（表明重启）
			for domain, currentStats := range report.DeltaStats {
				if lastStats, exists := lastReport.DeltaStats[domain]; exists {
					if currentStats.AccessCount < lastStats.AccessCount ||
						currentStats.BytesSent < lastStats.BytesSent ||
						currentStats.BytesReceived < lastStats.BytesReceived {
						result.IsRestart = true
						result.RestartCount = agentState.RestartCount + 1

						cm.logger.WithFields(logrus.Fields{
							"agent_id": agentID,
							"domain":   domain,
							"reason":   "metrics_decreased",
						}).Info("Agent restart detected by metrics comparison")
						break
					}
				}
			}
		}
	}

	return result
}

// handleAgentRestart 处理Agent重启
func (cm *CumulativeManager) handleAgentRestart(agentID string, report common.MetricsReport) {
	// 保存重启前的基线数据
	if lastReport, exists := cm.storage.RawMetrics[agentID]; exists {
		if cm.storage.RestartBaselines[agentID] == nil {
			cm.storage.RestartBaselines[agentID] = make(map[string]common.DomainMetrics)
		}

		// 将重启前的累计值作为基线
		for domain, stats := range lastReport.DeltaStats {
			if baseline, hasBaseline := cm.storage.RestartBaselines[agentID][domain]; hasBaseline {
				// 累加到现有基线
				cm.storage.RestartBaselines[agentID][domain] = common.MergeMetrics(baseline, stats)
			} else {
				// 创建新基线
				cm.storage.RestartBaselines[agentID][domain] = stats
			}
		}

		cm.logger.WithFields(logrus.Fields{
			"agent_id":         agentID,
			"baseline_domains": len(cm.storage.RestartBaselines[agentID]),
		}).Info("Saved restart baseline")
	}
}

// updateAgentState 更新Agent状态
func (cm *CumulativeManager) updateAgentState(agentID string, report common.MetricsReport, isRestart bool) {
	state, exists := cm.storage.AgentStates[agentID]
	if !exists {
		state = common.AgentState{
			AgentID: agentID,
		}
	}

	state.LastReportTime = report.ReportTime
	state.LastHeartbeat = time.Now()
	state.IsActive = true

	if isRestart || !exists {
		state.LastStartupTime = report.StartupTime
		if isRestart {
			state.RestartCount++
		}
	}

	cm.storage.AgentStates[agentID] = state
}

// updateCumulativeMetrics 更新累计指标
func (cm *CumulativeManager) updateCumulativeMetrics(agentID string, report common.MetricsReport, isRestart bool) {
	// 处理每个域名的指标
	for domain, currentStats := range report.DeltaStats {
		// 计算真实的累计值
		realCumulative := currentStats

		// 如果启用基线跟踪，加上重启前的基线值
		if cm.enableBaselineTracking {
			if baselines, hasAgent := cm.storage.RestartBaselines[agentID]; hasAgent {
				if baseline, hasBaseline := baselines[domain]; hasBaseline {
					realCumulative = common.MergeMetrics(baseline, currentStats)
				}
			}
		}

		// 更新全局累计指标
		if _, exists := cm.storage.GlobalCumulative[domain]; exists {
			// 需要减去该Agent之前的贡献，加上新的贡献
			cm.updateGlobalCumulativeForDomain(domain, agentID, realCumulative)
		} else {
			// 新域名，直接设置
			realCumulative.Domain = domain // 确保域名字段正确设置
			cm.storage.GlobalCumulative[domain] = realCumulative
		}
	}
}

// updateGlobalCumulativeForDomain 更新特定域名的全局累计指标
func (cm *CumulativeManager) updateGlobalCumulativeForDomain(domain, agentID string, newStats common.DomainMetrics) {
	// 获取当前全局统计
	globalStats := cm.storage.GlobalCumulative[domain]

	// 获取该Agent之前的贡献
	var previousContribution common.DomainMetrics
	if lastReport, exists := cm.storage.RawMetrics[agentID]; exists {
		if lastStats, hasDomain := lastReport.DeltaStats[domain]; hasDomain {
			previousContribution = lastStats

			// 如果启用基线跟踪，加上基线值
			if cm.enableBaselineTracking {
				if baselines, hasAgent := cm.storage.RestartBaselines[agentID]; hasAgent {
					if baseline, hasBaseline := baselines[domain]; hasBaseline {
						previousContribution = common.MergeMetrics(baseline, lastStats)
					}
				}
			}
		}
	}

	// 计算新的全局累计值：当前全局值 - 该Agent之前的贡献 + 该Agent新的贡献
	newGlobalStats := globalStats
	newGlobalStats.Domain = domain // 确保域名字段正确设置
	newGlobalStats.AccessCount = globalStats.AccessCount - previousContribution.AccessCount + newStats.AccessCount
	newGlobalStats.BytesSent = globalStats.BytesSent - previousContribution.BytesSent + newStats.BytesSent
	newGlobalStats.BytesReceived = globalStats.BytesReceived - previousContribution.BytesReceived + newStats.BytesReceived
	newGlobalStats.ConnectionCount = globalStats.ConnectionCount - previousContribution.ConnectionCount + newStats.ConnectionCount

	// 使用最新的访问时间
	if newStats.LastAccessTime.After(newGlobalStats.LastAccessTime) {
		newGlobalStats.LastAccessTime = newStats.LastAccessTime
	}

	// 更新协议和端口统计
	cm.updateProtocolAndPortStats(&newGlobalStats, previousContribution, newStats)

	cm.storage.GlobalCumulative[domain] = newGlobalStats
}

// updateProtocolAndPortStats 更新协议和端口统计
func (cm *CumulativeManager) updateProtocolAndPortStats(globalStats *common.DomainMetrics, previous, current common.DomainMetrics) {
	// 初始化映射
	if globalStats.ProtocolStats == nil {
		globalStats.ProtocolStats = make(map[string]int64)
	}
	if globalStats.PortStats == nil {
		globalStats.PortStats = make(map[int]int64)
	}

	// 更新协议统计
	for protocol, count := range previous.ProtocolStats {
		globalStats.ProtocolStats[protocol] -= count
		if globalStats.ProtocolStats[protocol] <= 0 {
			delete(globalStats.ProtocolStats, protocol)
		}
	}
	for protocol, count := range current.ProtocolStats {
		globalStats.ProtocolStats[protocol] += count
	}

	// 更新端口统计
	for port, count := range previous.PortStats {
		globalStats.PortStats[port] -= count
		if globalStats.PortStats[port] <= 0 {
			delete(globalStats.PortStats, port)
		}
	}
	for port, count := range current.PortStats {
		globalStats.PortStats[port] += count
	}
}

// GetCumulativeMetrics 获取累计指标
func (cm *CumulativeManager) GetCumulativeMetrics() (*common.CumulativeMetrics, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// 计算系统总计
	var totalConnections, totalBytesSent, totalBytesReceived int64
	var activeDomains int
	var dataSources []string

	for _, stats := range cm.storage.GlobalCumulative {
		totalConnections += stats.ConnectionCount
		totalBytesSent += stats.BytesSent
		totalBytesReceived += stats.BytesReceived
		activeDomains++
	}

	// 收集活跃的Agent列表
	for agentID, state := range cm.storage.AgentStates {
		if state.IsActive && time.Since(state.LastHeartbeat) < cm.agentTimeoutDuration {
			dataSources = append(dataSources, agentID)
		}
	}

	systemStats := common.SystemMetrics{
		TotalConnections:   totalConnections,
		TotalBytesSent:     totalBytesSent,
		TotalBytesReceived: totalBytesReceived,
		ActiveDomains:      activeDomains,
	}

	// 深拷贝域名统计
	domainStats := make(map[string]common.DomainMetrics)
	for k, v := range cm.storage.GlobalCumulative {
		domainStats[k] = v
	}

	return &common.CumulativeMetrics{
		DomainStats: domainStats,
		SystemStats: systemStats,
		GeneratedAt: time.Now(),
		DataSources: dataSources,
	}, nil
}

// GetDomainMetrics 获取指定域名的累计指标
func (cm *CumulativeManager) GetDomainMetrics(domain string) (*common.DomainMetrics, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if stats, exists := cm.storage.GlobalCumulative[domain]; exists {
		// 返回副本
		result := stats
		return &result, nil
	}

	return nil, fmt.Errorf("domain %s not found", domain)
}

// periodicCleanup 定期清理过期数据
func (cm *CumulativeManager) periodicCleanup() {
	ticker := time.NewTicker(cm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := cm.CleanupExpiredData(cm.agentTimeoutDuration * 2); err != nil {
				cm.logger.WithError(err).Error("Failed to cleanup expired data")
			}
		case <-cm.stopChan:
			return
		}
	}
}

// CleanupExpiredData 清理过期数据
func (cm *CumulativeManager) CleanupExpiredData(maxAge time.Duration) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	expiredAgents := make([]string, 0)

	// 找出过期的Agent
	for agentID, state := range cm.storage.AgentStates {
		if now.Sub(state.LastHeartbeat) > maxAge {
			expiredAgents = append(expiredAgents, agentID)
		}
	}

	// 清理过期Agent的数据
	for _, agentID := range expiredAgents {
		// 从全局累计中减去该Agent的贡献
		if report, exists := cm.storage.RawMetrics[agentID]; exists {
			for domain, stats := range report.DeltaStats {
				cm.removeAgentContributionFromGlobal(domain, agentID, stats)
			}
		}

		// 删除Agent相关数据
		delete(cm.storage.RawMetrics, agentID)
		delete(cm.storage.AgentStates, agentID)
		delete(cm.storage.RestartBaselines, agentID)

		cm.logger.WithField("agent_id", agentID).Info("Cleaned up expired agent data")
	}

	return nil
}

// removeAgentContributionFromGlobal 从全局累计中移除Agent的贡献
func (cm *CumulativeManager) removeAgentContributionFromGlobal(domain, agentID string, stats common.DomainMetrics) {
	if globalStats, exists := cm.storage.GlobalCumulative[domain]; exists {
		// 计算该Agent的真实贡献（包括基线）
		realContribution := stats
		if cm.enableBaselineTracking {
			if baselines, hasAgent := cm.storage.RestartBaselines[agentID]; hasAgent {
				if baseline, hasBaseline := baselines[domain]; hasBaseline {
					realContribution = common.MergeMetrics(baseline, stats)
				}
			}
		}

		// 从全局统计中减去该Agent的贡献
		newGlobalStats := globalStats
		newGlobalStats.Domain = domain // 确保域名字段正确设置
		newGlobalStats.AccessCount -= realContribution.AccessCount
		newGlobalStats.BytesSent -= realContribution.BytesSent
		newGlobalStats.BytesReceived -= realContribution.BytesReceived
		newGlobalStats.ConnectionCount -= realContribution.ConnectionCount

		// 确保不会出现负数
		if newGlobalStats.AccessCount < 0 {
			newGlobalStats.AccessCount = 0
		}
		if newGlobalStats.BytesSent < 0 {
			newGlobalStats.BytesSent = 0
		}
		if newGlobalStats.BytesReceived < 0 {
			newGlobalStats.BytesReceived = 0
		}
		if newGlobalStats.ConnectionCount < 0 {
			newGlobalStats.ConnectionCount = 0
		}

		// 如果所有指标都为0，删除该域名
		if newGlobalStats.AccessCount == 0 && newGlobalStats.BytesSent == 0 &&
			newGlobalStats.BytesReceived == 0 && newGlobalStats.ConnectionCount == 0 {
			delete(cm.storage.GlobalCumulative, domain)
		} else {
			cm.storage.GlobalCumulative[domain] = newGlobalStats
		}
	}
}

// GetStorageSnapshot 获取存储快照（用于调试）
func (cm *CumulativeManager) GetStorageSnapshot() (*common.ServerMetricsStorage, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// 深拷贝存储数据
	data, err := json.Marshal(cm.storage)
	if err != nil {
		return nil, err
	}

	var snapshot common.ServerMetricsStorage
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, err
	}

	return &snapshot, nil
}

// Close 关闭累计指标管理器
func (cm *CumulativeManager) Close() error {
	close(cm.stopChan)
	cm.logger.Info("Cumulative manager closed")
	return nil
}
