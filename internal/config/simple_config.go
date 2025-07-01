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

// SimpleLoadServerConfig 简单的Server配置加载
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
