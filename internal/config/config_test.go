package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadAgentConfig(t *testing.T) {
	// 创建临时配置文件
	configContent := `
server:
  host: "test-host"
  port: 9999

monitor:
  interface: "eth0"
  protocols: ["tcp", "udp"]
  report_interval: "60s"
  buffer_size: 2000

reporter:
  server_url: "http://test-server:8080/api/v1/metrics"
  timeout: "15s"
  retry_count: 5

log:
  level: "debug"
  format: "text"
  output: "stderr"
`

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "agent-config-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}
	tmpFile.Close()

	// 加载配置
	cfg, err := LoadAgentConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置
	if cfg.Server.Host != "test-host" {
		t.Errorf("期望 Server.Host 为 'test-host'，实际为 '%s'", cfg.Server.Host)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("期望 Server.Port 为 9999，实际为 %d", cfg.Server.Port)
	}

	if cfg.Monitor.Interface != "eth0" {
		t.Errorf("期望 Monitor.Interface 为 'eth0'，实际为 '%s'", cfg.Monitor.Interface)
	}

	if cfg.Monitor.ReportInterval != 60*time.Second {
		t.Errorf("期望 Monitor.ReportInterval 为 60s，实际为 %v", cfg.Monitor.ReportInterval)
	}

	if len(cfg.Monitor.Protocols) != 2 {
		t.Errorf("期望 Monitor.Protocols 长度为 2，实际为 %d", len(cfg.Monitor.Protocols))
	}

	if cfg.Reporter.RetryCount != 5 {
		t.Errorf("期望 Reporter.RetryCount 为 5，实际为 %d", cfg.Reporter.RetryCount)
	}

	if cfg.Log.Level != "debug" {
		t.Errorf("期望 Log.Level 为 'debug'，实际为 '%s'", cfg.Log.Level)
	}
}

func TestLoadServerConfig(t *testing.T) {
	// 创建临时配置文件
	configContent := `
http:
  host: "0.0.0.0"
  port: 8888
  read_timeout: "45s"
  write_timeout: "45s"

metrics:
  path: "/test-metrics"
  enabled: true
  interval: "20s"

storage:
  type: "memory"
  ttl: "2h"
  max_entries: 20000

log:
  level: "warn"
  format: "json"
  output: "stdout"
`

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "server-config-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}
	tmpFile.Close()

	// 加载配置
	cfg, err := LoadServerConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置
	if cfg.HTTP.Host != "0.0.0.0" {
		t.Errorf("期望 HTTP.Host 为 '0.0.0.0'，实际为 '%s'", cfg.HTTP.Host)
	}

	if cfg.HTTP.Port != 8888 {
		t.Errorf("期望 HTTP.Port 为 8888，实际为 %d", cfg.HTTP.Port)
	}

	if cfg.HTTP.ReadTimeout != 45*time.Second {
		t.Errorf("期望 HTTP.ReadTimeout 为 45s，实际为 %v", cfg.HTTP.ReadTimeout)
	}

	if cfg.Metrics.Path != "/test-metrics" {
		t.Errorf("期望 Metrics.Path 为 '/test-metrics'，实际为 '%s'", cfg.Metrics.Path)
	}

	if cfg.Storage.MaxEntries != 20000 {
		t.Errorf("期望 Storage.MaxEntries 为 20000，实际为 %d", cfg.Storage.MaxEntries)
	}

	if cfg.Log.Level != "warn" {
		t.Errorf("期望 Log.Level 为 'warn'，实际为 '%s'", cfg.Log.Level)
	}
}

func TestLoadConfigWithDefaults(t *testing.T) {
	// 创建空配置文件
	configContent := `{}`

	tmpFile, err := os.CreateTemp("", "empty-config-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("写入配置文件失败: %v", err)
	}
	tmpFile.Close()

	// 加载Agent配置（应该使用默认值）
	cfg, err := LoadAgentConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证默认值
	if cfg.Server.Host != "localhost" {
		t.Errorf("期望默认 Server.Host 为 'localhost'，实际为 '%s'", cfg.Server.Host)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("期望默认 Server.Port 为 8080，实际为 %d", cfg.Server.Port)
	}

	if cfg.Monitor.ReportInterval != 30*time.Second {
		t.Errorf("期望默认 Monitor.ReportInterval 为 30s，实际为 %v", cfg.Monitor.ReportInterval)
	}
}

func TestLoadNonExistentConfig(t *testing.T) {
	// 尝试加载不存在的配置文件
	_, err := LoadAgentConfig("/non/existent/config.yaml")
	if err == nil {
		t.Error("期望加载不存在的配置文件时返回错误")
	}
}
