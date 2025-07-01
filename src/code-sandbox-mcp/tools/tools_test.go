package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMockCallToolRequest(toolName string, params map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments,omitempty"`
			Meta      *struct {
				ProgressToken mcp.ProgressToken "json:\"progressToken,omitempty\""
			} "json:\"_meta,omitempty\""
		}{
			Name:      toolName,
			Arguments: params,
		},
	}
}

func TestSandboxLifecycle(t *testing.T) {
	ctx := context.Background()
	containerName := "mcp-test-container-lifecycle"

	// 1. Initialize
	initRequest := newMockCallToolRequest("sandbox_initialize", map[string]interface{}{
		"image": "alpine:latest",
		"name":  containerName,
	})
	initResult, err := InitializeEnvironment(ctx, initRequest)
	require.NoError(t, err)
	require.NotNil(t, initResult)
	require.Len(t, initResult.Content, 1)

	textContent, ok := initResult.Content[0].(mcp.TextContent)
	require.True(t, ok)

	parts := strings.Split(textContent.Text, ": ")
	require.Len(t, parts, 2, "Initialize result should be in 'container_id: xxx' format")
	containerID := parts[1]
	require.NotEmpty(t, containerID)

	// Defer Stop to ensure cleanup
	defer func() {
		stopRequest := newMockCallToolRequest("sandbox_stop", map[string]interface{}{
			"container_id_or_name": containerName,
		})
		_, err := StopContainer(ctx, stopRequest)
		assert.NoError(t, err, "Deferred stop should not fail")
	}()

	// 2. List
	listRequest := newMockCallToolRequest("sandbox_list", nil)
	listResult, err := ListSandboxes(ctx, listRequest)
	require.NoError(t, err)
	require.Len(t, listResult.Content, 1)

	listTextContent, ok := listResult.Content[0].(mcp.TextContent)
	require.True(t, ok)
	
	var sandboxes []SandboxInfo
	err = json.Unmarshal([]byte(listTextContent.Text), &sandboxes)
	require.NoError(t, err)

	found := false
	for _, s := range sandboxes {
		if s.Name == containerName {
			found = true
			assert.Equal(t, containerID[:12], s.ContainerID)
			assert.True(t, strings.HasPrefix(s.Image, "alpine:latest"), "Image should be alpine:latest")
			break
		}
	}
	assert.True(t, found, "Newly created container should be in the list")


	// 3. Exec
	execRequest := newMockCallToolRequest("sandbox_exec", map[string]interface{}{
		"container_id_or_name": containerName,
		"commands":             []string{"echo", "hello world"},
	})
	execResult, err := Exec(ctx, execRequest)
	require.NoError(t, err)
	require.Len(t, execResult.Content, 1)

	execTextContent, ok := execResult.Content[0].(mcp.TextContent)
	require.True(t, ok)
	assert.Contains(t, execTextContent.Text, "hello world")
}
