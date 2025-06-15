package wsl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"unicode"
	"unicode/utf16"

	apperrors "auto-claude-code/internal/errors"

	"go.uber.org/zap"
)

// WSLBridge WSL 桥接器接口
type WSLBridge interface {
	// CheckWSL 检查 WSL 环境是否可用
	CheckWSL() error

	// ListDistros 列出可用的 WSL 发行版
	ListDistros() ([]string, error)

	// GetDefaultDistro 获取默认的 WSL 发行版
	GetDefaultDistro() (string, error)

	// ExecuteCommand 在 WSL 中执行命令
	ExecuteCommand(distro, command string) error

	// ExecuteCommandWithOutput 在 WSL 中执行命令并返回输出
	ExecuteCommandWithOutput(distro, command string) (string, error)

	// StartClaudeCode 启动 Claude Code
	StartClaudeCode(distro, workingDir string, args []string) error

	// CheckClaudeCode 检查 Claude Code 是否可用
	CheckClaudeCode(distro string) error
}

// wslBridge WSL 桥接器实现
type wslBridge struct {
	logger *zap.Logger
}

// NewWSLBridge 创建新的 WSL 桥接器
func NewWSLBridge(logger *zap.Logger) WSLBridge {
	return &wslBridge{
		logger: logger,
	}
}

// CheckWSL 检查 WSL 环境是否可用
func (wb *wslBridge) CheckWSL() error {
	wb.logger.Debug("检查 WSL 环境")

	// 检查 wsl.exe 是否存在
	_, err := exec.LookPath("wsl")
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrWSLNotFound, "WSL 命令不可用")
	}

	// 尝试执行简单的 WSL 命令
	cmd := exec.Command("wsl", "--status")
	if err := cmd.Run(); err != nil {
		return apperrors.Wrap(err, apperrors.ErrWSLNotFound, "WSL 服务不可用")
	}

	wb.logger.Debug("WSL 环境检查通过")
	return nil
}

// cleanWSLOutput 清理 WSL 命令的输出，正确处理 UTF-16LE 编码
func cleanWSLOutput(output []byte) string {
	if len(output) == 0 {
		return ""
	}

	// 检查是否是 UTF-16LE 编码（Windows WSL 的默认输出格式）
	// UTF-16LE 的特征：字符串长度为偶数，且奇数位置多为 0x00
	isUTF16LE := len(output)%2 == 0
	if isUTF16LE && len(output) >= 4 {
		// 检查前几个字节是否符合 UTF-16LE 模式
		nullCount := 0
		for i := 1; i < len(output) && i < 20; i += 2 {
			if output[i] == 0x00 {
				nullCount++
			}
		}
		isUTF16LE = nullCount > 0
	}

	var result string

	if isUTF16LE {
		// 转换 UTF-16LE 到 UTF-8
		utf16Data := make([]uint16, len(output)/2)
		for i := 0; i < len(output); i += 2 {
			if i+1 < len(output) {
				utf16Data[i/2] = uint16(output[i]) | uint16(output[i+1])<<8
			}
		}

		// 移除 UTF-16 BOM（如果存在）
		if len(utf16Data) > 0 && utf16Data[0] == 0xfeff {
			utf16Data = utf16Data[1:]
		}

		// 解码为字符串
		result = string(utf16.Decode(utf16Data))
	} else {
		// 当作 UTF-8 处理
		result = string(output)
		// 移除 UTF-8 BOM
		result = strings.TrimPrefix(result, "\ufeff")
	}

	// 清理结果字符串
	var cleaned strings.Builder
	for _, r := range result {
		// 跳过控制字符，但保留换行、回车、制表符
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			continue
		}
		// 跳过非打印字符（除了空格和常见空白字符）
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			continue
		}
		cleaned.WriteRune(r)
	}

	result = cleaned.String()
	return strings.TrimSpace(result)
}

// ListDistros 列出可用的 WSL 发行版
func (wb *wslBridge) ListDistros() ([]string, error) {
	wb.logger.Debug("列出 WSL 发行版")

	cmd := exec.Command("wsl", "--list", "--quiet")
	output, err := cmd.Output()
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrWSLCommandFailed, "无法列出 WSL 发行版")
	}

	// 清理输出
	cleanedOutput := cleanWSLOutput(output)

	// 解析输出
	lines := strings.Split(cleanedOutput, "\n")
	var distros []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			distros = append(distros, line)
		}
	}

	wb.logger.Debug("找到 WSL 发行版", zap.Strings("distros", distros))
	return distros, nil
}

// GetDefaultDistro 获取默认的 WSL 发行版
func (wb *wslBridge) GetDefaultDistro() (string, error) {
	wb.logger.Debug("获取默认 WSL 发行版")

	cmd := exec.Command("wsl", "--list", "--verbose")
	output, err := cmd.Output()
	if err != nil {
		return "", apperrors.Wrap(err, apperrors.ErrWSLCommandFailed, "无法获取默认 WSL 发行版")
	}

	// 清理输出
	cleanedOutput := cleanWSLOutput(output)

	lines := strings.Split(cleanedOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "*") {
			// 提取发行版名称（移除 * 和状态信息）
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// 第一个字段是 "*"，第二个字段是发行版名称
				distro := parts[1]
				distro = strings.TrimSpace(distro)

				if distro != "" {
					wb.logger.Debug("找到默认发行版", zap.String("distro", distro))
					return distro, nil
				}
			} else if len(parts) == 1 {
				// 可能是 "*Ubuntu" 这种格式
				distro := strings.Trim(parts[0], "*")
				distro = strings.TrimSpace(distro)

				if distro != "" {
					wb.logger.Debug("找到默认发行版", zap.String("distro", distro))
					return distro, nil
				}
			}
		}
	}

	// 如果没有找到默认发行版，返回第一个可用的
	distros, err := wb.ListDistros()
	if err != nil {
		return "", err
	}

	if len(distros) == 0 {
		return "", apperrors.New(apperrors.ErrDistroNotFound, "没有找到可用的 WSL 发行版")
	}

	defaultDistro := distros[0]
	wb.logger.Debug("使用第一个可用发行版作为默认", zap.String("distro", defaultDistro))
	return defaultDistro, nil
}

// ExecuteCommand 在 WSL 中执行命令
func (wb *wslBridge) ExecuteCommand(distro, command string) error {
	wb.logger.Debug("在 WSL 中执行命令",
		zap.String("distro", distro),
		zap.String("command", command))

	var cmd *exec.Cmd
	if distro != "" {
		cmd = exec.Command("wsl", "-d", distro, "bash", "-l", "-c", command)
	} else {
		cmd = exec.Command("wsl", "bash", "-l", "-c", command)
	}

	// 连接标准输入输出
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return apperrors.Wrapf(err, apperrors.ErrWSLCommandFailed, "WSL 命令执行失败: %s", command)
	}

	return nil
}

// ExecuteCommandWithOutput 在 WSL 中执行命令并返回输出
func (wb *wslBridge) ExecuteCommandWithOutput(distro, command string) (string, error) {
	wb.logger.Debug("在 WSL 中执行命令并获取输出",
		zap.String("distro", distro),
		zap.String("command", command))

	var cmd *exec.Cmd
	if distro != "" {
		cmd = exec.Command("wsl", "-d", distro, "bash", "-l", "-c", command)
	} else {
		cmd = exec.Command("wsl", "bash", "-l", "-c", command)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", apperrors.Wrapf(err, apperrors.ErrWSLCommandFailed, "WSL 命令执行失败: %s", command)
	}

	// 清理输出
	cleanedOutput := cleanWSLOutput(output)
	return cleanedOutput, nil
}

// StartClaudeCode 启动 Claude Code
func (wb *wslBridge) StartClaudeCode(distro, workingDir string, args []string) error {
	wb.logger.Info("启动 Claude Code",
		zap.String("distro", distro),
		zap.String("workingDir", workingDir),
		zap.Strings("args", args))

	// 首先检查 Claude Code 是否可用
	if err := wb.CheckClaudeCode(distro); err != nil {
		return err
	}

	// 构建命令
	claudeArgs := []string{"claude-code"}
	claudeArgs = append(claudeArgs, args...)

	// 构建完整的命令字符串
	command := fmt.Sprintf("cd %s && %s",
		escapeShellArg(workingDir),
		strings.Join(claudeArgs, " "))

	wb.logger.Debug("执行 Claude Code 命令", zap.String("command", command))

	// 创建命令
	var cmd *exec.Cmd
	if distro != "" {
		cmd = exec.Command("wsl", "-d", distro, "bash", "-l", "-c", command)
	} else {
		cmd = exec.Command("wsl", "bash", "-l", "-c", command)
	}

	// 设置环境变量
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// 连接标准输入输出，实现 stdio 转发
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 启动命令
	if err := cmd.Start(); err != nil {
		return apperrors.Wrapf(err, apperrors.ErrClaudeCodeFailed, "Claude Code 启动失败")
	}

	wb.logger.Info("Claude Code 已启动", zap.Int("pid", cmd.Process.Pid))

	// 等待命令完成
	if err := cmd.Wait(); err != nil {
		// 如果是用户主动退出，不视为错误
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 130 { // Ctrl+C
				wb.logger.Info("Claude Code 被用户中断")
				return nil
			}
		}
		return apperrors.Wrapf(err, apperrors.ErrClaudeCodeFailed, "Claude Code 执行失败")
	}

	wb.logger.Info("Claude Code 执行完成")
	return nil
}

// CheckClaudeCode 检查 Claude Code 是否可用
func (wb *wslBridge) CheckClaudeCode(distro string) error {
	wb.logger.Debug("检查 Claude Code 可用性", zap.String("distro", distro))

	// 首先检查 claude-code 命令是否存在
	output, err := wb.ExecuteCommandWithOutput(distro, "which claude-code")
	if err != nil || output == "" {
		// 尝试检查常见的安装位置
		commonPaths := []string{
			"~/.local/bin/claude-code",
			"/usr/local/bin/claude-code",
			"/usr/bin/claude-code",
			"~/bin/claude-code",
		}

		for _, path := range commonPaths {
			checkCmd := fmt.Sprintf("test -x %s && echo 'found'", path)
			if result, err := wb.ExecuteCommandWithOutput(distro, checkCmd); err == nil && result == "found" {
				wb.logger.Debug("在非标准位置找到 Claude Code", zap.String("path", path))
				return apperrors.New(apperrors.ErrClaudeCodeNotFound,
					fmt.Sprintf("Claude Code 已安装在 %s 但不在 PATH 中，请将其添加到 PATH", path))
			}
		}

		return apperrors.New(apperrors.ErrClaudeCodeNotFound,
			"Claude Code 未安装或不在 PATH 中，请在 WSL 中安装 Claude Code")
	}

	wb.logger.Debug("Claude Code 已找到", zap.String("path", output))

	// 尝试获取版本信息来验证是否正常工作
	versionOutput, err := wb.ExecuteCommandWithOutput(distro, "claude-code --version 2>/dev/null || echo 'auth_required'")
	if err != nil {
		wb.logger.Warn("无法获取 Claude Code 版本信息", zap.Error(err))
		return apperrors.New(apperrors.ErrClaudeCodeNotFound,
			"Claude Code 已安装但无法执行，可能需要登录或配置")
	}

	if strings.Contains(versionOutput, "auth_required") || strings.Contains(versionOutput, "login") || strings.Contains(versionOutput, "authentication") {
		wb.logger.Info("Claude Code 需要登录")
		return apperrors.New(apperrors.ErrClaudeCodeNotFound,
			"Claude Code 已安装但需要登录，请先运行: claude-code auth login")
	}

	wb.logger.Debug("Claude Code 版本", zap.String("version", versionOutput))
	return nil
}

// StartClaudeCodeInteractive 启动交互式 Claude Code（带实时输出）
func (wb *wslBridge) StartClaudeCodeInteractive(distro, workingDir string, args []string) error {
	wb.logger.Info("启动交互式 Claude Code",
		zap.String("distro", distro),
		zap.String("workingDir", workingDir))

	// 检查 Claude Code 是否可用
	if err := wb.CheckClaudeCode(distro); err != nil {
		return err
	}

	// 构建命令
	claudeArgs := []string{"claude-code"}
	claudeArgs = append(claudeArgs, args...)

	command := fmt.Sprintf("cd %s && %s",
		escapeShellArg(workingDir),
		strings.Join(claudeArgs, " "))

	// 创建命令
	var cmd *exec.Cmd
	if distro != "" {
		cmd = exec.Command("wsl", "-d", distro, "bash", "-l", "-c", command)
	} else {
		cmd = exec.Command("wsl", "bash", "-l", "-c", command)
	}

	// 创建管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrClaudeCodeFailed, "无法创建输出管道")
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrClaudeCodeFailed, "无法创建错误管道")
	}

	cmd.Stdin = os.Stdin

	// 启动命令
	if err := cmd.Start(); err != nil {
		return apperrors.Wrapf(err, apperrors.ErrClaudeCodeFailed, "Claude Code 启动失败")
	}

	// 创建上下文用于取消
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动输出读取 goroutines
	go wb.streamOutput(ctx, stdout, os.Stdout, "stdout")
	go wb.streamOutput(ctx, stderr, os.Stderr, "stderr")

	// 等待命令完成
	err = cmd.Wait()
	cancel() // 取消输出流

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if exitError.ExitCode() == 130 {
				wb.logger.Info("Claude Code 被用户中断")
				return nil
			}
		}
		return apperrors.Wrapf(err, apperrors.ErrClaudeCodeFailed, "Claude Code 执行失败")
	}

	return nil
}

// streamOutput 流式输出处理
func (wb *wslBridge) streamOutput(ctx context.Context, src io.Reader, dst io.Writer, streamType string) {
	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
			line := scanner.Text()
			fmt.Fprintln(dst, line)
			wb.logger.Debug("输出流", zap.String("type", streamType), zap.String("line", line))
		}
	}

	if err := scanner.Err(); err != nil {
		wb.logger.Error("读取输出流失败", zap.String("type", streamType), zap.Error(err))
	}
}

// escapeShellArg 转义 shell 参数
func escapeShellArg(arg string) string {
	if strings.Contains(arg, " ") || strings.Contains(arg, "'") || strings.Contains(arg, "\"") {
		// 使用单引号包围，并转义内部的单引号
		escaped := strings.ReplaceAll(arg, "'", "'\"'\"'")
		return "'" + escaped + "'"
	}
	return arg
}

// GetWSLVersion 获取 WSL 版本信息
func (wb *wslBridge) GetWSLVersion() (string, error) {
	cmd := exec.Command("wsl", "--version")
	output, err := cmd.Output()
	if err != nil {
		// 如果 --version 不支持，尝试旧的方式
		cmd = exec.Command("wsl", "--help")
		output, err = cmd.Output()
		if err != nil {
			return "", apperrors.Wrap(err, apperrors.ErrWSLCommandFailed, "无法获取 WSL 版本信息")
		}
		return "WSL 1.x", nil
	}

	return strings.TrimSpace(string(output)), nil
}
