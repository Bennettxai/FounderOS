// Package container provides Docker container management for the application.
// app_container.go handles app-specific container operations for sandbox deployment.
package container

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

// App container constants
const (
	// Default resource limits for app containers
	appDefaultMemoryLimit = 512 * 1024 * 1024 // 512MB
	appDefaultCPUQuota    = 50000             // 50% of one CPU
	appDefaultPidsLimit   = 100               // Max processes

	// Additional container labels (labelApp, labelType, labelUserID are in volume.go)
	labelAppID     = "app_id"
	labelAppName   = "app_name"
	labelCreatedAt = "created_at"

	// Label values (labelAppValue is in volume.go as "businessos")
	labelValueBusinessOS = "businessos"
	labelValueSandboxApp = "sandbox-app"

	// Network settings
	appContainerNetwork = "bridge"

	// Healthcheck defaults
	healthCheckInterval = 30 * time.Second
	healthCheckTimeout  = 10 * time.Second
	healthCheckRetries  = 3
)

// AppContainerConfig holds configuration for creating an app container.
type AppContainerConfig struct {
	AppID         uuid.UUID
	AppName       string
	UserID        uuid.UUID
	Image         string
	WorkspacePath string // Host path to bind mount
	ContainerPort int    // Port inside container (e.g., 3000)
	HostPort      int    // Port on host (assigned by port allocator)
	Environment   map[string]string
	StartCommand  []string
	MemoryLimit   int64 // Optional: override default memory limit
	CPUQuota      int64 // Optional: override default CPU quota
	WorkingDir    string // Working directory inside container
}

// AppContainerInfo holds information about a running app container.
type AppContainerInfo struct {
	ContainerID   string
	AppID         uuid.UUID
	AppName       string
	UserID        uuid.UUID
	Image         string
	Status        string
	HostPort      int
	ContainerPort int
	CreatedAt     time.Time
	StartedAt     time.Time
	IPAddress     string
	HealthStatus  string
}

// AppContainerManager extends ContainerManager with app-specific operations.
type AppContainerManager struct {
	dockerClient *client.Client
	logger       *slog.Logger
	seccompPath  string
}

// NewAppContainerManager creates a new app container manager.
func NewAppContainerManager(dockerClient *client.Client, logger *slog.Logger, seccompPath string) *AppContainerManager {
	return &AppContainerManager{
		dockerClient: dockerClient,
		logger:       logger.With("component", "app_container_manager"),
		seccompPath:  seccompPath,
	}
}

// CreateAppContainer creates a new container for an app with security hardening.
func (m *AppContainerManager) CreateAppContainer(ctx context.Context, cfg AppContainerConfig) (*AppContainerInfo, error) {
	if m.dockerClient == nil {
		return nil, fmt.Errorf("docker client not initialized")
	}

	// Validate configuration
	if err := m.validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Set defaults for resource limits
	memoryLimit := cfg.MemoryLimit
	if memoryLimit == 0 {
		memoryLimit = appDefaultMemoryLimit
	}
	cpuQuota := cfg.CPUQuota
	if cpuQuota == 0 {
		cpuQuota = appDefaultCPUQuota
	}

	// Build container name
	containerName := fmt.Sprintf("businessos-app-%s-%s", cfg.AppName, cfg.AppID.String()[:8])

	// Build port bindings
	containerPortSpec := nat.Port(fmt.Sprintf("%d/tcp", cfg.ContainerPort))
	portBindings := nat.PortMap{
		containerPortSpec: []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d", cfg.HostPort),
			},
		},
	}

	// Build exposed ports
	exposedPorts := nat.PortSet{
		containerPortSpec: struct{}{},
	}

	// Build environment variables
	envVars := m.buildEnvironmentVars(cfg)

	// Build labels
	labels := map[string]string{
		labelApp:       labelValueBusinessOS,
		labelType:      labelValueSandboxApp,
		labelAppID:     cfg.AppID.String(),
		labelUserID:    cfg.UserID.String(),
		labelAppName:   cfg.AppName,
		labelCreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Build mounts
	mounts := []mount.Mount{}
	if cfg.WorkspacePath != "" {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   cfg.WorkspacePath,
			Target:   "/app",
			ReadOnly: false, // App needs write access to its workspace
		})
	}

	// Build security options
	securityOpts := []string{
		"no-new-privileges:true",
	}
	if m.seccompPath != "" {
		securityOpts = append(securityOpts, fmt.Sprintf("seccomp=%s", m.seccompPath))
	}

	// Container configuration
	containerConfig := &container.Config{
		Image:        cfg.Image,
		Env:          envVars,
		ExposedPorts: exposedPorts,
		Labels:       labels,
		WorkingDir:   cfg.WorkingDir,
		Cmd:          cfg.StartCommand,
		Healthcheck: &container.HealthConfig{
			Test:     []string{"CMD-SHELL", fmt.Sprintf("curl -f http://localhost:%d/ || exit 1", cfg.ContainerPort)},
			Interval: healthCheckInterval,
			Timeout:  healthCheckTimeout,
			Retries:  healthCheckRetries,
		},
	}

	// Set working directory default
	if containerConfig.WorkingDir == "" {
		containerConfig.WorkingDir = "/app"
	}

	// Host configuration with security hardening
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts:       mounts,
		Resources: container.Resources{
			Memory:     memoryLimit,
			CPUQuota:   cpuQuota,
			PidsLimit:  ptr(int64(appDefaultPidsLimit)),
		},
		SecurityOpt: securityOpts,
		CapDrop:     []string{"ALL"},
		CapAdd: []string{
			"NET_BIND_SERVICE", // Allow binding to ports
		},
		NetworkMode: container.NetworkMode(appContainerNetwork),
		AutoRemove:  false, // We want to keep containers for debugging
		RestartPolicy: container.RestartPolicy{
			Name:              container.RestartPolicyOnFailure,
			MaximumRetryCount: 3,
		},
	}

	// Network configuration
	networkConfig := &network.NetworkingConfig{}

	m.logger.Info("creating app container",
		"app_id", cfg.AppID,
		"app_name", cfg.AppName,
		"image", cfg.Image,
		"host_port", cfg.HostPort,
		"container_port", cfg.ContainerPort)

	// Create the container
	resp, err := m.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	m.logger.Info("app container created",
		"container_id", resp.ID[:12],
		"app_id", cfg.AppID,
		"warnings", resp.Warnings)

	return &AppContainerInfo{
		ContainerID:   resp.ID,
		AppID:         cfg.AppID,
		AppName:       cfg.AppName,
		UserID:        cfg.UserID,
		Image:         cfg.Image,
		Status:        "created",
		HostPort:      cfg.HostPort,
		ContainerPort: cfg.ContainerPort,
		CreatedAt:     time.Now().UTC(),
	}, nil
}

// StartAppContainer starts a previously created app container.
func (m *AppContainerManager) StartAppContainer(ctx context.Context, containerID string) error {
	if m.dockerClient == nil {
		return fmt.Errorf("docker client not initialized")
	}

	m.logger.Info("starting app container", "container_id", containerID[:12])

	if err := m.dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	m.logger.Info("app container started", "container_id", containerID[:12])
	return nil
}

// StopAppContainer stops a running app container.
func (m *AppContainerManager) StopAppContainer(ctx context.Context, containerID string, timeout *int) error {
	if m.dockerClient == nil {
		return fmt.Errorf("docker client not initialized")
	}

	m.logger.Info("stopping app container", "container_id", containerID[:12])

	stopOptions := container.StopOptions{}
	if timeout != nil {
		stopOptions.Timeout = timeout
	}

	if err := m.dockerClient.ContainerStop(ctx, containerID, stopOptions); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	m.logger.Info("app container stopped", "container_id", containerID[:12])
	return nil
}

// RemoveAppContainer removes an app container.
func (m *AppContainerManager) RemoveAppContainer(ctx context.Context, containerID string, force bool) error {
	if m.dockerClient == nil {
		return fmt.Errorf("docker client not initialized")
	}

	m.logger.Info("removing app container", "container_id", containerID[:12], "force", force)

	if err := m.dockerClient.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force:         force,
		RemoveVolumes: true,
	}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	m.logger.Info("app container removed", "container_id", containerID[:12])
	return nil
}

// GetAppContainerInfo retrieves information about an app container.
func (m *AppContainerManager) GetAppContainerInfo(ctx context.Context, containerID string) (*AppContainerInfo, error) {
	if m.dockerClient == nil {
		return nil, fmt.Errorf("docker client not initialized")
	}

	inspect, err := m.dockerClient.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Parse labels
	appIDStr := inspect.Config.Labels[labelAppID]
	userIDStr := inspect.Config.Labels[labelUserID]
	appName := inspect.Config.Labels[labelAppName]

	appID, _ := uuid.Parse(appIDStr)
	userID, _ := uuid.Parse(userIDStr)

	// Parse created time
	createdAt, _ := time.Parse(time.RFC3339, inspect.Created)

	// Parse started time
	var startedAt time.Time
	if inspect.State.StartedAt != "" {
		startedAt, _ = time.Parse(time.RFC3339Nano, inspect.State.StartedAt)
	}

	// Get IP address
	var ipAddress string
	if inspect.NetworkSettings != nil && inspect.NetworkSettings.Networks != nil {
		if bridge, ok := inspect.NetworkSettings.Networks["bridge"]; ok {
			ipAddress = bridge.IPAddress
		}
	}

	// Get host port
	var hostPort int
	for _, bindings := range inspect.NetworkSettings.Ports {
		if len(bindings) > 0 {
			fmt.Sscanf(bindings[0].HostPort, "%d", &hostPort)
			break
		}
	}

	// Get health status
	healthStatus := "unknown"
	if inspect.State.Health != nil {
		healthStatus = inspect.State.Health.Status
	}

	return &AppContainerInfo{
		ContainerID:   containerID,
		AppID:         appID,
		AppName:       appName,
		UserID:        userID,
		Image:         inspect.Config.Image,
		Status:        inspect.State.Status,
		HostPort:      hostPort,
		CreatedAt:     createdAt,
		StartedAt:     startedAt,
		IPAddress:     ipAddress,
		HealthStatus:  healthStatus,
	}, nil
}

// GetAppContainerLogs retrieves logs from an app container.
func (m *AppContainerManager) GetAppContainerLogs(ctx context.Context, containerID string, tail string, since string) (string, error) {
	if m.dockerClient == nil {
		return "", fmt.Errorf("docker client not initialized")
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Tail:       tail,
		Since:      since,
	}

	reader, err := m.dockerClient.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return "", fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	logs, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	// Strip Docker log header bytes (8 bytes per line for multiplexed streams)
	return stripDockerLogHeaders(string(logs)), nil
}

// StreamAppContainerLogs streams logs from an app container.
func (m *AppContainerManager) StreamAppContainerLogs(ctx context.Context, containerID string, follow bool) (io.ReadCloser, error) {
	if m.dockerClient == nil {
		return nil, fmt.Errorf("docker client not initialized")
	}

	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Timestamps: true,
		Follow:     follow,
		Tail:       "100", // Start with last 100 lines when streaming
	}

	reader, err := m.dockerClient.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to stream container logs: %w", err)
	}

	return reader, nil
}

// ExecInAppContainer executes a command in an app container.
func (m *AppContainerManager) ExecInAppContainer(ctx context.Context, containerID string, cmd []string) (string, error) {
	if m.dockerClient == nil {
		return "", fmt.Errorf("docker client not initialized")
	}

	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execID, err := m.dockerClient.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create exec: %w", err)
	}

	resp, err := m.dockerClient.ContainerExecAttach(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer resp.Close()

	output, err := io.ReadAll(resp.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to read exec output: %w", err)
	}

	return stripDockerLogHeaders(string(output)), nil
}

// ListAppContainers lists all BusinessOS app containers.
func (m *AppContainerManager) ListAppContainers(ctx context.Context) ([]*AppContainerInfo, error) {
	if m.dockerClient == nil {
		return nil, fmt.Errorf("docker client not initialized")
	}

	// Build filter args for BusinessOS sandbox containers
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("%s=%s", labelApp, labelValueBusinessOS))
	filterArgs.Add("label", fmt.Sprintf("%s=%s", labelType, labelValueSandboxApp))

	containers, err := m.dockerClient.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: filterArgs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	result := make([]*AppContainerInfo, 0, len(containers))
	for _, c := range containers {
		info, err := m.GetAppContainerInfo(ctx, c.ID)
		if err != nil {
			m.logger.Warn("failed to get container info", "container_id", c.ID[:12], "error", err)
			continue
		}
		result = append(result, info)
	}

	return result, nil
}

// ListUserAppContainers lists app containers for a specific user.
func (m *AppContainerManager) ListUserAppContainers(ctx context.Context, userID uuid.UUID) ([]*AppContainerInfo, error) {
	allContainers, err := m.ListAppContainers(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*AppContainerInfo, 0)
	for _, info := range allContainers {
		if info.UserID == userID {
			result = append(result, info)
		}
	}

	return result, nil
}

// validateConfig validates the app container configuration.
func (m *AppContainerManager) validateConfig(cfg AppContainerConfig) error {
	if cfg.AppID == uuid.Nil {
		return fmt.Errorf("app ID is required")
	}
	if cfg.AppName == "" {
		return fmt.Errorf("app name is required")
	}
	if cfg.Image == "" {
		return fmt.Errorf("image is required")
	}
	if cfg.ContainerPort <= 0 {
		return fmt.Errorf("container port must be positive")
	}
	if cfg.HostPort <= 0 {
		return fmt.Errorf("host port must be positive")
	}
	return nil
}

// buildEnvironmentVars builds the environment variables for the container.
func (m *AppContainerManager) buildEnvironmentVars(cfg AppContainerConfig) []string {
	envVars := []string{
		fmt.Sprintf("APP_ID=%s", cfg.AppID.String()),
		fmt.Sprintf("APP_NAME=%s", cfg.AppName),
		fmt.Sprintf("USER_ID=%s", cfg.UserID.String()),
		fmt.Sprintf("PORT=%d", cfg.ContainerPort),
		"NODE_ENV=production",
	}

	// Add custom environment variables
	for key, value := range cfg.Environment {
		// Sanitize environment variable names
		key = strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}

	return envVars
}

// stripDockerLogHeaders removes Docker multiplexed stream headers from log output.
func stripDockerLogHeaders(logs string) string {
	// Docker log format has 8-byte headers for multiplexed streams
	// Format: [STREAM_TYPE][0][0][0][SIZE1][SIZE2][SIZE3][SIZE4][payload]
	var result strings.Builder
	lines := strings.Split(logs, "\n")
	for _, line := range lines {
		if len(line) > 8 {
			// Skip the 8-byte header
			result.WriteString(line[8:])
		} else {
			result.WriteString(line)
		}
		result.WriteString("\n")
	}
	return strings.TrimSuffix(result.String(), "\n")
}

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}
