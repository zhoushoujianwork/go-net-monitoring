package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// SimpleLoadAgentConfig 简单的配置加载方式
func SimpleLoadAgentConfig(configPath string) (*AgentConfig, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML
	var config AgentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if config.Server.Host == "" {
		config.Server.Host = "localhost"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Monitor.ReportInterval == 0 {
		config.Monitor.ReportInterval = 30 * time.Second
	}
	if config.Monitor.BufferSize == 0 {
		config.Monitor.BufferSize = 1000
	}
	if config.Reporter.ServerURL == "" {
		config.Reporter.ServerURL = "http://localhost:8080/api/v1/metrics"
	}
	if config.Reporter.Timeout == 0 {
		config.Reporter.Timeout = 10 * time.Second
	}
	if config.Reporter.RetryCount == 0 {
		config.Reporter.RetryCount = 3
	}
	if config.Reporter.RetryDelay == 0 {
		config.Reporter.RetryDelay = 5 * time.Second
	}
	if config.Reporter.BatchSize == 0 {
		config.Reporter.BatchSize = 100
	}
	if config.Log.Level == "" {
		config.Log.Level = "info"
	}
	if config.Log.Format == "" {
		config.Log.Format = "json"
	}
	if config.Log.Output == "" {
		config.Log.Output = "stdout"
	}

	return &config, nil
}

// SimpleLoadServerConfig 简单的Server配置加载（支持环境变量）
func SimpleLoadServerConfig(configPath string) (*ServerAppConfig, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML
	var config ServerAppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 环境变量覆盖配置
	if host := os.Getenv("SERVER_HOST"); host != "" {
		config.HTTP.Host = host
	}
	if port := os.Getenv("SERVER_PORT"); port != "" {
		if p, err := parsePort(port); err == nil {
			config.HTTP.Port = p
		}
	}
	if storageType := os.Getenv("STORAGE_TYPE"); storageType != "" {
		config.Storage.Type = storageType
	}
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		config.Storage.Redis.Host = redisHost
	}
	if redisPort := os.Getenv("REDIS_PORT"); redisPort != "" {
		if p, err := parsePort(redisPort); err == nil {
			config.Storage.Redis.Port = p
		}
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		config.Storage.Redis.Password = redisPassword
	}
	if redisDB := os.Getenv("REDIS_DB"); redisDB != "" {
		if db, err := parseInt(redisDB); err == nil {
			config.Storage.Redis.DB = db
		}
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.Log.Level = logLevel
	}

	// 设置默认值
	if config.HTTP.Host == "" {
		config.HTTP.Host = "0.0.0.0"
	}
	if config.HTTP.Port == 0 {
		config.HTTP.Port = 8080
	}
	if config.HTTP.ReadTimeout == 0 {
		config.HTTP.ReadTimeout = 30 * time.Second
	}
	if config.HTTP.WriteTimeout == 0 {
		config.HTTP.WriteTimeout = 30 * time.Second
	}
	if config.Metrics.Path == "" {
		config.Metrics.Path = "/metrics"
	}
	if config.Metrics.Interval == 0 {
		config.Metrics.Interval = 15 * time.Second
	}
	if config.Storage.Type == "" {
		config.Storage.Type = "memory"
	}
	if config.Storage.TTL == 0 {
		config.Storage.TTL = 1 * time.Hour
	}
	if config.Storage.MaxEntries == 0 {
		config.Storage.MaxEntries = 10000
	}
	
	// Redis默认配置
	if config.Storage.Redis.Host == "" {
		config.Storage.Redis.Host = "localhost"
	}
	if config.Storage.Redis.Port == 0 {
		config.Storage.Redis.Port = 6379
	}
	if config.Storage.Redis.PoolSize == 0 {
		config.Storage.Redis.PoolSize = 10
	}
	if config.Storage.Redis.Timeout == 0 {
		config.Storage.Redis.Timeout = 5 * time.Second
	}
	
	if config.Log.Level == "" {
		config.Log.Level = "info"
	}
	if config.Log.Format == "" {
		config.Log.Format = "json"
	}
	if config.Log.Output == "" {
		config.Log.Output = "stdout"
	}

	return &config, nil
}
// parsePort 解析端口号
func parsePort(s string) (int, error) {
	var port int
	if _, err := fmt.Sscanf(s, "%d", &port); err != nil {
		return 0, err
	}
	if port <= 0 || port > 65535 {
		return 0, fmt.Errorf("invalid port: %d", port)
	}
	return port, nil
}

// parseInt 解析整数
func parseInt(s string) (int, error) {
	var i int
	if _, err := fmt.Sscanf(s, "%d", &i); err != nil {
		return 0, err
	}
	return i, nil
}
