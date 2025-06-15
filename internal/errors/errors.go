package errors

import (
	"fmt"

	"github.com/pkg/errors"
)

// ErrorCode 定义错误代码类型
type ErrorCode string

const (
	// 路径转换错误
	ErrInvalidPath    ErrorCode = "INVALID_PATH"
	ErrPathNotExists  ErrorCode = "PATH_NOT_EXISTS"
	ErrPathConversion ErrorCode = "PATH_CONVERSION_FAILED"

	// WSL 相关错误
	ErrWSLNotFound      ErrorCode = "WSL_NOT_FOUND"
	ErrDistroNotFound   ErrorCode = "DISTRO_NOT_FOUND"
	ErrWSLCommandFailed ErrorCode = "WSL_COMMAND_FAILED"

	// Claude Code 相关错误
	ErrClaudeCodeNotFound ErrorCode = "CLAUDE_CODE_NOT_FOUND"
	ErrClaudeCodeFailed   ErrorCode = "CLAUDE_CODE_FAILED"

	// 任务管理错误
	ErrTaskNotSupported ErrorCode = "TASK_NOT_SUPPORTED"
	ErrInstanceFailed   ErrorCode = "INSTANCE_FAILED"
	ErrGitOperation     ErrorCode = "GIT_OPERATION_FAILED"
	ErrTaskNotFound     ErrorCode = "TASK_NOT_FOUND"
	ErrTaskCancelled    ErrorCode = "TASK_CANCELLED"
	ErrTaskTimeout      ErrorCode = "TASK_TIMEOUT"
	ErrWorktreeNotFound ErrorCode = "WORKTREE_NOT_FOUND"
	ErrWorktreeFailed   ErrorCode = "WORKTREE_FAILED"

	// MCP 协议错误
	ErrMCPProtocolError ErrorCode = "MCP_PROTOCOL_ERROR"
	ErrMCPServerError   ErrorCode = "MCP_SERVER_ERROR"
	ErrMCPClientError   ErrorCode = "MCP_CLIENT_ERROR"

	// 配置错误
	ErrConfigInvalid  ErrorCode = "CONFIG_INVALID"
	ErrConfigNotFound ErrorCode = "CONFIG_NOT_FOUND"
)

// AppError 应用程序错误结构
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Cause   error     `json:"-"`
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 支持错误链
func (e *AppError) Unwrap() error {
	return e.Cause
}

// New 创建新的应用程序错误
func New(code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Newf 创建带格式化消息的应用程序错误
func Newf(code ErrorCode, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap 包装现有错误
func Wrap(err error, code ErrorCode, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// Wrapf 包装现有错误并格式化消息
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   err,
	}
}

// WithDetails 添加详细信息
func (e *AppError) WithDetails(details string) *AppError {
	e.Details = details
	return e
}

// WithDetailsf 添加格式化的详细信息
func (e *AppError) WithDetailsf(format string, args ...interface{}) *AppError {
	e.Details = fmt.Sprintf(format, args...)
	return e
}

// IsCode 检查错误是否为指定的错误代码
func IsCode(err error, code ErrorCode) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// GetCode 获取错误代码
func GetCode(err error) ErrorCode {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ""
}

// 预定义的常用错误
var (
	ErrWSLNotAvailable        = New(ErrWSLNotFound, "WSL 环境不可用，请确保已安装并启用 WSL")
	ErrInvalidWindowsPath     = New(ErrInvalidPath, "无效的 Windows 路径格式")
	ErrClaudeCodeNotInstalled = New(ErrClaudeCodeNotFound, "Claude Code 未安装或不在 PATH 中")
)
