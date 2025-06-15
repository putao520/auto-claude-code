package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"auto-claude-code/internal/config"
	apperrors "auto-claude-code/internal/errors"
	"auto-claude-code/internal/logger"
)

// worktreeManager Git worktree管理器实现
type worktreeManager struct {
	config    *config.MCPConfig
	logger    logger.Logger
	baseDir   string
	worktrees map[string]*WorktreeInfo
	mutex     sync.RWMutex

	// 生命周期管理
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewWorktreeManager 创建新的worktree管理器
func NewWorktreeManager(cfg *config.MCPConfig, log logger.Logger) WorktreeManager {
	baseDir := cfg.WorktreeBaseDir
	if baseDir == "" {
		baseDir = "./worktrees"
	}

	return &worktreeManager{
		config:    cfg,
		logger:    log,
		baseDir:   baseDir,
		worktrees: make(map[string]*WorktreeInfo),
	}
}

// Start 启动worktree管理器
func (wm *worktreeManager) Start(ctx context.Context) error {
	wm.ctx, wm.cancel = context.WithCancel(ctx)

	wm.logger.Info("启动Worktree管理器",
		zap.String("baseDir", wm.baseDir),
		zap.Int("maxWorktrees", wm.config.MaxWorktrees))

	// 确保基础目录存在
	if err := os.MkdirAll(wm.baseDir, 0755); err != nil {
		return apperrors.Wrap(err, apperrors.ErrWorktreeFailed, "无法创建worktree基础目录")
	}

	// 扫描现有的worktrees
	if err := wm.scanExistingWorktrees(); err != nil {
		wm.logger.Warn("扫描现有worktrees失败", zap.Error(err))
	}

	// 启动清理器
	if cleanupInterval, err := time.ParseDuration(wm.config.CleanupInterval); err == nil {
		wm.wg.Add(1)
		go wm.runCleaner(cleanupInterval)
	}

	return nil
}

// Stop 停止worktree管理器
func (wm *worktreeManager) Stop(ctx context.Context) error {
	wm.logger.Info("停止Worktree管理器")

	if wm.cancel != nil {
		wm.cancel()
	}

	// 等待清理器停止
	done := make(chan struct{})
	go func() {
		wm.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		wm.logger.Info("Worktree管理器已停止")
	case <-ctx.Done():
		wm.logger.Warn("Worktree管理器停止超时")
		return ctx.Err()
	}

	return nil
}

// CreateWorktree 创建新的worktree
func (wm *worktreeManager) CreateWorktree(ctx context.Context, projectPath string) (*WorktreeInfo, error) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	// 检查worktree数量限制
	if len(wm.worktrees) >= wm.config.MaxWorktrees {
		// 尝试清理空闲的worktrees
		if err := wm.cleanupIdleWorktrees(); err != nil {
			wm.logger.Warn("清理空闲worktrees失败", zap.Error(err))
		}

		// 再次检查
		if len(wm.worktrees) >= wm.config.MaxWorktrees {
			return nil, apperrors.New(apperrors.ErrWorktreeFailed, "已达到最大worktree数量限制")
		}
	}

	// 生成worktree ID
	worktreeID := fmt.Sprintf("wt_%d", time.Now().UnixNano())
	worktreePath := filepath.Join(wm.baseDir, worktreeID)

	wm.logger.Info("创建新的worktree",
		zap.String("worktreeId", worktreeID),
		zap.String("projectPath", projectPath),
		zap.String("worktreePath", worktreePath))

	// 检查项目是否为Git仓库
	if !wm.isGitRepository(projectPath) {
		// 如果不是Git仓库，直接复制目录
		if err := wm.copyDirectory(projectPath, worktreePath); err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrWorktreeFailed, "复制项目目录失败")
		}
	} else {
		// 创建Git worktree
		if err := wm.createGitWorktree(ctx, projectPath, worktreePath); err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrWorktreeFailed, "创建Git worktree失败")
		}
	}

	// 创建worktree信息
	worktree := &WorktreeInfo{
		ID:          worktreeID,
		ProjectPath: projectPath,
		WSLPath:     "/mnt/" + strings.ToLower(string(worktreePath[0])) + strings.ReplaceAll(worktreePath[2:], "\\", "/"),
		Branch:      "main", // 默认分支
		CreatedAt:   time.Now().Format(time.RFC3339),
		LastUsed:    time.Now().Format(time.RFC3339),
		Status:      "active",
	}

	// 如果是Git仓库，获取当前分支
	if wm.isGitRepository(projectPath) {
		if branch, err := wm.getCurrentBranch(projectPath); err == nil {
			worktree.Branch = branch
		}
	}

	// 保存worktree信息
	wm.worktrees[worktreeID] = worktree

	wm.logger.Info("Worktree创建成功",
		zap.String("worktreeId", worktreeID),
		zap.String("branch", worktree.Branch))

	return worktree, nil
}

// DeleteWorktree 删除worktree
func (wm *worktreeManager) DeleteWorktree(ctx context.Context, worktreeID string) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	worktree, exists := wm.worktrees[worktreeID]
	if !exists {
		return apperrors.Newf(apperrors.ErrWorktreeNotFound, "Worktree不存在: %s", worktreeID)
	}

	wm.logger.Info("删除worktree", zap.String("worktreeId", worktreeID))

	worktreePath := filepath.Join(wm.baseDir, worktreeID)

	// 如果是Git worktree，使用git worktree remove
	if wm.isGitRepository(worktree.ProjectPath) {
		if err := wm.removeGitWorktree(ctx, worktree.ProjectPath, worktreePath); err != nil {
			wm.logger.Warn("Git worktree删除失败，尝试直接删除目录", zap.Error(err))
		}
	}

	// 删除目录
	if err := os.RemoveAll(worktreePath); err != nil {
		return apperrors.Wrap(err, apperrors.ErrWorktreeFailed, "删除worktree目录失败")
	}

	// 从映射中删除
	delete(wm.worktrees, worktreeID)

	wm.logger.Info("Worktree删除成功", zap.String("worktreeId", worktreeID))
	return nil
}

// GetWorktree 获取worktree信息
func (wm *worktreeManager) GetWorktree(ctx context.Context, worktreeID string) (*WorktreeInfo, error) {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	worktree, exists := wm.worktrees[worktreeID]
	if !exists {
		return nil, apperrors.Newf(apperrors.ErrWorktreeNotFound, "Worktree不存在: %s", worktreeID)
	}

	// 更新最后使用时间
	worktree.LastUsed = time.Now().Format(time.RFC3339)

	// 返回副本
	worktreeCopy := *worktree
	return &worktreeCopy, nil
}

// ListWorktrees 列出所有worktrees
func (wm *worktreeManager) ListWorktrees(ctx context.Context) ([]*WorktreeInfo, error) {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	worktrees := make([]*WorktreeInfo, 0, len(wm.worktrees))
	for _, worktree := range wm.worktrees {
		worktreeCopy := *worktree
		worktrees = append(worktrees, &worktreeCopy)
	}

	return worktrees, nil
}

// CleanupWorktrees 清理过期的worktrees
func (wm *worktreeManager) CleanupWorktrees(ctx context.Context) error {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()

	return wm.cleanupIdleWorktrees()
}

// HealthCheck 健康检查
func (wm *worktreeManager) HealthCheck(ctx context.Context) error {
	// 检查基础目录是否存在
	if _, err := os.Stat(wm.baseDir); os.IsNotExist(err) {
		return apperrors.Wrap(err, apperrors.ErrWorktreeFailed, "Worktree基础目录不存在")
	}

	// 检查worktree数量
	wm.mutex.RLock()
	worktreeCount := len(wm.worktrees)
	wm.mutex.RUnlock()

	wm.logger.Debug("Worktree管理器健康检查通过",
		zap.Int("worktreeCount", worktreeCount),
		zap.Int("maxWorktrees", wm.config.MaxWorktrees))

	return nil
}

// isGitRepository 检查是否为Git仓库
func (wm *worktreeManager) isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return true
	}
	return false
}

// createGitWorktree 创建Git worktree
func (wm *worktreeManager) createGitWorktree(ctx context.Context, projectPath, worktreePath string) error {
	// 获取当前分支
	branch, err := wm.getCurrentBranch(projectPath)
	if err != nil {
		branch = "main" // 默认分支
	}

	// 创建唯一的分支名
	uniqueBranch := fmt.Sprintf("worktree_%d", time.Now().UnixNano())

	// 在项目目录中执行git worktree add
	cmd := exec.CommandContext(ctx, "git", "worktree", "add", "-b", uniqueBranch, worktreePath, branch)
	cmd.Dir = projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return apperrors.Wrapf(err, apperrors.ErrGitOperation, "Git worktree创建失败: %s", string(output))
	}

	wm.logger.Debug("Git worktree创建成功",
		zap.String("projectPath", projectPath),
		zap.String("worktreePath", worktreePath),
		zap.String("branch", uniqueBranch))

	return nil
}

// removeGitWorktree 删除Git worktree
func (wm *worktreeManager) removeGitWorktree(ctx context.Context, projectPath, worktreePath string) error {
	cmd := exec.CommandContext(ctx, "git", "worktree", "remove", worktreePath, "--force")
	cmd.Dir = projectPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return apperrors.Wrapf(err, apperrors.ErrGitOperation, "Git worktree删除失败: %s", string(output))
	}

	return nil
}

// getCurrentBranch 获取当前分支
func (wm *worktreeManager) getCurrentBranch(projectPath string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = projectPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "main", nil
	}

	return branch, nil
}

// copyDirectory 复制目录（用于非Git项目）
func (wm *worktreeManager) copyDirectory(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算目标路径
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		// 跳过.git目录
		if strings.Contains(relPath, ".git") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// 复制文件
		return wm.copyFile(path, dstPath)
	})
}

// copyFile 复制文件
func (wm *worktreeManager) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// 复制内容
	_, err = srcFile.WriteTo(dstFile)
	return err
}

// scanExistingWorktrees 扫描现有的worktrees
func (wm *worktreeManager) scanExistingWorktrees() error {
	entries, err := os.ReadDir(wm.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 目录不存在，没有现有的worktrees
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "wt_") {
			worktreeID := entry.Name()

			// 创建worktree信息（基本信息）
			info, err := entry.Info()
			if err != nil {
				continue
			}

			worktree := &WorktreeInfo{
				ID:        worktreeID,
				CreatedAt: info.ModTime().Format(time.RFC3339),
				LastUsed:  info.ModTime().Format(time.RFC3339),
				Status:    "idle",
			}

			wm.worktrees[worktreeID] = worktree
		}
	}

	wm.logger.Info("扫描到现有worktrees", zap.Int("count", len(wm.worktrees)))
	return nil
}

// cleanupIdleWorktrees 清理空闲的worktrees
func (wm *worktreeManager) cleanupIdleWorktrees() error {
	cutoff := time.Now().Add(-2 * time.Hour) // 2小时未使用的worktrees

	var toDelete []string
	for worktreeID, worktree := range wm.worktrees {
		if worktree.Status == "idle" {
			if lastUsed, err := time.Parse(time.RFC3339, worktree.LastUsed); err == nil {
				if lastUsed.Before(cutoff) {
					toDelete = append(toDelete, worktreeID)
				}
			}
		}
	}

	// 删除空闲的worktrees
	for _, worktreeID := range toDelete {
		worktreePath := filepath.Join(wm.baseDir, worktreeID)
		if err := os.RemoveAll(worktreePath); err != nil {
			wm.logger.Warn("删除空闲worktree失败",
				zap.String("worktreeId", worktreeID),
				zap.Error(err))
			continue
		}
		delete(wm.worktrees, worktreeID)
	}

	if len(toDelete) > 0 {
		wm.logger.Info("清理空闲worktrees", zap.Int("count", len(toDelete)))
	}

	return nil
}

// runCleaner 运行清理器
func (wm *worktreeManager) runCleaner(interval time.Duration) {
	defer wm.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-wm.ctx.Done():
			return
		case <-ticker.C:
			wm.CleanupWorktrees(wm.ctx)
		}
	}
}
