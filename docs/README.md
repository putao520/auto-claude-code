# Auto Claude Code - Windows to WSL Directory Bridge + MCP 任务分发系统

## 项目概述

Auto Claude Code 是一个用 Go 开发的智能编程助手，具有两大核心功能：

### 🔧 **核心功能 1：Windows to WSL 路径桥接**
专为 Windows + WSL 开发环境设计的轻量级代理工具。它能够智能地将 Windows 系统的当前工作目录转换为 WSL 内的挂载路径，并在 WSL 环境中启动 Claude Code，实现跨系统无缝的 AI 编程体验。

### 🚀 **核心功能 2：MCP 任务分发系统**
支持 MCP (Model Context Protocol) 的智能任务分发系统。主编程 AI 可以通过我们的 MCP 服务器分发专业编程任务给多个 Claude Code 实例，实现异步执行和结果聚合。

## 核心价值主张

### 🎯 解决的核心问题
- **环境割裂**：Windows 主机与 WSL 开发环境之间的切换摩擦
- **路径复杂**：Windows 路径到 WSL 挂载路径的手动转换繁琐
- **任务分工**：主 AI 与 Claude Code 之间缺乏明确的任务分工机制
- **并发执行**：无法同时运行多个独立的编程任务

### ✨ 提供的核心价值
- **一键启动**：在任意 Windows 目录下直接启动对应 WSL 路径的 Claude Code
- **智能转换**：自动处理 Windows 路径到 WSL 路径的转换逻辑
- **任务分发**：主 AI 可以将专业编程任务分发给 Claude Code 实例
- **异步执行**：支持并发执行多个编程任务，提高开发效率
- **工作隔离**：利用 Git Worktrees 实现完全独立的工作环境

## 功能特性

### 基础代理功能
1. **路径转换引擎**
   - Windows 绝对路径 → WSL 挂载路径
   - 支持所有盘符（C:, D:, E: 等）
   - 处理路径中的空格和特殊字符
   - 验证目标路径有效性

2. **WSL 集成**
   - 自动检测 WSL 发行版
   - 执行目录切换命令
   - 通过 stdio 方式启动 Claude Code
   - 处理 WSL 进程管理

### MCP 任务分发功能

#### Claude Code 专属任务类型
- 📁 **代码库维护**：Legacy 重构、依赖更新、代码清理、文档同步
- 🔧 **自动化开发**：测试生成、CI/CD 配置、构建优化、Git 工作流
- 📊 **代码分析**：安全审计、性能分析、依赖审计、代码质量指标
- 📝 **文档生成**：API 文档、架构文档、用户指南、变更日志
- 🔄 **迁移升级**：框架迁移、数据库迁移、API 版本升级、平台移植

#### 智能任务管理
- **任务分类器**：自动验证任务适配性，拒绝非专业任务
- **实例分配器**：根据任务类型分配专业化 Claude Code 实例
- **工作环境隔离**：每个任务在独立的 Git Worktree 中执行
- **结果聚合**：统一收集和展示所有任务的执行结果

## 架构设计

### 系统架构

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────────┐
│   主编程 AI     │───▶│  Auto Claude Code    │───▶│   Claude Code       │
│                 │    │  MCP 服务器          │    │   实例池            │
└─────────────────┘    └──────────────────────┘    └─────────────────────┘
                                │
                                ▼
                       ┌──────────────────────┐
                       │   Git Worktrees      │
                       │   工作环境隔离        │
                       └──────────────────────┘
                                │
                                ▼
                       ┌──────────────────────┐
                       │   WSL 环境           │
                       │   路径转换 + 执行     │
                       └──────────────────────┘
```

### 核心模块

1. **CLI 接口层**
   - 命令行参数解析
   - 基础路径转换功能
   - 单次 Claude Code 启动

2. **MCP 服务器层**
   - 标准 MCP 协议实现
   - 任务分类和验证
   - 工具接口定义

3. **任务管理层**
   - 任务队列管理
   - 实例生命周期管理
   - 结果聚合和监控

4. **执行环境层**
   - Git Worktree 管理
   - WSL 路径转换
   - Claude Code 实例控制

## API 设计

### CLI 接口
```bash
# 基础使用：在当前目录启动 Claude Code
auto-claude-code

# 指定目录
auto-claude-code --dir /path/to/project

# MCP 服务器模式
auto-claude-code --mcp-server --config config.yaml

# 查看支持的任务类型
auto-claude-code --list-task-types
```

### MCP 工具接口
```json
{
  "tools": [
    "create_coding_task",      // 创建编程任务
    "get_task_status",         // 获取任务状态  
    "list_active_tasks",       // 列出活跃任务
    "get_supported_task_types" // 获取支持的任务类型
  ]
}
```

### Go API 设计
```go
// 基础路径转换
type PathConverter interface {
    ConvertToWSL(windowsPath string) (string, error)
    ConvertToWindows(wslPath string) (string, error)
    ValidatePath(path string) error
}

// MCP 任务管理
type TaskManager interface {
    CreateTask(req *CreateTaskRequest) (*Task, error)
    GetTaskStatus(taskID string) (*TaskStatus, error)
    ListTasks(filter TaskFilter) ([]*Task, error)
    CancelTask(taskID string) error
}

// Claude Code 实例管理
type InstanceManager interface {
    AllocateInstance(taskType string) (*Instance, error)
    StartInstance(task *Task, worktree *Worktree) error
    MonitorInstance(instanceID string) (*InstanceStatus, error)
    StopInstance(instanceID string) error
}
```

## 错误处理

### 错误分类
1. **路径转换错误**：无效路径、权限问题、WSL 未安装
2. **任务验证错误**：不支持的任务类型、缺少必要参数
3. **实例管理错误**：Claude Code 启动失败、实例崩溃
4. **Git 操作错误**：Worktree 创建失败、分支冲突

### 错误处理策略
```go
type ErrorCode string

const (
    ErrInvalidPath     ErrorCode = "INVALID_PATH"
    ErrWSLNotFound     ErrorCode = "WSL_NOT_FOUND"
    ErrTaskNotSupported ErrorCode = "TASK_NOT_SUPPORTED"
    ErrInstanceFailed   ErrorCode = "INSTANCE_FAILED"
    ErrGitOperation     ErrorCode = "GIT_OPERATION_FAILED"
)

type AppError struct {
    Code    ErrorCode `json:"code"`
    Message string    `json:"message"`
    Details string    `json:"details,omitempty"`
}
```

## 测试策略

### 单元测试
- 路径转换逻辑测试
- 任务分类器测试
- MCP 协议实现测试
- Git Worktree 操作测试

### 集成测试
- WSL 环境集成测试
- Claude Code 启动测试
- 端到端任务执行测试

### 兼容性测试
- 不同 WSL 发行版测试
- 不同 Windows 版本测试
- 各种路径格式测试

## 部署方案

### 构建要求
- Go 1.21+
- Windows 10 1903+ 或 Windows 11
- WSL2（推荐）或 WSL1
- Git 2.25+（支持 worktree）

### 安装方式
1. **二进制发布**：GitHub Releases 下载
2. **包管理器**：Chocolatey、Scoop、Winget
3. **源码编译**：`go build` 本地编译

### 配置文件
```yaml
# config.yaml
server:
  name: "auto-claude-code"
  version: "1.0.0"
  
mcp:
  enabled: true
  max_concurrent_tasks: 5
  task_timeout: "30m"
  
claude_code:
  executable: "claude-code"
  default_args: ["--non-interactive"]
  
worktree:
  base_directory: "./worktrees"
  auto_cleanup: true
  cleanup_delay: "24h"
```

## 开发路线图

### 阶段 1：基础代理工具 (2周)
- [x] Windows 路径到 WSL 路径转换
- [x] WSL 环境检测和验证  
- [x] Claude Code 启动代理
- [x] 基本错误处理和日志
- [x] CLI 接口设计

### 阶段 2：MCP 服务器实现 (2周)
- [ ] MCP 协议实现
- [ ] 任务分类器开发
- [ ] 基础任务管理
- [ ] Git Worktree 集成

### 阶段 3：高级功能 (2周)  
- [ ] 智能实例分配
- [ ] 任务监控和日志
- [ ] Web 控制台界面
- [ ] 质量保证机制

### 阶段 4：优化和发布 (1周)
- [ ] 性能优化
- [ ] 文档完善
- [ ] 测试覆盖
- [ ] 发布准备

## 使用场景

### 基础使用场景
- Windows 主机 + WSL 开发环境的开发者
- 需要在 WSL 中使用 Claude Code 但当前在 Windows 目录下工作
- 希望简化跨系统工具启动流程的用户

### 高级使用场景
- 主编程 AI 需要分发专业编程任务
- 团队需要并行处理多个代码重构任务
- 自动化 CI/CD 流程中的代码质量检查
- 大型项目的文档生成和维护

这个设计将简单的路径转换工具升级为一个强大的 AI 编程助手生态系统，既保留了原有的便利性，又增加了强大的任务分发和管理能力。

---

*本文档遵循语义化版本控制，最后更新：2024年* 
*本文档遵循语义化版本控制，最后更新：2024年* 