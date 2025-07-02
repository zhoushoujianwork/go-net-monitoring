package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
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

// RedisStorage Redis存储实现
type RedisStorage struct {
	client *redis.Client
	config *config.StorageConfig
	ctx    context.Context
}

// NewRedisStorage 创建Redis存储
func NewRedisStorage(cfg *config.StorageConfig) (*RedisStorage, error) {
	// 创建Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		DialTimeout:  cfg.Redis.Timeout,
		ReadTimeout:  cfg.Redis.Timeout,
		WriteTimeout: cfg.Redis.Timeout,
	})

	ctx := context.Background()
	
	// 测试连接
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("连接Redis失败: %w", err)
	}

	storage := &RedisStorage{
		client: rdb,
		config: cfg,
		ctx:    ctx,
	}

	return storage, nil
}

// Store 存储数据到Redis
func (rs *RedisStorage) Store(key string, value interface{}) error {
	// 序列化数据
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %w", err)
	}

	// 存储到Redis
	if rs.config.TTL > 0 {
		// 设置过期时间
		return rs.client.Set(rs.ctx, key, data, rs.config.TTL).Err()
	} else {
		// 永不过期
		return rs.client.Set(rs.ctx, key, data, 0).Err()
	}
}

// Get 从Redis获取数据
func (rs *RedisStorage) Get(key string) (interface{}, error) {
	data, err := rs.client.Get(rs.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("key not found: %s", key)
		}
		return nil, fmt.Errorf("获取数据失败: %w", err)
	}

	// 反序列化数据
	var value interface{}
	if err := json.Unmarshal([]byte(data), &value); err != nil {
		return nil, fmt.Errorf("反序列化数据失败: %w", err)
	}

	return value, nil
}

// Delete 从Redis删除数据
func (rs *RedisStorage) Delete(key string) error {
	return rs.client.Del(rs.ctx, key).Err()
}

// List 从Redis列出数据
func (rs *RedisStorage) List(prefix string) ([]interface{}, error) {
	// 使用SCAN命令查找匹配的键
	pattern := prefix + "*"
	keys, err := rs.client.Keys(rs.ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("查找键失败: %w", err)
	}

	if len(keys) == 0 {
		return []interface{}{}, nil
	}

	// 批量获取数据
	values, err := rs.client.MGet(rs.ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("批量获取数据失败: %w", err)
	}

	var results []interface{}
	for _, val := range values {
		if val != nil {
			// 反序列化数据
			var value interface{}
			if data, ok := val.(string); ok {
				if err := json.Unmarshal([]byte(data), &value); err == nil {
					results = append(results, value)
				}
			}
		}
	}

	return results, nil
}

// Close 关闭Redis连接
func (rs *RedisStorage) Close() error {
	return rs.client.Close()
}

// NewStorage 根据配置创建存储实例
func NewStorage(cfg *config.StorageConfig) (Storage, error) {
	switch strings.ToLower(cfg.Type) {
	case "memory":
		return NewMemoryStorage(cfg)
	case "redis":
		return NewRedisStorage(cfg)
	default:
		return nil, fmt.Errorf("不支持的存储类型: %s", cfg.Type)
	}
}
