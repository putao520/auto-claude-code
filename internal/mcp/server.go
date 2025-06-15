package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"auto-claude-code/internal/config"
	apperrors "auto-claude-code/internal/errors"
	"auto-claude-code/internal/logger"
	"auto-claude-code/internal/wsl"
)

// MCPServer MCP服务器接口
type MCPServer interface {
	// Start 启动服务器
	Start(ctx context.Context) error

	// Stop 停止服务器
	Stop(ctx context.Context) error

	// GetAddress 获取服务器地址
	GetAddress() string
}

// mcpServer MCP服务器实现
type mcpServer struct {
	config          *config.MCPConfig
	logger          logger.Logger
	protocolHandler MCPProtocolHandler
	taskManager     TaskManager
	worktreeManager WorktreeManager

	// HTTP服务器
	httpServer *http.Server
	address    string
}

// NewMCPServer 创建新的MCP服务器
func NewMCPServer(cfg *config.MCPConfig, log logger.Logger, wslBridge wsl.WSLBridge) MCPServer {
	// 创建worktree管理器
	worktreeManager := NewWorktreeManager(cfg, log)

	// 创建任务管理器
	taskManager := NewTaskManager(cfg, log, wslBridge, worktreeManager)

	// 创建协议处理器
	protocolHandler := NewMCPProtocolHandler(taskManager, worktreeManager)

	address := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	server := &mcpServer{
		config:          cfg,
		logger:          log,
		protocolHandler: protocolHandler,
		taskManager:     taskManager,
		worktreeManager: worktreeManager,
		address:         address,
	}

	// 创建HTTP服务器
	mux := http.NewServeMux()
	server.setupRoutes(mux)

	server.httpServer = &http.Server{
		Addr:         address,
		Handler:      server.withMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server
}

// Start 启动服务器
func (s *mcpServer) Start(ctx context.Context) error {
	s.logger.Info("启动MCP服务器", zap.String("address", s.address))

	// 启动worktree管理器
	if err := s.worktreeManager.Start(ctx); err != nil {
		return apperrors.Wrap(err, apperrors.ErrMCPServerError, "启动worktree管理器失败")
	}

	// 启动任务管理器
	if err := s.taskManager.Start(ctx); err != nil {
		return apperrors.Wrap(err, apperrors.ErrMCPServerError, "启动任务管理器失败")
	}

	// 启动HTTP服务器
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP服务器启动失败", zap.Error(err))
		}
	}()

	s.logger.Info("MCP服务器启动成功", zap.String("address", s.address))
	return nil
}

// Stop 停止服务器
func (s *mcpServer) Stop(ctx context.Context) error {
	s.logger.Info("停止MCP服务器")

	// 停止HTTP服务器
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Warn("HTTP服务器停止失败", zap.Error(err))
	}

	// 停止任务管理器
	if err := s.taskManager.Stop(ctx); err != nil {
		s.logger.Warn("任务管理器停止失败", zap.Error(err))
	}

	// 停止worktree管理器
	if err := s.worktreeManager.Stop(ctx); err != nil {
		s.logger.Warn("worktree管理器停止失败", zap.Error(err))
	}

	s.logger.Info("MCP服务器已停止")
	return nil
}

// GetAddress 获取服务器地址
func (s *mcpServer) GetAddress() string {
	return s.address
}

// setupRoutes 设置路由
func (s *mcpServer) setupRoutes(mux *http.ServeMux) {
	// MCP协议端点
	mux.HandleFunc("/mcp", s.handleMCPRequest)

	// 健康检查端点
	if s.config.Monitoring.Enabled {
		mux.HandleFunc(s.config.Monitoring.HealthPath, s.handleHealth)
		mux.HandleFunc(s.config.Monitoring.MetricsPath, s.handleMetrics)
	}

	// 任务管理端点
	mux.HandleFunc("/tasks", s.handleTasks)
	mux.HandleFunc("/tasks/", s.handleTaskDetail)

	// Worktree管理端点
	mux.HandleFunc("/worktrees", s.handleWorktrees)
	mux.HandleFunc("/worktrees/", s.handleWorktreeDetail)
}

// withMiddleware 添加中间件
func (s *mcpServer) withMiddleware(handler http.Handler) http.Handler {
	// 日志中间件
	handler = s.loggingMiddleware(handler)

	// 认证中间件
	if s.config.Auth.Enabled {
		handler = s.authMiddleware(handler)
	}

	// CORS中间件
	handler = s.corsMiddleware(handler)

	return handler
}

// handleMCPRequest 处理MCP请求
func (s *mcpServer) handleMCPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "只支持POST方法")
		return
	}

	// 解析JSON-RPC请求
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONRPCError(w, nil, -32700, "解析错误", err.Error())
		return
	}

	// 处理请求
	ctx := r.Context()
	response := s.processJSONRPCRequest(ctx, &req)

	// 返回响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// processJSONRPCRequest 处理JSON-RPC请求
func (s *mcpServer) processJSONRPCRequest(ctx context.Context, req *JSONRPCRequest) *JSONRPCResponse {
	response := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
	}

	switch req.Method {
	case "initialize":
		var initReq InitializeRequest
		if err := s.parseParams(req.Params, &initReq); err != nil {
			response.Error = &JSONRPCError{Code: -32602, Message: "无效参数", Data: err.Error()}
			return response
		}

		result, err := s.protocolHandler.Initialize(ctx, &initReq)
		if err != nil {
			response.Error = &JSONRPCError{Code: -32603, Message: "内部错误", Data: err.Error()}
			return response
		}
		response.Result = result

	case "tools/list":
		result, err := s.protocolHandler.ListTools(ctx)
		if err != nil {
			response.Error = &JSONRPCError{Code: -32603, Message: "内部错误", Data: err.Error()}
			return response
		}
		response.Result = map[string]interface{}{"tools": result}

	case "tools/call":
		var callReq CallToolRequest
		if err := s.parseParams(req.Params, &callReq); err != nil {
			response.Error = &JSONRPCError{Code: -32602, Message: "无效参数", Data: err.Error()}
			return response
		}

		result, err := s.protocolHandler.CallTool(ctx, &callReq)
		if err != nil {
			response.Error = &JSONRPCError{Code: -32603, Message: "内部错误", Data: err.Error()}
			return response
		}
		response.Result = result

	default:
		response.Error = &JSONRPCError{Code: -32601, Message: "方法未找到"}
	}

	return response
}

// handleHealth 处理健康检查
func (s *mcpServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 检查各组件健康状态
	if err := s.protocolHandler.HealthCheck(ctx); err != nil {
		health["status"] = "error"
		health["error"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// handleMetrics 处理指标
func (s *mcpServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 获取任务统计
	tasks, _ := s.taskManager.ListTasks(ctx)
	taskStats := make(map[string]int)
	for _, task := range tasks {
		taskStats[task.Status]++
	}

	// 获取worktree统计
	worktrees, _ := s.worktreeManager.ListWorktrees(ctx)
	worktreeStats := make(map[string]int)
	for _, wt := range worktrees {
		worktreeStats[wt.Status]++
	}

	metrics := map[string]interface{}{
		"tasks": map[string]interface{}{
			"total":     len(tasks),
			"by_status": taskStats,
		},
		"worktrees": map[string]interface{}{
			"total":     len(worktrees),
			"by_status": worktreeStats,
		},
		"timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleTasks 处理任务列表
func (s *mcpServer) handleTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		tasks, err := s.taskManager.ListTasks(ctx)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"tasks": tasks})

	case http.MethodPost:
		var req TaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "无效的请求格式")
			return
		}

		status, err := s.taskManager.SubmitTask(ctx, &req)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(status)

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "不支持的方法")
	}
}

// handleTaskDetail 处理任务详情
func (s *mcpServer) handleTaskDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	taskID := r.URL.Path[len("/tasks/"):]

	switch r.Method {
	case http.MethodGet:
		status, err := s.taskManager.GetTaskStatus(ctx, taskID)
		if err != nil {
			if apperrors.IsCode(err, apperrors.ErrTaskNotFound) {
				s.writeError(w, http.StatusNotFound, err.Error())
			} else {
				s.writeError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)

	case http.MethodDelete:
		err := s.taskManager.CancelTask(ctx, taskID)
		if err != nil {
			if apperrors.IsCode(err, apperrors.ErrTaskNotFound) {
				s.writeError(w, http.StatusNotFound, err.Error())
			} else {
				s.writeError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "不支持的方法")
	}
}

// handleWorktrees 处理worktree列表
func (s *mcpServer) handleWorktrees(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "只支持GET方法")
		return
	}

	worktrees, err := s.worktreeManager.ListWorktrees(ctx)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"worktrees": worktrees})
}

// handleWorktreeDetail 处理worktree详情
func (s *mcpServer) handleWorktreeDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	worktreeID := r.URL.Path[len("/worktrees/"):]

	switch r.Method {
	case http.MethodGet:
		worktree, err := s.worktreeManager.GetWorktree(ctx, worktreeID)
		if err != nil {
			if apperrors.IsCode(err, apperrors.ErrWorktreeNotFound) {
				s.writeError(w, http.StatusNotFound, err.Error())
			} else {
				s.writeError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(worktree)

	case http.MethodDelete:
		err := s.worktreeManager.DeleteWorktree(ctx, worktreeID)
		if err != nil {
			if apperrors.IsCode(err, apperrors.ErrWorktreeNotFound) {
				s.writeError(w, http.StatusNotFound, err.Error())
			} else {
				s.writeError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		s.writeError(w, http.StatusMethodNotAllowed, "不支持的方法")
	}
}

// 中间件函数

// loggingMiddleware 日志中间件
func (s *mcpServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if s.config.Monitoring.LogRequests {
			s.logger.Info("HTTP请求",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("remote", r.RemoteAddr))
		}

		next.ServeHTTP(w, r)

		if s.config.Monitoring.LogRequests {
			s.logger.Info("HTTP响应",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Duration("duration", time.Since(start)))
		}
	})
}

// authMiddleware 认证中间件
func (s *mcpServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 跳过健康检查端点
		if r.URL.Path == s.config.Monitoring.HealthPath {
			next.ServeHTTP(w, r)
			return
		}

		// TODO: 实现认证逻辑
		// 这里可以添加Token验证、IP白名单等

		next.ServeHTTP(w, r)
	})
}

// corsMiddleware CORS中间件
func (s *mcpServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// 辅助函数

// parseParams 解析参数
func (s *mcpServer) parseParams(params interface{}, target interface{}) error {
	if params == nil {
		return nil
	}

	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, target)
}

// writeError 写入错误响应
func (s *mcpServer) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(errorResp)
}

// writeJSONRPCError 写入JSON-RPC错误响应
func (s *mcpServer) writeJSONRPCError(w http.ResponseWriter, id interface{}, code int, message, data string) {
	w.Header().Set("Content-Type", "application/json")

	response := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	json.NewEncoder(w).Encode(response)
}
