# MCP Stdio 支持

## 概述

Auto Claude Code 现在支持 MCP (Model Context Protocol) 的 stdio 传输模式，这使得程序可以通过标准输入输出进行 JSON-RPC 通信，而不仅仅依赖 HTTP 传输。

## 核心特性

### 🔗 多传输支持
- **HTTP传输**: 传统的 HTTP JSON-RPC 服务器
- **Stdio传输**: 通过 stdin/stdout 进行 JSON-RPC 通信  
- **多传输同时**: 可以同时启用多种传输方式

### 📡 标准协议
- 完全符合 MCP 2024-11-05 协议规范
- 标准 JSON-RPC 2.0 协议
- 每行一个 JSON-RPC 请求/响应

### 🚀 高性能
- 零拷贝的流式处理
- 异步任务执行
- 优雅的连接管理

## 快速开始

### 1. 启动 Stdio 服务器

```bash
# 基础启动
auto-claude-code mcp-stdio

# 使用自定义配置
auto-claude-code mcp-stdio --config config.stdio.example.yaml

# 调试模式
auto-claude-code mcp-stdio --debug
```

### 2. 发送 JSON-RPC 请求

```bash
# 初始化连接
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | auto-claude-code mcp-stdio

# 列出可用工具
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | auto-claude-code mcp-stdio
```

## 协议交互

### 初始化流程

1. **发送初始化请求**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "my-client",
      "version": "1.0.0"
    }
  }
}
```

2. **接收初始化响应**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2024-11-05",
    "capabilities": {
      "tools": {"listChanged": true},
      "logging": {}
    },
    "serverInfo": {
      "name": "auto-claude-code-mcp",
      "version": "1.0.0"
    }
  }
}
```

### 可用工具

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list"
}
```

响应包含所有可用工具:
```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [
      {
        "name": "execute_claude_code",
        "description": "在WSL中执行Claude Code任务",
        "inputSchema": {
          "type": "object",
          "properties": {
            "project_path": {"type": "string", "description": "项目路径"},
            "task_description": {"type": "string", "description": "任务描述"}
          },
          "required": ["project_path", "task_description"]
        }
      }
    ]
  }
}
```

### 执行任务

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
      "priority": "high",
      "timeout": "30m"
    }
  }
}
```

## 编程集成

### Python 示例

```python
import subprocess
import json
import threading

class MCPStdioClient:
    def __init__(self, executable_path="./auto-claude-code.exe"):
        self.process = subprocess.Popen(
            [executable_path, "mcp-stdio"],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1
        )
        self.request_id = 0
    
    def send_request(self, method, params=None):
        self.request_id += 1
        request = {
            "jsonrpc": "2.0",
            "id": self.request_id,
            "method": method,
            "params": params
        }
        
        request_line = json.dumps(request) + "\n"
        self.process.stdin.write(request_line)
        self.process.stdin.flush()
        
        response_line = self.process.stdout.readline()
        return json.loads(response_line.strip())
    
    def initialize(self):
        return self.send_request("initialize", {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "python-client", "version": "1.0"}
        })
    
    def list_tools(self):
        return self.send_request("tools/list")
    
    def execute_task(self, project_path, description):
        return self.send_request("tools/call", {
            "name": "execute_claude_code",
            "arguments": {
                "project_path": project_path,
                "task_description": description
            }
        })
    
    def close(self):
        self.process.stdin.close()
        self.process.wait()

# 使用示例
client = MCPStdioClient()
try:
    # 初始化
    init_result = client.initialize()
    print("初始化成功:", init_result)
    
    # 列出工具
    tools = client.list_tools()
    print("可用工具:", tools)
    
    # 执行任务
    task_result = client.execute_task("/path/to/project", "添加README文件")
    print("任务结果:", task_result)
    
finally:
    client.close()
```

### Node.js 示例

```javascript
const { spawn } = require('child_process');
const readline = require('readline');

class MCPStdioClient {
    constructor(executablePath = './auto-claude-code.exe') {
        this.process = spawn(executablePath, ['mcp-stdio']);
        this.rl = readline.createInterface({
            input: this.process.stdout,
            output: process.stdout,
            terminal: false
        });
        this.requestId = 0;
        this.pendingRequests = new Map();
        
        this.rl.on('line', (line) => {
            try {
                const response = JSON.parse(line);
                const pending = this.pendingRequests.get(response.id);
                if (pending) {
                    this.pendingRequests.delete(response.id);
                    if (response.error) {
                        pending.reject(new Error(response.error.message));
                    } else {
                        pending.resolve(response.result);
                    }
                }
            } catch (err) {
                console.error('解析响应失败:', err);
            }
        });
    }
    
    sendRequest(method, params = null) {
        return new Promise((resolve, reject) => {
            this.requestId++;
            const request = {
                jsonrpc: '2.0',
                id: this.requestId,
                method,
                params
            };
            
            this.pendingRequests.set(this.requestId, { resolve, reject });
            this.process.stdin.write(JSON.stringify(request) + '\n');
            
            // 设置超时
            setTimeout(() => {
                if (this.pendingRequests.has(this.requestId)) {
                    this.pendingRequests.delete(this.requestId);
                    reject(new Error('请求超时'));
                }
            }, 30000);
        });
    }
    
    async initialize() {
        return this.sendRequest('initialize', {
            protocolVersion: '2024-11-05',
            capabilities: {},
            clientInfo: { name: 'node-client', version: '1.0' }
        });
    }
    
    async listTools() {
        return this.sendRequest('tools/list');
    }
    
    async executeTask(projectPath, description) {
        return this.sendRequest('tools/call', {
            name: 'execute_claude_code',
            arguments: {
                project_path: projectPath,
                task_description: description
            }
        });
    }
    
    close() {
        this.process.stdin.end();
        this.process.kill();
    }
}

// 使用示例
async function main() {
    const client = new MCPStdioClient();
    
    try {
        // 初始化
        const initResult = await client.initialize();
        console.log('初始化成功:', initResult);
        
        // 列出工具
        const tools = await client.listTools();
        console.log('可用工具:', tools);
        
        // 执行任务
        const taskResult = await client.executeTask('/path/to/project', '添加单元测试');
        console.log('任务结果:', taskResult);
        
    } catch (error) {
        console.error('错误:', error);
    } finally {
        client.close();
    }
}

main();
```

## 配置选项

### Stdio 传输配置

```yaml
mcp:
  # 传输配置
  http:
    enabled: false    # 禁用HTTP（纯stdio模式）
  
  stdio:
    enabled: true     # 启用stdio传输
    # reader和writer在运行时自动设置
```

### 混合模式配置

```yaml
mcp:
  # 同时启用HTTP和stdio
  http:
    enabled: true
  
  stdio:
    enabled: true
```

## 使用场景

### 1. AI Assistant 集成
- Claude Desktop 等AI工具的MCP服务器
- 通过stdio与AI进行直接通信
- 异步任务分发和管理

### 2. 自动化脚本
- CI/CD 流水线中的代码处理
- 批量项目管理和维护
- 定时任务执行

### 3. 开发工具集成
- IDE插件开发
- 编辑器扩展
- 开发环境自动化

### 4. 微服务架构
- 容器化部署
- 进程间通信
- 服务编排

## 调试和监控

### 启用调试日志

```bash
auto-claude-code mcp-stdio --debug --log-level debug
```

### 请求/响应跟踪

```yaml
mcp:
  monitoring:
    log_requests: true    # 记录所有请求
    log_responses: true   # 记录所有响应
```

### 常见问题

1. **连接超时**: 检查stdin/stdout是否正确连接
2. **JSON解析错误**: 确保每行只有一个JSON对象
3. **任务执行失败**: 检查WSL环境和Claude Code安装

## 性能优化

### 1. 连接池管理
- 复用stdio连接
- 异步请求处理
- 连接超时控制

### 2. 批量操作
- 多个请求批量发送
- 流水线式处理
- 并发任务执行

### 3. 资源管理
- 及时关闭连接
- 内存使用监控
- 进程生命周期管理

## 安全考虑

### 1. 输入验证
- JSON格式验证
- 参数类型检查
- 路径安全验证

### 2. 资源限制
- 并发任务数限制
- 内存使用限制
- 执行时间限制

### 3. 权限控制
- 文件访问权限
- WSL环境隔离
- 任务执行权限

## 总结

MCP Stdio 支持为 Auto Claude Code 提供了更灵活的集成方式，特别适合：

- ✅ AI工具集成
- ✅ 自动化脚本
- ✅ 容器化部署
- ✅ 进程间通信
- ✅ 开发工具集成

通过标准的stdin/stdout接口，可以轻松地与各种编程语言和工具进行集成，实现强大的代码处理和任务管理功能。 