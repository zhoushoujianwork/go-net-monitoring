package server

import (
	"fmt"
	"sync"
	"time"

	"go-net-monitoring/internal/common"

	"github.com/sirupsen/logrus"
)

// SimpleCumulativeManager 简化的累计指标管理器
type SimpleCumulativeManager struct {
	// 存储Agent的累计数据
	agentMetrics map[string]map[string]common.DomainMetrics // agentID -> domain -> metrics

	// 存储Agent状态
	agentStates map[string]AgentState

	// 全局累计缓存
	globalCache map[string]common.DomainMetrics
	cacheTime   time.Time

	mutex  sync.RWMutex
	logger *logrus.Logger

	// 配置
	enablePersistence bool
	cacheExpiry       time.Duration
}

// AgentState 简化的Agent状态
type AgentState struct {
	AgentID        string    `json:"agent_id"`
	StartupTime    time.Time `json:"startup_time"`
	LastReportTime time.Time `json:"last_report_time"`
	IsActive       bool      `json:"is_active"`
	RestartCount   int       `json:"restart_count"`
}

// NewSimpleCumulativeManager 创建简化的累计指标管理器
func NewSimpleCumulativeManager(logger *logrus.Logger, enablePersistence bool) *SimpleCumulativeManager {
	return &SimpleCumulativeManager{
		agentMetrics:      make(map[string]map[string]common.DomainMetrics),
		agentStates:       make(map[string]AgentState),
		globalCache:       make(map[string]common.DomainMetrics),
		logger:            logger,
		enablePersistence: enablePersistence,
		cacheExpiry:       time.Second * 30, // 30秒缓存过期
	}
}

// ProcessMetrics 处理Agent上报的指标
func (scm *SimpleCumulativeManager) ProcessMetrics(report common.MetricsReport) error {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	agentID := report.AgentID

	// 检测Agent重启
	isRestart := scm.detectRestart(agentID, report)

	// 更新Agent状态
	scm.updateAgentState(agentID, report, isRestart)

	// 存储Agent的累计指标
	if scm.agentMetrics[agentID] == nil {
		scm.agentMetrics[agentID] = make(map[string]common.DomainMetrics)
	}

	// 如果是重启，需要将之前的累计值加到新的值上
	if isRestart && scm.enablePersistence {
		scm.handleRestart(agentID, report)
	} else {
		// 直接更新累计值
		for domain, metrics := range report.DeltaStats {
			metrics.Domain = domain
			scm.agentMetrics[agentID][domain] = metrics
		}
	}

	// 清除全局缓存，强制重新计算
	scm.globalCache = make(map[string]common.DomainMetrics)
	scm.cacheTime = time.Time{}

	scm.logger.WithFields(logrus.Fields{
		"agent_id":   agentID,
		"is_restart": isRestart,
		"domains":    len(report.DeltaStats),
	}).Debug("Processed metrics report")

	return nil
}

// detectRestart 检测Agent重启
func (scm *SimpleCumulativeManager) detectRestart(agentID string, report common.MetricsReport) bool {
	state, exists := scm.agentStates[agentID]
	if !exists {
		return false // 首次上报
	}

	// 通过启动时间检测重启
	return report.StartupTime.After(state.StartupTime)
}

// updateAgentState 更新Agent状态
func (scm *SimpleCumulativeManager) updateAgentState(agentID string, report common.MetricsReport, isRestart bool) {
	state := scm.agentStates[agentID]

	state.AgentID = agentID
	state.LastReportTime = report.ReportTime
	state.IsActive = true

	if isRestart || state.StartupTime.IsZero() {
		state.StartupTime = report.StartupTime
		if isRestart {
			state.RestartCount++
		}
	}

	scm.agentStates[agentID] = state
}

// handleRestart 处理Agent重启
func (scm *SimpleCumulativeManager) handleRestart(agentID string, report common.MetricsReport) {
	// 获取重启前的累计值
	previousMetrics := scm.agentMetrics[agentID]

	// 将新的累计值与重启前的值合并
	for domain, newMetrics := range report.DeltaStats {
		if prevMetrics, exists := previousMetrics[domain]; exists {
			// 合并指标
			mergedMetrics := common.DomainMetrics{
				Domain:          domain,
				AccessCount:     prevMetrics.AccessCount + newMetrics.AccessCount,
				BytesSent:       prevMetrics.BytesSent + newMetrics.BytesSent,
				BytesReceived:   prevMetrics.BytesReceived + newMetrics.BytesReceived,
				ConnectionCount: prevMetrics.ConnectionCount + newMetrics.ConnectionCount,
				LastAccessTime:  newMetrics.LastAccessTime,
			}

			// 合并协议统计
			if mergedMetrics.ProtocolStats == nil {
				mergedMetrics.ProtocolStats = make(map[string]int64)
			}
			for protocol, count := range prevMetrics.ProtocolStats {
				mergedMetrics.ProtocolStats[protocol] += count
			}
			for protocol, count := range newMetrics.ProtocolStats {
				mergedMetrics.ProtocolStats[protocol] += count
			}

			// 合并端口统计
			if mergedMetrics.PortStats == nil {
				mergedMetrics.PortStats = make(map[int]int64)
			}
			for port, count := range prevMetrics.PortStats {
				mergedMetrics.PortStats[port] += count
			}
			for port, count := range newMetrics.PortStats {
				mergedMetrics.PortStats[port] += count
			}

			scm.agentMetrics[agentID][domain] = mergedMetrics
		} else {
			// 新域名，直接设置
			newMetrics.Domain = domain
			scm.agentMetrics[agentID][domain] = newMetrics
		}
	}
}

// GetCumulativeMetrics 获取累计指标
func (scm *SimpleCumulativeManager) GetCumulativeMetrics() (*common.CumulativeMetrics, error) {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()

	// 检查缓存是否有效
	if time.Since(scm.cacheTime) < scm.cacheExpiry && len(scm.globalCache) > 0 {
		return scm.buildCumulativeMetricsFromCache(), nil
	}

	// 重新计算全局累计指标
	globalMetrics := make(map[string]common.DomainMetrics)

	// 遍历所有Agent的指标
	for agentID, agentMetrics := range scm.agentMetrics {
		// 只处理活跃的Agent
		if state, exists := scm.agentStates[agentID]; exists && state.IsActive {
			for domain, metrics := range agentMetrics {
				if existing, exists := globalMetrics[domain]; exists {
					// 合并指标
					existing.AccessCount += metrics.AccessCount
					existing.BytesSent += metrics.BytesSent
					existing.BytesReceived += metrics.BytesReceived
					existing.ConnectionCount += metrics.ConnectionCount

					// 使用最新的访问时间
					if metrics.LastAccessTime.After(existing.LastAccessTime) {
						existing.LastAccessTime = metrics.LastAccessTime
					}

					// 合并协议统计
					if existing.ProtocolStats == nil {
						existing.ProtocolStats = make(map[string]int64)
					}
					for protocol, count := range metrics.ProtocolStats {
						existing.ProtocolStats[protocol] += count
					}

					// 合并端口统计
					if existing.PortStats == nil {
						existing.PortStats = make(map[int]int64)
					}
					for port, count := range metrics.PortStats {
						existing.PortStats[port] += count
					}

					globalMetrics[domain] = existing
				} else {
					// 新域名
					globalMetrics[domain] = metrics
				}
			}
		}
	}

	// 更新缓存
	scm.globalCache = globalMetrics
	scm.cacheTime = time.Now()

	return scm.buildCumulativeMetricsFromCache(), nil
}

// buildCumulativeMetricsFromCache 从缓存构建累计指标
func (scm *SimpleCumulativeManager) buildCumulativeMetricsFromCache() *common.CumulativeMetrics {
	var totalConnections, totalBytesSent, totalBytesReceived int64
	activeDomains := len(scm.globalCache)

	for _, metrics := range scm.globalCache {
		totalConnections += metrics.ConnectionCount
		totalBytesSent += metrics.BytesSent
		totalBytesReceived += metrics.BytesReceived
	}

	// 收集活跃Agent列表
	var dataSources []string
	for agentID, state := range scm.agentStates {
		if state.IsActive {
			dataSources = append(dataSources, agentID)
		}
	}

	return &common.CumulativeMetrics{
		DomainStats: scm.globalCache,
		SystemStats: common.SystemMetrics{
			TotalConnections:   totalConnections,
			TotalBytesSent:     totalBytesSent,
			TotalBytesReceived: totalBytesReceived,
			ActiveDomains:      activeDomains,
		},
		GeneratedAt: time.Now(),
		DataSources: dataSources,
	}
}

// GetDomainMetrics 获取指定域名的累计指标
func (scm *SimpleCumulativeManager) GetDomainMetrics(domain string) (*common.DomainMetrics, error) {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()

	// 如果缓存有效，直接从缓存获取
	if time.Since(scm.cacheTime) < scm.cacheExpiry {
		if metrics, exists := scm.globalCache[domain]; exists {
			result := metrics
			return &result, nil
		}
	}

	// 实时计算该域名的累计指标
	var result common.DomainMetrics
	result.Domain = domain
	found := false

	for agentID, agentMetrics := range scm.agentMetrics {
		if state, exists := scm.agentStates[agentID]; exists && state.IsActive {
			if metrics, exists := agentMetrics[domain]; exists {
				if !found {
					result = metrics
					found = true
				} else {
					result.AccessCount += metrics.AccessCount
					result.BytesSent += metrics.BytesSent
					result.BytesReceived += metrics.BytesReceived
					result.ConnectionCount += metrics.ConnectionCount

					if metrics.LastAccessTime.After(result.LastAccessTime) {
						result.LastAccessTime = metrics.LastAccessTime
					}
				}
			}
		}
	}

	if !found {
		return nil, fmt.Errorf("domain %s not found", domain)
	}

	return &result, nil
}

// CleanupExpiredAgents 清理过期的Agent
func (scm *SimpleCumulativeManager) CleanupExpiredAgents(maxAge time.Duration) error {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()

	now := time.Now()
	expiredAgents := make([]string, 0)

	// 找出过期的Agent
	for agentID, state := range scm.agentStates {
		if now.Sub(state.LastReportTime) > maxAge {
			expiredAgents = append(expiredAgents, agentID)
		}
	}

	// 清理过期Agent
	for _, agentID := range expiredAgents {
		delete(scm.agentMetrics, agentID)
		delete(scm.agentStates, agentID)

		scm.logger.WithField("agent_id", agentID).Info("Cleaned up expired agent")
	}

	// 如果有Agent被清理，清除缓存
	if len(expiredAgents) > 0 {
		scm.globalCache = make(map[string]common.DomainMetrics)
		scm.cacheTime = time.Time{}
	}

	return nil
}

// GetAgentStates 获取所有Agent状态
func (scm *SimpleCumulativeManager) GetAgentStates() map[string]AgentState {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()

	// 深拷贝
	result := make(map[string]AgentState)
	for k, v := range scm.agentStates {
		result[k] = v
	}

	return result
}

// Close 关闭管理器
func (scm *SimpleCumulativeManager) Close() error {
	scm.logger.Info("Simple cumulative manager closed")
	return nil
}
