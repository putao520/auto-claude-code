package converter

import (
	"testing"

	apperrors "auto-claude-code/internal/errors"
)

func TestPathConverter_ConvertToWSL(t *testing.T) {
	pc := NewPathConverter()

	tests := []struct {
		name        string
		windowsPath string
		expected    string
		expectError bool
	}{
		{
			name:        "C盘路径转换",
			windowsPath: "C:\\Users\\test",
			expected:    "/mnt/c/Users/test",
			expectError: false,
		},
		{
			name:        "D盘路径转换",
			windowsPath: "D:\\Projects\\app",
			expected:    "/mnt/d/Projects/app",
			expectError: false,
		},
		{
			name:        "正斜杠路径转换",
			windowsPath: "C:/Users/test",
			expected:    "/mnt/c/Users/test",
			expectError: false,
		},
		{
			name:        "带空格的路径",
			windowsPath: "C:\\Program Files\\test",
			expected:    "/mnt/c/Program Files/test",
			expectError: false,
		},
		{
			name:        "空路径",
			windowsPath: "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "无效路径格式",
			windowsPath: "invalid/path",
			expected:    "",
			expectError: true,
		},
		{
			name:        "相对路径",
			windowsPath: "./test",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pc.ConvertToWSL(tt.windowsPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误但没有返回错误")
				}
				return
			}

			if err != nil {
				t.Errorf("意外的错误: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("期望 %s，但得到 %s", tt.expected, result)
			}
		})
	}
}

func TestPathConverter_ConvertToWindows(t *testing.T) {
	pc := NewPathConverter()

	tests := []struct {
		name        string
		wslPath     string
		expected    string
		expectError bool
	}{
		{
			name:        "C盘WSL路径转换",
			wslPath:     "/mnt/c/Users/test",
			expected:    "C:\\Users\\test",
			expectError: false,
		},
		{
			name:        "D盘WSL路径转换",
			wslPath:     "/mnt/d/Projects/app",
			expected:    "D:\\Projects\\app",
			expectError: false,
		},
		{
			name:        "根目录路径",
			wslPath:     "/mnt/c",
			expected:    "C:",
			expectError: false,
		},
		{
			name:        "空路径",
			wslPath:     "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "无效WSL路径",
			wslPath:     "/home/user",
			expected:    "",
			expectError: true,
		},
		{
			name:        "缺少盘符的路径",
			wslPath:     "/mnt/",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pc.ConvertToWindows(tt.wslPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("期望错误但没有返回错误")
				}
				return
			}

			if err != nil {
				t.Errorf("意外的错误: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("期望 %s，但得到 %s", tt.expected, result)
			}
		})
	}
}

func TestPathConverter_IsWindowsPath(t *testing.T) {
	pc := NewPathConverter()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "标准Windows路径",
			path:     "C:\\Users\\test",
			expected: true,
		},
		{
			name:     "正斜杠Windows路径",
			path:     "C:/Users/test",
			expected: true,
		},
		{
			name:     "D盘路径",
			path:     "D:\\Projects",
			expected: true,
		},
		{
			name:     "WSL路径",
			path:     "/mnt/c/Users/test",
			expected: false,
		},
		{
			name:     "Linux路径",
			path:     "/home/user",
			expected: false,
		},
		{
			name:     "相对路径",
			path:     "./test",
			expected: false,
		},
		{
			name:     "空路径",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pc.IsWindowsPath(tt.path)
			if result != tt.expected {
				t.Errorf("期望 %v，但得到 %v", tt.expected, result)
			}
		})
	}
}

func TestPathConverter_IsWSLPath(t *testing.T) {
	pc := NewPathConverter()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "标准WSL路径",
			path:     "/mnt/c/Users/test",
			expected: true,
		},
		{
			name:     "D盘WSL路径",
			path:     "/mnt/d/Projects",
			expected: true,
		},
		{
			name:     "Windows路径",
			path:     "C:\\Users\\test",
			expected: false,
		},
		{
			name:     "Linux路径",
			path:     "/home/user",
			expected: false,
		},
		{
			name:     "根路径",
			path:     "/",
			expected: false,
		},
		{
			name:     "空路径",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pc.IsWSLPath(tt.path)
			if result != tt.expected {
				t.Errorf("期望 %v，但得到 %v", tt.expected, result)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "反斜杠转换",
			path:     "C:\\Users\\test",
			expected: "C:/Users/test",
		},
		{
			name:     "混合斜杠",
			path:     "C:\\Users/test\\file",
			expected: "C:/Users/test/file",
		},
		{
			name:     "已经是正斜杠",
			path:     "C:/Users/test",
			expected: "C:/Users/test",
		},
		{
			name:     "相对路径",
			path:     "./test/../file",
			expected: "file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.path)
			if result != tt.expected {
				t.Errorf("期望 %s，但得到 %s", tt.expected, result)
			}
		})
	}
}

func TestEscapePathForShell(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "无空格路径",
			path:     "/mnt/c/Users/test",
			expected: "/mnt/c/Users/test",
		},
		{
			name:     "带空格路径",
			path:     "/mnt/c/Program Files/test",
			expected: "\"/mnt/c/Program Files/test\"",
		},
		{
			name:     "空路径",
			path:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapePathForShell(tt.path)
			if result != tt.expected {
				t.Errorf("期望 %s，但得到 %s", tt.expected, result)
			}
		})
	}
}

// 测试错误类型
func TestPathConverter_ErrorTypes(t *testing.T) {
	pc := NewPathConverter()

	// 测试空路径错误
	_, err := pc.ConvertToWSL("")
	if !apperrors.IsCode(err, apperrors.ErrInvalidPath) {
		t.Errorf("期望 ErrInvalidPath 错误，但得到 %v", err)
	}

	// 测试无效路径格式错误
	_, err = pc.ConvertToWSL("invalid")
	if !apperrors.IsCode(err, apperrors.ErrInvalidPath) {
		t.Errorf("期望 ErrInvalidPath 错误，但得到 %v", err)
	}

	// 测试WSL路径转换错误
	_, err = pc.ConvertToWindows("/invalid/path")
	if !apperrors.IsCode(err, apperrors.ErrInvalidPath) {
		t.Errorf("期望 ErrInvalidPath 错误，但得到 %v", err)
	}
}
