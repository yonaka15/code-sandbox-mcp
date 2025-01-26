package dependencies

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// Language configurations
var languageConfigs = map[Language]struct {
	image           string
	installCommand  string
	fileExtension   string
	runCommand      []string
	requirementsGen func([]string) string
}{
	Python: {
		image:          "python:3.12-slim-bookworm",
		installCommand: "pip install",
		fileExtension:  ".py",
		runCommand:     []string{"python", "-c"},
		requirementsGen: func(deps []string) string {
			return strings.Join(deps, "\n")
		},
	},
	NodeJS: {
		image:          "node:23-slim",
		installCommand: "npm install --no-save",
		fileExtension:  ".js",
		runCommand:     []string{"node", "-e"},
		requirementsGen: func(deps []string) string {
			// Create a minimal package.json
			pkgJSON := struct {
				Dependencies map[string]string `json:"dependencies"`
			}{
				Dependencies: make(map[string]string),
			}
			for _, dep := range deps {
				pkgJSON.Dependencies[dep] = "latest"
			}
			return fmt.Sprintf(`{"dependencies":%s}`, pkgJSON.Dependencies)
		},
	},
	Go: {
		image:          "golang:1.21-alpine",
		installCommand: "go get",
		fileExtension:  ".go",
		runCommand:     []string{"go", "run"},
		requirementsGen: func(deps []string) string {
			// Create a minimal go.mod file
			return fmt.Sprintf("module sandbox\n\ngo 1.21\n\nrequire (\n\t%s\n)\n",
				strings.Join(deps, " latest\n\t")+" latest")
		},
	},
}

// RunWithDependencies runs code with the specified dependencies in a Docker container
func RunWithDependencies(ctx context.Context, code string, lang Language, deps []string) (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer cli.Close()

	config := languageConfigs[lang]

	// Create a temporary directory for the code and dependencies
	tmpDir, err := os.MkdirTemp("", "code-sandbox-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write the code file
	codeFile := filepath.Join(tmpDir, "main"+config.fileExtension)
	if err := os.WriteFile(codeFile, []byte(code), 0644); err != nil {
		return "", fmt.Errorf("failed to write code file: %w", err)
	}

	// Write the requirements file
	var requirementsFile string
	var installCmd string
	switch lang {
	case Python:
		requirementsFile = filepath.Join(tmpDir, "requirements.txt")
		installCmd = fmt.Sprintf("%s -r requirements.txt", config.installCommand)
	case NodeJS:
		requirementsFile = filepath.Join(tmpDir, "package.json")
		installCmd = config.installCommand
	case Go:
		requirementsFile = filepath.Join(tmpDir, "go.mod")
		installCmd = fmt.Sprintf("%s ./...", config.installCommand)
	}

	if err := os.WriteFile(requirementsFile, []byte(config.requirementsGen(deps)), 0644); err != nil {
		return "", fmt.Errorf("failed to write requirements file: %w", err)
	}

	// Pull the Docker image
	reader, err := cli.ImagePull(ctx, "docker.io/library/"+config.image, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to pull Docker image %s: %w", config.image, err)
	}
	io.Copy(os.Stdout, reader)

	// Create container config
	containerConfig := &container.Config{
		Image:      config.image,
		WorkingDir: "/app",
		Cmd: []string{
			"/bin/sh", "-c",
			fmt.Sprintf("%s && %s %s",
				installCmd,
				strings.Join(config.runCommand, " "),
				"main"+config.fileExtension),
		},
	}

	// Mount the temporary directory
	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/app", tmpDir),
		},
	}

	// Create and start the container
	resp, err := cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	// Wait for the container to finish
	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("error waiting for container: %w", err)
		}
	case <-statusCh:
	}

	// Get the container logs
	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}

	var outBuf, errBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&outBuf, &errBuf, out)
	if err != nil {
		return "", fmt.Errorf("failed to copy container output: %w", err)
	}

	// Combine stdout and stderr
	output := outBuf.String()
	if errBuf.Len() > 0 {
		output += "\nErrors:\n" + errBuf.String()
	}

	return output, nil
}
