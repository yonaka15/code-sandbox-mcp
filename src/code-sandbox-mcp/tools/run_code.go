package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Automata-Labs-team/code-sandbox-mcp/languages"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RunCodeSandbox(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	server := server.ServerFromContext(ctx)
	progressToken := request.Params.Meta.ProgressToken
	language, ok := request.Params.Arguments["language"].(string)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("Language not supported: %s", request.Params.Arguments["language"])), nil
	}
	code, ok := request.Params.Arguments["code"].(string)
	if !ok {
		return mcp.NewToolResultError("language must be a string"), nil
	}
	parsed := languages.Language(language)
	config := languages.SupportedLanguages[languages.Language(language)]
	if err := server.SendNotificationToClient(
		"notifications/progress",
		map[string]interface{}{
			"progress":      10,
			"progressToken": progressToken,
		},
	); err != nil {
		return mcp.NewToolResultError("Could not send progress to client"), nil
	}

	cmd := config.RunCommand

	// Escape the Python code for shell execution
	escapedCode := strings.ToValidUTF8(code, "")
	if parsed == languages.Go {
		// For Go, we need to write the code to a file
		cmd = []string{"/bin/sh", "-c", "go mod init sandbox && go mod tidy && go run main.go"}
	} else if parsed == languages.Python {
		// For Python leverage something like https://github.com/tliron/py4go to run pipreqs (https://github.com/bndr/pipreqs)
		// natively to generate requirements.txt

	}

	server.SendNotificationToClient(
		"notifications/progress",
		map[string]interface{}{
			"progress":      50,
			"progressToken": progressToken,
		},
	)

	logs, err := runInDocker(ctx, progressToken, cmd, config.Image, escapedCode, parsed, nil)
	server.SendNotificationToClient(
		"notifications/progress",
		map[string]interface{}{
			"progress":      100,
			"progressToken": progressToken,
		},
	)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
	}
	return mcp.NewToolResultText(logs), nil
}

func runInDocker(ctx context.Context, progressToken mcp.ProgressToken, cmd []string, dockerImage string, code string, language languages.Language, hostConfig *container.HostConfig) (string, error) {
	server := server.ServerFromContext(ctx)
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Pull the Docker image
	reader, err := cli.ImagePull(ctx, dockerImage, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to pull Docker image %s: %w", dockerImage, err)
	}
	io.Copy(os.Stdout, reader)

	// Create container config
	config := &container.Config{
		Image: dockerImage,
		Cmd:   cmd,
	}

	// For Go, we need to write the code to a file and set up a module
	if language == languages.Go {
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

		// Mount the temporary directory to /app and set it as working directory
		hostConfig = &container.HostConfig{
			Binds: []string{
				fmt.Sprintf("%s:/app", tmpDir),
			},
		}

		// Update container config to work in the mounted directory
		config.WorkingDir = "/app"
	}
	if language == languages.NodeJS {
		// Create a temporary directory for the Go file
		tmpDir, err := os.MkdirTemp("", "docker-sandbox-*")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary directory: %w", err)
		}
		// Clean up the temporary directory after we're done
		defer os.RemoveAll(tmpDir)

		// Write the code to a file in the temporary directory
		tmpFile := filepath.Join(tmpDir, "main.ts")
		err = os.WriteFile(tmpFile, []byte(code), 0644)
		if err != nil {
			return "", fmt.Errorf("failed to write code to temporary file: %w", err)
		}

		// Mount the temporary directory to /app and set it as working directory
		hostConfig = &container.HostConfig{
			Binds: []string{
				fmt.Sprintf("%s:/app", tmpDir),
			},
		}

		// Update container config to work in the mounted directory
		config.WorkingDir = "/app"
	}

	sandboxContainer, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := cli.ContainerStart(ctx, sandboxContainer.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	// Create a ticker for status updates (e.g., every 5 seconds)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Start ContainerWait in a separate goroutine
	waitDone := make(chan struct{})
	var waitErr error
	go func() {
		statusCh, errCh := cli.ContainerWait(ctx, sandboxContainer.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				waitErr = fmt.Errorf("error waiting for container: %w", err)
			}
		case <-statusCh:
		}
		close(waitDone)
	}()
	progress := 50
	// Main select loop to handle both container wait and status updates
loop:
	for {
		select {
		case <-ticker.C:
			progress = progress + 5
			// Send your status update here
			// For example:
			if err := server.SendNotificationToClient(
				"notifications/progress",
				map[string]interface{}{
					"progress":      progress,
					"progressToken": progressToken,
				},
			); err != nil {
				fmt.Printf("failed to send status update: %v", err)
			}
		case <-ctx.Done():
			return "", ctx.Err()
		case <-waitDone:
			if waitErr != nil {
				return "", waitErr
			}
			break loop
		}
	}

	out, err := cli.ContainerLogs(ctx, sandboxContainer.ID, container.LogsOptions{})
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
