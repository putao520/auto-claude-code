# Auto Claude Code - 用户使用指南

## 快速开始

### 系统要求

- **操作系统**：Windows 10 1903+ 或 Windows 11
- **WSL**：已安装并配置 WSL2（推荐）或 WSL1
- **Claude Code**：已在 WSL 环境中安装 Claude Code

### 验证环境

在开始使用之前，请确保你的系统满足以下要求：

1. **检查 WSL 状态**
   ```powershell
   wsl --status
   ```

2. **列出已安装的 WSL 发行版**
   ```powershell
   wsl --list --verbose
   ```

3. **验证 Claude Code 安装**
   ```bash
   # 在 WSL 中执行
   wsl -- which claude-code
   ```

## 安装方法

### 方法 1：二进制下载（推荐）

1. 访问 [GitHub Releases](https://github.com/username/auto-claude-code/releases)
2. 下载最新版本的 `auto-claude-code.exe`
3. 将文件放到 PATH 环境变量包含的目录中，或直接使用

### 方法 2：包管理器安装

#### Chocolatey
```powershell
choco install auto-claude-code
```

#### Scoop
```powershell
scoop bucket add extras
scoop install auto-claude-code
```

#### Winget
```powershell
winget install auto-claude-code
```

### 方法 3：源码编译

```bash
# 克隆项目
git clone https://github.com/username/auto-claude-code.git
cd auto-claude-code

# 编译
go build -o auto-claude-code.exe ./cmd/auto-claude-code

# 可选：安装到系统
copy auto-claude-code.exe %GOPATH%\bin\
```

## 基本使用

### 最简使用

在任何 Windows 目录下打开 PowerShell 或 CMD，直接运行：

```powershell
auto-claude-code
```

程序会自动：
1. 获取当前 Windows 工作目录
2. 转换为对应的 WSL 路径
3. 在 WSL 中切换到该目录
4. 启动 Claude Code

### 指定 WSL 发行版

如果你有多个 WSL 发行版，可以指定使用哪一个：

```powershell
auto-claude-code --distro Ubuntu-20.04
```

### 调试模式

启用调试模式以查看详细的执行过程：

```powershell
auto-claude-code --debug
```

输出示例：
```
Windows Path: C:\Users\username\projects\myapp
WSL Path: /mnt/c/Users/username/projects/myapp
Using WSL distribution: Ubuntu-20.04
Starting Claude Code in WSL at: /mnt/c/Users/username/projects/myapp
```

## 高级配置

### 配置文件

Auto Claude Code 支持通过配置文件进行个性化设置。配置文件会按以下优先级搜索：

1. 当前目录的 `config.yaml`
2. `%USERPROFILE%\.auto-claude-code\config.yaml`
3. `C:\ProgramData\auto-claude-code\config.yaml`

### 创建配置文件

```powershell
# 创建默认配置
auto-claude-code config init
```

这会在用户目录下创建配置文件模板：

```yaml
# 默认 WSL 发行版
default_distro: "Ubuntu-20.04"

# 调试模式
debug: false

# 日志级别 (debug, info, warn, error)
log_level: "info"

# Claude Code 命令路径
claude_code_path: "claude-code"

# 超时时间（秒）
timeout: 30

# 自定义路径映射
path_mappings:
  "D:\\Projects": "/home/username/projects"
  "E:\\Data": "/mnt/data"
```

### 配置项说明

#### default_distro
- **类型**：字符串
- **默认值**：空（自动检测）
- **说明**：指定默认使用的 WSL 发行版名称

#### debug
- **类型**：布尔
- **默认值**：false
- **说明**：是否启用调试模式，显示详细执行信息

#### log_level
- **类型**：字符串
- **默认值**："info"
- **可选值**："debug", "info", "warn", "error"
- **说明**：日志输出级别

#### claude_code_path
- **类型**：字符串
- **默认值**："claude-code"
- **说明**：Claude Code 的命令名称或完整路径

#### timeout
- **类型**：整数
- **默认值**：30
- **说明**：命令执行超时时间（秒）

#### path_mappings
- **类型**：键值对映射
- **默认值**：空
- **说明**：自定义路径映射规则，优先级高于默认转换规则

### 环境变量

除了配置文件，还可以通过环境变量进行配置：

```powershell
# 设置默认发行版
$env:AUTO_CLAUDE_CODE_DISTRO = "Ubuntu-20.04"

# 启用调试模式
$env:AUTO_CLAUDE_CODE_DEBUG = "true"

# 设置日志级别
$env:AUTO_CLAUDE_CODE_LOG_LEVEL = "debug"
```

### 查看当前配置

```powershell
auto-claude-code config show
```

## 使用场景

### 场景 1：快速启动

当你在 Windows 文件资源管理器中浏览项目目录时，可以：

1. 在地址栏输入 `cmd` 或 `powershell`
2. 运行 `auto-claude-code`
3. 立即在对应的 WSL 路径中启动 Claude Code

### 场景 2：脚本集成

将 Auto Claude Code 集成到你的开发脚本中：

```powershell
# develop.ps1
cd C:\Projects\MyApp
auto-claude-code --distro Ubuntu-20.04
```

### 场景 3：多项目管理

为不同的项目设置不同的 WSL 发行版：

```powershell
# 项目 A 使用 Ubuntu
cd C:\Projects\ProjectA
auto-claude-code --distro Ubuntu-20.04

# 项目 B 使用 Debian  
cd C:\Projects\ProjectB
auto-claude-code --distro Debian
```

## 路径转换规则

### 基本规则

| Windows 路径 | WSL 路径 | 说明 |
|-------------|----------|------|
| `C:\Users\username` | `/mnt/c/Users/username` | 基本盘符映射 |
| `D:\Projects\app` | `/mnt/d/Projects/app` | 支持所有盘符 |
| `C:\Program Files\App` | `/mnt/c/Program Files/App` | 保留空格 |
| `C:\Users\用户名\项目` | `/mnt/c/Users/用户名/项目` | 支持中文路径 |

### 特殊路径处理

1. **UNC 网络路径**
   - `\\server\share\path` → `/mnt/wsl/server/share/path`

2. **相对路径**
   - 自动转换为绝对路径后处理

3. **路径规范化**
   - 处理 `.` 和 `..` 路径组件
   - 移除多余的路径分隔符

### 自定义映射

通过配置文件可以覆盖默认的转换规则：

```yaml
path_mappings:
  "C:\\workspace": "/home/username/workspace"
  "D:\\tools": "/opt/tools"
```

当 Windows 路径匹配到自定义映射的前缀时，会使用自定义规则进行转换。

## 故障排除

### 常见问题

#### 1. "WSL not available" 错误

**原因**：WSL 未安装或未启用

**解决方案**：
```powershell
# 启用 WSL 功能
dism.exe /online /enable-feature /featurename:Microsoft-Windows-Subsystem-Linux /all /norestart

# 启用虚拟机平台（WSL2 需要）
dism.exe /online /enable-feature /featurename:VirtualMachinePlatform /all /norestart

# 重启计算机后安装 WSL 发行版
wsl --install Ubuntu-20.04
```

#### 2. "claude-code not found" 错误

**原因**：Claude Code 未在 WSL 中安装或不在 PATH 中

**解决方案**：
```bash
# 在 WSL 中安装 Claude Code
# 具体安装方法请参考 Claude Code 官方文档
```

#### 3. 路径转换失败

**原因**：路径格式不符合 Windows 标准

**解决方案**：
- 确保使用绝对路径
- 检查路径中是否包含非法字符
- 使用调试模式查看详细错误信息

#### 4. 权限错误

**原因**：WSL 或目标目录权限不足

**解决方案**：
```bash
# 在 WSL 中修改目录权限
sudo chmod 755 /mnt/c/path/to/directory
```

### 调试步骤

1. **启用调试模式**
   ```powershell
   auto-claude-code --debug
   ```

2. **检查 WSL 状态**
   ```powershell
   wsl --status
   wsl --list --verbose
   ```

3. **验证路径转换**
   ```powershell
   # 手动测试路径转换
   echo "Current directory: $(Get-Location)"
   ```

4. **测试 WSL 连接**
   ```powershell
   wsl -- pwd
   wsl -- ls -la
   ```

### 日志分析

Auto Claude Code 的日志文件位置：
- Windows：`%USERPROFILE%\.auto-claude-code\logs\`
- 格式：JSON 结构化日志

查看日志：
```powershell
Get-Content "$env:USERPROFILE\.auto-claude-code\logs\auto-claude-code.log" | ConvertFrom-Json | Format-Table
```

## 性能优化

### 启动优化

1. **使用 SSD 存储**：将项目放在 SSD 上可以显著提升启动速度

2. **WSL2 优化**：确保使用 WSL2 而不是 WSL1

3. **缓存配置**：启用路径转换缓存（默认启用）

### 内存优化

Auto Claude Code 本身内存占用很小（< 10MB），但可以通过以下方式优化：

1. **关闭不必要的 WSL 发行版**
   ```powershell
   wsl --shutdown
   ```

2. **限制 WSL 内存使用**
   创建 `%USERPROFILE%\.wslconfig`：
   ```ini
   [wsl2]
   memory=4GB
   processors=2
   ```

## 更新和维护

### 检查更新

```powershell
auto-claude-code version
```

### 自动更新（未来功能）

```powershell
auto-claude-code update
```

### 卸载

1. **删除二进制文件**：移除 `auto-claude-code.exe`
2. **清理配置**：删除 `%USERPROFILE%\.auto-claude-code\` 目录
3. **清理环境变量**：移除相关环境变量

## 集成示例

### Visual Studio Code 集成

在 VS Code 的任务配置中添加：

```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Open Claude Code",
            "type": "shell",
            "command": "auto-claude-code",
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "new"
            }
        }
    ]
}
```

### PowerShell Profile 集成

在 PowerShell 配置文件中添加别名：

```powershell
# $PROFILE
function Start-ClaudeCode {
    auto-claude-code @args
}

Set-Alias cc Start-ClaudeCode
```

### Windows Terminal 集成

在 Windows Terminal 的设置中添加快捷操作：

```json
{
    "actions": [
        {
            "command": {
                "action": "sendInput",
                "input": "auto-claude-code\r"
            },
            "keys": "ctrl+alt+c"
        }
    ]
}
```

## 获取帮助

### 命令行帮助

```powershell
auto-claude-code --help
auto-claude-code config --help
```

### 社区支持

- **GitHub Issues**：[问题报告和功能请求](https://github.com/username/auto-claude-code/issues)
- **讨论区**：[GitHub Discussions](https://github.com/username/auto-claude-code/discussions)
- **文档**：[完整文档](https://github.com/username/auto-claude-code/docs)

### 贡献

欢迎提交 Pull Request 或报告问题！请参考 [贡献指南](CONTRIBUTING.md) 了解详细信息。 