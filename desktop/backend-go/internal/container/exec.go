package container

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

const (
	// Timeout for exec operations
	execCreateTimeout  = 10 * time.Second
	execAttachTimeout  = 5 * time.Second
	execResizeTimeout  = 3 * time.Second
	execInspectTimeout = 3 * time.Second
)

// CreateExec creates a new exec instance in a container
// This sets up an exec session that can be attached to for interactive I/O
func (m *ContainerManager) CreateExec(containerID string, cmd []string, tty bool) (string, error) {
	return m.CreateExecWithEnv(containerID, cmd, tty, nil)
}

// CreateExecWithEnv creates a new exec instance with custom environment variables
func (m *ContainerManager) CreateExecWithEnv(containerID string, cmd []string, tty bool, envVars map[string]string) (string, error) {
	ctx, cancel := context.WithTimeout(m.ctx, execCreateTimeout)
	defer cancel()

	// Validate input
	if containerID == "" {
		return "", fmt.Errorf("container ID is required")
	}
	if len(cmd) == 0 {
		return "", fmt.Errorf("command is required")
	}

	slog.Info("[Container] Creating exec in container", "container_id", containerID, "cmd", cmd, "tty", tty)

	// Build environment variables array
	env := []string{
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"LANG=en_US.UTF-8",
	}

	// Add custom environment variables
	for key, value := range envVars {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Create exec configuration
	execConfig := container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          tty,
		Cmd:          cmd,
		WorkingDir:   "/workspace",
		Env:          env,
	}

	// Create the exec instance
	response, err := m.cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		slog.Error("[Container] Failed to create exec", "error", err)
		return "", fmt.Errorf("failed to create exec instance: %w", err)
	}

	slog.Info("[Container] Exec created successfully", "exec_id", response.ID)
	return response.ID, nil
}

// AttachExec attaches to an existing exec instance and returns the connection
// The returned HijackedResponse provides Reader/Writer for bidirectional I/O
func (m *ContainerManager) AttachExec(execID string) (types.HijackedResponse, error) {
	ctx, cancel := context.WithTimeout(m.ctx, execAttachTimeout)
	defer cancel()

	if execID == "" {
		return types.HijackedResponse{}, fmt.Errorf("exec ID is required")
	}

	slog.Info("[Container] Attaching to exec", "exec_id", execID)

	// Check exec configuration
	execConfig := container.ExecStartOptions{
		Tty: true, // Enable TTY for interactive sessions
	}

	// Attach to the exec instance
	response, err := m.cli.ContainerExecAttach(ctx, execID, execConfig)
	if err != nil {
		slog.Error("[Container] Failed to attach to exec", "exec_id", execID, "error", err)
		return types.HijackedResponse{}, fmt.Errorf("failed to attach to exec: %w", err)
	}

	slog.Info("[Container] Successfully attached to exec", "exec_id", execID)
	return response, nil
}

// ResizeExec resizes the TTY of an exec instance
// This should be called when the terminal window is resized
func (m *ContainerManager) ResizeExec(execID string, height, width uint) error {
	ctx, cancel := context.WithTimeout(m.ctx, execResizeTimeout)
	defer cancel()

	if execID == "" {
		return fmt.Errorf("exec ID is required")
	}

	slog.Info("[Container] Resizing exec", "exec_id", execID, "width", width, "height", height)

	// Resize the exec TTY
	err := m.cli.ContainerExecResize(ctx, execID, container.ResizeOptions{
		Height: height,
		Width:  width,
	})

	if err != nil {
		slog.Error("[Container] Failed to resize exec", "exec_id", execID, "error", err)
		return fmt.Errorf("failed to resize exec TTY: %w", err)
	}

	slog.Info("[Container] Successfully resized exec", "exec_id", execID)
	return nil
}

// GetExecInfo retrieves information about an exec instance
// Useful for checking if the exec is still running
func (m *ContainerManager) GetExecInfo(execID string) (container.ExecInspect, error) {
	ctx, cancel := context.WithTimeout(m.ctx, execInspectTimeout)
	defer cancel()

	if execID == "" {
		return container.ExecInspect{}, fmt.Errorf("exec ID is required")
	}

	slog.Info("[Container] Inspecting exec", "exec_id", execID)

	// Inspect the exec instance
	inspect, err := m.cli.ContainerExecInspect(ctx, execID)
	if err != nil {
		slog.Error("[Container] Failed to inspect exec", "exec_id", execID, "error", err)
		return container.ExecInspect{}, fmt.Errorf("failed to inspect exec: %w", err)
	}

	slog.Info("[Container] Exec inspected",
		"exec_id", execID, "running", inspect.Running, "exit_code", inspect.ExitCode)

	return inspect, nil
}

// StartExec is a convenience method that creates and starts an exec instance
// Returns the exec ID and the hijacked response for I/O
func (m *ContainerManager) StartExec(containerID string, cmd []string, tty bool) (string, types.HijackedResponse, error) {
	return m.StartExecWithEnv(containerID, cmd, tty, nil)
}

// StartExecWithEnv creates and starts an exec instance with custom environment variables
func (m *ContainerManager) StartExecWithEnv(containerID string, cmd []string, tty bool, envVars map[string]string) (string, types.HijackedResponse, error) {
	// Create exec instance with environment variables
	execID, err := m.CreateExecWithEnv(containerID, cmd, tty, envVars)
	if err != nil {
		return "", types.HijackedResponse{}, fmt.Errorf("failed to create exec: %w", err)
	}

	// Attach to exec instance
	hijacked, err := m.AttachExec(execID)
	if err != nil {
		return execID, types.HijackedResponse{}, fmt.Errorf("failed to attach to exec: %w", err)
	}

	// Start the exec instance
	ctx, cancel := context.WithTimeout(m.ctx, execCreateTimeout)
	defer cancel()

	startOptions := container.ExecStartOptions{
		Tty: tty,
	}

	if err := m.cli.ContainerExecStart(ctx, execID, startOptions); err != nil {
		hijacked.Close()
		return execID, types.HijackedResponse{}, fmt.Errorf("failed to start exec: %w", err)
	}

	slog.Info("[Container] Exec started successfully", "exec_id", execID)
	return execID, hijacked, nil
}

// WaitExec waits for an exec instance to complete and returns the exit code
func (m *ContainerManager) WaitExec(execID string) (int, error) {
	// Poll the exec status until it completes
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute) // Max wait time

	for {
		select {
		case <-ticker.C:
			inspect, err := m.GetExecInfo(execID)
			if err != nil {
				return -1, fmt.Errorf("failed to get exec info: %w", err)
			}

			if !inspect.Running {
				slog.Info("[Container] Exec completed", "exec_id", execID, "exit_code", inspect.ExitCode)
				return inspect.ExitCode, nil
			}

		case <-timeout:
			return -1, fmt.Errorf("exec timeout waiting for completion")
		}
	}
}

// StopExec attempts to stop a running exec instance
// Note: Docker doesn't provide a direct way to stop exec, but we can inspect it
func (m *ContainerManager) StopExec(execID string) error {
	inspect, err := m.GetExecInfo(execID)
	if err != nil {
		return fmt.Errorf("failed to get exec info: %w", err)
	}

	if !inspect.Running {
		slog.Info("[Container] Exec is not running", "exec_id", execID)
		return nil
	}

	// Unfortunately, Docker API doesn't provide a direct stop for exec
	// The exec will stop when the process inside exits or the connection closes
	slog.Info("[Container] Exec is still running, will stop when connection closes", "exec_id", execID)
	return nil
}
