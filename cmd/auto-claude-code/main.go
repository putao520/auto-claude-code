package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"auto-claude-code/internal/config"
	"auto-claude-code/internal/converter"
	apperrors "auto-claude-code/internal/errors"
	"auto-claude-code/internal/logger"
	"auto-claude-code/internal/mcp"
	"auto-claude-code/internal/wsl"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

var (
	// 版本信息
	version = "1.0.0"
	commit  = "dev"
	date    = "unknown"

	// 全局配置
	cfg *config.Config
	log logger.Logger

	// 命令行参数
	configFile  string
	debug       bool
	logLevel    string
	targetDir   string
	distro      string
	claudeArgs  []string
	showVersion bool
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "auto-claude-code",
	Short: "Windows to WSL Claude Code 桥接工具",
	Long: `Auto Claude Code 是一个智能的 Windows 到 WSL 路径转换工具，
可以自动将当前 Windows 工作目录转换为 WSL 路径，并在 WSL 环境中启动 Claude Code。

支持功能：
- 自动路径转换（Windows → WSL）
- WSL 环境检测和管理
- Claude Code 启动代理
- 配置文件管理
- 详细的日志记录`,
	Example: `  # 在当前目录启动 Claude Code
  auto-claude-code

  # 指定目录启动
  auto-claude-code --dir /path/to/project

  # 指定 WSL 发行版
  auto-claude-code --distro Ubuntu-20.04

  # 调试模式
  auto-claude-code --debug

  # 传递参数给 Claude Code
  auto-claude-code -- --help`,
	RunE: runMain,
}

func main() {
	// 设置命令行参数
	setupFlags()

	// 执行命令
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

// setupFlags 设置命令行参数
func setupFlags() {
	// 全局参数
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "配置文件路径")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "启用调试模式")
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "info", "日志级别 (debug, info, warn, error, fatal)")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "显示版本信息")

	// 主命令参数
	rootCmd.Flags().StringVar(&targetDir, "dir", "", "目标目录（默认为当前目录）")
	rootCmd.Flags().StringVar(&distro, "distro", "", "WSL 发行版名称（默认使用系统默认）")

	// 版本命令
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			printVersion()
		},
	}
	rootCmd.AddCommand(versionCmd)

	// 检查命令
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "检查系统环境",
		Long:  "检查 WSL 环境、Claude Code 安装状态等",
		RunE:  runCheck,
	}
	rootCmd.AddCommand(checkCmd)

	// 配置命令
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "配置管理",
		Long:  "管理应用程序配置",
	}

	configShowCmd := &cobra.Command{
		Use:   "show",
		Short: "显示当前配置",
		RunE:  runConfigShow,
	}

	configInitCmd := &cobra.Command{
		Use:   "init",
		Short: "初始化配置文件",
		RunE:  runConfigInit,
	}

	configCmd.AddCommand(configShowCmd, configInitCmd)
	rootCmd.AddCommand(configCmd)

	// MCP服务器命令
	mcpCmd := &cobra.Command{
		Use:   "mcp-server",
		Short: "启动MCP服务器",
		Long:  "启动MCP服务器，提供Claude Code任务分发和管理功能",
		RunE:  runMCPServer,
	}
	rootCmd.AddCommand(mcpCmd)

	// 任务管理命令
	taskCmd := &cobra.Command{
		Use:   "task",
		Short: "任务管理",
		Long:  "管理MCP服务器上的任务",
	}

	// 列出任务命令
	taskListCmd := &cobra.Command{
		Use:   "list",
		Short: "列出所有任务",
		Long:  "列出MCP服务器上的所有任务及其状态",
		RunE:  runTaskList,
	}

	// 查看任务详情命令
	taskShowCmd := &cobra.Command{
		Use:   "show <task-id>",
		Short: "查看任务详情",
		Long:  "查看指定任务的详细信息",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskShow,
	}

	// 取消任务命令
	taskCancelCmd := &cobra.Command{
		Use:   "cancel <task-id>",
		Short: "取消任务",
		Long:  "取消指定的任务",
		Args:  cobra.ExactArgs(1),
		RunE:  runTaskCancel,
	}

	// 提交任务命令
	taskSubmitCmd := &cobra.Command{
		Use:   "submit",
		Short: "提交新任务",
		Long:  "向MCP服务器提交新的编程任务",
		RunE:  runTaskSubmit,
	}

	// 任务状态监控命令
	taskWatchCmd := &cobra.Command{
		Use:   "watch",
		Short: "实时监控任务状态",
		Long:  "实时监控所有任务的执行状态",
		RunE:  runTaskWatch,
	}

	// TUI监控命令
	taskTUICmd := &cobra.Command{
		Use:   "tui",
		Short: "TUI界面监控任务",
		Long:  "使用类似top命令的TUI界面实时监控任务状态",
		RunE:  runTaskTUI,
	}

	// 添加任务提交的参数
	taskSubmitCmd.Flags().StringP("project", "p", "", "项目路径（必需）")
	taskSubmitCmd.Flags().String("description", "", "任务描述（必需）")
	taskSubmitCmd.Flags().StringP("priority", "r", "medium", "任务优先级 (low, medium, high)")
	taskSubmitCmd.Flags().StringP("timeout", "t", "30m", "任务超时时间")
	taskSubmitCmd.Flags().StringSliceP("args", "a", []string{}, "传递给Claude Code的参数")
	taskSubmitCmd.MarkFlagRequired("project")
	taskSubmitCmd.MarkFlagRequired("description")

	// 添加服务器地址参数
	taskCmd.PersistentFlags().StringP("server", "s", "http://localhost:8080", "MCP服务器地址")
	taskWatchCmd.Flags().IntP("interval", "i", 2, "刷新间隔（秒）")
	taskTUICmd.Flags().IntP("interval", "i", 2, "刷新间隔（秒）")

	taskCmd.AddCommand(taskListCmd, taskShowCmd, taskCancelCmd, taskSubmitCmd, taskWatchCmd, taskTUICmd)
	rootCmd.AddCommand(taskCmd)
}

// runMain 主命令执行函数
func runMain(cmd *cobra.Command, args []string) error {
	// 处理版本显示
	if showVersion {
		printVersion()
		return nil
	}

	// 初始化应用程序
	if err := initApp(); err != nil {
		return err
	}

	// 获取目标目录
	workingDir, err := getWorkingDirectory()
	if err != nil {
		return err
	}

	log.Info("开始执行 Claude Code 启动流程",
		zap.String("workingDir", workingDir),
		zap.String("distro", distro))

	// 创建路径转换器
	pathConverter := converter.NewPathConverter()

	// 验证路径
	if err := pathConverter.ValidatePath(workingDir); err != nil {
		return fmt.Errorf("路径验证失败: %w", err)
	}

	// 转换路径
	wslPath, err := pathConverter.ConvertToWSL(workingDir)
	if err != nil {
		return fmt.Errorf("路径转换失败: %w", err)
	}

	log.Info("路径转换成功",
		zap.String("windowsPath", workingDir),
		zap.String("wslPath", wslPath))

	// 创建 WSL 桥接器
	wslBridge := wsl.NewWSLBridge(log.GetZapLogger())

	// 检查 WSL 环境
	if err := wslBridge.CheckWSL(); err != nil {
		return fmt.Errorf("WSL 环境检查失败: %w", err)
	}

	// 获取 WSL 发行版
	if distro == "" {
		if cfg.WSL.DefaultDistro != "" {
			distro = cfg.WSL.DefaultDistro
		} else {
			distro, err = wslBridge.GetDefaultDistro()
			if err != nil {
				return fmt.Errorf("获取默认 WSL 发行版失败: %w", err)
			}
		}
	}

	log.Info("使用 WSL 发行版", zap.String("distro", distro))

	// 检查 Claude Code
	if err := wslBridge.CheckClaudeCode(distro); err != nil {
		return fmt.Errorf("Claude Code 检查失败: %w", err)
	}

	// 准备 Claude Code 参数
	claudeCodeArgs := append(cfg.ClaudeCode.DefaultArgs, args...)

	log.Info("启动 Claude Code",
		zap.String("distro", distro),
		zap.String("wslPath", wslPath),
		zap.Strings("args", claudeCodeArgs))

	// 启动 Claude Code
	if err := wslBridge.StartClaudeCode(distro, wslPath, claudeCodeArgs); err != nil {
		return fmt.Errorf("Claude Code 启动失败: %w", err)
	}

	log.Info("Claude Code 执行完成")
	return nil
}

// runCheck 检查命令执行函数
func runCheck(cmd *cobra.Command, args []string) error {
	if err := initApp(); err != nil {
		return err
	}

	fmt.Println("🔍 系统环境检查")
	fmt.Println("================")

	// 检查 WSL
	wslBridge := wsl.NewWSLBridge(log.GetZapLogger())

	fmt.Print("WSL 环境: ")
	if err := wslBridge.CheckWSL(); err != nil {
		fmt.Printf("❌ 失败 - %v\n", err)
		return nil
	}
	fmt.Println("✅ 可用")

	// 列出 WSL 发行版
	fmt.Print("WSL 发行版: ")
	distros, err := wslBridge.ListDistros()
	if err != nil {
		fmt.Printf("❌ 获取失败 - %v\n", err)
		return nil
	}

	if len(distros) == 0 {
		fmt.Println("❌ 未找到可用的发行版")
		return nil
	}

	fmt.Printf("✅ 找到 %d 个发行版\n", len(distros))
	for i, d := range distros {
		fmt.Printf("  %d. %s\n", i+1, d)
	}

	// 获取默认发行版
	fmt.Print("默认发行版: ")
	defaultDistro, err := wslBridge.GetDefaultDistro()
	if err != nil {
		fmt.Printf("❌ 获取失败 - %v\n", err)
	} else {
		fmt.Printf("✅ %s\n", defaultDistro)

		// 检查 Claude Code
		fmt.Print("Claude Code: ")
		if err := wslBridge.CheckClaudeCode(defaultDistro); err != nil {
			fmt.Printf("❌ 不可用 - %v\n", err)
		} else {
			fmt.Println("✅ 可用")
		}
	}

	// 检查路径转换
	fmt.Print("路径转换: ")
	pathConverter := converter.NewPathConverter()
	currentDir, err := converter.GetCurrentDirectory()
	if err != nil {
		fmt.Printf("❌ 获取当前目录失败 - %v\n", err)
		return nil
	}

	wslPath, err := pathConverter.ConvertToWSL(currentDir)
	if err != nil {
		fmt.Printf("❌ 转换失败 - %v\n", err)
		return nil
	}

	fmt.Printf("✅ 成功\n")
	fmt.Printf("  Windows: %s\n", currentDir)
	fmt.Printf("  WSL:     %s\n", wslPath)

	fmt.Println("\n✅ 系统环境检查完成")
	return nil
}

// runConfigShow 显示配置命令
func runConfigShow(cmd *cobra.Command, args []string) error {
	if err := initApp(); err != nil {
		return err
	}

	fmt.Println("📋 当前配置")
	fmt.Println("============")
	fmt.Printf("调试模式: %v\n", cfg.Debug)
	fmt.Printf("日志级别: %s\n", cfg.LogLevel)
	fmt.Printf("默认 WSL 发行版: %s\n", cfg.WSL.DefaultDistro)
	fmt.Printf("Claude Code 可执行文件: %s\n", cfg.ClaudeCode.Executable)
	fmt.Printf("Claude Code 默认参数: %v\n", cfg.ClaudeCode.DefaultArgs)
	fmt.Printf("交互模式: %v\n", cfg.ClaudeCode.Interactive)

	// 显示配置文件路径
	cm := config.NewConfigManager()
	if configFile != "" {
		cm.SetConfigPath(configFile)
	}
	fmt.Printf("配置文件路径: %s\n", cm.GetConfigPath())

	return nil
}

// runConfigInit 初始化配置命令
func runConfigInit(cmd *cobra.Command, args []string) error {
	cm := config.NewConfigManager()
	if configFile != "" {
		cm.SetConfigPath(configFile)
	}

	// 创建默认配置
	defaultConfig := config.GetDefaultConfig()

	// 保存配置
	if err := cm.SaveConfig(defaultConfig); err != nil {
		return fmt.Errorf("保存配置失败: %w", err)
	}

	fmt.Printf("✅ 配置文件已创建: %s\n", cm.GetConfigPath())
	return nil
}

// initApp 初始化应用程序
func initApp() error {
	// 加载配置
	var err error
	if configFile != "" {
		cfg, err = config.LoadConfigFromFile(configFile)
	} else {
		cm := config.NewConfigManager()
		cfg, err = cm.LoadConfig()
	}

	if err != nil {
		// 如果配置加载失败，使用默认配置
		cfg = config.GetDefaultConfig()
	}

	// 命令行参数覆盖配置
	if debug {
		cfg.Debug = true
	}
	if logLevel != "info" {
		cfg.LogLevel = logLevel
	}

	// 初始化日志器
	log, err = logger.CreateLoggerFromConfig(cfg.LogLevel, cfg.Debug, "")
	if err != nil {
		return fmt.Errorf("初始化日志器失败: %w", err)
	}

	// 设置全局日志器
	logger.SetGlobalLogger(log)

	log.Debug("应用程序初始化完成",
		zap.Bool("debug", cfg.Debug),
		zap.String("logLevel", cfg.LogLevel))

	return nil
}

// getWorkingDirectory 获取工作目录
func getWorkingDirectory() (string, error) {
	if targetDir != "" {
		// 使用指定目录
		absPath, err := filepath.Abs(targetDir)
		if err != nil {
			return "", apperrors.Wrapf(err, apperrors.ErrInvalidPath, "无法获取绝对路径: %s", targetDir)
		}
		return absPath, nil
	}

	// 使用当前目录
	return converter.GetCurrentDirectory()
}

// printVersion 打印版本信息
func printVersion() {
	fmt.Printf("Auto Claude Code v%s\n", version)
	fmt.Printf("Commit: %s\n", commit)
	fmt.Printf("Build Date: %s\n", date)
	fmt.Printf("Go Version: %s\n", "go1.21+")
}

// runMCPServer MCP服务器命令执行函数
func runMCPServer(cmd *cobra.Command, args []string) error {
	if err := initApp(); err != nil {
		return err
	}

	// 检查MCP配置
	if !cfg.MCP.Enabled {
		return fmt.Errorf("MCP服务器未启用，请在配置文件中设置 mcp.enabled = true")
	}

	log.Info("启动MCP服务器",
		zap.String("host", cfg.MCP.Host),
		zap.Int("port", cfg.MCP.Port),
		zap.Int("maxConcurrentTasks", cfg.MCP.MaxConcurrentTasks))

	// 创建WSL桥接器
	wslBridge := wsl.NewWSLBridge(log.GetZapLogger())

	// 检查WSL环境
	if err := wslBridge.CheckWSL(); err != nil {
		return fmt.Errorf("WSL环境检查失败: %w", err)
	}

	// 创建MCP服务器
	mcpServer := mcp.NewMCPServer(&cfg.MCP, log, wslBridge)

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动服务器
	if err := mcpServer.Start(ctx); err != nil {
		return fmt.Errorf("MCP服务器启动失败: %w", err)
	}

	log.Info("MCP服务器启动成功", zap.String("address", mcpServer.GetAddress()))

	// 等待信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞等待信号
	sig := <-sigChan
	log.Info("收到信号，开始关闭服务器", zap.String("signal", sig.String()))

	// 优雅关闭
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := mcpServer.Stop(shutdownCtx); err != nil {
		log.Error("MCP服务器关闭失败", zap.Error(err))
		return err
	}

	log.Info("MCP服务器已关闭")
	return nil
}

// runTaskList 列出所有任务
func runTaskList(cmd *cobra.Command, args []string) error {
	serverURL, _ := cmd.Flags().GetString("server")

	resp, err := http.Get(serverURL + "/tasks")
	if err != nil {
		return fmt.Errorf("连接MCP服务器失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误: %s", resp.Status)
	}

	var result struct {
		Tasks []map[string]interface{} `json:"tasks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 打印任务列表
	fmt.Println("📋 任务列表")
	fmt.Println("=" + strings.Repeat("=", 80))

	if len(result.Tasks) == 0 {
		fmt.Println("暂无任务")
		return nil
	}

	// 按状态分组统计
	statusCount := make(map[string]int)
	for _, task := range result.Tasks {
		if status, ok := task["status"].(string); ok {
			statusCount[status]++
		}
	}

	// 显示统计信息
	fmt.Printf("总计: %d 个任务", len(result.Tasks))
	for status, count := range statusCount {
		emoji := getStatusEmoji(status)
		fmt.Printf(" | %s %s: %d", emoji, status, count)
	}
	fmt.Println("\n")

	// 显示任务详情
	fmt.Printf("%-12s %-10s %-20s %-30s %-15s\n", "任务ID", "状态", "优先级", "描述", "创建时间")
	fmt.Println(strings.Repeat("-", 90))

	for _, task := range result.Tasks {
		taskID := getStringField(task, "id", "")
		status := getStringField(task, "status", "unknown")
		priority := getStringField(task, "priority", "medium")
		description := getStringField(task, "task_description", "")
		createdAt := getStringField(task, "created_at", "")

		// 截断长描述
		if len(description) > 28 {
			description = description[:25] + "..."
		}

		// 格式化时间
		if createdAt != "" {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				createdAt = t.Format("01-02 15:04")
			}
		}

		emoji := getStatusEmoji(status)
		fmt.Printf("%-12s %s %-8s %-20s %-30s %-15s\n",
			taskID[:min(12, len(taskID))], emoji, status, priority, description, createdAt)
	}

	return nil
}

// runTaskShow 查看任务详情
func runTaskShow(cmd *cobra.Command, args []string) error {
	serverURL, _ := cmd.Flags().GetString("server")
	taskID := args[0]

	resp, err := http.Get(serverURL + "/tasks/" + taskID)
	if err != nil {
		return fmt.Errorf("连接MCP服务器失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误: %s", resp.Status)
	}

	var task map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	// 打印任务详情
	fmt.Printf("🔍 任务详情: %s\n", taskID)
	fmt.Println("=" + strings.Repeat("=", 50))

	status := getStringField(task, "status", "unknown")
	emoji := getStatusEmoji(status)

	fmt.Printf("状态: %s %s\n", emoji, status)
	fmt.Printf("优先级: %s\n", getStringField(task, "priority", "medium"))
	fmt.Printf("描述: %s\n", getStringField(task, "task_description", ""))
	fmt.Printf("项目路径: %s\n", getStringField(task, "project_path", ""))
	fmt.Printf("创建时间: %s\n", formatTime(getStringField(task, "created_at", "")))
	fmt.Printf("开始时间: %s\n", formatTime(getStringField(task, "started_at", "")))
	fmt.Printf("完成时间: %s\n", formatTime(getStringField(task, "completed_at", "")))

	if worktreeID := getStringField(task, "worktree_id", ""); worktreeID != "" {
		fmt.Printf("Worktree ID: %s\n", worktreeID)
	}

	if errorMsg := getStringField(task, "error", ""); errorMsg != "" {
		fmt.Printf("错误信息: %s\n", errorMsg)
	}

	if output := getStringField(task, "output", ""); output != "" {
		fmt.Printf("\n📄 输出:\n%s\n", output)
	}

	return nil
}

// runTaskCancel 取消任务
func runTaskCancel(cmd *cobra.Command, args []string) error {
	serverURL, _ := cmd.Flags().GetString("server")
	taskID := args[0]

	req, err := http.NewRequest(http.MethodDelete, serverURL+"/tasks/"+taskID, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("连接MCP服务器失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("取消任务失败: %s", resp.Status)
	}

	fmt.Printf("✅ 任务已取消: %s\n", taskID)
	return nil
}

// runTaskSubmit 提交新任务
func runTaskSubmit(cmd *cobra.Command, args []string) error {
	serverURL, _ := cmd.Flags().GetString("server")
	projectPath, _ := cmd.Flags().GetString("project")
	description, _ := cmd.Flags().GetString("description")
	priority, _ := cmd.Flags().GetString("priority")
	timeout, _ := cmd.Flags().GetString("timeout")
	claudeArgs, _ := cmd.Flags().GetStringSlice("args")

	// 构建任务请求
	taskReq := map[string]interface{}{
		"project_path":     projectPath,
		"task_description": description,
		"priority":         priority,
		"timeout":          timeout,
		"claude_args":      claudeArgs,
	}

	reqBody, err := json.Marshal(taskReq)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	resp, err := http.Post(serverURL+"/tasks", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("连接MCP服务器失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("提交任务失败: %s", resp.Status)
	}

	var task map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	taskID := getStringField(task, "id", "")
	fmt.Printf("✅ 任务已提交: %s\n", taskID)
	fmt.Printf("状态: %s\n", getStringField(task, "status", ""))
	fmt.Printf("优先级: %s\n", priority)
	fmt.Printf("描述: %s\n", description)

	return nil
}

// runTaskWatch 实时监控任务状态
func runTaskWatch(cmd *cobra.Command, args []string) error {
	serverURL, _ := cmd.Flags().GetString("server")
	interval, _ := cmd.Flags().GetInt("interval")

	fmt.Println("🔄 实时监控任务状态 (按 Ctrl+C 退出)")
	fmt.Println("=" + strings.Repeat("=", 50))

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// 立即显示一次
	if err := displayTaskStatus(serverURL); err != nil {
		return err
	}

	for {
		select {
		case <-sigChan:
			fmt.Println("\n👋 监控已停止")
			return nil
		case <-ticker.C:
			// 清屏
			fmt.Print("\033[2J\033[H")
			fmt.Println("🔄 实时监控任务状态 (按 Ctrl+C 退出)")
			fmt.Println("=" + strings.Repeat("=", 50))

			if err := displayTaskStatus(serverURL); err != nil {
				fmt.Printf("❌ 获取任务状态失败: %v\n", err)
			}
		}
	}
}

// displayTaskStatus 显示任务状态
func displayTaskStatus(serverURL string) error {
	resp, err := http.Get(serverURL + "/tasks")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("服务器返回错误: %s", resp.Status)
	}

	var result struct {
		Tasks []map[string]interface{} `json:"tasks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	// 按状态分组
	statusGroups := make(map[string][]map[string]interface{})
	for _, task := range result.Tasks {
		status := getStringField(task, "status", "unknown")
		statusGroups[status] = append(statusGroups[status], task)
	}

	// 显示统计
	fmt.Printf("📊 总计: %d 个任务 | 更新时间: %s\n\n",
		len(result.Tasks), time.Now().Format("15:04:05"))

	// 按状态显示
	statusOrder := []string{"running", "pending", "completed", "failed", "cancelled", "timeout"}
	for _, status := range statusOrder {
		tasks := statusGroups[status]
		if len(tasks) == 0 {
			continue
		}

		emoji := getStatusEmoji(status)
		fmt.Printf("%s %s (%d):\n", emoji, strings.ToUpper(status), len(tasks))

		for _, task := range tasks {
			taskID := getStringField(task, "id", "")
			description := getStringField(task, "task_description", "")
			if len(description) > 40 {
				description = description[:37] + "..."
			}

			fmt.Printf("  • %s - %s\n", taskID[:min(8, len(taskID))], description)
		}
		fmt.Println()
	}

	return nil
}

// 辅助函数
func getStatusEmoji(status string) string {
	switch status {
	case "pending":
		return "⏳"
	case "running":
		return "🔄"
	case "completed":
		return "✅"
	case "failed":
		return "❌"
	case "cancelled":
		return "🚫"
	case "timeout":
		return "⏰"
	default:
		return "❓"
	}
}

func getStringField(m map[string]interface{}, key, defaultValue string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return defaultValue
}

func formatTime(timeStr string) string {
	if timeStr == "" {
		return "-"
	}
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t.Format("2006-01-02 15:04:05")
	}
	return timeStr
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// runTaskTUI 运行TUI界面监控
func runTaskTUI(cmd *cobra.Command, args []string) error {
	serverURL, _ := cmd.Flags().GetString("server")
	interval, _ := cmd.Flags().GetInt("interval")

	if err := ui.Init(); err != nil {
		return fmt.Errorf("初始化TUI失败: %v", err)
	}
	defer ui.Close()

	// 创建TUI组件
	tui := NewTaskTUI(serverURL, interval)
	return tui.Run()
}

// TaskInfo 任务信息结构
type TaskInfo struct {
	ID          string     `json:"id"`
	Status      string     `json:"status"`
	ProjectPath string     `json:"project_path"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// TaskTUI TUI界面结构
type TaskTUI struct {
	serverURL    string
	interval     int
	tasks        []TaskInfo
	systemInfo   SystemInfo
	lastUpdate   time.Time
	selectedTask int
}

// SystemInfo 系统信息
type SystemInfo struct {
	TotalTasks     int
	RunningTasks   int
	CompletedTasks int
	FailedTasks    int
	Uptime         time.Duration
	StartTime      time.Time
}

// NewTaskTUI 创建新的TUI实例
func NewTaskTUI(serverURL string, interval int) *TaskTUI {
	return &TaskTUI{
		serverURL: serverURL,
		interval:  interval,
		tasks:     []TaskInfo{},
		systemInfo: SystemInfo{
			StartTime: time.Now(),
		},
	}
}

// Run 运行TUI界面
func (t *TaskTUI) Run() error {
	// 创建UI组件
	header := widgets.NewParagraph()
	header.Title = "Auto Claude Code - 任务监控"
	header.Text = "正在加载..."
	header.SetRect(0, 0, 80, 3)
	header.BorderStyle.Fg = ui.ColorCyan

	summary := widgets.NewParagraph()
	summary.Title = "系统概览"
	summary.SetRect(0, 3, 40, 8)
	summary.BorderStyle.Fg = ui.ColorGreen

	taskTable := widgets.NewTable()
	taskTable.Title = "任务列表"
	taskTable.SetRect(0, 8, 120, 25)
	taskTable.BorderStyle.Fg = ui.ColorYellow
	taskTable.RowSeparator = false
	taskTable.FillRow = true

	details := widgets.NewParagraph()
	details.Title = "任务详情"
	details.SetRect(40, 3, 120, 8)
	details.BorderStyle.Fg = ui.ColorMagenta

	help := widgets.NewParagraph()
	help.Title = "快捷键"
	help.Text = "↑/↓: 选择任务 | Enter: 查看详情 | c: 取消任务 | r: 刷新 | q: 退出"
	help.SetRect(0, 25, 120, 28)
	help.BorderStyle.Fg = ui.ColorWhite

	// 初始渲染
	ui.Render(header, summary, taskTable, details, help)

	// 创建定时器
	ticker := time.NewTicker(time.Duration(t.interval) * time.Second)
	defer ticker.Stop()

	// 立即更新一次
	t.updateData()
	t.renderAll(header, summary, taskTable, details)

	// 事件循环
	uiEvents := ui.PollEvents()
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return nil
			case "<Up>":
				if t.selectedTask > 0 {
					t.selectedTask--
					t.renderTaskTable(taskTable)
					t.renderTaskDetails(details)
					ui.Render(taskTable, details)
				}
			case "<Down>":
				if t.selectedTask < len(t.tasks)-1 {
					t.selectedTask++
					t.renderTaskTable(taskTable)
					t.renderTaskDetails(details)
					ui.Render(taskTable, details)
				}
			case "<Enter>":
				if len(t.tasks) > 0 && t.selectedTask < len(t.tasks) {
					t.showTaskDetails()
				}
			case "c":
				if len(t.tasks) > 0 && t.selectedTask < len(t.tasks) {
					t.cancelTask(t.tasks[t.selectedTask].ID)
				}
			case "r":
				t.updateData()
				t.renderAll(header, summary, taskTable, details)
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				header.SetRect(0, 0, payload.Width, 3)
				summary.SetRect(0, 3, payload.Width/3, 8)
				details.SetRect(payload.Width/3, 3, payload.Width, 8)
				taskTable.SetRect(0, 8, payload.Width, payload.Height-6)
				help.SetRect(0, payload.Height-3, payload.Width, payload.Height)
				ui.Clear()
				t.renderAll(header, summary, taskTable, details)
				ui.Render(help)
			}
		case <-ticker.C:
			t.updateData()
			t.renderAll(header, summary, taskTable, details)
		}
	}
}

// updateData 更新数据
func (t *TaskTUI) updateData() {
	// 获取任务列表
	resp, err := http.Get(fmt.Sprintf("%s/api/tasks", t.serverURL))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result struct {
		Tasks []TaskInfo `json:"tasks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return
	}

	t.tasks = result.Tasks
	t.lastUpdate = time.Now()

	// 更新系统信息
	t.systemInfo.TotalTasks = len(t.tasks)
	t.systemInfo.RunningTasks = 0
	t.systemInfo.CompletedTasks = 0
	t.systemInfo.FailedTasks = 0
	t.systemInfo.Uptime = time.Since(t.systemInfo.StartTime)

	for _, task := range t.tasks {
		switch task.Status {
		case "running":
			t.systemInfo.RunningTasks++
		case "completed":
			t.systemInfo.CompletedTasks++
		case "failed":
			t.systemInfo.FailedTasks++
		}
	}

	// 确保选中的任务索引有效
	if t.selectedTask >= len(t.tasks) {
		t.selectedTask = len(t.tasks) - 1
	}
	if t.selectedTask < 0 {
		t.selectedTask = 0
	}
}

// renderAll 渲染所有组件
func (t *TaskTUI) renderAll(header, summary *widgets.Paragraph, taskTable *widgets.Table, details *widgets.Paragraph) {
	t.renderHeader(header)
	t.renderSummary(summary)
	t.renderTaskTable(taskTable)
	t.renderTaskDetails(details)
	ui.Render(header, summary, taskTable, details)
}

// renderHeader 渲染头部
func (t *TaskTUI) renderHeader(header *widgets.Paragraph) {
	header.Text = fmt.Sprintf("Auto Claude Code 任务监控 | 服务器: %s | 最后更新: %s",
		t.serverURL, t.lastUpdate.Format("15:04:05"))
}

// renderSummary 渲染概览
func (t *TaskTUI) renderSummary(summary *widgets.Paragraph) {
	summary.Text = fmt.Sprintf(`总任务数: %d
运行中: [%d](fg:green)
已完成: [%d](fg:blue)
失败: [%d](fg:red)
运行时间: %s`,
		t.systemInfo.TotalTasks,
		t.systemInfo.RunningTasks,
		t.systemInfo.CompletedTasks,
		t.systemInfo.FailedTasks,
		formatDuration(t.systemInfo.Uptime))
}

// renderTaskTable 渲染任务表格
func (t *TaskTUI) renderTaskTable(taskTable *widgets.Table) {
	// 表头
	taskTable.Rows = [][]string{
		{"ID", "状态", "项目", "描述", "优先级", "创建时间", "耗时"},
	}

	// 任务行
	for i, task := range t.tasks {
		status := getStatusEmoji(task.Status)
		if i == t.selectedTask {
			status = fmt.Sprintf("[%s](bg:blue)", status)
		}

		duration := ""
		if task.StartedAt != nil && !task.StartedAt.IsZero() {
			if task.CompletedAt != nil && !task.CompletedAt.IsZero() {
				duration = task.CompletedAt.Sub(*task.StartedAt).Truncate(time.Second).String()
			} else {
				duration = time.Since(*task.StartedAt).Truncate(time.Second).String()
			}
		}

		row := []string{
			task.ID[:8],
			status,
			truncateString(extractProjectName(task.ProjectPath), 15),
			truncateString(task.Description, 30),
			task.Priority,
			task.CreatedAt.Format("15:04:05"),
			duration,
		}

		if i == t.selectedTask {
			for j := range row {
				if j != 1 { // 不要给状态列添加背景色，因为它已经有了
					row[j] = fmt.Sprintf("[%s](bg:blue)", row[j])
				}
			}
		}

		taskTable.Rows = append(taskTable.Rows, row)
	}
}

// renderTaskDetails 渲染任务详情
func (t *TaskTUI) renderTaskDetails(details *widgets.Paragraph) {
	if len(t.tasks) == 0 || t.selectedTask >= len(t.tasks) {
		details.Text = "无任务选中"
		return
	}

	task := t.tasks[t.selectedTask]
	details.Text = fmt.Sprintf(`ID: %s
状态: %s
项目: %s
描述: %s
优先级: %s
创建时间: %s
开始时间: %s
完成时间: %s`,
		task.ID,
		task.Status,
		task.ProjectPath,
		task.Description,
		task.Priority,
		task.CreatedAt.Format("2006-01-02 15:04:05"),
		formatTimePtr(task.StartedAt),
		formatTimePtr(task.CompletedAt))
}

// showTaskDetails 显示任务详细信息（弹窗）
func (t *TaskTUI) showTaskDetails() {
	// 这里可以实现一个详细信息弹窗
	// 暂时使用简单的实现
}

// cancelTask 取消任务
func (t *TaskTUI) cancelTask(taskID string) {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/api/tasks/%s", t.serverURL, taskID), nil)
	if err != nil {
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	client.Do(req)
}

// formatDuration 格式化时间间隔
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	} else {
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	}
}

// formatTimePtr 格式化时间指针
func formatTimePtr(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "-"
	}
	return t.Format("15:04:05")
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// extractProjectName 从项目路径提取项目名
func extractProjectName(path string) string {
	if path == "" {
		return "未知项目"
	}

	// 处理Windows和Unix路径
	parts := strings.Split(strings.ReplaceAll(path, "\\", "/"), "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return "未知项目"
}
