package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ========================================
// RUN COMMAND TOOL
// ========================================

type RunCommandInput struct {
	Command    string `json:"command"`
	WorkingDir string `json:"working_dir,omitempty"`
	Timeout    int    `json:"timeout,omitempty"` // seconds
}

type RunCommandOutput struct {
	Command  string `json:"command"`
	Output   string `json:"output"`
	ExitCode int    `json:"exit_code"`
	Duration string `json:"duration"`
}

func (r *CodeToolRegistry) RunCommand(ctx context.Context, input json.RawMessage) (string, error) {
	var params RunCommandInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	if params.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	// Parse command
	parts := strings.Fields(params.Command)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	// Check if command is allowed
	cmdName := parts[0]
	if !r.allowedCmds[cmdName] {
		return "", fmt.Errorf("command not allowed: %s (allowed: go, npm, git, etc.)", cmdName)
	}

	// Set working directory
	workDir := r.workspaceRoot
	if params.WorkingDir != "" {
		absPath, err := r.resolveAndValidatePath(params.WorkingDir)
		if err != nil {
			return "", err
		}
		workDir = absPath
	}

	// Set timeout
	timeout := 30 * time.Second
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Second
	}
	if timeout > 5*time.Minute {
		timeout = 5 * time.Minute // Max 5 minutes
	}

	// Create command with timeout
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, parts[0], parts[1:]...)
	cmd.Dir = workDir

	// Capture output
	start := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if cmdCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command timed out after %v", timeout)
		}
	}

	result := RunCommandOutput{
		Command:  params.Command,
		Output:   truncateString(string(output), 10000),
		ExitCode: exitCode,
		Duration: duration.String(),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal result: %w", err)
	}
	return string(resultJSON), nil
}
