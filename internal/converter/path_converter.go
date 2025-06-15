package converter

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	apperrors "auto-claude-code/internal/errors"
)

// PathConverter 路径转换器接口
type PathConverter interface {
	// ConvertToWSL 将 Windows 路径转换为 WSL 路径
	ConvertToWSL(windowsPath string) (string, error)

	// ConvertToWindows 将 WSL 路径转换为 Windows 路径
	ConvertToWindows(wslPath string) (string, error)

	// ValidatePath 验证路径有效性
	ValidatePath(path string) error

	// IsWindowsPath 检查是否为 Windows 路径
	IsWindowsPath(path string) bool

	// IsWSLPath 检查是否为 WSL 路径
	IsWSLPath(path string) bool
}

// pathConverter 路径转换器实现
type pathConverter struct {
	// Windows 路径正则表达式
	windowsPathRegex *regexp.Regexp
	// WSL 路径正则表达式
	wslPathRegex *regexp.Regexp
}

// NewPathConverter 创建新的路径转换器
func NewPathConverter() PathConverter {
	return &pathConverter{
		// Windows 路径格式：C:\path\to\file 或 C:/path/to/file
		windowsPathRegex: regexp.MustCompile(`^[A-Za-z]:[/\\].*`),
		// WSL 路径格式：/mnt/c/path/to/file
		wslPathRegex: regexp.MustCompile(`^/mnt/[a-z]/.*`),
	}
}

// ConvertToWSL 将 Windows 路径转换为 WSL 路径
func (pc *pathConverter) ConvertToWSL(windowsPath string) (string, error) {
	if windowsPath == "" {
		return "", apperrors.New(apperrors.ErrInvalidPath, "路径不能为空")
	}

	// 清理路径
	cleanPath := filepath.Clean(windowsPath)

	// 检查是否为有效的 Windows 路径
	if !pc.IsWindowsPath(cleanPath) {
		return "", apperrors.Newf(apperrors.ErrInvalidPath, "无效的 Windows 路径格式: %s", windowsPath)
	}

	// 提取盘符
	driveLetter := strings.ToLower(string(cleanPath[0]))

	// 获取路径部分（去掉盘符和冒号）
	pathPart := cleanPath[2:]

	// 将反斜杠转换为正斜杠
	pathPart = strings.ReplaceAll(pathPart, "\\", "/")

	// 构建 WSL 路径
	wslPath := "/mnt/" + driveLetter + pathPart

	return wslPath, nil
}

// ConvertToWindows 将 WSL 路径转换为 Windows 路径
func (pc *pathConverter) ConvertToWindows(wslPath string) (string, error) {
	if wslPath == "" {
		return "", apperrors.New(apperrors.ErrInvalidPath, "路径不能为空")
	}

	// 检查是否为有效的 WSL 路径
	if !pc.IsWSLPath(wslPath) {
		return "", apperrors.Newf(apperrors.ErrInvalidPath, "无效的 WSL 路径格式: %s", wslPath)
	}

	// 移除 /mnt/ 前缀
	pathWithoutMnt := strings.TrimPrefix(wslPath, "/mnt/")

	// 提取盘符
	if len(pathWithoutMnt) < 1 {
		return "", apperrors.Newf(apperrors.ErrInvalidPath, "WSL 路径缺少盘符: %s", wslPath)
	}

	driveLetter := strings.ToUpper(string(pathWithoutMnt[0]))

	// 获取路径部分
	var pathPart string
	if len(pathWithoutMnt) > 1 {
		pathPart = pathWithoutMnt[1:]
		// 将正斜杠转换为反斜杠
		pathPart = strings.ReplaceAll(pathPart, "/", "\\")
	}

	// 构建 Windows 路径
	windowsPath := driveLetter + ":" + pathPart

	return windowsPath, nil
}

// ValidatePath 验证路径有效性
func (pc *pathConverter) ValidatePath(path string) error {
	if path == "" {
		return apperrors.New(apperrors.ErrInvalidPath, "路径不能为空")
	}

	// 检查路径是否存在（仅对 Windows 路径进行检查）
	if pc.IsWindowsPath(path) {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return apperrors.Wrapf(err, apperrors.ErrPathNotExists, "路径不存在: %s", path)
		}
	}

	return nil
}

// IsWindowsPath 检查是否为 Windows 路径
func (pc *pathConverter) IsWindowsPath(path string) bool {
	return pc.windowsPathRegex.MatchString(path)
}

// IsWSLPath 检查是否为 WSL 路径
func (pc *pathConverter) IsWSLPath(path string) bool {
	return pc.wslPathRegex.MatchString(path)
}

// GetCurrentDirectory 获取当前工作目录
func GetCurrentDirectory() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", apperrors.Wrap(err, apperrors.ErrPathConversion, "无法获取当前工作目录")
	}
	return wd, nil
}

// NormalizePath 标准化路径格式
func NormalizePath(path string) string {
	// 清理路径
	cleanPath := filepath.Clean(path)

	// 将所有反斜杠转换为正斜杠（用于内部处理）
	normalizedPath := strings.ReplaceAll(cleanPath, "\\", "/")

	return normalizedPath
}

// EscapePathForShell 为 shell 命令转义路径
func EscapePathForShell(path string) string {
	// 如果路径包含空格，用引号包围
	if strings.Contains(path, " ") {
		return `"` + path + `"`
	}
	return path
}
