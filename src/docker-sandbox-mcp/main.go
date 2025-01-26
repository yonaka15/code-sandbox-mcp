package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/stdio"
)

// Version information (set by build flags)
var (
	Version   = "dev"         // Version number (from git tag or specified)
	BuildMode = "development" // Build mode (development or release)
)

// MCPConfig represents the Claude Desktop config file structure
type MCPConfig struct {
	MCPServers map[string]MCPServer `json:"mcpServers"`
}

// MCPServer represents a single MCP server configuration
type MCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	var configDir string
	switch runtime.GOOS {
	case "darwin":
		configDir = filepath.Join(homeDir, "Library", "Application Support", "Claude")
	case "windows":
		configDir = filepath.Join(os.Getenv("APPDATA"), "Claude")
	default: // linux and others
		configDir = filepath.Join(homeDir, ".config", "Claude")
	}

	return filepath.Join(configDir, "claude_desktop_config.json"), nil
}

func installConfig() error {
	configPath, err := getConfigPath()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Get the absolute path of the current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	var config MCPConfig
	if _, err := os.Stat(configPath); err == nil {
		// Read existing config
		configData, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		if err := json.Unmarshal(configData, &config); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	} else {
		// Create new config
		config = MCPConfig{
			MCPServers: make(map[string]MCPServer),
		}
	}

	// Add or update our server config
	config.MCPServers["code-sandbox-mcp"] = MCPServer{
		Command: execPath,
		Args:    []string{},
		Env:     map[string]string{},
	}

	// Write the updated config
	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Added code-sandbox-mcp to %s\n", configPath)
	return nil
}

// Language represents a supported programming language
type Language string

// Supported languages
const (
	Python Language = "python"
	Go     Language = "go"
	NodeJS Language = "nodejs"
)

// AllLanguages contains all supported languages in a specific order
var AllLanguages = []Language{Python, Go, NodeJS}

// String returns the string representation of the language
func (l Language) String() string {
	return string(l)
}

// IsValid checks if the language is supported
func (l Language) IsValid() bool {
	for _, valid := range AllLanguages {
		if l == valid {
			return true
		}
	}
	return false
}

// GenerateEnumTag generates the jsonschema enum tag for all supported languages
func GenerateEnumTag() string {
	var tags []string
	for _, lang := range AllLanguages {
		tags = append(tags, fmt.Sprintf("enum=%s", lang))
	}
	return strings.Join(tags, ",")
}

// Language configurations
type LanguageConfig struct {
	Image string // Docker image to use
	// Dependency management
	DependencyFiles []string // Files that indicate dependencies (e.g., go.mod, requirements.txt)
	InstallCommand  []string // Command to install dependencies (e.g., pip install -r requirements.txt)
	// File extensions and their specific run commands
	FileExtensions map[string][]string // Map of file extensions to their run commands
}

// supportedLanguages maps Language to their configurations
var supportedLanguages = map[Language]LanguageConfig{
	Python: {
		Image:           "python:3.12-slim-bookworm",
		DependencyFiles: []string{"requirements.txt", "pyproject.toml", "setup.py"},
		InstallCommand:  []string{"pip", "install", "-r", "requirements.txt"},
	},
	Go: {
		Image:           "golang:1.21-alpine",
		DependencyFiles: []string{"go.mod"},
		InstallCommand:  []string{"go", "mod", "download"},
	},
	NodeJS: {
		Image:           "node:23-slim",
		DependencyFiles: []string{"package.json"},
		InstallCommand:  []string{"npm", "install"},
		FileExtensions: map[string][]string{
			".js":  {"node"},
			".ts":  {"node", "--experimental-strip-types", "--experimental-transform-types"},
			".tsx": {"node", "--experimental-strip-types", "--experimental-transform-types"},
			".jsx": {"node"},
		},
	},
}

// Tool arguments are just structs, annotated with jsonschema tags
// More at https://mcpgolang.com/tools#schema-generation
type Content struct {
	Title       string  `json:"title" jsonschema:"required,description=The title to submit"`
	Description *string `json:"description" jsonschema:"description=The description to submit"`
}

type RunCodeArguments struct {
	Code       string   `json:"code" jsonschema:"required,description=The code to run"`
	Language   Language `json:"language" jsonschema:"required,description=The programming language to use,enum=python,enum=go,enum=nodejs"`
	Entrypoint []string `json:"entrypoint" jsonschema:"required,description=The command to run the code. Examples: ['python', '-c'] for Python, ['node', '-e'] for Node.js, ['go', 'run'] for Go"`
}

type RunProjectArguments struct {
	ProjectDir string   `json:"project_dir" jsonschema:"required,description=The directory containing the project to run"`
	Language   Language `json:"language" jsonschema:"required,description=The programming language to use,enum=python,enum=go,enum=nodejs"`
	Entrypoint []string `json:"entrypoint" jsonschema:"required,description=The command to run the project. Examples: ['python', 'main.py'], ['node', 'index.js'], ['go', 'run', '.'], ['./start.sh']"`
	Background bool     `json:"background" jsonschema:"description=Whether to run the project in the background (for servers, APIs, etc.)"`
}

func main() {
	// Check for --install flag
	installFlag := flag.Bool("install", false, "Add this binary to Claude Desktop config")
	flag.Parse()

	if *installFlag {
		if err := installConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	done := make(chan struct{})

	server := mcp_golang.NewServer(stdio.NewStdioServerTransport())

	// Register a tool to run code in a docker container
	err := server.RegisterTool("run_code", "Run code in a docker container. The supported languages are: "+GenerateEnumTag(), func(arguments RunCodeArguments) (*mcp_golang.ToolResponse, error) {
		// Validate language
		if !arguments.Language.IsValid() {
			return nil, fmt.Errorf("unsupported language: %s", arguments.Language)
		}

		config := supportedLanguages[arguments.Language]
		var cmd []string
		if arguments.Language == Go {
			// For Go, we need to write the code to a file
			cmd = append(arguments.Entrypoint, "/tmp/main.go")
		} else {
			// For other languages, pass code directly
			cmd = append(arguments.Entrypoint, arguments.Code)
		}

		logs, err := runInDocker(context.Background(), cmd, config.Image, arguments.Code, arguments.Language)
		if err != nil {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Error: %v", err))), nil
		}
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(logs)), nil
	})
	if err != nil {
		panic(err)
	}

	// Register the new run_project tool
	err = server.RegisterTool("run_project", "Run a project directory in a docker container. The supported languages are: "+GenerateEnumTag(), func(arguments RunProjectArguments) (*mcp_golang.ToolResponse, error) {
		// Validate language
		if !arguments.Language.IsValid() {
			return nil, fmt.Errorf("unsupported language: %s", arguments.Language)
		}

		// Validate project directory
		projectDir, err := filepath.Abs(arguments.ProjectDir)
		if err != nil {
			return nil, fmt.Errorf("invalid project directory: %v", err)
		}
		if _, err := os.Stat(projectDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("project directory does not exist: %s", projectDir)
		}

		config := supportedLanguages[arguments.Language]
		logs, containerId, err := runProjectInDocker(context.Background(), arguments.Entrypoint, config.Image, projectDir, arguments.Language, arguments.Background)
		if err != nil {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Error: %v", err))), nil
		}

		if arguments.Background {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Container started successfully with ID: %s\nLogs:\n%s", containerId, logs))), nil
		}
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(logs)), nil
	})
	if err != nil {
		panic(err)
	}

	err = server.RegisterPrompt("promt_test", "This is a test prompt", func(arguments Content) (*mcp_golang.PromptResponse, error) {
		return mcp_golang.NewPromptResponse("description", mcp_golang.NewPromptMessage(mcp_golang.NewTextContent(fmt.Sprintf("Hello, %server!", arguments.Title)), mcp_golang.RoleUser)), nil
	})
	if err != nil {
		panic(err)
	}

	err = server.RegisterResource("test://resource", "resource_test", "This is a test resource", "application/json", func() (*mcp_golang.ResourceResponse, error) {
		return mcp_golang.NewResourceResponse(mcp_golang.NewTextEmbeddedResource("test://resource", "This is a test resource", "application/json")), nil
	})
	if err != nil {
		panic(err)
	}

	err = server.Serve()
	if err != nil {
		panic(err)
	}

	<-done
}

func runInDocker(ctx context.Context, cmd []string, dockerImage string, code string, language Language) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Pull the Docker image
	reader, err := cli.ImagePull(ctx, "docker.io/library/"+dockerImage, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to pull Docker image %s: %w", dockerImage, err)
	}
	io.Copy(os.Stdout, reader)

	// Create container config
	config := &container.Config{
		Image: dockerImage,
		Cmd:   cmd,
	}

	// For Go, we need to write the code to a file and mount it
	if language == Go {
		// Create a temporary directory for the Go file
		tmpDir, err := os.MkdirTemp("", "docker-sandbox-*")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary directory: %w", err)
		}
		// Clean up the temporary directory after we're done
		defer os.RemoveAll(tmpDir)

		// Write the code to a file in the temporary directory
		tmpFile := filepath.Join(tmpDir, "main.go")
		err = os.WriteFile(tmpFile, []byte(code), 0644)
		if err != nil {
			return "", fmt.Errorf("failed to write code to temporary file: %w", err)
		}

		// Mount the temporary directory
		hostConfig := &container.HostConfig{
			Binds: []string{
				fmt.Sprintf("%s:/tmp", tmpDir),
			},
		}

		resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
		if err != nil {
			return "", fmt.Errorf("failed to create container: %w", err)
		}

		if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
			return "", fmt.Errorf("failed to start container: %w", err)
		}

		statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				return "", fmt.Errorf("error waiting for container: %w", err)
			}
		case <-statusCh:
		}

		out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
		if err != nil {
			return "", fmt.Errorf("failed to get container logs: %w", err)
		}

		var outBuf, errBuf bytes.Buffer
		_, err = stdcopy.StdCopy(&outBuf, &errBuf, out)
		if err != nil {
			return "", fmt.Errorf("failed to copy container output: %w", err)
		}

		return outBuf.String() + errBuf.String(), nil
	}

	// For other languages (e.g., Python)
	resp, err := cli.ContainerCreate(ctx, config, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("error waiting for container: %w", err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}

	var outBuf, errBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&outBuf, &errBuf, out)
	if err != nil {
		return "", fmt.Errorf("failed to copy container output: %w", err)
	}

	return outBuf.String() + errBuf.String(), nil
}

func runProjectInDocker(ctx context.Context, cmd []string, dockerImage string, projectDir string, language Language, background bool) (string, string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", "", fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Pull the Docker image
	reader, err := cli.ImagePull(ctx, "docker.io/library/"+dockerImage, image.PullOptions{})
	if err != nil {
		return "", "", fmt.Errorf("failed to pull Docker image %s: %w", dockerImage, err)
	}
	io.Copy(os.Stdout, reader)

	// Check for dependency files and prepare install command
	config := supportedLanguages[language]
	var hasDepFile bool
	var depFile string
	for _, file := range config.DependencyFiles {
		if _, err := os.Stat(filepath.Join(projectDir, file)); err == nil {
			hasDepFile = true
			depFile = file
			break
		}
	}

	// Create container config with working directory set to /app
	containerConfig := &container.Config{
		Image:      dockerImage,
		WorkingDir: "/app",
	}

	// If we have dependencies, modify the command to install them first
	if hasDepFile {
		switch language {
		case Python:
			if depFile == "requirements.txt" {
				containerConfig.Cmd = []string{
					"/bin/sh", "-c",
					fmt.Sprintf("pip install -r %s && %s", depFile, strings.Join(cmd, " ")),
				}
			} else if depFile == "pyproject.toml" || depFile == "setup.py" {
				containerConfig.Cmd = []string{
					"/bin/sh", "-c",
					fmt.Sprintf("pip install . && %s", strings.Join(cmd, " ")),
				}
			}
		case Go:
			containerConfig.Cmd = []string{
				"/bin/sh", "-c",
				fmt.Sprintf("go mod download && %s", strings.Join(cmd, " ")),
			}
		case NodeJS:
			// For Node.js, we need to check the file extension and use appropriate flags
			lastArg := cmd[len(cmd)-1]
			if ext := filepath.Ext(lastArg); ext != "" {
				if nodeCmd, ok := config.FileExtensions[ext]; ok {
					// Replace the first part of the command with the appropriate node command
					cmd = append(nodeCmd, cmd[1:]...)
				}
			}
			containerConfig.Cmd = []string{
				"/bin/sh", "-c",
				fmt.Sprintf("npm install && %s", strings.Join(cmd, " ")),
			}
		}
	} else {
		if language == NodeJS {
			// Even without package.json, we need to check file extension for TypeScript support
			lastArg := cmd[len(cmd)-1]
			if ext := filepath.Ext(lastArg); ext != "" {
				if nodeCmd, ok := config.FileExtensions[ext]; ok {
					// Replace the first part of the command with the appropriate node command
					cmd = append(nodeCmd, cmd[1:]...)
				}
			}
		}
		containerConfig.Cmd = cmd
	}

	// Mount the project directory to /app in the container
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/app", projectDir),
		},
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return "", "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", "", fmt.Errorf("failed to start container: %w", err)
	}

	// For background processes, return after container starts successfully
	if background {
		return fmt.Sprintf("Container started in background mode. Use 'docker logs %s' to view logs.", resp.ID), resp.ID, nil
	}

	// For regular processes, wait for completion and return logs
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", resp.ID, fmt.Errorf("error waiting for container: %w", err)
		}
	case <-statusCh:
	}

	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", resp.ID, fmt.Errorf("failed to get container logs: %w", err)
	}

	var outBuf, errBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&outBuf, &errBuf, out)
	if err != nil {
		return "", resp.ID, fmt.Errorf("failed to copy container output: %w", err)
	}

	return outBuf.String() + errBuf.String(), resp.ID, nil
}
