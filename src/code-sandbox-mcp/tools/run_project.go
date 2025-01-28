package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	deps "github.com/Automata-Labs-team/code-sandbox-mcp/languages"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/mark3labs/mcp-go/mcp"
)

func RunProjectSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	language, ok := request.Params.Arguments["language"].(deps.Language)
	if !ok {
		return nil, fmt.Errorf("invalid language")
	}
	entrypoint, ok := request.Params.Arguments["entrypoint"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid entrypoint")
	}
	projectDir, ok := request.Params.Arguments["projectDir"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid projectDir")
	}
	background, ok := request.Params.Arguments["background"].(bool)
	if !ok {
		return nil, fmt.Errorf("invalid background")
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
	logs, containerId, err := runProjectInDocker(context.Background(), strings.Fields(entrypoint), config.Image, projectDir, language, background)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
	}

	if background {
		return mcp.NewToolResultText(fmt.Sprintf("Container started successfully with ID: %s\nLogs:\n%s", containerId, logs)), nil
	}
	return mcp.NewToolResultText(logs), nil
}

func runProjectInDocker(ctx context.Context, cmd []string, dockerImage string, projectDir string, language deps.Language, background bool) (string, string, error) {
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
			// For Node.js, we need to check the file extension and use appropriate flags
			lastArg := cmd[len(cmd)-1]
			if ext := filepath.Ext(lastArg); ext != "" {
				if deps.SupportedLanguages[language].RunCommand != nil {
					// Replace the first part of the command with the appropriate node command
					cmd = append(deps.SupportedLanguages[language].RunCommand, cmd[1:]...)
				}
			}
			containerConfig.Cmd = []string{
				"/bin/sh", "-c",
				fmt.Sprintf("npm install && %s", strings.Join(cmd, " ")),
			}
		}
	} else {
		if language == deps.NodeJS {
			// Even without package.json, we need to check file extension for TypeScript support
			lastArg := cmd[len(cmd)-1]
			if ext := filepath.Ext(lastArg); ext != "" {
				if deps.SupportedLanguages[language].RunCommand != nil {
					// Replace the first part of the command with the appropriate node command
					cmd = append(deps.SupportedLanguages[language].RunCommand, cmd[1:]...)
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
