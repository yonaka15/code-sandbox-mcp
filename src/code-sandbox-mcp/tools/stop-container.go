package tools

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// StopContainer stops and removes a container by its ID or name
func StopContainer(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get the container ID or name from the request using new API
	containerIdOrName, err := request.RequireString("container_id_or_name")
	if err != nil {
		return mcp.NewToolResultText("Error: container_id_or_name is required"), nil
	}

	// Stop and remove the container
	if err := stopAndRemoveContainer(ctx, containerIdOrName); err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully stopped and removed container: %s", containerIdOrName)), nil
}

// stopAndRemoveContainer stops and removes a Docker container
func stopAndRemoveContainer(ctx context.Context, containerIdOrName string) error {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Stop the container with a timeout
	timeout := 10 // seconds
	if err := cli.ContainerStop(ctx, containerIdOrName, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Remove the container
	if err := cli.ContainerRemove(ctx, containerIdOrName, container.RemoveOptions{
		RemoveVolumes: true,
		Force:         true,
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	return nil
}
