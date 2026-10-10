package container

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// ContainerManager manages Docker container lifecycle
type ContainerManager struct {
	cli          *client.Client
	ctx          context.Context
	defaultImage string
	mu           sync.RWMutex
	containers   map[string]*ContainerInfo // containerID -> info
}

// ContainerInfo tracks container metadata
type ContainerInfo struct {
	ID           string
	UserID       string
	Image        string
	Status       string
	CreatedAt    time.Time
	LastActivity time.Time
}

// NewContainerManager creates a new container manager
func NewContainerManager(ctx context.Context, imageName string) (*ContainerManager, error) {
	slog.Info("[Container] Creating new container manager", "image", imageName)

	// Create Docker client with version negotiation
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Verify Docker daemon is available with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := cli.Ping(pingCtx); err != nil {
		cli.Close()
		return nil, fmt.Errorf("Docker daemon not available: %w", err)
	}

	slog.Info("[Container] Docker daemon connection verified")

	// Set default image if not provided
	if imageName == "" {
		imageName = "ubuntu:22.04"
		slog.Info("[Container] Using default image", "image", imageName)
	}

	manager := &ContainerManager{
		cli:          cli,
		ctx:          ctx,
		defaultImage: imageName,
		containers:   make(map[string]*ContainerInfo),
	}

	slog.Info("[Container] Container manager initialized successfully")
	return manager, nil
}

// IsDockerAvailable checks if Docker daemon is available
func (m *ContainerManager) IsDockerAvailable() bool {
	ctx, cancel := context.WithTimeout(m.ctx, 3*time.Second)
	defer cancel()

	_, err := m.cli.Ping(ctx)
	return err == nil
}

// GetContainerInfo retrieves detailed container information
func (m *ContainerManager) GetContainerInfo(containerID string) (types.ContainerJSON, error) {
	ctx, cancel := context.WithTimeout(m.ctx, 10*time.Second)
	defer cancel()

	info, err := m.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return types.ContainerJSON{}, fmt.Errorf("failed to inspect container: %w", err)
	}

	return info, nil
}

// ListContainers lists all containers managed by this manager
func (m *ContainerManager) ListContainers(userID string) []*ContainerInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var containers []*ContainerInfo
	for _, info := range m.containers {
		if userID == "" || info.UserID == userID {
			containers = append(containers, info)
		}
	}

	return containers
}

// UpdateActivity updates the last activity timestamp for a container
func (m *ContainerManager) UpdateActivity(containerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, exists := m.containers[containerID]; exists {
		info.LastActivity = time.Now()
	}
}

// GetDefaultImage returns the default container image
func (m *ContainerManager) GetDefaultImage() string {
	return m.defaultImage
}

// Close cleanups the container manager and Docker client
func (m *ContainerManager) Close() error {
	slog.Info("[Container] Closing container manager")

	if m.cli != nil {
		if err := m.cli.Close(); err != nil {
			return fmt.Errorf("failed to close Docker client: %w", err)
		}
	}

	slog.Info("[Container] Container manager closed")
	return nil
}

// Shutdown gracefully shuts down all containers and closes the manager
func (m *ContainerManager) Shutdown() error {
	slog.Info("[Container] Shutting down container manager")

	m.mu.Lock()
	containerIDs := make([]string, 0, len(m.containers))
	for id := range m.containers {
		containerIDs = append(containerIDs, id)
	}
	m.mu.Unlock()

	// Stop all containers
	for _, id := range containerIDs {
		slog.Info("[Container] Stopping container", "container_id", id[:12])
		if err := m.StopContainer(id, 10); err != nil {
			slog.Warn("[Container] Failed to stop container", "container_id", id[:12], "error", err)
		}
	}

	// Close Docker client
	return m.Close()
}
