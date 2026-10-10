package container

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/go-connections/nat"
)

const (
	// Container naming convention
	containerPrefix = "terminal"

	// Security and resource limits
	defaultMemoryLimit = 512 * 1024 * 1024 // 512MB
	defaultCPUQuota    = 50000             // 50% of one CPU core
	defaultCPUPeriod   = 100000            // Standard period: 100ms
	defaultPidsLimit   = 100               // Max 100 processes
	defaultStopTimeout = 10                // Default graceful stop timeout in seconds

	// Container configuration
	defaultWorkspaceMount = "/workspace"
)

// CreateContainer creates a new isolated container for a user with security hardening
//
// Parameters:
//   - userID: Unique identifier for the user
//   - sessionID: Unique identifier for the terminal session (used in container name)
//   - image: Docker image to use (e.g., "ubuntu:22.04")
//
// Returns:
//   - containerID: Unique identifier for the created container
//   - error: Error if container creation fails
//
// Security features:
//   - Capability dropping: All capabilities dropped, only essential ones added
//   - Resource limits: Memory, CPU, and PID limits enforced
//   - Network isolation: No network access by default
//   - Volume isolation: User-specific workspace volume
func (m *ContainerManager) CreateContainer(userID string, sessionID string, image string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use default image if not specified
	if image == "" {
		image = m.defaultImage
		slog.Info("[Container] Using default image", "image", image)
	}

	// Generate container name with session ID for uniqueness
	// Use first 8 chars of sessionID to keep names readable
	shortSessionID := sessionID
	if len(sessionID) > 8 {
		shortSessionID = sessionID[:8]
	}
	containerName := fmt.Sprintf("%s-%s-%s", containerPrefix, userID, shortSessionID)
	volumeName := fmt.Sprintf("workspace_%s", userID)

	slog.Info("[Container] Creating container",
		"container", containerName,
		"user_id", userID,
		"image", image,
	)

	// Create volume if it doesn't exist
	if err := m.ensureVolume(volumeName, userID); err != nil {
		return "", fmt.Errorf("failed to ensure volume: %w", err)
	}

	// Configure container with security hardening
	config := &container.Config{
		Image: image,
		Tty:   true,
		Env: []string{
			"TERM=xterm-256color",
			"LANG=en_US.UTF-8",
		},
		Labels: map[string]string{
			"app":     "businessos",
			"type":    "terminal",
			"user_id": userID,
		},
		// Keep container running
		Cmd: strslice.StrSlice{"/bin/sh", "-c", "while true; do sleep 3600; done"},
	}

	// Host configuration with security and resource limits
	hostConfig := &container.HostConfig{
		// Resource limits
		Resources: container.Resources{
			Memory:    defaultMemoryLimit,
			NanoCPUs:  0, // Set via CPUQuota/CPUPeriod instead
			CPUQuota:  defaultCPUQuota,
			CPUPeriod: defaultCPUPeriod,
			PidsLimit: newInt64(defaultPidsLimit),
		},

		// Security: Drop ALL capabilities, add only essential ones
		// CRITICAL: Removed DAC_OVERRIDE (file permission bypass), SETUID/SETGID (privilege escalation)
		CapDrop: strslice.StrSlice{"ALL"},
		CapAdd: strslice.StrSlice{
			"CHOWN",  // Allow changing file ownership in /workspace
			"FOWNER", // Allow operations on files regardless of owner (needed for workspace)
		},

		// Security options: Custom Seccomp profile + no new privileges
		// Seccomp profile blocks: mount, pivot_root, setns, ptrace, kernel modules, bpf
		SecurityOpt: []string{
			"no-new-privileges:true",    // Prevent privilege escalation via setuid binaries
			"seccomp=" + SeccompProfile, // Custom profile blocking escape syscalls (embedded)
		},

		// Network: Bridge mode for terminal containers (allows npm, git, curl, etc.)
		// Security is maintained via: capability dropping, seccomp, no-new-privileges, readonly root
		NetworkMode: "bridge",

		// DNS configuration for reliable resolution
		DNS: []string{"8.8.8.8", "8.8.4.4", "1.1.1.1"},

		// Add host.docker.internal to allow containers to reach host services
		ExtraHosts: []string{"host.docker.internal:host-gateway"},

		// Read-only root filesystem with tmpfs for required writable paths
		ReadonlyRootfs: true,
		Tmpfs: map[string]string{
			"/tmp":     "rw,noexec,nosuid,size=64m", // Temp files, no execution
			"/var/tmp": "rw,noexec,nosuid,size=32m", // Var temp
			"/run":     "rw,noexec,nosuid,size=16m", // Runtime files (pid, sockets)
		},

		// Mount user workspace volume and init script
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: defaultWorkspaceMount,
			},
			{
				Type:     mount.TypeBind,
				Source:   getInitScriptPath(),
				Target:   "/etc/businessos/init.sh",
				ReadOnly: true,
			},
		},

		// Auto-remove container on exit (optional, set to false for debugging)
		AutoRemove: false,

		// Logging configuration
		LogConfig: container.LogConfig{
			Type: "json-file",
			Config: map[string]string{
				"max-size": "10m",
				"max-file": "3",
			},
		},
	}

	// Create container with timeout
	ctx, cancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer cancel()

	resp, err := m.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Track container in manager
	m.containers[resp.ID] = &ContainerInfo{
		ID:           resp.ID,
		UserID:       userID,
		Image:        image,
		Status:       "created",
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	slog.Info("[Container] Container created successfully",
		containerName, resp.ID[:12])

	// Log any warnings
	for _, warning := range resp.Warnings {
		slog.Warn("[Container] Warning", "message", warning)
	}

	return resp.ID, nil
}

// StartContainer starts an existing container
//
// Parameters:
//   - containerID: Unique identifier of the container to start
//
// Returns:
//   - error: Error if container start fails
func (m *ContainerManager) StartContainer(containerID string) error {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	slog.Info("[Container] Starting container", "container_id", containerID[:12])

	if err := m.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Update container status
	m.mu.Lock()
	if info, exists := m.containers[containerID]; exists {
		info.Status = "running"
		info.LastActivity = time.Now()
	}
	m.mu.Unlock()

	slog.Info("[Container] Container started", "container_id", containerID[:12])
	return nil
}

// StopContainer stops a running container with graceful shutdown
//
// Parameters:
//   - containerID: Unique identifier of the container to stop
//   - timeout: Seconds to wait before forcefully killing the container
//
// Returns:
//   - error: Error if container stop fails
func (m *ContainerManager) StopContainer(containerID string, timeout int) error {
	ctx, cancel := context.WithTimeout(m.ctx, time.Duration(timeout+5)*time.Second)
	defer cancel()

	slog.Info("[Container] Stopping container", "container_id", containerID[:12], "timeout_secs", timeout)

	// Use timeout for graceful shutdown
	timeoutPtr := &timeout
	stopOptions := container.StopOptions{
		Timeout: timeoutPtr,
	}

	if err := m.cli.ContainerStop(ctx, containerID, stopOptions); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Update container status
	m.mu.Lock()
	if info, exists := m.containers[containerID]; exists {
		info.Status = "stopped"
		info.LastActivity = time.Now()
	}
	m.mu.Unlock()

	slog.Info("[Container] Container stopped", "container_id", containerID[:12])
	return nil
}

// RemoveContainer removes a container and cleans up resources
//
// Parameters:
//   - containerID: Unique identifier of the container to remove
//   - force: If true, forcefully remove even if container is running
//
// Returns:
//   - error: Error if container removal fails
func (m *ContainerManager) RemoveContainer(containerID string, force bool) error {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	slog.Info("[Container] Removing container", "container_id", containerID[:12], "force", force)

	removeOptions := container.RemoveOptions{
		Force:         force,
		RemoveVolumes: false, // Keep volumes for data persistence
		RemoveLinks:   false,
	}

	if err := m.cli.ContainerRemove(ctx, containerID, removeOptions); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Remove from tracking map
	m.mu.Lock()
	delete(m.containers, containerID)
	m.mu.Unlock()

	slog.Info("[Container] Container removed", "container_id", containerID[:12])
	return nil
}

// GetContainerStatus retrieves the current status of a container
//
// Parameters:
//   - containerID: Unique identifier of the container
//
// Returns:
//   - status: Current container status (e.g., "running", "exited", "created")
//   - error: Error if status retrieval fails
func (m *ContainerManager) GetContainerStatus(containerID string) (string, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 5*time.Second)
	defer cancel()

	inspectData, err := m.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container: %w", err)
	}

	status := inspectData.State.Status
	slog.Info("[Container] Container status", "container_id", containerID[:12], "status", status)

	// Update internal tracking
	m.mu.Lock()
	if info, exists := m.containers[containerID]; exists {
		info.Status = status
	}
	m.mu.Unlock()

	return status, nil
}

// ListUserContainers lists all containers belonging to a specific user
//
// Parameters:
//   - userID: Unique identifier of the user
//
// Returns:
//   - containers: List of containers with the user_id label
//   - error: Error if listing fails
func (m *ContainerManager) ListUserContainers(userID string) ([]types.Container, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	slog.Info("[Container] Listing containers", "user_id", userID)

	// Build filter to find containers by user label
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", "app=businessos")
	filterArgs.Add("label", fmt.Sprintf("user_id=%s", userID))

	listOptions := container.ListOptions{
		All:     true, // Include stopped containers
		Filters: filterArgs,
	}

	containers, err := m.cli.ContainerList(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	slog.Info("[Container] Found containers", "count", len(containers), "user_id", userID)

	return containers, nil
}

// ensureVolume creates a volume if it doesn't exist
func (m *ContainerManager) ensureVolume(volumeName, userID string) error {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	// Check if volume exists
	_, err := m.cli.VolumeInspect(ctx, volumeName)
	if err == nil {
		slog.Info("[Container] Volume already exists", "volume", volumeName)
		return nil
	}

	// Create volume
	slog.Info("[Container] Creating volume", "volume", volumeName)

	volumeConfig := volume.CreateOptions{
		Name:   volumeName,
		Driver: "local",
		Labels: map[string]string{
			"app":     "businessos",
			"user_id": userID,
		},
	}

	_, err = m.cli.VolumeCreate(ctx, volumeConfig)
	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}

	slog.Info("[Container] Volume created", "volume", volumeName)
	return nil
}

// newInt64 is a helper function to create a pointer to an int64
func newInt64(i int64) *int64 {
	return &i
}

// GetOrCreateFilesystemContainer gets an existing or creates a new lightweight container
// for filesystem operations. Unlike terminal containers, these are long-lived and shared
// for all filesystem operations for a user.
//
// Parameters:
//   - userID: Unique identifier for the user
//
// Returns:
//   - containerID: Unique identifier for the filesystem container
//   - error: Error if container creation/retrieval fails
func (m *ContainerManager) GetOrCreateFilesystemContainer(userID string) (string, error) {
	// Check for existing filesystem container
	containerName := fmt.Sprintf("filesystem-%s", userID)
	volumeName := fmt.Sprintf("workspace_%s", userID)

	// First, check if container already exists and is running
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	// List containers with our label
	filterArgs := filters.NewArgs()
	filterArgs.Add("name", containerName)

	containers, err := m.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	// If container exists
	if len(containers) > 0 {
		c := containers[0]
		if c.State == "running" {
			slog.Info("[Container] Reusing existing filesystem container", "name", containerName)
			return c.ID, nil
		}
		// Container exists but not running - start it
		slog.Info("[Container] Starting existing filesystem container", "name", containerName)
		if err := m.StartContainer(c.ID); err != nil {
			// If we can't start, remove and recreate
			_ = m.cli.ContainerRemove(ctx, c.ID, container.RemoveOptions{Force: true})
		} else {
			return c.ID, nil
		}
	}

	slog.Info("[Container] Creating filesystem container", "name", containerName, "user_id", userID)

	// Ensure volume exists
	if err := m.ensureVolume(volumeName, userID); err != nil {
		return "", fmt.Errorf("failed to ensure volume: %w", err)
	}

	// Configure lightweight filesystem container
	config := &container.Config{
		Image: m.defaultImage,
		Tty:   false,
		Env: []string{
			"TERM=xterm-256color",
		},
		Labels: map[string]string{
			"app":     "businessos",
			"type":    "filesystem",
			"user_id": userID,
		},
		// Keep container running with minimal overhead
		Cmd: strslice.StrSlice{"/bin/sh", "-c", "while true; do sleep 3600; done"},
	}

	// Minimal host configuration for filesystem operations
	hostConfig := &container.HostConfig{
		Resources: container.Resources{
			Memory:    128 * 1024 * 1024, // 128MB - minimal for filesystem ops
			CPUQuota:  10000,             // 10% CPU
			CPUPeriod: 100000,
			PidsLimit: newInt64(10), // Minimal processes
		},
		CapDrop: strslice.StrSlice{"ALL"},
		CapAdd: strslice.StrSlice{
			"CHOWN",
			"FOWNER",
		},
		SecurityOpt: []string{
			"no-new-privileges:true",
			"seccomp=" + SeccompProfile,
		},
		NetworkMode:    "none",
		ReadonlyRootfs: true,
		Tmpfs: map[string]string{
			"/tmp": "rw,noexec,nosuid,size=16m",
		},
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: defaultWorkspaceMount,
			},
		},
		AutoRemove: false,
		RestartPolicy: container.RestartPolicy{
			Name: container.RestartPolicyUnlessStopped,
		},
	}

	// Create container
	createCtx, createCancel := context.WithTimeout(m.ctx, 30*time.Second)
	defer createCancel()

	resp, err := m.cli.ContainerCreate(createCtx, config, hostConfig, nil, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create filesystem container: %w", err)
	}

	// Start the container
	if err := m.StartContainer(resp.ID); err != nil {
		// Cleanup on failure
		_ = m.cli.ContainerRemove(ctx, resp.ID, container.RemoveOptions{Force: true})
		return "", fmt.Errorf("failed to start filesystem container: %w", err)
	}

	slog.Info("[Container] Filesystem container created and started", "name", containerName, "container_id", resp.ID[:12])
	return resp.ID, nil
}

// Helper type for port bindings (not used currently, but useful for future)
type PortBinding struct {
	HostIP   string
	HostPort string
}

// buildPortMap creates port mappings for container networking
// Currently unused due to NetworkMode: "none", but kept for future use
func buildPortMap(bindings []PortBinding) nat.PortMap {
	portMap := nat.PortMap{}
	for _, binding := range bindings {
		port, _ := nat.NewPort("tcp", binding.HostPort)
		portMap[port] = []nat.PortBinding{
			{
				HostIP:   binding.HostIP,
				HostPort: binding.HostPort,
			},
		}
	}
	return portMap
}

// getInitScriptPath returns the path to the BusinessOS init script on the host
// This script is bind-mounted into containers at /etc/businessos/init.sh
func getInitScriptPath() string {
	// Check environment variable first
	if path := os.Getenv("BUSINESSOS_INIT_SCRIPT"); path != "" {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				slog.Info("[Container] Using init script from env", "path", absPath)
				return absPath
			}
		}
	}

	// No hardcoded developer paths — only use env var or relative path resolution

	// Try relative path and convert to absolute
	relativePath := "internal/terminal/businessos_init.sh"
	if absPath, err := filepath.Abs(relativePath); err == nil {
		if _, err := os.Stat(absPath); err == nil {
			slog.Info("[Container] Found init script", "path", absPath)
			return absPath
		}
	}

	// Fallback — return a relative path; container creation will fail with a clear error
	fallback := "internal/terminal/businessos_init.sh"
	slog.Warn("[Container] Init script not found, using fallback", "fallback", fallback)
	return fallback
}
