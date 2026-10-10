package e2b

import (
	"context"
	"fmt"
	"net/http"
)

// ---- Sandbox lifecycle methods ----------------------------------------------

// TestExecution runs a single sandbox execution against projectPath.
func (c *Client) TestExecution(ctx context.Context, projectPath string) (*ExecutionResult, error) {
	return c.testExecutionWithID(ctx, projectPath, "")
}

// testExecutionWithID is the internal implementation that accepts an optional
// iteration ID.
func (c *Client) testExecutionWithID(ctx context.Context, projectPath, iterationID string) (*ExecutionResult, error) {
	req := testExecutionRequest{
		Path:        projectPath,
		IterationID: iterationID,
		KeepSandbox: c.config.KeepSandbox,
	}

	var result ExecutionResult
	if err := c.makeRequest(ctx, http.MethodPost, "/test-execution", req, &result); err != nil {
		return nil, fmt.Errorf("test execution: %w", err)
	}
	return &result, nil
}

// UpdateSandboxFiles replaces or creates files inside a running sandbox.
func (c *Client) UpdateSandboxFiles(ctx context.Context, sandboxID string, files map[string]string) (*UpdateFilesResult, error) {
	if sandboxID == "" {
		return nil, NewValidationError("sandbox ID must not be empty")
	}
	if len(files) == 0 {
		return nil, NewValidationError("files map must not be empty")
	}

	req := updateFilesRequest{
		SandboxID: sandboxID,
		Files:     files,
	}

	var result UpdateFilesResult
	if err := c.makeRequest(ctx, http.MethodPost, "/update-sandbox-files", req, &result); err != nil {
		return nil, fmt.Errorf("update sandbox files: %w", err)
	}
	return &result, nil
}

// GetSandboxStatus queries the running status of a sandbox.
func (c *Client) GetSandboxStatus(ctx context.Context, sandboxID string) (*SandboxStatus, error) {
	if sandboxID == "" {
		return nil, NewValidationError("sandbox ID must not be empty")
	}

	var result SandboxStatus
	endpoint := fmt.Sprintf("/sandbox-status/%s", sandboxID)
	if err := c.makeRequest(ctx, http.MethodGet, endpoint, nil, &result); err != nil {
		return nil, fmt.Errorf("get sandbox status: %w", err)
	}
	return &result, nil
}

// CleanupSandbox destroys a sandbox and releases its resources.
func (c *Client) CleanupSandbox(ctx context.Context, sandboxID string) (*CleanupResult, error) {
	if sandboxID == "" {
		return nil, NewValidationError("sandbox ID must not be empty")
	}

	var result CleanupResult
	endpoint := fmt.Sprintf("/sandbox/%s", sandboxID)
	if err := c.makeRequest(ctx, http.MethodDelete, endpoint, nil, &result); err != nil {
		return nil, fmt.Errorf("cleanup sandbox: %w", err)
	}
	return &result, nil
}
