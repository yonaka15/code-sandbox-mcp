# Code Sandbox MCP 🐳

A secure sandbox environment for executing code within Docker containers. This MCP server provides AI applications with a safe and isolated environment for running code while maintaining security through containerization.

## 🌟 Features

- **Multi-Language Support**: Run Python, Go, and Node.js code in isolated Docker containers
- **TypeScript Support**: Built-in support for TypeScript and JSX/TSX files
- **Dependency Management**: Automatic handling of project dependencies (pip, go mod, npm)
- **Flexible Execution**: Custom entrypoints for both single-file code and full projects
- **Background Mode**: Run long-running services in the background
- **Real-time Output**: Capture and stream container logs in real-time

## 🚀 Installation

### Prerequisites

- Docker installed and running
  - [Install Docker for Linux](https://docs.docker.com/engine/install/)
  - [Install Docker Desktop for macOS](https://docs.docker.com/desktop/install/mac/)
  - [Install Docker Desktop for Windows](https://docs.docker.com/desktop/install/windows-install/)

### Quick Install

#### Unix-like Systems (Linux, macOS)
```bash
curl -fsSL https://raw.githubusercontent.com/Automata-Labs-team/code-sandbox-mcp/main/install.sh | bash
```

#### Windows
```powershell
# Run in PowerShell
irm https://raw.githubusercontent.com/Automata-Labs-team/code-sandbox-mcp/main/install.ps1 | iex
```

The installer will:
1. Check for Docker installation
2. Download the appropriate binary for your system
3. Create Claude Desktop configuration

### Manual Installation

1. Download the latest release for your platform from the [releases page](https://github.com/Automata-Labs-team/code-sandbox-mcp/releases)
2. Place the binary in a directory in your PATH
3. Make it executable (Unix-like systems only):
   ```bash
   chmod +x code-sandbox-mcp
   ```

## 🔧 Configuration

### Claude Desktop

The installer automatically creates the configuration file. If you need to manually configure it:

#### Linux
```json
// ~/.config/Claude/claude_desktop_config.json
{
    "mcpServers": {
        "code-sandbox-mcp": {
            "command": "/path/to/code-sandbox-mcp",
            "args": [],
            "env": {}
        }
    }
}
```

#### macOS
```json
// ~/Library/Application Support/Claude/claude_desktop_config.json
{
    "mcpServers": {
        "code-sandbox-mcp": {
            "command": "/path/to/code-sandbox-mcp",
            "args": [],
            "env": {}
        }
    }
}
```

#### Windows
```json
// %APPDATA%\Claude\claude_desktop_config.json
{
    "mcpServers": {
        "code-sandbox-mcp": {
            "command": "C:\\path\\to\\code-sandbox-mcp.exe",
            "args": [],
            "env": {}
        }
    }
}
```

### Other AI Applications

For other AI applications that support MCP servers, configure them to use the `code-sandbox-mcp` binary as their code execution backend.

## 🔧 Technical Details

### Supported Languages

| Language | File Extensions | Docker Image |
|----------|----------------|--------------|
| Python | .py | python:3.12-slim-bookworm |
| Go | .go | golang:1.21-alpine |
| Node.js | .js, .ts, .tsx, .jsx | node:23-slim |

### Dependency Management

The sandbox automatically detects and installs dependencies:

- **Python**: requirements.txt, pyproject.toml, setup.py
- **Go**: go.mod
- **Node.js**: package.json

### TypeScript Support

Node.js 23+ includes built-in TypeScript support:
- `--experimental-strip-types`: Enabled by default for .ts files
- `--experimental-transform-types`: Used for .tsx files

## 🔐 Security Features

- Isolated execution environment using Docker containers
- Resource limitations through Docker container constraints
- Separate stdout and stderr streams
- Clean container cleanup after execution
- Project files mounted read-only in containers

## 🛠️ Development

If you want to build the project locally or contribute to its development, see [DEVELOPMENT.md](DEVELOPMENT.md).

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
