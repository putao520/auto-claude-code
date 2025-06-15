package mcp

import (
	"context"
	"testing"

	"auto-claude-code/internal/config"
	"auto-claude-code/internal/logger"
	"auto-claude-code/internal/wsl"
)

func TestMCPProtocolHandler_Initialize(t *testing.T) {
	// 创建测试配置
	cfg := &config.MCPConfig{
		Enabled:            true,
		Port:               8080,
		Host:               "localhost",
		MaxConcurrentTasks: 5,
		TaskTimeout:        "30m",
		WorktreeBaseDir:    "./test_worktrees",
		CleanupInterval:    "1h",
		MaxWorktrees:       10,
	}

	// 创建测试日志器
	log, err := logger.CreateLoggerFromConfig("info", false, "")
	if err != nil {
		t.Fatalf("创建日志器失败: %v", err)
	}

	// 创建模拟的WSL桥接器
	wslBridge := wsl.NewWSLBridge(log.GetZapLogger())

	// 创建worktree管理器
	worktreeManager := NewWorktreeManager(cfg, log)

	// 创建任务管理器
	taskManager := NewTaskManager(cfg, log, wslBridge, worktreeManager)

	// 创建协议处理器
	handler := NewMCPProtocolHandler(taskManager, worktreeManager)

	// 测试初始化
	ctx := context.Background()
	req := &InitializeRequest{
		ProtocolVersion: MCPVersion,
		Capabilities:    ClientCapabilities{},
		ClientInfo: ClientInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	result, err := handler.Initialize(ctx, req)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 验证结果
	if result.ProtocolVersion != MCPVersion {
		t.Errorf("协议版本不匹配: 期望 %s, 得到 %s", MCPVersion, result.ProtocolVersion)
	}

	if result.ServerInfo.Name != "auto-claude-code-mcp" {
		t.Errorf("服务器名称不匹配: 期望 %s, 得到 %s", "auto-claude-code-mcp", result.ServerInfo.Name)
	}
}

func TestMCPProtocolHandler_ListTools(t *testing.T) {
	// 创建测试配置
	cfg := &config.MCPConfig{
		Enabled:            true,
		Port:               8080,
		Host:               "localhost",
		MaxConcurrentTasks: 5,
		TaskTimeout:        "30m",
		WorktreeBaseDir:    "./test_worktrees",
		CleanupInterval:    "1h",
		MaxWorktrees:       10,
	}

	// 创建测试日志器
	log, err := logger.CreateLoggerFromConfig("info", false, "")
	if err != nil {
		t.Fatalf("创建日志器失败: %v", err)
	}

	// 创建模拟的WSL桥接器
	wslBridge := wsl.NewWSLBridge(log.GetZapLogger())

	// 创建worktree管理器
	worktreeManager := NewWorktreeManager(cfg, log)

	// 创建任务管理器
	taskManager := NewTaskManager(cfg, log, wslBridge, worktreeManager)

	// 创建协议处理器
	handler := NewMCPProtocolHandler(taskManager, worktreeManager)

	// 测试列出工具
	ctx := context.Background()
	tools, err := handler.ListTools(ctx)
	if err != nil {
		t.Fatalf("列出工具失败: %v", err)
	}

	// 验证工具列表
	expectedTools := []string{
		"execute_claude_code",
		"get_task_status",
		"cancel_task",
		"list_tasks",
	}

	if len(tools) != len(expectedTools) {
		t.Errorf("工具数量不匹配: 期望 %d, 得到 %d", len(expectedTools), len(tools))
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	for _, expectedTool := range expectedTools {
		if !toolNames[expectedTool] {
			t.Errorf("缺少工具: %s", expectedTool)
		}
	}
}

func TestMCPProtocolHandler_HealthCheck(t *testing.T) {
	// 创建测试配置
	cfg := &config.MCPConfig{
		Enabled:            true,
		Port:               8080,
		Host:               "localhost",
		MaxConcurrentTasks: 5,
		TaskTimeout:        "30m",
		WorktreeBaseDir:    "./test_worktrees",
		CleanupInterval:    "1h",
		MaxWorktrees:       10,
	}

	// 创建测试日志器
	log, err := logger.CreateLoggerFromConfig("info", false, "")
	if err != nil {
		t.Fatalf("创建日志器失败: %v", err)
	}

	// 创建模拟的WSL桥接器
	wslBridge := wsl.NewWSLBridge(log.GetZapLogger())

	// 创建worktree管理器
	worktreeManager := NewWorktreeManager(cfg, log)

	// 启动worktree管理器
	ctx := context.Background()
	if err := worktreeManager.Start(ctx); err != nil {
		t.Fatalf("启动worktree管理器失败: %v", err)
	}
	defer worktreeManager.Stop(ctx)

	// 创建任务管理器
	taskManager := NewTaskManager(cfg, log, wslBridge, worktreeManager)

	// 启动任务管理器
	if err := taskManager.Start(ctx); err != nil {
		t.Fatalf("启动任务管理器失败: %v", err)
	}
	defer taskManager.Stop(ctx)

	// 创建协议处理器
	handler := NewMCPProtocolHandler(taskManager, worktreeManager)

	// 测试健康检查
	err = handler.HealthCheck(ctx)
	if err != nil {
		t.Errorf("健康检查失败: %v", err)
	}
}
