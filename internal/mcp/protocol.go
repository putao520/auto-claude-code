package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	apperrors "auto-claude-code/internal/errors"
)

// MCPVersion MCP协议版本
const MCPVersion = "2024-11-05"

// 具体的参数类型定义

// ExecuteClaudeCodeParams 执行Claude Code的参数
type ExecuteClaudeCodeParams struct {
	ProjectPath string            `json:"projectPath"`
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Context     map[string]string `json:"context,omitempty"`
	Priority    int               `json:"priority,omitempty"`
}

// GetTaskStatusParams 获取任务状态的参数
type GetTaskStatusParams struct {
	TaskID string `json:"taskId"`
}

// CancelTaskParams 取消任务的参数
type CancelTaskParams struct {
	TaskID string `json:"taskId"`
}

// ListTasksParams 列出任务的参数
type ListTasksParams struct {
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// TaskResult 任务执行结果
type TaskResult struct {
	Output    string            `json:"output,omitempty"`
	ExitCode  int               `json:"exitCode,omitempty"`
	Error     string            `json:"error,omitempty"`
	Artifacts []string          `json:"artifacts,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// SchemaProperty JSON Schema属性定义
type SchemaProperty struct {
	Type        string                    `json:"type"`
	Description string                    `json:"description,omitempty"`
	Enum        []string                  `json:"enum,omitempty"`
	Default     interface{}               `json:"default,omitempty"`
	Properties  map[string]SchemaProperty `json:"properties,omitempty"`
	Items       *SchemaProperty           `json:"items,omitempty"`
	Required    []string                  `json:"required,omitempty"`
}

// 改进的JSON-RPC类型定义

// JSONRPCID JSON-RPC ID类型（可以是字符串或数字）
type JSONRPCID interface{}

// JSONRPCParams JSON-RPC参数的联合类型
type JSONRPCParams interface{}

// JSONRPCResult JSON-RPC结果的联合类型
type JSONRPCResult interface{}

// JSONRPCRequest JSON-RPC 2.0 请求结构
type JSONRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      JSONRPCID     `json:"id,omitempty"`
	Method  string        `json:"method"`
	Params  JSONRPCParams `json:"params,omitempty"`
}

// JSONRPCResponse JSON-RPC 2.0 响应结构
type JSONRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      JSONRPCID     `json:"id,omitempty"`
	Result  JSONRPCResult `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError JSON-RPC 2.0 错误结构
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"` // 保留interface{}，因为错误数据可以是任意类型
}

// MCPCapabilities MCP服务器能力声明
type MCPCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"` // 保留interface{}，实验性功能需要灵活性
	Logging      *LoggingCapability     `json:"logging,omitempty"`
	Prompts      *PromptsCapability     `json:"prompts,omitempty"`
	Resources    *ResourcesCapability   `json:"resources,omitempty"`
	Tools        *ToolsCapability       `json:"tools,omitempty"`
}

// LoggingCapability 日志能力
type LoggingCapability struct{}

// PromptsCapability 提示能力
type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability 资源能力
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolsCapability 工具能力
type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// InitializeRequest 初始化请求
type InitializeRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

// ClientCapabilities 客户端能力
type ClientCapabilities struct {
	Experimental map[string]interface{} `json:"experimental,omitempty"`
	Sampling     map[string]interface{} `json:"sampling,omitempty"`
}

// ClientInfo 客户端信息
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// InitializeResult 初始化结果
type InitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    MCPCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo      `json:"serverInfo"`
}

// ServerInfo 服务器信息
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tool 工具定义
type Tool struct {
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	InputSchema ToolSchema `json:"inputSchema"`
}

// ToolSchema 工具参数模式
type ToolSchema struct {
	Type       string                    `json:"type"`
	Properties map[string]SchemaProperty `json:"properties,omitempty"`
	Required   []string                  `json:"required,omitempty"`
}

// CallToolRequest 调用工具请求
type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// CallToolResult 调用工具结果
type CallToolResult struct {
	Content []ToolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ToolContent 工具内容
type ToolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// TaskRequest 任务请求
type TaskRequest struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	ProjectPath string                 `json:"projectPath"`
	Command     string                 `json:"command,omitempty"`
	Args        []string               `json:"args,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Priority    int                    `json:"priority,omitempty"`
	Timeout     time.Duration          `json:"timeout,omitempty"`
}

// TaskStatus 任务状态
type TaskStatus struct {
	ID         string                 `json:"id"`
	Status     string                 `json:"status"` // "pending", "running", "completed", "failed", "cancelled"
	Progress   float64                `json:"progress,omitempty"`
	Message    string                 `json:"message,omitempty"`
	Result     interface{}            `json:"result,omitempty"`
	Error      string                 `json:"error,omitempty"`
	StartTime  time.Time              `json:"startTime,omitempty"`
	EndTime    time.Time              `json:"endTime,omitempty"`
	WorktreeID string                 `json:"worktreeId,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// MCPProtocolHandler MCP协议处理器接口
type MCPProtocolHandler interface {
	// 协议方法
	Initialize(ctx context.Context, req *InitializeRequest) (*InitializeResult, error)
	ListTools(ctx context.Context) ([]Tool, error)
	CallTool(ctx context.Context, req *CallToolRequest) (*CallToolResult, error)

	// 任务管理方法
	SubmitTask(ctx context.Context, req *TaskRequest) (*TaskStatus, error)
	GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error)
	CancelTask(ctx context.Context, taskID string) error
	ListTasks(ctx context.Context) ([]*TaskStatus, error)

	// 健康检查
	HealthCheck(ctx context.Context) error
}

// protocolHandler MCP协议处理器实现
type protocolHandler struct {
	serverInfo      ServerInfo
	capabilities    MCPCapabilities
	taskManager     TaskManager
	worktreeManager WorktreeManager
}

// NewMCPProtocolHandler 创建新的MCP协议处理器
func NewMCPProtocolHandler(taskManager TaskManager, worktreeManager WorktreeManager) MCPProtocolHandler {
	return &protocolHandler{
		serverInfo: ServerInfo{
			Name:    "auto-claude-code-mcp",
			Version: "1.0.0",
		},
		capabilities: MCPCapabilities{
			Tools: &ToolsCapability{
				ListChanged: true,
			},
			Logging: &LoggingCapability{},
		},
		taskManager:     taskManager,
		worktreeManager: worktreeManager,
	}
}

// Initialize 初始化MCP连接
func (h *protocolHandler) Initialize(ctx context.Context, req *InitializeRequest) (*InitializeResult, error) {
	// 验证协议版本
	if req.ProtocolVersion != MCPVersion {
		return nil, apperrors.Newf(apperrors.ErrMCPProtocolError,
			"不支持的协议版本: %s，期望: %s", req.ProtocolVersion, MCPVersion)
	}

	return &InitializeResult{
		ProtocolVersion: MCPVersion,
		Capabilities:    h.capabilities,
		ServerInfo:      h.serverInfo,
	}, nil
}

// ListTools 列出可用工具
func (h *protocolHandler) ListTools(ctx context.Context) ([]Tool, error) {
	tools := []Tool{
		{
			Name:        "execute_claude_code",
			Description: "在WSL环境中执行Claude Code任务",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]SchemaProperty{
					"projectPath": stringProperty("项目路径（Windows路径）"),
					"command":     stringProperty("要执行的命令", ""),
					"args":        arrayProperty("命令参数", "string"),
					"priority":    integerProperty("任务优先级 (1-3)", 2, 1, 3),
					"timeout":     stringProperty("任务超时时间 (如: 30m, 1h)", "30m"),
				},
				Required: []string{"projectPath"},
			},
		},
		{
			Name:        "get_task_status",
			Description: "获取任务执行状态",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]SchemaProperty{
					"taskId": stringProperty("任务ID"),
				},
				Required: []string{"taskId"},
			},
		},
		{
			Name:        "cancel_task",
			Description: "取消正在执行的任务",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]SchemaProperty{
					"taskId": stringProperty("任务ID"),
				},
				Required: []string{"taskId"},
			},
		},
		{
			Name:        "list_tasks",
			Description: "列出所有任务状态",
			InputSchema: ToolSchema{
				Type: "object",
				Properties: map[string]SchemaProperty{
					"status": enumProperty("过滤任务状态", []string{"pending", "running", "completed", "failed", "cancelled"}),
				},
			},
		},
	}

	return tools, nil
}

// CallTool 调用工具
func (h *protocolHandler) CallTool(ctx context.Context, req *CallToolRequest) (*CallToolResult, error) {
	switch req.Name {
	case "execute_claude_code":
		return h.handleExecuteClaudeCode(ctx, req.Arguments)
	case "get_task_status":
		return h.handleGetTaskStatus(ctx, req.Arguments)
	case "cancel_task":
		return h.handleCancelTask(ctx, req.Arguments)
	case "list_tasks":
		return h.handleListTasks(ctx, req.Arguments)
	default:
		return &CallToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("未知工具: %s", req.Name),
			}},
			IsError: true,
		}, nil
	}
}

// handleExecuteClaudeCode 处理执行Claude Code工具调用
func (h *protocolHandler) handleExecuteClaudeCode(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	// 解析参数
	projectPath, ok := args["projectPath"].(string)
	if !ok || projectPath == "" {
		return &CallToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: "缺少必需参数: projectPath",
			}},
			IsError: true,
		}, nil
	}

	// 构建任务请求
	taskReq := &TaskRequest{
		Type:        "claude_code",
		ProjectPath: projectPath,
		Priority:    2, // 默认优先级
	}

	// 解析可选参数
	if command, ok := args["command"].(string); ok {
		taskReq.Command = command
	}

	if argsSlice, ok := args["args"].([]interface{}); ok {
		for _, arg := range argsSlice {
			if argStr, ok := arg.(string); ok {
				taskReq.Args = append(taskReq.Args, argStr)
			}
		}
	}

	if priority, ok := args["priority"].(float64); ok {
		taskReq.Priority = int(priority)
	}

	if timeoutStr, ok := args["timeout"].(string); ok {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			taskReq.Timeout = timeout
		}
	}

	// 提交任务
	status, err := h.SubmitTask(ctx, taskReq)
	if err != nil {
		return &CallToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("任务提交失败: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// 返回任务状态
	statusJSON, _ := json.MarshalIndent(status, "", "  ")
	return &CallToolResult{
		Content: []ToolContent{{
			Type: "text",
			Text: fmt.Sprintf("任务已提交:\n%s", string(statusJSON)),
		}},
	}, nil
}

// handleGetTaskStatus 处理获取任务状态工具调用
func (h *protocolHandler) handleGetTaskStatus(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	taskID, ok := args["taskId"].(string)
	if !ok || taskID == "" {
		return &CallToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: "缺少必需参数: taskId",
			}},
			IsError: true,
		}, nil
	}

	status, err := h.GetTaskStatus(ctx, taskID)
	if err != nil {
		return &CallToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("获取任务状态失败: %v", err),
			}},
			IsError: true,
		}, nil
	}

	statusJSON, _ := json.MarshalIndent(status, "", "  ")
	return &CallToolResult{
		Content: []ToolContent{{
			Type: "text",
			Text: string(statusJSON),
		}},
	}, nil
}

// handleCancelTask 处理取消任务工具调用
func (h *protocolHandler) handleCancelTask(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	taskID, ok := args["taskId"].(string)
	if !ok || taskID == "" {
		return &CallToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: "缺少必需参数: taskId",
			}},
			IsError: true,
		}, nil
	}

	err := h.CancelTask(ctx, taskID)
	if err != nil {
		return &CallToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("取消任务失败: %v", err),
			}},
			IsError: true,
		}, nil
	}

	return &CallToolResult{
		Content: []ToolContent{{
			Type: "text",
			Text: fmt.Sprintf("任务 %s 已取消", taskID),
		}},
	}, nil
}

// handleListTasks 处理列出任务工具调用
func (h *protocolHandler) handleListTasks(ctx context.Context, args map[string]interface{}) (*CallToolResult, error) {
	tasks, err := h.ListTasks(ctx)
	if err != nil {
		return &CallToolResult{
			Content: []ToolContent{{
				Type: "text",
				Text: fmt.Sprintf("获取任务列表失败: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// 过滤任务状态
	if statusFilter, ok := args["status"].(string); ok && statusFilter != "" {
		var filteredTasks []*TaskStatus
		for _, task := range tasks {
			if task.Status == statusFilter {
				filteredTasks = append(filteredTasks, task)
			}
		}
		tasks = filteredTasks
	}

	tasksJSON, _ := json.MarshalIndent(tasks, "", "  ")
	return &CallToolResult{
		Content: []ToolContent{{
			Type: "text",
			Text: string(tasksJSON),
		}},
	}, nil
}

// SubmitTask 提交任务
func (h *protocolHandler) SubmitTask(ctx context.Context, req *TaskRequest) (*TaskStatus, error) {
	return h.taskManager.SubmitTask(ctx, req)
}

// GetTaskStatus 获取任务状态
func (h *protocolHandler) GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	return h.taskManager.GetTaskStatus(ctx, taskID)
}

// CancelTask 取消任务
func (h *protocolHandler) CancelTask(ctx context.Context, taskID string) error {
	return h.taskManager.CancelTask(ctx, taskID)
}

// ListTasks 列出任务
func (h *protocolHandler) ListTasks(ctx context.Context) ([]*TaskStatus, error) {
	return h.taskManager.ListTasks(ctx)
}

// HealthCheck 健康检查
func (h *protocolHandler) HealthCheck(ctx context.Context) error {
	// 检查任务管理器状态
	if err := h.taskManager.HealthCheck(ctx); err != nil {
		return err
	}

	// 检查worktree管理器状态
	if err := h.worktreeManager.HealthCheck(ctx); err != nil {
		return err
	}

	return nil
}

// SchemaProperty助手函数

// stringProperty 创建字符串类型的属性
func stringProperty(description string, defaultValue ...string) SchemaProperty {
	prop := SchemaProperty{
		Type:        "string",
		Description: description,
	}
	if len(defaultValue) > 0 {
		prop.Default = defaultValue[0]
	}
	return prop
}

// arrayProperty 创建数组类型的属性
func arrayProperty(description string, itemType string) SchemaProperty {
	return SchemaProperty{
		Type:        "array",
		Description: description,
		Items: &SchemaProperty{
			Type: itemType,
		},
	}
}

// integerProperty 创建整数类型的属性
func integerProperty(description string, defaultValue int, min int, max int) SchemaProperty {
	return SchemaProperty{
		Type:        "integer",
		Description: description,
		Default:     defaultValue,
	}
}

// enumProperty 创建枚举类型的属性
func enumProperty(description string, values []string) SchemaProperty {
	return SchemaProperty{
		Type:        "string",
		Description: description,
		Enum:        values,
	}
}
