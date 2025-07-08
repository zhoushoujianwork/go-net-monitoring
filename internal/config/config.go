package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// AgentConfig Agent配置结构
type AgentConfig struct {
	Server      ServerConfig      `yaml:"server"`
	Monitor     MonitorConfig     `yaml:"monitor"`
	Reporter    ReporterConfig    `yaml:"reporter"`
	Persistence PersistenceConfig `yaml:"persistence"`
	EBPF        EBPFConfig        `yaml:"ebpf"`
	Log         LogConfig         `yaml:"log"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	Interface      string        `yaml:"interface"`       // 监控的网络接口
	Protocols      []string      `yaml:"protocols"`       // 监控的协议 tcp,udp,http,https
	ReportInterval time.Duration `yaml:"report_interval"` // 上报间隔
	BufferSize     int           `yaml:"buffer_size"`     // 缓冲区大小
	Filters        FilterConfig  `yaml:"filters"`         // 过滤规则
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
	ServerURL     string        `yaml:"server_url"`
	Timeout       time.Duration `yaml:"timeout"`
	RetryCount    int           `yaml:"retry_count"`
	RetryDelay    time.Duration `yaml:"retry_delay"`
	BatchSize     int           `yaml:"batch_size"`
	EnableTLS     bool          `yaml:"enable_tls"`
	TLSCertPath   string        `yaml:"tls_cert_path"`
	TLSKeyPath    string        `yaml:"tls_key_path"`
	Mode          string        `yaml:"mode"`           // "incremental" 或 "cumulative"
	IncludeTotals bool          `yaml:"include_totals"` // 是否包含总计数据
	AgentID       string        `yaml:"agent_id"`       // Agent唯一标识
}

// PersistenceConfig 持久化配置
type PersistenceConfig struct {
	Enabled      bool          `yaml:"enabled"`       // 是否启用持久化
	StateFile    string        `yaml:"state_file"`    // 状态文件路径
	SaveInterval time.Duration `yaml:"save_interval"` // 保存间隔
	BackupCount  int           `yaml:"backup_count"`  // 备份文件数量
}

// EBPFConfig eBPF程序配置
type EBPFConfig struct {
	ProgramPath    string   `yaml:"program_path"`    // eBPF程序文件路径
	FallbackPaths  []string `yaml:"fallback_paths"`  // 备用路径列表
	EnableFallback bool     `yaml:"enable_fallback"` // 是否启用模拟模式回退
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

// ServerAppConfig Server应用配置
type ServerAppConfig struct {
	HTTP    HTTPConfig    `yaml:"http"`
	Metrics MetricsConfig `yaml:"metrics"`
	Storage StorageConfig `yaml:"storage"`
	Log     LogConfig     `yaml:"log"`
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
	Debug        bool          `yaml:"debug"` // Debug模式
}

// MetricsConfig Prometheus指标配置
type MetricsConfig struct {
	Path     string        `yaml:"path"`
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
}

// StorageConfig 存储配置
type StorageConfig struct {
	Type                  string        `yaml:"type"`                    // memory, redis
	TTL                   time.Duration `yaml:"ttl"`                     // 数据保留时间
	MaxEntries            int           `yaml:"max_entries"`             // 最大条目数 (仅memory)
	CumulativeMode        bool          `yaml:"cumulative_mode"`         // 累计模式
	BaselineTracking      bool          `yaml:"baseline_tracking"`       // 基线跟踪
	AgentRestartDetection bool          `yaml:"agent_restart_detection"` // Agent重启检测

	// Redis配置
	Redis RedisConfig `yaml:"redis"`
}

// RedisConfig Redis存储配置
type RedisConfig struct {
	Host     string        `yaml:"host"`      // Redis主机地址
	Port     int           `yaml:"port"`      // Redis端口
	Password string        `yaml:"password"`  // Redis密码
	DB       int           `yaml:"db"`        // Redis数据库编号
	PoolSize int           `yaml:"pool_size"` // 连接池大小
	Timeout  time.Duration `yaml:"timeout"`   // 连接超时
}

// LoadAgentConfig 加载Agent配置
func LoadAgentConfig(configPath string) (*AgentConfig, error) {
	// 使用独立的 viper 实例避免全局状态冲突
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 设置默认值
	setAgentDefaultsForViper(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config AgentConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 手动处理可能的配置问题
	if config.Reporter.ServerURL == "" {
		config.Reporter.ServerURL = v.GetString("reporter.server_url")
	}

	// 手动处理 eBPF 配置（如果解析失败）
	if config.EBPF.ProgramPath == "" && v.IsSet("ebpf.program_path") {
		config.EBPF.ProgramPath = v.GetString("ebpf.program_path")
		config.EBPF.EnableFallback = v.GetBool("ebpf.enable_fallback")
		config.EBPF.FallbackPaths = v.GetStringSlice("ebpf.fallback_paths")
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

// setAgentDefaultsForViper 为指定的 viper 实例设置Agent默认配置
func setAgentDefaultsForViper(v *viper.Viper) {
	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.port", 8080)

	v.SetDefault("monitor.interface", "")
	v.SetDefault("monitor.protocols", []string{"tcp", "udp"})
	v.SetDefault("monitor.report_interval", 30*time.Second)
	v.SetDefault("monitor.buffer_size", 1000)
	v.SetDefault("monitor.filters.ignore_localhost", true)
	v.SetDefault("monitor.filters.ignore_ports", []int{22, 53})

	v.SetDefault("reporter.server_url", "http://localhost:8080/api/v1/metrics")
	v.SetDefault("reporter.timeout", 10*time.Second)
	v.SetDefault("reporter.retry_count", 3)
	v.SetDefault("reporter.retry_delay", 5*time.Second)
	v.SetDefault("reporter.batch_size", 100)
	v.SetDefault("reporter.enable_tls", false)

	// eBPF配置默认值
	v.SetDefault("ebpf.program_path", "/opt/go-net-monitoring/bpf/xdp_monitor.o")
	v.SetDefault("ebpf.fallback_paths", []string{
		"bpf/xdp_monitor.o",
		"bin/bpf/xdp_monitor.o",
		"bin/bpf/xdp_monitor_linux.o",
		"/usr/local/bin/bpf/xdp_monitor.o",
	})
	v.SetDefault("ebpf.enable_fallback", true)

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")
}

// setAgentDefaults 设置Agent默认配置 (保持向后兼容)
func setAgentDefaults() {
	setAgentDefaultsForViper(viper.GetViper())
}

// setServerDefaults 设置Server默认配置
func setServerDefaults() {
	viper.SetDefault("http.host", "0.0.0.0")
	viper.SetDefault("http.port", 8080)
	viper.SetDefault("http.read_timeout", 30*time.Second)
	viper.SetDefault("http.write_timeout", 30*time.Second)
	viper.SetDefault("http.enable_tls", false)
	viper.SetDefault("http.debug", false)

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
