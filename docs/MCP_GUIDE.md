# Auto Claude Code MCP 服务器使用指南

## 概述

Auto Claude Code 现在支持 MCP (Model Context Protocol) 服务器功能，允许主编程 AI 异步分发任务到多个 Claude Code 实例，实现并行处理和任务管理。

## 核心特性

### 🚀 任务分发系统
- **异步任务处理**：主 AI 可以提交任务后立即返回，无需等待完成
- **并发执行**：支持多个 Claude Code 实例同时运行
- **任务状态跟踪**：实时监控任务执行状态和进度
- **智能队列管理**：自动排队和优先级处理

### 🌳 Git Worktree 管理
- **隔离环境**：每个任务在独立的 Git worktree 中执行
- **自动清理**：定期清理过期的 worktree，节省磁盘空间
- **分支管理**：自动创建和管理临时分支
- **非 Git 项目支持**：自动复制目录结构

### 🔧 MCP 协议支持
- **标准兼容**：完全符合 MCP 2024-11-05 协议规范
- **工具集成**：提供丰富的工具接口供 AI 调用
- **JSON-RPC 2.0**：基于标准的 JSON-RPC 2.0 协议
- **RESTful API**：同时提供 HTTP REST 接口

## 快速开始

### 1. 配置启用

创建或编辑配置文件 `config.yaml`：

```yaml
mcp:
  enabled: true
  host: "localhost"
  port: 8080
  max_concurrent_tasks: 5
  task_timeout: "30m"
  
  # Git Worktree 配置
  worktree_base_dir: "./worktrees"
  cleanup_interval: "1h"
  max_worktrees: 10
  
  # 监控配置
  monitoring:
    enabled: true
    metrics_path: "/metrics"
    health_path: "/health"
    log_requests: true
```

### 2. 启动服务器

```bash
# 启动 MCP 服务器
auto-claude-code mcp-server

# 使用自定义配置文件
auto-claude-code mcp-server --config /path/to/config.yaml

# 调试模式
auto-claude-code mcp-server --debug
```

### 3. 验证服务

```bash
# 健康检查
curl http://localhost:8080/health

# 查看指标
curl http://localhost:8080/metrics

# 列出任务
curl http://localhost:8080/tasks
```

## MCP 协议接口

### 初始化连接

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "main-ai",
      "version": "1.0.0"
    }
  }
}
```

### 列出可用工具

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}
```

### 执行 Claude Code 任务

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "execute_claude_code",
    "arguments": {
      "project_path": "/path/to/project",
      "task_description": "实现用户登录功能",
      "claude_args": ["--help"],
      "priority": "high",
      "timeout": "30m"
    }
  }
}
```

## REST API 接口

### 任务管理

```bash
# 提交任务
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "project_path": "/path/to/project",
    "task_description": "实现用户登录功能",
    "claude_args": ["--help"],
    "priority": "high"
  }'

# 获取任务状态
curl http://localhost:8080/tasks/{task_id}

# 取消任务
curl -X DELETE http://localhost:8080/tasks/{task_id}

# 列出所有任务
curl http://localhost:8080/tasks
```

### Worktree 管理

```bash
# 列出所有 worktrees
curl http://localhost:8080/worktrees

# 获取 worktree 详情
curl http://localhost:8080/worktrees/{worktree_id}

# 删除 worktree
curl -X DELETE http://localhost:8080/worktrees/{worktree_id}
```

## 任务状态说明

| 状态 | 描述 |
|------|------|
| `pending` | 任务已提交，等待执行 |
| `running` | 任务正在执行中 |
| `completed` | 任务执行成功完成 |
| `failed` | 任务执行失败 |
| `cancelled` | 任务被取消 |
| `timeout` | 任务执行超时 |

## 配置选项详解

### 基础配置

```yaml
mcp:
  enabled: true              # 是否启用 MCP 服务器
  host: "localhost"          # 监听地址
  port: 8080                # 监听端口
  max_concurrent_tasks: 5    # 最大并发任务数
  task_timeout: "30m"        # 任务超时时间
```

### Git Worktree 配置

```yaml
mcp:
  worktree_base_dir: "./worktrees"  # worktree 基础目录
  cleanup_interval: "1h"            # 清理间隔
  max_worktrees: 10                 # 最大 worktree 数量
```

### 认证配置

```yaml
mcp:
  auth:
    enabled: false           # 是否启用认证
    method: "token"          # 认证方法: "token", "oauth2", "none"
    token_file: "tokens.txt" # Token 文件路径
    allowed_ips:             # 允许的 IP 地址
      - "127.0.0.1"
      - "::1"
```

### 队列配置

```yaml
mcp:
  queue:
    max_size: 100           # 队列最大大小
    retry_attempts: 3       # 重试次数
    retry_interval: "5s"    # 重试间隔
    priority_levels: 3      # 优先级级别数
```

### 监控配置

```yaml
mcp:
  monitoring:
    enabled: true           # 是否启用监控
    metrics_path: "/metrics" # 指标端点路径
    health_path: "/health"   # 健康检查端点路径
    log_requests: true       # 是否记录请求日志
    log_responses: false     # 是否记录响应日志
```

## 使用场景

### 1. 并行开发任务

主 AI 可以同时分发多个开发任务：

```bash
# 任务1：实现用户认证
curl -X POST http://localhost:8080/tasks -d '{
  "project_path": "/project",
  "task_description": "实现JWT用户认证系统",
  "priority": "high"
}'

# 任务2：编写单元测试
curl -X POST http://localhost:8080/tasks -d '{
  "project_path": "/project", 
  "task_description": "为用户模块编写单元测试",
  "priority": "medium"
}'

# 任务3：优化数据库查询
curl -X POST http://localhost:8080/tasks -d '{
  "project_path": "/project",
  "task_description": "优化用户查询的数据库性能",
  "priority": "low"
}'
```

### 2. 代码审查和重构

```bash
# 代码审查任务
curl -X POST http://localhost:8080/tasks -d '{
  "project_path": "/project",
  "task_description": "审查并重构用户服务代码",
  "claude_args": ["--review", "--suggest-improvements"]
}'
```

### 3. 文档生成

```bash
# 文档生成任务
curl -X POST http://localhost:8080/tasks -d '{
  "project_path": "/project",
  "task_description": "生成API文档和用户手册",
  "claude_args": ["--generate-docs"]
}'
```

## 监控和调试

### 查看服务状态

```bash
# 健康检查
curl http://localhost:8080/health

# 响应示例
{
  "status": "ok",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 查看指标

```bash
# 获取指标
curl http://localhost:8080/metrics

# 响应示例
{
  "tasks": {
    "total": 15,
    "by_status": {
      "pending": 2,
      "running": 3,
      "completed": 8,
      "failed": 2
    }
  },
  "worktrees": {
    "total": 5,
    "by_status": {
      "active": 3,
      "idle": 2
    }
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 日志分析

启用调试模式查看详细日志：

```bash
auto-claude-code mcp-server --debug --log-level debug
```

## 故障排除

### 常见问题

1. **服务器启动失败**
   - 检查端口是否被占用
   - 验证配置文件格式
   - 确认 WSL 环境可用

2. **任务执行失败**
   - 检查项目路径是否存在
   - 验证 Claude Code 是否安装
   - 查看任务日志获取详细错误信息

3. **Worktree 创建失败**
   - 确认项目是 Git 仓库
   - 检查磁盘空间是否充足
   - 验证 Git 命令是否可用

### 调试技巧

1. **启用详细日志**：
   ```bash
   auto-claude-code mcp-server --debug --log-level debug
   ```

2. **检查任务状态**：
   ```bash
   curl http://localhost:8080/tasks/{task_id}
   ```

3. **查看 worktree 状态**：
   ```bash
   curl http://localhost:8080/worktrees
   ```

## 最佳实践

### 1. 任务设计
- 将大任务拆分为小的独立任务
- 设置合理的任务超时时间
- 使用优先级管理重要任务

### 2. 资源管理
- 定期清理过期的 worktrees
- 监控磁盘空间使用情况
- 合理设置并发任务数量

### 3. 错误处理
- 实现任务重试机制
- 记录详细的错误日志
- 设置任务失败通知

### 4. 性能优化
- 使用 SSD 存储 worktrees
- 调整任务队列大小
- 监控系统资源使用情况

## 扩展开发

### 自定义工具

可以扩展 MCP 协议处理器添加自定义工具：

```go
// 在 protocol.go 中添加新工具
func (h *mcpProtocolHandler) registerCustomTool() {
    tool := &Tool{
        Name: "custom_tool",
        Description: "自定义工具描述",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "param1": map[string]interface{}{
                    "type": "string",
                    "description": "参数1描述",
                },
            },
        },
    }
    h.tools = append(h.tools, tool)
}
```

### 中间件扩展

可以添加自定义中间件：

```go
// 在 server.go 中添加中间件
func (s *mcpServer) customMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 自定义逻辑
        next.ServeHTTP(w, r)
    })
}
```

## 总结

Auto Claude Code 的 MCP 服务器功能为 AI 协作开发提供了强大的基础设施，支持：

- ✅ 异步任务分发和管理
- ✅ 并行 Claude Code 实例执行
- ✅ Git worktree 隔离环境
- ✅ 标准 MCP 协议兼容
- ✅ RESTful API 接口
- ✅ 实时监控和指标
- ✅ 灵活的配置选项

通过这些功能，主编程 AI 可以高效地管理和分发编程任务，实现真正的并行开发工作流。 