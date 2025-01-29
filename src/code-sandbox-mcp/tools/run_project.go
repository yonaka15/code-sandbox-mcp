package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	deps "github.com/Automata-Labs-team/code-sandbox-mcp/languages"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RunProjectSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	language, ok := request.Params.Arguments["language"].(string)
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
	projectDir = filepath.Clean(projectDir)
	if _, err := os.Stat(projectDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("project directory does not exist: %s", projectDir)
	}

	config := deps.SupportedLanguages[deps.Language(language)]
	containerId, err := runProjectInDocker(ctx, request.Params.Meta.ProgressToken, strings.Fields(entrypoint), config.Image, projectDir, deps.Language(language))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
	}

	// Fetch the logs so far and attach it to the tool result
	logFile, _ := os.Create(containerId + ".log")
	defer logFile.Close()

	// Set default ContainerLogsOptions
	logOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false, // we just want to grab logs and return
		Tail:       "all",
	}

	// Actually fetch the logs
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()
	reader, err := cli.ContainerLogs(ctx, containerId, logOpts)
	if err != nil {
		return nil, fmt.Errorf("error fetching container logs: %w", err)
	}
	defer reader.Close()

	// Docker returns a multiplexed stream if the container was started without TTY.
	// We use stdcopy.StdCopy to split stdout and stderr.
	var stdoutBuf, stderrBuf bytes.Buffer
	if _, err := stdcopy.StdCopy(&stdoutBuf, &stderrBuf, reader); err != nil {
		return nil, fmt.Errorf("error copying container logs: %w", err)
	}

	// Combine them. You could also return them separately if you prefer.
	combined := stdoutBuf.String() + stderrBuf.String()

	return mcp.NewToolResultResource(combined, mcp.ResourceContents{
		URI:      fmt.Sprintf("containers://%s/logs", containerId),
		MIMEType: "text/plain",
	}), nil
}

func runProjectInDocker(ctx context.Context, progressToken mcp.ProgressToken, cmd []string, dockerImage string, projectDir string, language deps.Language) (string, error) {
	server := server.ServerFromContext(ctx)
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()
	if err := server.SendNotificationToClient(
		"notifications/progress",
		map[string]interface{}{
			"progress":      10,
			"progressToken": progressToken,
		},
	); err != nil {
		return "", fmt.Errorf("failed to send progress notification: %w", err)
	}
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
