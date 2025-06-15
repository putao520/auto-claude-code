# Auto Claude Code - 技术实现文档

## 技术栈选择

### 核心技术
- **编程语言**：Go 1.21+
- **并发模型**：Goroutines + Channels
- **配置管理**：YAML + viper
- **日志系统**：zap 结构化日志
- **错误处理**：pkg/errors 增强错误信息

### 第三方依赖
```go
require (
	github.com/spf13/cobra v1.8.0      // CLI框架
	github.com/spf13/viper v1.18.0     // 配置管理
	go.uber.org/zap v1.26.0            // 结构化日志
	github.com/pkg/errors v0.9.1       // 错误处理
	gopkg.in/yaml.v3 v3.0.1            // YAML解析
)
```

## 项目结构设计

### 目录布局
```
auto-claude-code/
├── cmd/
│   └── auto-claude-code/           # 主程序入口
│       └── main.go
├── internal/                       # 内部包，不对外暴露
│   ├── converter/                  # 路径转换模块
│   │   ├── path_converter.go
│   │   └── path_converter_test.go
│   ├── wsl/                       # WSL桥接模块
│   │   ├── bridge.go
│   │   ├── bridge_test.go
│   │   └── command.go
│   ├── config/                    # 配置管理模块
│   │   ├── config.go
│   │   ├── config_test.go
│   │   └── defaults.go
│   ├── errors/                    # 错误定义模块
│   │   └── errors.go
│   └── logger/                    # 日志模块
│       └── logger.go
├── pkg/                           # 公共包，可供外部使用
│   └── version/                   # 版本信息
│       └── version.go
├── configs/                       # 配置文件模板
│   └── config.example.yaml
├── scripts/                       # 构建和部署脚本
│   ├── build.sh
│   └── install.ps1
├── docs/                          # 文档
│   ├── README.md
│   ├── TECHNICAL.md
│   └── API.md
├── go.mod
├── go.sum
├── Makefile
└── .gitignore
```

## 核心算法实现

### 1. 路径转换算法

#### 算法原理
Windows 路径到 WSL 路径的转换遵循以下规则：
- `C:\path\to\dir` → `/mnt/c/path/to/dir`
- 盘符转换为小写
- 反斜杠转换为正斜杠
- 处理空格和特殊字符

#### 实现代码
```go
package converter

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

type PathConverter struct {
	// 路径验证正则表达式
	windowsPathRegex *regexp.Regexp
}

func NewPathConverter() *PathConverter {
	// Windows绝对路径正则：C:\... 或 \\server\share\...
	regex := regexp.MustCompile(`^[A-Za-z]:\\|^\\\\`)
	return &PathConverter{
		windowsPathRegex: regex,
	}
}

func (pc *PathConverter) ConvertPath(windowsPath string) (string, error) {
	// 1. 验证输入路径格式
	if !pc.windowsPathRegex.MatchString(windowsPath) {
		return "", fmt.Errorf("invalid Windows path format: %s", windowsPath)
	}
	
	// 2. 标准化路径
	absPath, err := filepath.Abs(windowsPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	
	// 3. 处理UNC路径（网络路径）
	if strings.HasPrefix(absPath, "\\\\") {
		return pc.convertUNCPath(absPath)
	}
	
	// 4. 处理本地磁盘路径
	return pc.convertLocalPath(absPath)
}

func (pc *PathConverter) convertLocalPath(path string) (string, error) {
	// 检查路径格式：C:\path\to\dir
	if len(path) < 3 || path[1] != ':' || path[2] != '\\' {
		return "", fmt.Errorf("invalid local path format: %s", path)
	}
	
	// 提取盘符
	drive := strings.ToLower(string(path[0]))
	if !unicode.IsLetter(rune(path[0])) {
		return "", fmt.Errorf("invalid drive letter: %c", path[0])
	}
	
	// 提取路径部分
	pathPart := path[3:] // 跳过 "C:\"
	
	// 转换路径分隔符
	unixPath := strings.ReplaceAll(pathPart, "\\", "/")
	
	// 构建WSL路径
	wslPath := fmt.Sprintf("/mnt/%s/%s", drive, unixPath)
	
	// 处理末尾的斜杠
	if strings.HasSuffix(path, "\\") && !strings.HasSuffix(wslPath, "/") {
		wslPath += "/"
	}
	
	return wslPath, nil
}

func (pc *PathConverter) convertUNCPath(path string) (string, error) {
	// UNC路径：\\server\share\path → /mnt/wsl/server/share/path
	pathParts := strings.Split(path[2:], "\\") // 移除开头的 "\\"
	if len(pathParts) < 2 {
		return "", fmt.Errorf("invalid UNC path format: %s", path)
	}
	
	server := pathParts[0]
	share := pathParts[1]
	remaining := strings.Join(pathParts[2:], "/")
	
	wslPath := fmt.Sprintf("/mnt/wsl/%s/%s", server, share)
	if remaining != "" {
		wslPath += "/" + remaining
	}
	
	return wslPath, nil
}

func (pc *PathConverter) ValidatePath(path string) error {
	// 检查路径是否存在（在Windows环境中）
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", path)
		}
		return fmt.Errorf("cannot access path %s: %w", path, err)
	}
	return nil
}
```

### 2. WSL桥接实现

#### WSL命令执行器
```go
package wsl

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Bridge struct {
	defaultDistro string
	timeout       time.Duration
}

func NewBridge(defaultDistro string) *Bridge {
	return &Bridge{
		defaultDistro: defaultDistro,
		timeout:       30 * time.Second,
	}
}

func (b *Bridge) CheckWSL() error {
	// 检查WSL是否可用
	cmd := exec.Command("wsl", "--status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("WSL not available: %w", err)
	}
	
	// 解析WSL状态
	status := string(output)
	if strings.Contains(status, "not installed") {
		return fmt.Errorf("WSL is not installed")
	}
	
	return nil
}

func (b *Bridge) ListDistros() ([]string, error) {
	cmd := exec.Command("wsl", "--list", "--quiet")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list WSL distributions: %w", err)
	}
	
	var distros []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.Contains(line, "docker-desktop") {
			distros = append(distros, line)
		}
	}
	
	return distros, nil
}

func (b *Bridge) ExecuteInWSL(distro, directory, command string) error {
	if distro == "" {
		distro = b.defaultDistro
	}
	
	// 构建WSL命令
	var cmd *exec.Cmd
	if directory != "" {
		// 带目录切换的命令
		fullCommand := fmt.Sprintf("cd '%s' && %s", directory, command)
		cmd = exec.Command("wsl", "-d", distro, "--", "bash", "-c", fullCommand)
	} else {
		// 直接执行命令
		cmd = exec.Command("wsl", "-d", distro, "--", "bash", "-c", command)
	}
	
	// 设置超时
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Args[0], cmd.Args[1:]...)
	
	// 连接标准输入输出
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

func (b *Bridge) StartClaudeCode(distro, workingDir string) error {
	// 检查Claude Code是否可用
	checkCmd := "which claude-code"
	if err := b.ExecuteInWSL(distro, "", checkCmd); err != nil {
		return fmt.Errorf("claude-code not found in WSL distribution %s", distro)
	}
	
	// 启动Claude Code
	claudeCmd := "claude-code"
	return b.ExecuteInWSL(distro, workingDir, claudeCmd)
}
```

### 3. 配置管理实现

#### 配置结构定义
```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/spf13/viper"
)

type Config struct {
	DefaultDistro    string            `mapstructure:"default_distro" yaml:"default_distro"`
	Debug           bool              `mapstructure:"debug" yaml:"debug"`
	LogLevel        string            `mapstructure:"log_level" yaml:"log_level"`
	PathMappings    map[string]string `mapstructure:"path_mappings" yaml:"path_mappings"`
	ClaudeCodePath  string            `mapstructure:"claude_code_path" yaml:"claude_code_path"`
	Timeout         int               `mapstructure:"timeout" yaml:"timeout"`
}

func LoadConfig() (*Config, error) {
	v := viper.New()
	
	// 设置配置文件名和类型
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	
	// 添加配置文件搜索路径
	v.AddConfigPath(".")                                    // 当前目录
	v.AddConfigPath("$HOME/.auto-claude-code")             // 用户目录
	v.AddConfigPath("/etc/auto-claude-code")               // 系统目录
	
	// 设置环境变量前缀
	v.SetEnvPrefix("AUTO_CLAUDE_CODE")
	v.AutomaticEnv()
	
	// 设置默认值
	setDefaults(v)
 
	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// 配置文件不存在，使用默认值
	}
	
	// 解析配置到结构体
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	return &config, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("default_distro", "")
	v.SetDefault("debug", false)
	v.SetDefault("log_level", "info")
	v.SetDefault("claude_code_path", "claude-code")
	v.SetDefault("timeout", 30)
	v.SetDefault("path_mappings", map[string]string{})
}

func (c *Config) GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(home, ".auto-claude-code", "config.yaml")
}

func (c *Config) SaveConfig() error {
	configPath := c.GetConfigPath()
	
	// 确保配置目录存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	v := viper.New()
	v.SetConfigFile(configPath)
	
	// 设置配置值
	v.Set("default_distro", c.DefaultDistro)
	v.Set("debug", c.Debug)
	v.Set("log_level", c.LogLevel)
	v.Set("claude_code_path", c.ClaudeCodePath)
	v.Set("timeout", c.Timeout)
	v.Set("path_mappings", c.PathMappings)
	
	return v.WriteConfig()
}
```

## CLI 接口实现

### 主命令结构
```go
package main

import (
	"fmt"
	"os"
	
	"github.com/spf13/cobra"
	"github.com/username/auto-claude-code/internal/config"
	"github.com/username/auto-claude-code/internal/converter"
	"github.com/username/auto-claude-code/internal/wsl"
	"github.com/username/auto-claude-code/pkg/version"
)

var (
	cfgFile string
	debug   bool
	distro  string
)

var rootCmd = &cobra.Command{
	Use:   "auto-claude-code",
	Short: "Windows to WSL Claude Code bridge",
	Long: `Auto Claude Code is a lightweight proxy tool designed for Windows + WSL development environments.
It intelligently converts Windows current working directory to WSL mount paths and launches Claude Code in WSL environment.`,
	RunE: runClaudeCode,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	
	// 全局标志
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.auto-claude-code/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug mode")
	rootCmd.PersistentFlags().StringVar(&distro, "distro", "", "specify WSL distribution")
	
	// 版本子命令
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Auto Claude Code %s\n", version.Version)
			fmt.Printf("Build Date: %s\n", version.BuildDate)
			fmt.Printf("Git Commit: %s\n", version.GitCommit)
		},
	}
	rootCmd.AddCommand(versionCmd)
 
	// 配置子命令
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}
	
	configCmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE:  showConfig,
	})
	
	configCmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Initialize configuration file",
		RunE:  initConfigFile,
	})
	
	rootCmd.AddCommand(configCmd)
}

func runClaudeCode(cmd *cobra.Command, args []string) error {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	// 覆盖配置
	if debug {
		cfg.Debug = true
	}
	if distro != "" {
		cfg.DefaultDistro = distro
	}
	
	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	
	// 转换路径
	pathConverter := converter.NewPathConverter()
	wslPath, err := pathConverter.ConvertPath(currentDir)
	if err != nil {
		return fmt.Errorf("failed to convert path: %w", err)
	}
	
	if cfg.Debug {
		fmt.Printf("Windows Path: %s\n", currentDir)
		fmt.Printf("WSL Path: %s\n", wslPath)
	}
	
	// 创建WSL桥接
	wslBridge := wsl.NewBridge(cfg.DefaultDistro)
	
	// 检查WSL环境
	if err := wslBridge.CheckWSL(); err != nil {
		return fmt.Errorf("WSL check failed: %w", err)
	}
	
	// 启动Claude Code
	fmt.Printf("Starting Claude Code in WSL at: %s\n", wslPath)
	if err := wslBridge.StartClaudeCode(cfg.DefaultDistro, wslPath); err != nil {
		return fmt.Errorf("failed to start Claude Code: %w", err)
	}
	
	return nil
}
```

## 错误处理策略

### 自定义错误类型
```go
package errors

import (
	"fmt"
)

type ErrorType string

const (
	ErrTypePathConversion ErrorType = "PATH_CONVERSION"
	ErrTypeWSL           ErrorType = "WSL"
	ErrTypeClaudeCode    ErrorType = "CLAUDE_CODE"
	ErrTypeConfig        ErrorType = "CONFIG"
)

type AppError struct {
	Type     ErrorType
	Message  string
	Cause    error
	Metadata map[string]interface{}
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

// 路径转换错误
func NewPathConversionError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrTypePathConversion,
		Message: message,
		Cause:   cause,
	}
}

// WSL错误
func NewWSLError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrTypeWSL,
		Message: message,
		Cause:   cause,
	}
}

// Claude Code错误
func NewClaudeCodeError(message string, cause error) *AppError {
	return &AppError{
		Type:    ErrTypeClaudeCode,
		Message: message,
		Cause:   cause,
	}
}
```

## 性能优化策略

### 1. 缓存机制
```go
type PathCache struct {
	cache map[string]string
	mutex sync.RWMutex
	ttl   time.Duration
}

func (pc *PathCache) Get(key string) (string, bool) {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	
	value, exists := pc.cache[key]
	return value, exists
}

func (pc *PathCache) Set(key, value string) {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	
	pc.cache[key] = value
}
```

### 2. 并发处理
```go
func (b *Bridge) ExecuteInWSLAsync(distro, directory, command string) <-chan error {
	errChan := make(chan error, 1)
	
	go func() {
		defer close(errChan)
		err := b.ExecuteInWSL(distro, directory, command)
		errChan <- err
	}()
	
	return errChan
}
```

## 测试实现

### 单元测试示例
```go
func TestPathConverter_ConvertPath(t *testing.T) {
	converter := NewPathConverter()
	
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "Simple C drive path",
			input:    "C:\\Users\\test",
			expected: "/mnt/c/Users/test",
			hasError: false,
		},
		{
			name:     "D drive with spaces",
			input:    "D:\\Program Files\\App",
			expected: "/mnt/d/Program Files/App",
			hasError: false,
		},
		{
			name:     "Invalid path",
			input:    "invalid",
			expected: "",
			hasError: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.ConvertPath(tt.input)
			
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
```

## 部署和构建

### Makefile
```makefile
.PHONY: build test clean install

BINARY_NAME=auto-claude-code
VERSION=$(shell git describe --tags --always)
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=$(shell git rev-parse HEAD)

LDFLAGS=-ldflags "-s -w -X github.com/username/auto-claude-code/pkg/version.Version=$(VERSION) -X github.com/username/auto-claude-code/pkg/version.BuildDate=$(BUILD_DATE) -X github.com/username/auto-claude-code/pkg/version.GitCommit=$(GIT_COMMIT)"

build:
	go build $(LDFLAGS) -o $(BINARY_NAME).exe ./cmd/auto-claude-code

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME).exe

install: build
	cp $(BINARY_NAME).exe $(GOPATH)/bin/
```

这个技术文档提供了完整的实现方案，包括详细的算法设计、代码结构和最佳实践。所有代码都遵循 Go 语言的惯例和 SOLID 原则，确保代码的可维护性和可扩展性。 