package mcp

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"auto-claude-code/internal/config"
	"auto-claude-code/internal/converter"
	apperrors "auto-claude-code/internal/errors"
	"auto-claude-code/internal/logger"
	"auto-claude-code/internal/wsl"
)

// taskManager 任务管理器实现
type taskManager struct {
	config          *config.MCPConfig
	logger          logger.Logger
	wslBridge       wsl.WSLBridge
	pathConverter   converter.PathConverter
	worktreeManager WorktreeManager

	// 任务管理
	tasks       map[string]*TaskStatus
	tasksMutex  sync.RWMutex
	taskQueue   chan *TaskRequest
	workers     []*taskWorker
	workerCount int

	// 生命周期管理
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// taskWorker 任务工作器
type taskWorker struct {
	id          int
	manager     *taskManager
	ctx         context.Context
	cancel      context.CancelFunc
	currentTask *TaskStatus
	mutex       sync.RWMutex
}

// NewTaskManager 创建新的任务管理器
func NewTaskManager(cfg *config.MCPConfig, log logger.Logger, wslBridge wsl.WSLBridge, worktreeManager WorktreeManager) TaskManager {
	return &taskManager{
		config:          cfg,
		logger:          log,
		wslBridge:       wslBridge,
		pathConverter:   converter.NewPathConverter(),
		worktreeManager: worktreeManager,
		tasks:           make(map[string]*TaskStatus),
		taskQueue:       make(chan *TaskRequest, cfg.Queue.MaxSize),
		workerCount:     cfg.MaxConcurrentTasks,
	}
}

// Start 启动任务管理器
func (tm *taskManager) Start(ctx context.Context) error {
	tm.ctx, tm.cancel = context.WithCancel(ctx)

	tm.logger.Info("启动任务管理器",
		zap.Int("workerCount", tm.workerCount),
		zap.Int("queueSize", tm.config.Queue.MaxSize))

	// 启动工作器
	tm.workers = make([]*taskWorker, tm.workerCount)
	for i := 0; i < tm.workerCount; i++ {
		worker := &taskWorker{
			id:      i,
			manager: tm,
		}
		worker.ctx, worker.cancel = context.WithCancel(tm.ctx)
		tm.workers[i] = worker

		tm.wg.Add(1)
		go worker.run()
	}

	// 启动任务清理器
	tm.wg.Add(1)
	go tm.runTaskCleaner()

	return nil
}

// Stop 停止任务管理器
func (tm *taskManager) Stop(ctx context.Context) error {
	tm.logger.Info("停止任务管理器")

	// 取消所有工作器
	if tm.cancel != nil {
		tm.cancel()
	}

	// 等待所有工作器停止
	done := make(chan struct{})
	go func() {
		tm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		tm.logger.Info("任务管理器已停止")
	case <-ctx.Done():
		tm.logger.Warn("任务管理器停止超时")
		return ctx.Err()
	}

	return nil
}

// SubmitTask 提交任务
func (tm *taskManager) SubmitTask(ctx context.Context, req *TaskRequest) (*TaskStatus, error) {
	// 生成任务ID
	if req.ID == "" {
		req.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}

	// 设置默认超时
	if req.Timeout == 0 {
		if timeout, err := time.ParseDuration(tm.config.TaskTimeout); err == nil {
			req.Timeout = timeout
		} else {
			req.Timeout = 30 * time.Minute
		}
	}

	// 创建任务状态
	status := &TaskStatus{
		ID:       req.ID,
		Status:   "pending",
		Progress: 0,
		Message:  "任务已提交，等待执行",
		Metadata: make(map[string]interface{}),
	}

	// 保存任务状态
	tm.tasksMutex.Lock()
	tm.tasks[req.ID] = status
	tm.tasksMutex.Unlock()

	// 提交到队列
	select {
	case tm.taskQueue <- req:
		tm.logger.Info("任务已提交到队列",
			zap.String("taskId", req.ID),
			zap.String("type", req.Type),
			zap.String("projectPath", req.ProjectPath))
		return status, nil
	case <-ctx.Done():
		// 清理任务状态
		tm.tasksMutex.Lock()
		delete(tm.tasks, req.ID)
		tm.tasksMutex.Unlock()
		return nil, ctx.Err()
	default:
		// 队列已满
		tm.tasksMutex.Lock()
		delete(tm.tasks, req.ID)
		tm.tasksMutex.Unlock()
		return nil, apperrors.New(apperrors.ErrTaskNotSupported, "任务队列已满")
	}
}

// GetTaskStatus 获取任务状态
func (tm *taskManager) GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error) {
	tm.tasksMutex.RLock()
	status, exists := tm.tasks[taskID]
	tm.tasksMutex.RUnlock()

	if !exists {
		return nil, apperrors.Newf(apperrors.ErrTaskNotFound, "任务不存在: %s", taskID)
	}

	// 返回状态副本
	statusCopy := *status
	return &statusCopy, nil
}

// CancelTask 取消任务
func (tm *taskManager) CancelTask(ctx context.Context, taskID string) error {
	tm.tasksMutex.Lock()
	status, exists := tm.tasks[taskID]
	if !exists {
		tm.tasksMutex.Unlock()
		return apperrors.Newf(apperrors.ErrTaskNotFound, "任务不存在: %s", taskID)
	}

	// 检查任务状态
	if status.Status == "completed" || status.Status == "failed" || status.Status == "cancelled" {
		tm.tasksMutex.Unlock()
		return apperrors.Newf(apperrors.ErrTaskCancelled, "任务已完成或已取消: %s", taskID)
	}

	// 标记为取消
	status.Status = "cancelled"
	status.Message = "任务已取消"
	status.EndTime = time.Now()
	tm.tasksMutex.Unlock()

	// 通知工作器取消任务
	for _, worker := range tm.workers {
		worker.mutex.RLock()
		if worker.currentTask != nil && worker.currentTask.ID == taskID {
			worker.cancel()
		}
		worker.mutex.RUnlock()
	}

	tm.logger.Info("任务已取消", zap.String("taskId", taskID))
	return nil
}

// ListTasks 列出所有任务
func (tm *taskManager) ListTasks(ctx context.Context) ([]*TaskStatus, error) {
	tm.tasksMutex.RLock()
	defer tm.tasksMutex.RUnlock()

	tasks := make([]*TaskStatus, 0, len(tm.tasks))
	for _, status := range tm.tasks {
		statusCopy := *status
		tasks = append(tasks, &statusCopy)
	}

	return tasks, nil
}

// HealthCheck 健康检查
func (tm *taskManager) HealthCheck(ctx context.Context) error {
	// 检查工作器状态
	activeWorkers := 0
	for _, worker := range tm.workers {
		select {
		case <-worker.ctx.Done():
			// 工作器已停止
		default:
			activeWorkers++
		}
	}

	if activeWorkers == 0 {
		return apperrors.New(apperrors.ErrInstanceFailed, "没有活跃的任务工作器")
	}

	// 检查队列状态
	queueLen := len(tm.taskQueue)
	if queueLen >= tm.config.Queue.MaxSize {
		return apperrors.New(apperrors.ErrTaskNotSupported, "任务队列已满")
	}

	tm.logger.Debug("任务管理器健康检查通过",
		zap.Int("activeWorkers", activeWorkers),
		zap.Int("queueLength", queueLen))

	return nil
}

// runTaskCleaner 运行任务清理器
func (tm *taskManager) runTaskCleaner() {
	defer tm.wg.Done()

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-tm.ctx.Done():
			return
		case <-ticker.C:
			tm.cleanupCompletedTasks()
		}
	}
}

// cleanupCompletedTasks 清理已完成的任务
func (tm *taskManager) cleanupCompletedTasks() {
	tm.tasksMutex.Lock()
	defer tm.tasksMutex.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour) // 保留24小时内的任务
	var toDelete []string

	for taskID, status := range tm.tasks {
		if (status.Status == "completed" || status.Status == "failed" || status.Status == "cancelled") &&
			!status.EndTime.IsZero() && status.EndTime.Before(cutoff) {
			toDelete = append(toDelete, taskID)
		}
	}

	for _, taskID := range toDelete {
		delete(tm.tasks, taskID)
	}

	if len(toDelete) > 0 {
		tm.logger.Info("清理已完成的任务", zap.Int("count", len(toDelete)))
	}
}

// run 工作器运行循环
func (w *taskWorker) run() {
	defer w.manager.wg.Done()

	w.manager.logger.Debug("任务工作器启动", zap.Int("workerId", w.id))

	for {
		select {
		case <-w.ctx.Done():
			w.manager.logger.Debug("任务工作器停止", zap.Int("workerId", w.id))
			return
		case req := <-w.manager.taskQueue:
			w.executeTask(req)
		}
	}
}

// executeTask 执行任务
func (w *taskWorker) executeTask(req *TaskRequest) {
	w.manager.logger.Info("开始执行任务",
		zap.Int("workerId", w.id),
		zap.String("taskId", req.ID),
		zap.String("type", req.Type))

	// 获取任务状态
	w.manager.tasksMutex.Lock()
	status, exists := w.manager.tasks[req.ID]
	if !exists {
		w.manager.tasksMutex.Unlock()
		return
	}

	// 检查任务是否已被取消
	if status.Status == "cancelled" {
		w.manager.tasksMutex.Unlock()
		return
	}

	// 更新任务状态
	status.Status = "running"
	status.Message = "任务正在执行"
	status.StartTime = time.Now()
	status.Progress = 0.1
	w.manager.tasksMutex.Unlock()

	// 设置当前任务
	w.mutex.Lock()
	w.currentTask = status
	w.mutex.Unlock()

	// 创建任务上下文
	taskCtx, taskCancel := context.WithTimeout(w.ctx, req.Timeout)
	defer taskCancel()

	// 执行任务
	var err error
	switch req.Type {
	case "claude_code":
		err = w.executeClaudeCodeTask(taskCtx, req, status)
	default:
		err = apperrors.Newf(apperrors.ErrTaskNotSupported, "不支持的任务类型: %s", req.Type)
	}

	// 更新最终状态
	w.manager.tasksMutex.Lock()
	if err != nil {
		status.Status = "failed"
		status.Error = err.Error()
		status.Message = "任务执行失败"
	} else {
		status.Status = "completed"
		status.Message = "任务执行成功"
		status.Progress = 1.0
	}
	status.EndTime = time.Now()
	w.manager.tasksMutex.Unlock()

	// 清除当前任务
	w.mutex.Lock()
	w.currentTask = nil
	w.mutex.Unlock()

	w.manager.logger.Info("任务执行完成",
		zap.Int("workerId", w.id),
		zap.String("taskId", req.ID),
		zap.String("status", status.Status),
		zap.Error(err))
}

// executeClaudeCodeTask 执行Claude Code任务
func (w *taskWorker) executeClaudeCodeTask(ctx context.Context, req *TaskRequest, status *TaskStatus) error {
	// 验证路径
	if err := w.manager.pathConverter.ValidatePath(req.ProjectPath); err != nil {
		return apperrors.Wrap(err, apperrors.ErrInvalidPath, "项目路径验证失败")
	}

	// 更新进度
	w.manager.tasksMutex.Lock()
	status.Progress = 0.2
	status.Message = "正在转换路径"
	w.manager.tasksMutex.Unlock()

	// 转换路径
	wslPath, err := w.manager.pathConverter.ConvertToWSL(req.ProjectPath)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrPathConversion, "路径转换失败")
	}

	// 更新进度
	w.manager.tasksMutex.Lock()
	status.Progress = 0.4
	status.Message = "正在创建工作树"
	w.manager.tasksMutex.Unlock()

	// 创建worktree
	worktree, err := w.manager.worktreeManager.CreateWorktree(ctx, req.ProjectPath)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrWorktreeFailed, "创建工作树失败")
	}

	// 记录worktree ID
	w.manager.tasksMutex.Lock()
	status.WorktreeID = worktree.ID
	status.Progress = 0.6
	status.Message = "正在启动Claude Code"
	w.manager.tasksMutex.Unlock()

	// 构建Claude Code参数
	args := append([]string{}, req.Args...)
	if req.Command != "" {
		args = append([]string{req.Command}, args...)
	}

	// 启动Claude Code
	err = w.manager.wslBridge.StartClaudeCode("", wslPath, args)
	if err != nil {
		// 清理worktree
		w.manager.worktreeManager.DeleteWorktree(context.Background(), worktree.ID)
		return apperrors.Wrap(err, apperrors.ErrClaudeCodeFailed, "Claude Code启动失败")
	}

	// 更新进度
	w.manager.tasksMutex.Lock()
	status.Progress = 0.9
	status.Message = "Claude Code执行完成"
	status.Result = map[string]interface{}{
		"wslPath":     wslPath,
		"worktreeId":  worktree.ID,
		"projectPath": req.ProjectPath,
	}
	w.manager.tasksMutex.Unlock()

	return nil
}
