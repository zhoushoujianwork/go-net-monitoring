package agent

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// PersistentState Agent持久化状态
type PersistentState struct {
	AgentID         string                 `json:"agent_id"`
	StartupTime     time.Time              `json:"startup_time"`
	LastReportTime  time.Time              `json:"last_report_time"`
	CumulativeStats map[string]DomainStats `json:"cumulative_stats"`
	LastSaveTime    time.Time              `json:"last_save_time"`
	Version         string                 `json:"version"`
}

// DomainStats 域名统计信息
type DomainStats struct {
	Domain          string    `json:"domain"`
	AccessCount     int64     `json:"access_count"`
	BytesSent       int64     `json:"bytes_sent"`
	BytesReceived   int64     `json:"bytes_received"`
	ConnectionCount int64     `json:"connection_count"`
	LastAccessTime  time.Time `json:"last_access_time"`
}

// PersistenceManager 持久化管理器
type PersistenceManager struct {
	stateFile    string
	saveInterval time.Duration
	backupCount  int
	state        *PersistentState
	mutex        sync.RWMutex
	logger       *logrus.Logger
	stopChan     chan struct{}
	enabled      bool
}

// NewPersistenceManager 创建持久化管理器
func NewPersistenceManager(stateFile string, saveInterval time.Duration, backupCount int, agentID string, logger *logrus.Logger) *PersistenceManager {
	pm := &PersistenceManager{
		stateFile:    stateFile,
		saveInterval: saveInterval,
		backupCount:  backupCount,
		logger:       logger,
		stopChan:     make(chan struct{}),
		enabled:      stateFile != "",
		state: &PersistentState{
			AgentID:         agentID,
			StartupTime:     time.Now(),
			CumulativeStats: make(map[string]DomainStats),
			Version:         "1.0",
		},
	}

	if pm.enabled {
		pm.loadState()
		go pm.periodicSave()
	}

	return pm
}

// loadState 加载持久化状态
func (pm *PersistenceManager) loadState() error {
	if !pm.enabled {
		return nil
	}

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// 检查状态文件是否存在
	if _, err := os.Stat(pm.stateFile); os.IsNotExist(err) {
		pm.logger.Info("State file does not exist, starting with fresh state")
		return nil
	}

	// 读取状态文件
	data, err := ioutil.ReadFile(pm.stateFile)
	if err != nil {
		pm.logger.WithError(err).Error("Failed to read state file")
		return err
	}

	// 解析状态数据
	var loadedState PersistentState
	if err := json.Unmarshal(data, &loadedState); err != nil {
		pm.logger.WithError(err).Error("Failed to parse state file")
		return err
	}

	// 验证状态版本
	if loadedState.Version != pm.state.Version {
		pm.logger.Warnf("State version mismatch, expected %s, got %s", pm.state.Version, loadedState.Version)
	}

	// 更新启动时间，保留累计统计
	loadedState.StartupTime = time.Now()
	pm.state = &loadedState

	pm.logger.WithFields(logrus.Fields{
		"agent_id":         pm.state.AgentID,
		"cumulative_stats": len(pm.state.CumulativeStats),
		"last_report_time": pm.state.LastReportTime,
	}).Info("Loaded persistent state")

	return nil
}

// saveState 保存持久化状态
func (pm *PersistenceManager) saveState() error {
	if !pm.enabled {
		return nil
	}

	pm.mutex.RLock()
	pm.state.LastSaveTime = time.Now()
	data, err := json.MarshalIndent(pm.state, "", "  ")
	pm.mutex.RUnlock()

	if err != nil {
		pm.logger.WithError(err).Error("Failed to marshal state")
		return err
	}

	// 创建目录
	dir := filepath.Dir(pm.stateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		pm.logger.WithError(err).Error("Failed to create state directory")
		return err
	}

	// 备份现有文件
	if pm.backupCount > 0 {
		pm.rotateBackups()
	}

	// 写入新状态
	if err := ioutil.WriteFile(pm.stateFile, data, 0644); err != nil {
		pm.logger.WithError(err).Error("Failed to write state file")
		return err
	}

	pm.logger.Debug("State saved successfully")
	return nil
}

// rotateBackups 轮转备份文件
func (pm *PersistenceManager) rotateBackups() {
	for i := pm.backupCount - 1; i >= 1; i-- {
		oldFile := fmt.Sprintf("%s.%d", pm.stateFile, i)
		newFile := fmt.Sprintf("%s.%d", pm.stateFile, i+1)
		os.Rename(oldFile, newFile)
	}

	// 备份当前文件
	if _, err := os.Stat(pm.stateFile); err == nil {
		backupFile := fmt.Sprintf("%s.1", pm.stateFile)
		os.Rename(pm.stateFile, backupFile)
	}
}

// periodicSave 定期保存状态
func (pm *PersistenceManager) periodicSave() {
	ticker := time.NewTicker(pm.saveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := pm.saveState(); err != nil {
				pm.logger.WithError(err).Error("Failed to save state periodically")
			}
		case <-pm.stopChan:
			return
		}
	}
}

// UpdateStats 更新统计信息
func (pm *PersistenceManager) UpdateStats(domain string, stats DomainStats) {
	if !pm.enabled {
		return
	}

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	existing, exists := pm.state.CumulativeStats[domain]
	if exists {
		// 累加统计信息
		existing.AccessCount += stats.AccessCount
		existing.BytesSent += stats.BytesSent
		existing.BytesReceived += stats.BytesReceived
		existing.ConnectionCount += stats.ConnectionCount
		existing.LastAccessTime = stats.LastAccessTime
	} else {
		// 新域名
		existing = stats
	}

	pm.state.CumulativeStats[domain] = existing
	pm.state.LastReportTime = time.Now()
}

// GetCumulativeStats 获取累计统计信息
func (pm *PersistenceManager) GetCumulativeStats() map[string]DomainStats {
	if !pm.enabled {
		return make(map[string]DomainStats)
	}

	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// 深拷贝
	result := make(map[string]DomainStats)
	for k, v := range pm.state.CumulativeStats {
		result[k] = v
	}

	return result
}

// GetState 获取持久化状态
func (pm *PersistenceManager) GetState() PersistentState {
	if !pm.enabled {
		return PersistentState{
			AgentID:         "unknown",
			StartupTime:     time.Now(),
			CumulativeStats: make(map[string]DomainStats),
		}
	}

	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// 深拷贝状态
	state := *pm.state
	state.CumulativeStats = make(map[string]DomainStats)
	for k, v := range pm.state.CumulativeStats {
		state.CumulativeStats[k] = v
	}

	return state
}

// Close 关闭持久化管理器
func (pm *PersistenceManager) Close() error {
	if !pm.enabled {
		return nil
	}

	close(pm.stopChan)

	// 最后保存一次状态
	if err := pm.saveState(); err != nil {
		pm.logger.WithError(err).Error("Failed to save state on close")
		return err
	}

	pm.logger.Info("Persistence manager closed")
	return nil
}

// IsEnabled 检查是否启用持久化
func (pm *PersistenceManager) IsEnabled() bool {
	return pm.enabled
}
