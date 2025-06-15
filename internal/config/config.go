package config

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	apperrors "auto-claude-code/internal/errors"

	"github.com/spf13/viper"
)

// Config 应用程序配置结构
type Config struct {
	// 基础配置
	Debug    bool   `mapstructure:"debug" yaml:"debug"`
	LogLevel string `mapstructure:"log_level" yaml:"log_level"`

	// WSL 配置
	WSL WSLConfig `mapstructure:"wsl" yaml:"wsl"`

	// Claude Code 配置
	ClaudeCode ClaudeCodeConfig `mapstructure:"claude_code" yaml:"claude_code"`

	// MCP 配置（为后续功能预留）
	MCP MCPConfig `mapstructure:"mcp" yaml:"mcp"`
}

// WSLConfig WSL 相关配置
type WSLConfig struct {
	DefaultDistro string            `mapstructure:"default_distro" yaml:"default_distro"`
	PathMappings  map[string]string `mapstructure:"path_mappings" yaml:"path_mappings"`
	Timeout       string            `mapstructure:"timeout" yaml:"timeout"`
}

// ClaudeCodeConfig Claude Code 相关配置
type ClaudeCodeConfig struct {
	Executable   string   `mapstructure:"executable" yaml:"executable"`
	DefaultArgs  []string `mapstructure:"default_args" yaml:"default_args"`
	Interactive  bool     `mapstructure:"interactive" yaml:"interactive"`
	WorkspaceDir string   `mapstructure:"workspace_dir" yaml:"workspace_dir"`
}

// MCPConfig MCP 服务器配置
type MCPConfig struct {
	// 基础配置
	Enabled            bool   `mapstructure:"enabled" yaml:"enabled"`
	Port               int    `mapstructure:"port" yaml:"port"`
	Host               string `mapstructure:"host" yaml:"host"`
	MaxConcurrentTasks int    `mapstructure:"max_concurrent_tasks" yaml:"max_concurrent_tasks"`
	TaskTimeout        string `mapstructure:"task_timeout" yaml:"task_timeout"`

	// Git Worktree 配置
	WorktreeBaseDir string `mapstructure:"worktree_base_dir" yaml:"worktree_base_dir"`
	CleanupInterval string `mapstructure:"cleanup_interval" yaml:"cleanup_interval"`
	MaxWorktrees    int    `mapstructure:"max_worktrees" yaml:"max_worktrees"`

	// 传输配置
	HTTP  MCPHTTPConfig  `mapstructure:"http" yaml:"http"`
	Stdio MCPStdioConfig `mapstructure:"stdio" yaml:"stdio"`

	// 认证配置
	Auth MCPAuthConfig `mapstructure:"auth" yaml:"auth"`

	// 任务队列配置
	Queue MCPQueueConfig `mapstructure:"queue" yaml:"queue"`

	// 监控配置
	Monitoring MCPMonitoringConfig `mapstructure:"monitoring" yaml:"monitoring"`
}

// MCPAuthConfig MCP 认证配置
type MCPAuthConfig struct {
	Enabled    bool     `mapstructure:"enabled" yaml:"enabled"`
	Method     string   `mapstructure:"method" yaml:"method"` // "token", "oauth2", "none"
	TokenFile  string   `mapstructure:"token_file" yaml:"token_file"`
	AllowedIPs []string `mapstructure:"allowed_ips" yaml:"allowed_ips"`
}

// MCPQueueConfig MCP 任务队列配置
type MCPQueueConfig struct {
	MaxSize        int    `mapstructure:"max_size" yaml:"max_size"`
	RetryAttempts  int    `mapstructure:"retry_attempts" yaml:"retry_attempts"`
	RetryInterval  string `mapstructure:"retry_interval" yaml:"retry_interval"`
	PriorityLevels int    `mapstructure:"priority_levels" yaml:"priority_levels"`
}

// MCPMonitoringConfig MCP 监控配置
type MCPMonitoringConfig struct {
	Enabled      bool   `mapstructure:"enabled" yaml:"enabled"`
	MetricsPath  string `mapstructure:"metrics_path" yaml:"metrics_path"`
	HealthPath   string `mapstructure:"health_path" yaml:"health_path"`
	LogRequests  bool   `mapstructure:"log_requests" yaml:"log_requests"`
	LogResponses bool   `mapstructure:"log_responses" yaml:"log_responses"`
}

// MCPHTTPConfig MCP HTTP传输配置
type MCPHTTPConfig struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
}

// MCPStdioConfig MCP stdio传输配置
type MCPStdioConfig struct {
	Enabled bool `mapstructure:"enabled" yaml:"enabled"`
	// Reader和Writer在运行时设置，不序列化
	Reader io.Reader `mapstructure:"-" yaml:"-"`
	Writer io.Writer `mapstructure:"-" yaml:"-"`
}

// ConfigManager 配置管理器接口
type ConfigManager interface {
	// LoadConfig 加载配置
	LoadConfig() (*Config, error)

	// SaveConfig 保存配置
	SaveConfig(config *Config) error

	// GetConfigPath 获取配置文件路径
	GetConfigPath() string

	// SetConfigPath 设置配置文件路径
	SetConfigPath(path string)
}

// configManager 配置管理器实现
type configManager struct {
	configPath string
	viper      *viper.Viper
}

// NewConfigManager 创建新的配置管理器
func NewConfigManager() ConfigManager {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	return &configManager{
		viper: v,
	}
}

// LoadConfig 加载配置
func (cm *configManager) LoadConfig() (*Config, error) {
	// 设置配置文件搜索路径
	cm.setupConfigPaths()

	// 设置环境变量前缀
	cm.viper.SetEnvPrefix("AUTO_CLAUDE_CODE")
	cm.viper.AutomaticEnv()

	// 替换环境变量中的点和横线
	cm.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// 尝试读取配置文件
	if err := cm.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, apperrors.Wrap(err, apperrors.ErrConfigInvalid, "配置文件读取失败")
		}
		// 配置文件不存在，使用默认配置
	}

	// 解析配置
	var config Config
	if err := cm.viper.Unmarshal(&config); err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrConfigInvalid, "配置解析失败")
	}

	// 验证配置
	if err := cm.validateConfig(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig 保存配置
func (cm *configManager) SaveConfig(config *Config) error {
	// 验证配置
	if err := cm.validateConfig(config); err != nil {
		return err
	}

	// 确保配置目录存在
	configPath := cm.GetConfigPath()
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return apperrors.Wrap(err, apperrors.ErrConfigInvalid, "无法创建配置目录")
	}

	// 设置配置值
	cm.viper.Set("debug", config.Debug)
	cm.viper.Set("log_level", config.LogLevel)
	cm.viper.Set("wsl", config.WSL)
	cm.viper.Set("claude_code", config.ClaudeCode)
	cm.viper.Set("mcp", config.MCP)

	// 写入配置文件
	if err := cm.viper.WriteConfigAs(configPath); err != nil {
		return apperrors.Wrap(err, apperrors.ErrConfigInvalid, "配置文件写入失败")
	}

	return nil
}

// GetConfigPath 获取配置文件路径
func (cm *configManager) GetConfigPath() string {
	if cm.configPath != "" {
		return cm.configPath
	}

	// 默认配置路径
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./config.yaml"
	}

	return filepath.Join(homeDir, ".auto-claude-code", "config.yaml")
}

// SetConfigPath 设置配置文件路径
func (cm *configManager) SetConfigPath(path string) {
	cm.configPath = path
	cm.viper.SetConfigFile(path)
}

// setupConfigPaths 设置配置文件搜索路径
func (cm *configManager) setupConfigPaths() {
	if cm.configPath != "" {
		cm.viper.SetConfigFile(cm.configPath)
		return
	}

	// 设置配置文件名和类型
	cm.viper.SetConfigName("config")
	cm.viper.SetConfigType("yaml")

	// 添加搜索路径
	cm.viper.AddConfigPath(".")        // 当前目录
	cm.viper.AddConfigPath("./config") // config 子目录

	// 用户配置目录
	if homeDir, err := os.UserHomeDir(); err == nil {
		cm.viper.AddConfigPath(filepath.Join(homeDir, ".auto-claude-code"))
	}

	// Windows 应用数据目录
	if appData := os.Getenv("APPDATA"); appData != "" {
		cm.viper.AddConfigPath(filepath.Join(appData, "auto-claude-code"))
	}
}

// setDefaults 设置默认配置值
func setDefaults(v *viper.Viper) {
	// 基础配置默认值
	v.SetDefault("debug", false)
	v.SetDefault("log_level", "info")

	// WSL 配置默认值
	v.SetDefault("wsl.default_distro", "")
	v.SetDefault("wsl.timeout", "30s")
	v.SetDefault("wsl.path_mappings", map[string]string{})

	// Claude Code 配置默认值
	v.SetDefault("claude_code.executable", "claude-code")
	v.SetDefault("claude_code.default_args", []string{})
	v.SetDefault("claude_code.interactive", true)
	v.SetDefault("claude_code.workspace_dir", "")

	// MCP 配置默认值
	v.SetDefault("mcp.enabled", false)
	v.SetDefault("mcp.port", 8080)
	v.SetDefault("mcp.host", "localhost")
	v.SetDefault("mcp.max_concurrent_tasks", 5)
	v.SetDefault("mcp.task_timeout", "30m")
	v.SetDefault("mcp.worktree_base_dir", "./worktrees")
	v.SetDefault("mcp.cleanup_interval", "1h")
	v.SetDefault("mcp.max_worktrees", 10)

	// MCP 认证配置默认值
	v.SetDefault("mcp.auth.enabled", false)
	v.SetDefault("mcp.auth.method", "none")
	v.SetDefault("mcp.auth.token_file", "")
	v.SetDefault("mcp.auth.allowed_ips", []string{"127.0.0.1", "::1"})

	// MCP 队列配置默认值
	v.SetDefault("mcp.queue.max_size", 100)
	v.SetDefault("mcp.queue.retry_attempts", 3)
	v.SetDefault("mcp.queue.retry_interval", "5s")
	v.SetDefault("mcp.queue.priority_levels", 3)

	// MCP 传输配置默认值
	v.SetDefault("mcp.http.enabled", true)
	v.SetDefault("mcp.stdio.enabled", false)

	// MCP 监控配置默认值
	v.SetDefault("mcp.monitoring.enabled", true)
	v.SetDefault("mcp.monitoring.metrics_path", "/metrics")
	v.SetDefault("mcp.monitoring.health_path", "/health")
	v.SetDefault("mcp.monitoring.log_requests", true)
	v.SetDefault("mcp.monitoring.log_responses", false)
}

// validateConfig 验证配置
func (cm *configManager) validateConfig(config *Config) error {
	// 验证日志级别
	validLogLevels := []string{"debug", "info", "warn", "error", "fatal"}
	if !contains(validLogLevels, config.LogLevel) {
		return apperrors.Newf(apperrors.ErrConfigInvalid,
			"无效的日志级别: %s，支持的级别: %v", config.LogLevel, validLogLevels)
	}

	// 验证 Claude Code 可执行文件
	if config.ClaudeCode.Executable == "" {
		return apperrors.New(apperrors.ErrConfigInvalid, "Claude Code 可执行文件路径不能为空")
	}

	// 验证 MCP 配置
	if config.MCP.Enabled {
		if config.MCP.Port <= 0 || config.MCP.Port > 65535 {
			return apperrors.Newf(apperrors.ErrConfigInvalid,
				"无效的 MCP 端口号: %d", config.MCP.Port)
		}

		if config.MCP.MaxConcurrentTasks <= 0 {
			return apperrors.Newf(apperrors.ErrConfigInvalid,
				"最大并发任务数必须大于 0: %d", config.MCP.MaxConcurrentTasks)
		}
	}

	return nil
}

// contains 检查字符串切片是否包含指定值
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// GetDefaultConfig 获取默认配置
func GetDefaultConfig() *Config {
	return &Config{
		Debug:    false,
		LogLevel: "info",
		WSL: WSLConfig{
			DefaultDistro: "",
			PathMappings:  make(map[string]string),
			Timeout:       "30s",
		},
		ClaudeCode: ClaudeCodeConfig{
			Executable:   "claude-code",
			DefaultArgs:  []string{},
			Interactive:  true,
			WorkspaceDir: "",
		},
		MCP: MCPConfig{
			Enabled:            false,
			Port:               8080,
			MaxConcurrentTasks: 5,
			TaskTimeout:        "30m",
			WorktreeBaseDir:    "./worktrees",
		},
	}
}

// LoadConfigFromFile 从指定文件加载配置
func LoadConfigFromFile(path string) (*Config, error) {
	cm := NewConfigManager()
	cm.SetConfigPath(path)
	return cm.LoadConfig()
}

// LoadConfigFromEnv 从环境变量加载配置
func LoadConfigFromEnv() (*Config, error) {
	config := GetDefaultConfig()

	// 从环境变量读取配置
	if debug := os.Getenv("AUTO_CLAUDE_CODE_DEBUG"); debug != "" {
		config.Debug = strings.ToLower(debug) == "true"
	}

	if logLevel := os.Getenv("AUTO_CLAUDE_CODE_LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	if distro := os.Getenv("AUTO_CLAUDE_CODE_WSL_DEFAULT_DISTRO"); distro != "" {
		config.WSL.DefaultDistro = distro
	}

	if executable := os.Getenv("AUTO_CLAUDE_CODE_CLAUDE_CODE_EXECUTABLE"); executable != "" {
		config.ClaudeCode.Executable = executable
	}

	return config, nil
}
