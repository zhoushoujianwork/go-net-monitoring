package server

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go-net-monitoring/internal/config"
)

// MemoryStorage 内存存储实现
type MemoryStorage struct {
	data   map[string]*StorageItem
	mu     sync.RWMutex
	config *config.StorageConfig
}

// StorageItem 存储项
type StorageItem struct {
	Value     interface{}
	Timestamp time.Time
	TTL       time.Duration
}

// NewMemoryStorage 创建内存存储
func NewMemoryStorage(cfg *config.StorageConfig) (*MemoryStorage, error) {
	storage := &MemoryStorage{
		data:   make(map[string]*StorageItem),
		config: cfg,
	}

	// 启动清理协程
	go storage.cleanupWorker()

	return storage, nil
}

// Store 存储数据
func (ms *MemoryStorage) Store(key string, value interface{}) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// 检查是否超过最大条目数
	if len(ms.data) >= ms.config.MaxEntries {
		// 删除最旧的条目
		ms.evictOldest()
	}

	ms.data[key] = &StorageItem{
		Value:     value,
		Timestamp: time.Now(),
		TTL:       ms.config.TTL,
	}

	return nil
}

// Get 获取数据
func (ms *MemoryStorage) Get(key string) (interface{}, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	item, exists := ms.data[key]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", key)
	}

	// 检查是否过期
	if ms.isExpired(item) {
		delete(ms.data, key)
		return nil, fmt.Errorf("key expired: %s", key)
	}

	return item.Value, nil
}

// Delete 删除数据
func (ms *MemoryStorage) Delete(key string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	delete(ms.data, key)
	return nil
}

// List 列出数据
func (ms *MemoryStorage) List(prefix string) ([]interface{}, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var results []interface{}
	for key, item := range ms.data {
		if strings.HasPrefix(key, prefix) {
			// 检查是否过期
			if !ms.isExpired(item) {
				results = append(results, item.Value)
			}
		}
	}

	return results, nil
}

// Close 关闭存储
func (ms *MemoryStorage) Close() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.data = nil
	return nil
}

// isExpired 检查是否过期
func (ms *MemoryStorage) isExpired(item *StorageItem) bool {
	if item.TTL <= 0 {
		return false // 永不过期
	}
	return time.Since(item.Timestamp) > item.TTL
}

// evictOldest 删除最旧的条目
func (ms *MemoryStorage) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range ms.data {
		if oldestKey == "" || item.Timestamp.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.Timestamp
		}
	}

	if oldestKey != "" {
		delete(ms.data, oldestKey)
	}
}

// cleanupWorker 清理工作协程
func (ms *MemoryStorage) cleanupWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ms.cleanup()
	}
}

// cleanup 清理过期数据
func (ms *MemoryStorage) cleanup() {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	var expiredKeys []string
	for key, item := range ms.data {
		if ms.isExpired(item) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(ms.data, key)
	}
}
