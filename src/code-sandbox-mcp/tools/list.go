package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// SandboxInfo holds information about a running sandbox container.
type SandboxInfo struct {
	ContainerID string `json:"container_id"`
	Name        string `json:"name"`
	Image       string `json:"image"`
	Status      string `json:"status"`
}

// ListSandboxes lists all running sandbox containers.
func ListSandboxes(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, mcp.NewToolResultError("DOCKER_CLIENT_ERROR", fmt.Sprintf("failed to create Docker client: %v", err))
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, mcp.NewToolResultError("CONTAINER_LIST_ERROR", fmt.Sprintf("failed to list containers: %v", err))
	}

	var sandboxes []SandboxInfo
	for _, container := range containers {
		sandboxes = append(sandboxes, SandboxInfo{
			ContainerID: container.ID[:12],
			Name:        strings.TrimPrefix(container.Names[0], "/"),
			Image:       container.Image,
			Status:      container.Status,
		})
	}

	return mcp.NewToolResultResource(sandboxes), nil
}
