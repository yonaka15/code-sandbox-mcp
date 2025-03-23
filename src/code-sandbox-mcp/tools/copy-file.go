package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// CopyFile copies a single local file to a container's filesystem
func CopyFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	containerID, ok := request.Params.Arguments["container_id"].(string)
	if !ok || containerID == "" {
		return mcp.NewToolResultText("container_id is required"), nil
	}

	localSrcFile, ok := request.Params.Arguments["local_src_file"].(string)
	if !ok || localSrcFile == "" {
		return mcp.NewToolResultText("local_src_file is required"), nil
	}

	// Clean and validate the source path
	localSrcFile = filepath.Clean(localSrcFile)
	info, err := os.Stat(localSrcFile)
	if err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error accessing source file: %v", err)), nil
	}

	if info.IsDir() {
		return mcp.NewToolResultText("local_src_file must be a file, not a directory"), nil
	}

	// Get the destination path (optional parameter)
	destPath, ok := request.Params.Arguments["dest_path"].(string)
	if !ok || destPath == "" {
		// Default: use the name of the source file
		destPath = filepath.Join("/app", filepath.Base(localSrcFile))
	} else {
		// If provided but doesn't start with /, prepend /app/
		if !strings.HasPrefix(destPath, "/") {
			destPath = filepath.Join("/app", destPath)
		}
	}

	// Create destination directory in container if it doesn't exist
	destDir := filepath.Dir(destPath)
	if err := createDirectoryInContainer(ctx, containerID, destDir); err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error creating destination directory: %v", err)), nil
	}

	// Copy the file to the container
	if err := copyFileToContainer(ctx, containerID, localSrcFile, destPath); err != nil {
		return mcp.NewToolResultText(fmt.Sprintf("Error copying file to container: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully copied %s to %s in container %s", localSrcFile, destPath, containerID)), nil
}

// createDirectoryInContainer creates a directory in the container if it doesn't exist
func createDirectoryInContainer(ctx context.Context, containerID string, dirPath string) error {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	createDirCmd := []string{"mkdir", "-p", dirPath}
	exec, err := cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          createDirCmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return fmt.Errorf("failed to create exec: %w", err)
	}

	if err := cli.ContainerExecStart(ctx, exec.ID, container.ExecStartOptions{}); err != nil {
		return fmt.Errorf("failed to start exec: %w", err)
	}

	return nil
}

// copyFileToContainer copies a single file to the container
func copyFileToContainer(ctx context.Context, containerID string, srcPath string, destPath string) error {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	// Open the source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Copy the file content to the container
	err = cli.CopyToContainer(ctx, containerID, filepath.Dir(destPath), srcFile, container.CopyToContainerOptions{})
	if err != nil {
		return fmt.Errorf("failed to copy to container: %w", err)
	}

	return nil
}
