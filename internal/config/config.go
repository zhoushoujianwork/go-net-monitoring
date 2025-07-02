package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// AgentConfig Agent配置结构
type AgentConfig struct {
	Server   ServerConfig   `yaml:"server"`
	Monitor  MonitorConfig  `yaml:"monitor"`
	Reporter ReporterConfig `yaml:"reporter"`
	Log      LogConfig      `yaml:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Interface    string        `yaml:"interface"`     // 监控的网络接口
	Protocols    []string      `yaml:"protocols"`     // 监控的协议 tcp,udp,http,https
	ReportInterval time.Duration `yaml:"report_interval"` // 上报间隔
	BufferSize   int           `yaml:"buffer_size"`   // 缓冲区大小
	Filters      FilterConfig  `yaml:"filters"`       // 过滤规则
}

// FilterConfig 过滤配置
type FilterConfig struct {
	IgnoreLocalhost bool     `yaml:"ignore_localhost"` // 忽略本地回环
	IgnorePorts     []int    `yaml:"ignore_ports"`     // 忽略的端口
	IgnoreIPs       []string `yaml:"ignore_ips"`       // 忽略的IP
	OnlyDomains     []string `yaml:"only_domains"`     // 只监控特定域名
}

// ReporterConfig 上报配置
type ReporterConfig struct {
	ServerURL    string        `yaml:"server_url"`
	Timeout      time.Duration `yaml:"timeout"`
	RetryCount   int           `yaml:"retry_count"`
	RetryDelay   time.Duration `yaml:"retry_delay"`
	BatchSize    int           `yaml:"batch_size"`
	EnableTLS    bool          `yaml:"enable_tls"`
	TLSCertPath  string        `yaml:"tls_cert_path"`
	TLSKeyPath   string        `yaml:"tls_key_path"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// ServerAppConfig Server应用配置
type ServerAppConfig struct {
	HTTP       HTTPConfig       `yaml:"http"`
	Metrics    MetricsConfig    `yaml:"metrics"`
	Storage    StorageConfig    `yaml:"storage"`
	Log        LogConfig        `yaml:"log"`
}

// HTTPConfig HTTP服务配置
type HTTPConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	EnableTLS    bool          `yaml:"enable_tls"`
	TLSCertPath  string        `yaml:"tls_cert_path"`
	TLSKeyPath   string        `yaml:"tls_key_path"`
}

// MetricsConfig Prometheus指标配置
type MetricsConfig struct {
	Path     string `yaml:"path"`
	Enabled  bool   `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type       string `yaml:"type"`        // memory, redis
	TTL        time.Duration `yaml:"ttl"`  // 数据保留时间
	MaxEntries int    `yaml:"max_entries"` // 最大条目数 (仅memory)
	
	// Redis配置
	Redis RedisConfig `yaml:"redis"`
}

// RedisConfig Redis存储配置
type RedisConfig struct {
	Host     string `yaml:"host"`     // Redis主机地址
	Port     int    `yaml:"port"`     // Redis端口
	Password string `yaml:"password"` // Redis密码
	DB       int    `yaml:"db"`       // Redis数据库编号
	PoolSize int    `yaml:"pool_size"` // 连接池大小
	Timeout  time.Duration `yaml:"timeout"` // 连接超时
}

// LoadAgentConfig 加载Agent配置
func LoadAgentConfig(configPath string) (*AgentConfig, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	setAgentDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config AgentConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 手动处理可能的配置问题
	if config.Reporter.ServerURL == "" {
		config.Reporter.ServerURL = viper.GetString("reporter.server_url")
	}
	
	// 验证配置
	if err := validateAgentConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &config, nil
}

// validateAgentConfig 验证Agent配置
func validateAgentConfig(config *AgentConfig) error {
	// 验证时间间隔
	if config.Monitor.ReportInterval <= 0 {
		config.Monitor.ReportInterval = 30 * time.Second
	}
	
	if config.Reporter.Timeout <= 0 {
		config.Reporter.Timeout = 10 * time.Second
	}
	
	if config.Reporter.RetryDelay <= 0 {
		config.Reporter.RetryDelay = 5 * time.Second
	}
	
	// 验证其他必要字段
	if config.Reporter.ServerURL == "" {
		return fmt.Errorf("reporter.server_url 不能为空")
	}
	
	if config.Monitor.BufferSize <= 0 {
		config.Monitor.BufferSize = 1000
	}
	
	return nil
}

// LoadServerConfig 加载Server配置
func LoadServerConfig(configPath string) (*ServerAppConfig, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	setServerDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config ServerAppConfig
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := validateServerConfig(&config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &config, nil
}

// validateServerConfig 验证Server配置
func validateServerConfig(config *ServerAppConfig) error {
	// 验证时间间隔
	if config.HTTP.ReadTimeout <= 0 {
		config.HTTP.ReadTimeout = 30 * time.Second
	}
	
	if config.HTTP.WriteTimeout <= 0 {
		config.HTTP.WriteTimeout = 30 * time.Second
	}
	
	if config.Metrics.Interval <= 0 {
		config.Metrics.Interval = 15 * time.Second
	}
	
	if config.Storage.TTL <= 0 {
		config.Storage.TTL = 1 * time.Hour
	}
	
	// 验证端口
	if config.HTTP.Port <= 0 || config.HTTP.Port > 65535 {
		return fmt.Errorf("无效的HTTP端口: %d", config.HTTP.Port)
	}
	
	return nil
}

// setAgentDefaults 设置Agent默认配置
func setAgentDefaults() {
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", 8080)
	
	viper.SetDefault("monitor.interface", "")
	viper.SetDefault("monitor.protocols", []string{"tcp", "udp"})
	viper.SetDefault("monitor.report_interval", 30*time.Second)
	viper.SetDefault("monitor.buffer_size", 1000)
	viper.SetDefault("monitor.filters.ignore_localhost", true)
	viper.SetDefault("monitor.filters.ignore_ports", []int{22, 53})
	
	viper.SetDefault("reporter.server_url", "http://localhost:8080/api/v1/metrics")
	viper.SetDefault("reporter.timeout", 10*time.Second)
	viper.SetDefault("reporter.retry_count", 3)
	viper.SetDefault("reporter.retry_delay", 5*time.Second)
	viper.SetDefault("reporter.batch_size", 100)
	viper.SetDefault("reporter.enable_tls", false)
	
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", "stdout")
}

// setServerDefaults 设置Server默认配置
func setServerDefaults() {
	viper.SetDefault("http.host", "0.0.0.0")
	viper.SetDefault("http.port", 8080)
	viper.SetDefault("http.read_timeout", 30*time.Second)
	viper.SetDefault("http.write_timeout", 30*time.Second)
	viper.SetDefault("http.enable_tls", false)
	
	viper.SetDefault("metrics.path", "/metrics")
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.interval", 15*time.Second)
	
	viper.SetDefault("storage.type", "memory")
	viper.SetDefault("storage.ttl", 1*time.Hour)
	viper.SetDefault("storage.max_entries", 10000)
	
	// Redis默认配置
	viper.SetDefault("storage.redis.host", "localhost")
	viper.SetDefault("storage.redis.port", 6379)
	viper.SetDefault("storage.redis.password", "")
	viper.SetDefault("storage.redis.db", 0)
	viper.SetDefault("storage.redis.pool_size", 10)
	viper.SetDefault("storage.redis.timeout", 5*time.Second)
	
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", "stdout")
}
