# Development Guide 🛠️

This guide is for developers who want to build the project locally or contribute to its development.

## Prerequisites

- Go 1.21 or later
- Docker installed and running
- Git (for version information)
- Make (optional, for build automation)

## Building from Source

1. Clone the repository:
```bash
git clone https://github.com/Automata-Labs-team/docker-sandbox-mcp.git
cd docker-sandbox-mcp
```

2. Build the project:
```bash
# Development build
./build.sh

# Release build
./build.sh --release

# Release with specific version
./build.sh --release --version v1.0.0
```

The binaries will be available in the `bin` directory.

## Build Options

The `build.sh` script supports several options:

| Option | Description |
|--------|-------------|
| `--release` | Build in release mode with version information |
| `--version <ver>` | Specify a version number (e.g., v1.0.0) |

## Project Structure

```
docker-sandbox-mcp/
├── src/
│   └── docker-sandbox-mcp/
│       └── main.go       # Main application code
├── bin/                  # Compiled binaries
├── build.sh             # Build script
├── install.sh           # Unix-like systems installer
├── install.ps1          # Windows installer
├── README.md            # User documentation
└── DEVELOPMENT.md       # This file
```

## API Documentation

The project implements the MCP (Machine Code Protocol) server interface for executing code in Docker containers.

### Core Functions

- `runInDocker`: Executes single-file code in a Docker container
- `runProjectInDocker`: Runs project directories in containers
- `RegisterTool`: Registers new tool endpoints
- `NewServer`: Creates a new MCP server instance

### Tool Arguments

#### RunCodeArguments
```go
type RunCodeArguments struct {
    Code       string   `json:"code"`       // The code to run
    Language   Language `json:"language"`   // Programming language
    Entrypoint []string `json:"entrypoint"` // Command to run the code
}
```

#### RunProjectArguments
```go
type RunProjectArguments struct {
    ProjectDir string   `json:"project_dir"` // Project directory
    Language   Language `json:"language"`    // Programming language
    Entrypoint []string `json:"entrypoint"` // Command to run the project
    Background bool     `json:"background"`  // Run in background
}
```

## Adding Language Support

To add support for a new programming language:

1. Add a new language constant:
```go
const NewLang Language = "newlang"
```

2. Add it to `AllLanguages`:
```go
var AllLanguages = []Language{Python, Go, NodeJS, NewLang}
```

3. Add language configuration:
```go
var supportedLanguages = map[Language]LanguageConfig{
    NewLang: {
        Image:           "official/image:tag",
        DependencyFiles: []string{"deps.txt"},
        InstallCommand:  []string{"install", "deps"},
        FileExtensions:  map[string][]string{
            ".ext": {"command", "--flags"},
        },
    },
}
```

## Performance Considerations

- Use slim/alpine Docker images where possible
- Optimize container startup time
- Implement efficient log capturing
- Clean up containers and resources

## Security Guidelines

When implementing new features:
1. Never mount sensitive host directories
2. Always run containers with minimal privileges
3. Clean up containers after execution
4. Validate and sanitize all inputs
5. Use read-only mounts where possible

## Testing

Run the test suite:
```bash
go test ./...
```

## Version Information

Version information is embedded in release builds:
```go
var (
    Version   = "dev"
    BuildMode = "development"
)
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

Please follow the existing code style and include appropriate documentation. 