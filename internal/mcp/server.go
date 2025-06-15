package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
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

	// 传输层
	multiTransport *MultiTransport
	address        string
}

// NewMCPServer 创建新的MCP服务器
func NewMCPServer(cfg *config.MCPConfig, log logger.Logger, wslBridge wsl.WSLBridge) MCPServer {
	// 创建worktree管理器
	worktreeManager := NewWorktreeManager(cfg, log)

	// 创建任务管理器
	taskManager := NewTaskManager(cfg, log, wslBridge, worktreeManager)

	// 创建协议处理器
	protocolHandler := NewMCPProtocolHandler(taskManager, worktreeManager)

	server := &mcpServer{
		config:          cfg,
		logger:          log,
		protocolHandler: protocolHandler,
		taskManager:     taskManager,
		worktreeManager: worktreeManager,
		multiTransport:  NewMultiTransport(log),
		address:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
	}

	// 创建传输处理器适配器
	transportHandler := &transportHandlerAdapter{server: server}

	// 配置HTTP传输
	if cfg.HTTP.Enabled {
		mux := http.NewServeMux()
		server.setupRoutes(mux)

		httpServer := &http.Server{
			Addr:         server.address,
			Handler:      server.withMiddleware(mux),
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		httpTransport := NewHTTPTransport(httpServer, server.address, transportHandler, log)
		server.multiTransport.AddTransport(httpTransport)
	}

	// 配置stdio传输
	if cfg.Stdio.Enabled {
		stdioTransport := NewStdioTransport(transportHandler, log, cfg.Stdio.Reader, cfg.Stdio.Writer)
		server.multiTransport.AddTransport(stdioTransport)
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

	// 启动多传输服务器
	if err := s.multiTransport.Start(ctx); err != nil {
		return apperrors.Wrap(err, apperrors.ErrMCPServerError, "启动传输层失败")
	}

	s.logger.Info("MCP服务器启动成功", zap.String("address", s.address))
	return nil
}

// Stop 停止服务器
func (s *mcpServer) Stop(ctx context.Context) error {
	s.logger.Info("停止MCP服务器")

	// 停止传输层
	if err := s.multiTransport.Stop(ctx); err != nil {
		s.logger.Warn("传输层停止失败", zap.Error(err))
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

		// 如果认证未启用，直接通过
		if !s.config.Auth.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// IP白名单验证
		if !s.validateClientIP(r) {
			s.logger.Warn("访问被拒绝 - IP不在白名单",
				zap.String("remote_ip", s.getClientIP(r)),
				zap.String("path", r.URL.Path))
			s.writeError(w, http.StatusForbidden, "访问被拒绝：IP地址不被允许")
			return
		}

		// Token验证
		if s.config.Auth.Method == "token" {
			if !s.validateToken(r) {
				s.logger.Warn("访问被拒绝 - Token验证失败",
					zap.String("remote_ip", s.getClientIP(r)),
					zap.String("path", r.URL.Path))
				s.writeError(w, http.StatusUnauthorized, "未授权访问：Token验证失败")
				return
			}
		}

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

// 认证相关方法

// validateClientIP 验证客户端IP是否在白名单中
func (s *mcpServer) validateClientIP(r *http.Request) bool {
	// 如果没有配置IP白名单，默认允许所有IP
	if len(s.config.Auth.AllowedIPs) == 0 {
		return true
	}

	clientIP := s.getClientIP(r)

	// 检查IP是否在白名单中
	for _, allowedIP := range s.config.Auth.AllowedIPs {
		if allowedIP == "*" || allowedIP == clientIP {
			return true
		}
		// 支持CIDR格式的IP范围
		if strings.Contains(allowedIP, "/") {
			if s.isIPInCIDR(clientIP, allowedIP) {
				return true
			}
		}
	}

	return false
}

// getClientIP 获取真实的客户端IP地址
func (s *mcpServer) getClientIP(r *http.Request) string {
	// 检查常见的代理头
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// X-Forwarded-For 可能包含多个IP，取第一个
		if ips := strings.Split(ip, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}

	if ip := r.Header.Get("X-Forwarded"); ip != "" {
		return ip
	}

	// 从RemoteAddr获取IP（移除端口）
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}

	return r.RemoteAddr
}

// isIPInCIDR 检查IP是否在CIDR范围内
func (s *mcpServer) isIPInCIDR(ip, cidr string) bool {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		s.logger.Warn("无效的CIDR格式", zap.String("cidr", cidr), zap.Error(err))
		return false
	}

	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		s.logger.Warn("无效的IP地址", zap.String("ip", ip))
		return false
	}

	return network.Contains(parsedIP)
}

// validateToken 验证Token
func (s *mcpServer) validateToken(r *http.Request) bool {
	// 从Authorization头获取token
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	// 支持Bearer token格式
	var token string
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		token = authHeader
	}

	if token == "" {
		return false
	}

	// 从文件读取有效的tokens
	validTokens, err := s.loadValidTokens()
	if err != nil {
		s.logger.Error("加载token文件失败", zap.Error(err))
		return false
	}

	// 验证token
	for _, validToken := range validTokens {
		if validToken == token {
			return true
		}
	}

	return false
}

// loadValidTokens 从文件加载有效的tokens
func (s *mcpServer) loadValidTokens() ([]string, error) {
	if s.config.Auth.TokenFile == "" {
		return nil, fmt.Errorf("未配置token文件")
	}

	data, err := os.ReadFile(s.config.Auth.TokenFile)
	if err != nil {
		return nil, fmt.Errorf("读取token文件失败: %w", err)
	}

	var tokens []string
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过空行和注释行
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		tokens = append(tokens, line)
	}

	return tokens, nil
}
