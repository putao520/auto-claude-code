#!/bin/bash

# MCP Stdio 模式测试脚本
echo "🚀 MCP Stdio 模式测试"
echo "===================="

# 检查可执行文件
if [ ! -f "./auto-claude-code.exe" ]; then
    echo "❌ 找不到 auto-claude-code.exe"
    exit 1
fi

echo "✅ 找到可执行文件"

# 生成测试请求
cat > examples/test-requests.jsonl << 'EOF'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test-client","version":"1.0.0"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
EOF

echo "✅ 测试文件已生成: examples/test-requests.jsonl"
echo ""
echo "🔗 使用方法："
echo "1. 启动服务器: ./auto-claude-code.exe mcp-stdio"
echo "2. 发送请求: cat examples/test-requests.jsonl | ./auto-claude-code.exe mcp-stdio" 