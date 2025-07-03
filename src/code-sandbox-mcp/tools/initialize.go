package tools

import (
	"context"
	"fmt"

	dockerImage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// InitializeEnvironment creates a new container for code execution
func InitializeEnvironment(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Get the requested Docker image or use default using new API
	image := request.GetString("image", "python:3.12-slim-bookworm")

	// Get the optional container name
	name := request.GetString("name", "")

	// Create and start the container
	containerID, err := createContainer(ctx, image, name)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("container_id: %s", containerID)), nil
}

// createContainer creates a new Docker container and returns its ID
func createContainer(ctx context.Context, image string, name string) (string, error) {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Pull the Docker image if not already available
	reader, err := cli.ImagePull(ctx, image, dockerImage.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to pull Docker image %s: %w", image, err)
	}
	defer reader.Close()

	// Create container config with a working directory
	config := &container.Config{
		Image:       image,
		WorkingDir:  "/app",
		Tty:         true,
		OpenStdin:   true,
		StdinOnce:   false,
	}

	// Create host config
	hostConfig := &container.HostConfig{
		// Add any resource constraints here if needed
	}

	// Create the container
	resp, err := cli.ContainerCreate(
		ctx,
		config,
		hostConfig,
		nil,
		nil,
		name, // Use the provided name here
	)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return resp.ID, nil
}
