package mcp

import (
	"context"
)

// TaskManager 任务管理器接口
type TaskManager interface {
	// SubmitTask 提交任务
	SubmitTask(ctx context.Context, req *TaskRequest) (*TaskStatus, error)

	// GetTaskStatus 获取任务状态
	GetTaskStatus(ctx context.Context, taskID string) (*TaskStatus, error)

	// CancelTask 取消任务
	CancelTask(ctx context.Context, taskID string) error

	// ListTasks 列出所有任务
	ListTasks(ctx context.Context) ([]*TaskStatus, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error

	// Start 启动任务管理器
	Start(ctx context.Context) error

	// Stop 停止任务管理器
	Stop(ctx context.Context) error
}

// WorktreeManager Git worktree管理器接口
type WorktreeManager interface {
	// CreateWorktree 创建新的worktree
	CreateWorktree(ctx context.Context, projectPath string) (*WorktreeInfo, error)

	// DeleteWorktree 删除worktree
	DeleteWorktree(ctx context.Context, worktreeID string) error

	// GetWorktree 获取worktree信息
	GetWorktree(ctx context.Context, worktreeID string) (*WorktreeInfo, error)

	// ListWorktrees 列出所有worktrees
	ListWorktrees(ctx context.Context) ([]*WorktreeInfo, error)

	// CleanupWorktrees 清理过期的worktrees
	CleanupWorktrees(ctx context.Context) error

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error

	// Start 启动worktree管理器
	Start(ctx context.Context) error

	// Stop 停止worktree管理器
	Stop(ctx context.Context) error
}

// WorktreeInfo Worktree信息
type WorktreeInfo struct {
	ID          string `json:"id"`
	ProjectPath string `json:"projectPath"`
	WSLPath     string `json:"wslPath"`
	Branch      string `json:"branch"`
	CreatedAt   string `json:"createdAt"`
	LastUsed    string `json:"lastUsed"`
	Status      string `json:"status"` // "active", "idle", "cleanup"
}
