package container

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
)

const (
	// Volume configuration
	volumePrefix      = "workspace_"
	volumeDriver      = "local"
	defaultVolumeType = "workspace"

	// Label keys
	labelApp    = "app"
	labelType   = "type"
	labelUserID = "user_id"

	// Label values
	labelAppValue = "businessos"

	// Timeouts
	volumeCreateTimeout  = 30 * time.Second
	volumeRemoveTimeout  = 30 * time.Second
	volumeListTimeout    = 15 * time.Second
	volumeInspectTimeout = 10 * time.Second
)

// CreateVolume creates a new Docker volume for a user workspace
// Returns the volume name and error if any
func (m *ContainerManager) CreateVolume(userID string) (string, error) {
	if userID == "" {
		return "", fmt.Errorf("userID cannot be empty")
	}

	volumeName := m.getVolumeName(userID)

	// Check if volume already exists
	exists, err := m.VolumeExists(userID)
	if err != nil {
		return "", fmt.Errorf("failed to check if volume exists: %w", err)
	}

	if exists {
		slog.Info("[Container] Volume already exists", "volume", volumeName)
		return volumeName, nil
	}

	// Create volume with timeout
	ctx, cancel := context.WithTimeout(m.ctx, volumeCreateTimeout)
	defer cancel()

	// Build volume creation options
	volumeCreateOptions := volume.CreateOptions{
		Driver: volumeDriver,
		Name:   volumeName,
		Labels: map[string]string{
			labelApp:    labelAppValue,
			labelType:   defaultVolumeType,
			labelUserID: userID,
		},
	}

	// Create the volume
	vol, err := m.cli.VolumeCreate(ctx, volumeCreateOptions)
	if err != nil {
		return "", fmt.Errorf("failed to create volume %s: %w", volumeName, err)
	}

	slog.Info("[Container] Volume created", "volume", vol.Name)
	return vol.Name, nil
}

// RemoveVolume removes a user's Docker volume
// Only removes if no containers are currently using it
func (m *ContainerManager) RemoveVolume(userID string) error {
	if userID == "" {
		return fmt.Errorf("userID cannot be empty")
	}

	volumeName := m.getVolumeName(userID)

	// Check if volume exists
	exists, err := m.VolumeExists(userID)
	if err != nil {
		return fmt.Errorf("failed to check if volume exists: %w", err)
	}

	if !exists {
		return fmt.Errorf("volume %s does not exist", volumeName)
	}

	// Get volume info to check if it's in use
	volumeInfo, err := m.GetVolumeInfo(volumeName)
	if err != nil {
		return fmt.Errorf("failed to get volume info: %w", err)
	}

	// Check if volume is in use
	// Docker returns UsageData with RefCount for volumes in use
	if volumeInfo.UsageData != nil && volumeInfo.UsageData.RefCount > 0 {
		return fmt.Errorf("volume %s is currently in use by %d container(s)", volumeName, volumeInfo.UsageData.RefCount)
	}

	// Remove volume with timeout
	ctx, cancel := context.WithTimeout(m.ctx, volumeRemoveTimeout)
	defer cancel()

	// Force removal is set to false for safety
	err = m.cli.VolumeRemove(ctx, volumeName, false)
	if err != nil {
		return fmt.Errorf("failed to remove volume %s: %w", volumeName, err)
	}

	slog.Info("[Container] Volume removed", "volume", volumeName)
	return nil
}

// VolumeExists checks if a volume exists for the given user
func (m *ContainerManager) VolumeExists(userID string) (bool, error) {
	if userID == "" {
		return false, fmt.Errorf("userID cannot be empty")
	}

	volumeName := m.getVolumeName(userID)

	ctx, cancel := context.WithTimeout(m.ctx, volumeInspectTimeout)
	defer cancel()

	_, err := m.cli.VolumeInspect(ctx, volumeName)
	if err != nil {
		// If volume doesn't exist, Docker returns an error
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to inspect volume %s: %w", volumeName, err)
	}

	return true, nil
}

// ListUserVolumes lists all volumes associated with a specific user
func (m *ContainerManager) ListUserVolumes(userID string) ([]*volume.Volume, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	ctx, cancel := context.WithTimeout(m.ctx, volumeListTimeout)
	defer cancel()

	// Build filters to find volumes for this user
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("%s=%s", labelApp, labelAppValue))
	filterArgs.Add("label", fmt.Sprintf("%s=%s", labelUserID, userID))

	// List volumes
	volumeListOptions := volume.ListOptions{
		Filters: filterArgs,
	}

	volumeList, err := m.cli.VolumeList(ctx, volumeListOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes for user %s: %w", userID, err)
	}

	slog.Info("[Container] Listed volumes", "count", len(volumeList.Volumes), "user_id", userID)
	return volumeList.Volumes, nil
}

// GetVolumeInfo retrieves detailed information about a specific volume
func (m *ContainerManager) GetVolumeInfo(volumeName string) (volume.Volume, error) {
	if volumeName == "" {
		return volume.Volume{}, fmt.Errorf("volumeName cannot be empty")
	}

	ctx, cancel := context.WithTimeout(m.ctx, volumeInspectTimeout)
	defer cancel()

	vol, err := m.cli.VolumeInspect(ctx, volumeName)
	if err != nil {
		if client.IsErrNotFound(err) {
			return volume.Volume{}, fmt.Errorf("volume %s not found", volumeName)
		}
		return volume.Volume{}, fmt.Errorf("failed to inspect volume %s: %w", volumeName, err)
	}

	return vol, nil
}

// ListAllWorkspaceVolumes lists all workspace volumes in the system
func (m *ContainerManager) ListAllWorkspaceVolumes() ([]*volume.Volume, error) {
	ctx, cancel := context.WithTimeout(m.ctx, volumeListTimeout)
	defer cancel()

	// Build filters to find all workspace volumes
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("%s=%s", labelApp, labelAppValue))
	filterArgs.Add("label", fmt.Sprintf("%s=%s", labelType, defaultVolumeType))

	// List volumes
	volumeListOptions := volume.ListOptions{
		Filters: filterArgs,
	}

	volumeList, err := m.cli.VolumeList(ctx, volumeListOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list workspace volumes: %w", err)
	}

	slog.Info("[Container] Listed workspace volumes", "count", len(volumeList.Volumes))
	return volumeList.Volumes, nil
}

// PruneUnusedVolumes removes all unused workspace volumes
// Returns the list of removed volume names and any error
func (m *ContainerManager) PruneUnusedVolumes() ([]string, error) {
	ctx, cancel := context.WithTimeout(m.ctx, volumeRemoveTimeout)
	defer cancel()

	// Build filters to only prune workspace volumes
	filterArgs := filters.NewArgs()
	filterArgs.Add("label", fmt.Sprintf("%s=%s", labelApp, labelAppValue))
	filterArgs.Add("label", fmt.Sprintf("%s=%s", labelType, defaultVolumeType))

	pruneReport, err := m.cli.VolumesPrune(ctx, filterArgs)
	if err != nil {
		return nil, fmt.Errorf("failed to prune volumes: %w", err)
	}

	slog.Info("[Container] Pruned unused volumes", "count", len(pruneReport.VolumesDeleted))
	return pruneReport.VolumesDeleted, nil
}

// GetVolumeStats returns statistics about workspace volumes
func (m *ContainerManager) GetVolumeStats() (*VolumeStats, error) {
	volumes, err := m.ListAllWorkspaceVolumes()
	if err != nil {
		return nil, err
	}

	stats := &VolumeStats{
		TotalVolumes:  len(volumes),
		VolumesInUse:  0,
		UnusedVolumes: 0,
	}

	for _, vol := range volumes {
		if vol.UsageData != nil && vol.UsageData.RefCount > 0 {
			stats.VolumesInUse++
		} else {
			stats.UnusedVolumes++
		}
	}

	return stats, nil
}

// Helper functions

// getVolumeName generates a volume name from userID
func (m *ContainerManager) getVolumeName(userID string) string {
	return fmt.Sprintf("%s%s", volumePrefix, userID)
}

// VolumeStats holds statistics about workspace volumes
type VolumeStats struct {
	TotalVolumes  int `json:"total_volumes"`
	VolumesInUse  int `json:"volumes_in_use"`
	UnusedVolumes int `json:"unused_volumes"`
}
