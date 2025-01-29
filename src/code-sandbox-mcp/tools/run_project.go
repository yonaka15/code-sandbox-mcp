package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	deps "github.com/Automata-Labs-team/code-sandbox-mcp/languages"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RunProjectSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	language, ok := request.Params.Arguments["language"].(deps.Language)
	if !ok {
		return nil, fmt.Errorf("invalid language")
	}
	entrypoint, ok := request.Params.Arguments["entrypointCmd"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid entrypoint")
	}
	projectDir, ok := request.Params.Arguments["projectDir"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid projectDir")
	}

	// Validate project directory
	projectDir, err := filepath.Abs(projectDir)
	if err != nil {
		return nil, fmt.Errorf("invalid project directory: %v", err)
	}
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("project directory does not exist: %s", projectDir)
	}

	config := deps.SupportedLanguages[language]
	containerId, err := runProjectInDocker(context.Background(), request.Params.Meta.ProgressToken, strings.Fields(entrypoint), config.Image, projectDir, language)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
	}

	return mcp.NewToolResultText(containerId), nil
}

func runProjectInDocker(ctx context.Context, progressToken mcp.ProgressToken, cmd []string, dockerImage string, projectDir string, language deps.Language) (string, error) {
	server := server.ServerFromContext(ctx)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()
	server.SendNotificationToClient(
		"notifications/progress",
		map[string]interface{}{
			"progress":      10,
			"progressToken": progressToken,
		},
	)
	// Pull the Docker image
	_, err = cli.ImagePull(ctx, dockerImage, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to pull Docker image %s: %w", dockerImage, err)
	}


	// Check for dependency files and prepare install command
	var hasDepFile bool
	var depFile string
	for _, file := range deps.SupportedLanguages[language].DependencyFiles {
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
		case deps.Python:
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
		case deps.Go:
			containerConfig.Cmd = []string{
				"/bin/sh", "-c",
				fmt.Sprintf("go mod download && %s", strings.Join(cmd, " ")),
			}
		case deps.NodeJS:
			containerConfig.Cmd = []string{
				"/bin/sh", "-c",
				fmt.Sprintf("npm install && %s", strings.Join(cmd, " ")),
			}
		}
	}

	server.SendNotificationToClient(
		"notifications/progress",
		map[string]interface{}{
			"progress":      50,
			"progressToken": progressToken,
		},
	)

	// Mount the project directory to /app in the container
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/app", projectDir),
		},
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}
	server.SendNotificationToClient(
		"notifications/progress",
		map[string]interface{}{
			"progress":      75,
			"progressToken": progressToken,
		},
	)

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	server.SendNotificationToClient(
		"notifications/progress",
		map[string]interface{}{
			"progress":      100,
			"progressToken": progressToken,
		},
	)

	return resp.ID, nil
}
