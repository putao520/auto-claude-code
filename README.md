# Auto Claude Code - Windows to WSL Bridge + MCP Task Distribution System

🌉 A smart Windows-to-WSL bridge for seamless Claude Code integration with MCP (Model Context Protocol) task distribution.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square)](https://opensource.org/licenses/MIT)
[![Windows](https://img.shields.io/badge/Windows-0078D6?style=flat-square&logo=windows&logoColor=white)](https://www.microsoft.com/windows)
[![WSL](https://img.shields.io/badge/WSL-4E9A06?style=flat-square&logo=linux&logoColor=white)](https://docs.microsoft.com/windows/wsl/)

## 🚀 What is Auto Claude Code?

Auto Claude Code is an intelligent programming assistant with two core functionalities:

### 🔧 **Core Feature 1: Windows-to-WSL Path Bridge**
A lightweight proxy tool designed for Windows + WSL development environments. It intelligently converts Windows working directories to WSL mount paths and launches Claude Code in the WSL environment, enabling seamless cross-system AI programming experiences.

### 🤖 **Core Feature 2: MCP Task Distribution System**
An intelligent task distribution system supporting MCP (Model Context Protocol). Main programming AIs can distribute specialized programming tasks to multiple Claude Code instances through our MCP server, enabling asynchronous execution and result aggregation.

## ✨ Key Features

- **🎯 One-Click Launch**: Start Claude Code in the corresponding WSL path from any Windows directory
- **🔄 Smart Path Conversion**: Automatic Windows-to-WSL path translation
- **📋 Task Distribution**: Main AI can distribute specialized programming tasks to Claude Code instances
- **⚡ Async Execution**: Support concurrent execution of multiple programming tasks
- **🔒 Work Isolation**: Complete independent work environments using Git Worktrees
- **🌐 UTF-16LE Support**: Proper handling of WSL command output encoding

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────────┐    ┌─────────────────────┐
│   Main AI       │───▶│  Auto Claude Code    │───▶│   Claude Code       │
│                 │    │  MCP Server          │    │   Instance Pool     │
└─────────────────┘    └──────────────────────┘    └─────────────────────┘
                                │
                                ▼
                       ┌──────────────────────┐
                       │   Git Worktrees      │
                       │   Work Isolation     │
                       └──────────────────────┘
                                │
                                ▼
                       ┌──────────────────────┐
                       │   WSL Environment    │
                       │   Path Conversion    │
                       └──────────────────────┘
```

## 🚀 Quick Start

### Prerequisites

- Windows 10/11 with WSL2 installed
- Go 1.21+ (for building from source)
- Claude Code installed in WSL environment

### Installation

#### Option 1: Download Binary (Recommended)
```bash
# Download the latest release
curl -L https://github.com/putao520/auto-claude-code/releases/latest/download/auto-claude-code.exe -o auto-claude-code.exe
```

#### Option 2: Build from Source
```bash
git clone https://github.com/putao520/auto-claude-code.git
cd auto-claude-code
go build -o auto-claude-code.exe ./cmd/auto-claude-code
```

### Basic Usage

```bash
# Launch Claude Code in current directory
./auto-claude-code.exe

# Launch in specific directory
./auto-claude-code.exe --dir "C:\Projects\MyApp"

# Check system environment
./auto-claude-code.exe check

# Start MCP server mode
./auto-claude-code.exe mcp-server --config config.yaml
```

## 🛠️ Claude Code Specialized Tasks

Auto Claude Code supports intelligent task distribution for specialized programming tasks:

- 📁 **Codebase Maintenance**: Legacy refactoring, dependency updates, code cleanup
- 🔧 **Development Automation**: Test generation, CI/CD configuration, build optimization
- 📊 **Code Analysis**: Security audits, performance analysis, quality metrics
- 📝 **Documentation**: API docs, architecture docs, user guides
- 🔄 **Migration & Upgrades**: Framework migrations, database migrations, API upgrades

## 📚 Documentation

- **[User Guide](docs/USER_GUIDE.md)** - Comprehensive usage guide
- **[MCP Integration](docs/MCP_INTEGRATION.md)** - MCP protocol implementation details
- **[Technical Documentation](docs/TECHNICAL.md)** - Architecture and implementation
- **[Implementation Roadmap](docs/IMPLEMENTATION_ROADMAP.md)** - Development roadmap
- **[TUI Demo](docs/TUI_DEMO.md)** - Terminal UI demonstration

## 🎯 Use Cases

### For Individual Developers
- **Cross-Platform Development**: Seamlessly switch between Windows UI tools and WSL development environment
- **Directory-Based AI Assistance**: Get AI help contextual to your current working directory
- **Automated Development Tasks**: Let Claude Code handle routine programming tasks

### For Development Teams
- **Task Distribution**: Main AI coordinates and distributes specialized tasks
- **Parallel Development**: Multiple Claude Code instances working on different aspects
- **Quality Assurance**: Automated code analysis and documentation generation

## 🔧 Configuration

Create a `config.yaml` file:

```yaml
# Basic configuration
wsl:
  defaultDistro: "Ubuntu"
  timeout: 30s

# MCP Server configuration
mcp:
  enabled: true
  port: 8080
  maxInstances: 5
  taskTimeout: 300s

# Logging
logging:
  level: "info"
  file: "auto-claude-code.log"
```

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guidelines](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📋 System Requirements

- **Operating System**: Windows 10/11
- **WSL**: WSL2 with Ubuntu/Debian distribution
- **Memory**: 2GB+ available RAM
- **Storage**: 100MB+ free space
- **Network**: Internet connection for Claude Code authentication

## 🆘 Troubleshooting

### Common Issues

1. **WSL not detected**
   ```bash
   # Check WSL installation
   wsl --list --verbose
   ```

2. **Claude Code not found**
   ```bash
   # Verify Claude Code installation in WSL
   wsl which claude-code
   ```

3. **Path conversion failed**
   ```bash
   # Test path conversion
   ./auto-claude-code.exe check --debug
   ```

For more troubleshooting help, see our [User Guide](docs/USER_GUIDE.md).

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Claude AI](https://claude.ai) - For the amazing AI programming assistant
- [Microsoft WSL](https://docs.microsoft.com/windows/wsl/) - For the Windows Subsystem for Linux
- [Model Context Protocol](https://modelcontextprotocol.io/) - For the standardized AI tool protocol

## 📊 Project Status

- ✅ Windows-to-WSL path conversion
- ✅ Claude Code integration  
- ✅ UTF-16LE encoding support
- 🚧 MCP server implementation (in progress)
- 🚧 Task distribution system (in progress)
- 📋 TUI interface (planned)

---

<div align="center">
  <p>Made with ❤️ for the Windows + WSL + AI development community</p>
  <p>
    <a href="https://github.com/putao520/auto-claude-code/issues">Report Bug</a>
    ·
    <a href="https://github.com/putao520/auto-claude-code/issues">Request Feature</a>
    ·
    <a href="docs/USER_GUIDE.md">Documentation</a>
  </p>
</div> 