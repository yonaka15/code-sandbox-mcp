package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/container"
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
		return nil, fmt.Errorf("DOCKER_CLIENT_ERROR: failed to create Docker client: %v", err)
	}
	defer cli.Close()

	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("CONTAINER_LIST_ERROR: failed to list containers: %v", err)
	}

	var sandboxes []SandboxInfo
	for _, c := range containers {
		var name string
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		sandboxes = append(sandboxes, SandboxInfo{
			ContainerID: c.ID[:12],
			Name:        name,
			Image:       c.Image,
			Status:      c.Status,
		})
	}

	jsonData, err := json.Marshal(sandboxes)
	if err != nil {
		return nil, fmt.Errorf("JSON_SERIALIZE_ERROR: failed to serialize sandbox list: %v", err)
	}

	return mcp.NewToolResultText(string(jsonData)), nil
}
