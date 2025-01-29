package resources

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/docker/docker/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func GetContainerLogs(ctx context.Context, request mcp.ReadResourceRequest) ([]interface{}, error) {

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	containerID := request.Params.URI // Extract ID from the full URI

	logFile, _ := os.Create(containerID + ".log")
	defer logFile.Close()

	// Set default ContainerLogsOptions
	logOpts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     false, // we just want to grab logs and return
		Tail:       "all",
	}

	// Actually fetch the logs
	reader, err := cli.ContainerLogs(ctx, containerID, logOpts)
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

	return []interface{}{
		mcp.TextResourceContents{
			ResourceContents: mcp.ResourceContents{
				URI:      fmt.Sprintf("containers://%s/logs", containerID),
				MIMEType: "text/plain",
			},
			Text: combined,
		},
	}, nil
}
