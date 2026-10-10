// Package services provides business logic services for the application.
// sandbox_quota.go manages per-user sandbox resource quotas.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// Quota enforcement errors
var (
	ErrQuotaExceeded       = errors.New("quota exceeded")
	ErrMemoryLimitExceeded = errors.New("memory limit exceeded")
	ErrCPULimitExceeded    = errors.New("CPU limit exceeded")
	ErrStorageLimitExceeded = errors.New("storage limit exceeded")
	ErrInvalidQuotaRequest  = errors.New("invalid quota request")
)

// QuotaRequest represents a resource request that needs quota validation.
type QuotaRequest struct {
	// Number of sandboxes being requested (typically 1)
	SandboxCount int
	// Memory requested per sandbox (bytes)
	MemoryPerSandbox int64
	// CPU quota per sandbox (100000 = 1 CPU)
	CPUPerSandbox int64
	// Storage requested per sandbox (bytes)
	StoragePerSandbox int64
}

// UserQuota defines the limits for a user.
type UserQuota struct {
	UserID uuid.UUID

	// Max concurrent sandboxes
	MaxSandboxes int
	// Max memory per sandbox (bytes)
	MaxMemoryPerSandbox int64
	// Max CPU per sandbox (100000 = 1 CPU)
	MaxCPUPerSandbox int64
	// Total memory across all sandboxes (bytes)
	MaxTotalMemory int64
	// Total storage for workspaces (bytes)
	MaxTotalStorage int64

	// Custom overrides (set by admin)
	IsOverride bool
}

// QuotaUsage represents current resource usage for a user.
type QuotaUsage struct {
	UserID uuid.UUID

	// Current number of running sandboxes
	CurrentSandboxes int
	// Total memory currently allocated (bytes)
	CurrentTotalMemory int64
	// Total CPU currently allocated
	CurrentTotalCPU int64
	// Total storage used (bytes)
	CurrentTotalStorage int64

	// Per-sandbox details
	Sandboxes []SandboxResourceUsage
}

// SandboxResourceUsage tracks resources for a single sandbox.
type SandboxResourceUsage struct {
	AppID       uuid.UUID
	AppName     string
	Memory      int64
	CPU         int64
	Storage     int64
	Status      SandboxStatus
}

// QuotaService manages per-user sandbox quotas.
type QuotaService struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	config  *config.Config
	logger  *slog.Logger
	mu      sync.RWMutex

	// In-memory cache of quota overrides (admin-set)
	quotaOverrides map[uuid.UUID]*UserQuota
}

// NewQuotaService creates a new quota management service.
func NewQuotaService(
	pool *pgxpool.Pool,
	cfg *config.Config,
	logger *slog.Logger,
) *QuotaService {
	return &QuotaService{
		pool:           pool,
		queries:        sqlc.New(pool),
		config:         cfg,
		logger:         logger.With("service", "sandbox_quota"),
		quotaOverrides: make(map[uuid.UUID]*UserQuota),
	}
}

// CheckQuota validates if a user can allocate the requested resources.
// Returns nil if quota allows the request, otherwise returns a quota error.
func (s *QuotaService) CheckQuota(ctx context.Context, userID uuid.UUID, requested QuotaRequest) error {
	if requested.SandboxCount < 0 || requested.MemoryPerSandbox < 0 ||
	   requested.CPUPerSandbox < 0 || requested.StoragePerSandbox < 0 {
		return ErrInvalidQuotaRequest
	}

	// Get user's quota limits
	quota, err := s.GetUserQuota(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get user quota, using defaults",
			"user_id", userID,
			"error", err)
		quota = s.getDefaultQuota(userID)
	}

	// Get current usage
	usage, err := s.GetUserUsage(ctx, userID)
	if err != nil {
		s.logger.Warn("failed to get user usage",
			"user_id", userID,
			"error", err)
		// Continue with check - if we can't get usage, be permissive
		usage = &QuotaUsage{UserID: userID}
	}

	// Check 1: Max sandboxes count
	newSandboxCount := usage.CurrentSandboxes + requested.SandboxCount
	if newSandboxCount > quota.MaxSandboxes {
		s.logger.Info("sandbox count quota exceeded",
			"user_id", userID,
			"current", usage.CurrentSandboxes,
			"requested", requested.SandboxCount,
			"max", quota.MaxSandboxes)
		return fmt.Errorf("%w: sandboxes (current: %d, requested: %d, max: %d)",
			ErrMaxSandboxesReached,
			usage.CurrentSandboxes,
			requested.SandboxCount,
			quota.MaxSandboxes)
	}

	// Check 2: Memory per sandbox limit
	if requested.MemoryPerSandbox > quota.MaxMemoryPerSandbox {
		s.logger.Info("per-sandbox memory limit exceeded",
			"user_id", userID,
			"requested", requested.MemoryPerSandbox,
			"max", quota.MaxMemoryPerSandbox)
		return fmt.Errorf("%w: per-sandbox (requested: %d bytes, max: %d bytes)",
			ErrMemoryLimitExceeded,
			requested.MemoryPerSandbox,
			quota.MaxMemoryPerSandbox)
	}

	// Check 3: Total memory limit
	newTotalMemory := usage.CurrentTotalMemory + (requested.MemoryPerSandbox * int64(requested.SandboxCount))
	if newTotalMemory > quota.MaxTotalMemory {
		s.logger.Info("total memory quota exceeded",
			"user_id", userID,
			"current", usage.CurrentTotalMemory,
			"requested", requested.MemoryPerSandbox * int64(requested.SandboxCount),
			"max", quota.MaxTotalMemory)
		return fmt.Errorf("%w: total memory (current: %d bytes, new total: %d bytes, max: %d bytes)",
			ErrMemoryLimitExceeded,
			usage.CurrentTotalMemory,
			newTotalMemory,
			quota.MaxTotalMemory)
	}

	// Check 4: CPU per sandbox limit
	if requested.CPUPerSandbox > quota.MaxCPUPerSandbox {
		s.logger.Info("per-sandbox CPU limit exceeded",
			"user_id", userID,
			"requested", requested.CPUPerSandbox,
			"max", quota.MaxCPUPerSandbox)
		return fmt.Errorf("%w: per-sandbox (requested: %d, max: %d)",
			ErrCPULimitExceeded,
			requested.CPUPerSandbox,
			quota.MaxCPUPerSandbox)
	}

	// Check 5: Storage limit
	newTotalStorage := usage.CurrentTotalStorage + (requested.StoragePerSandbox * int64(requested.SandboxCount))
	if newTotalStorage > quota.MaxTotalStorage {
		s.logger.Info("storage quota exceeded",
			"user_id", userID,
			"current", usage.CurrentTotalStorage,
			"requested", requested.StoragePerSandbox * int64(requested.SandboxCount),
			"max", quota.MaxTotalStorage)
		return fmt.Errorf("%w: (current: %d bytes, new total: %d bytes, max: %d bytes)",
			ErrStorageLimitExceeded,
			usage.CurrentTotalStorage,
			newTotalStorage,
			quota.MaxTotalStorage)
	}

	s.logger.Debug("quota check passed",
		"user_id", userID,
		"requested_sandboxes", requested.SandboxCount,
		"current_sandboxes", usage.CurrentSandboxes,
		"max_sandboxes", quota.MaxSandboxes)

	return nil
}

// GetUserQuota retrieves the quota limits for a user.
// Returns admin overrides if set, otherwise returns default quota.
func (s *QuotaService) GetUserQuota(ctx context.Context, userID uuid.UUID) (*UserQuota, error) {
	// Check in-memory overrides first
	s.mu.RLock()
	if override, exists := s.quotaOverrides[userID]; exists {
		s.mu.RUnlock()
		s.logger.Debug("using quota override",
			"user_id", userID,
			"max_sandboxes", override.MaxSandboxes)
		return override, nil
	}
	s.mu.RUnlock()

	// TODO: In future, check database for persisted quota overrides
	// For now, return default quota
	quota := s.getDefaultQuota(userID)

	s.logger.Debug("using default quota",
		"user_id", userID,
		"max_sandboxes", quota.MaxSandboxes)

	return quota, nil
}

// GetUserUsage retrieves current resource usage for a user.
func (s *QuotaService) GetUserUsage(ctx context.Context, userID uuid.UUID) (*QuotaUsage, error) {
	if s.pool == nil {
		// Without database, return empty usage
		return &QuotaUsage{UserID: userID}, nil
	}

	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}

	// Get count of running sandboxes
	count, err := s.queries.CountUserRunningSandboxes(ctx, pgUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to count running sandboxes: %w", err)
	}

	// Get all sandboxes for this user
	dbSandboxes, err := s.queries.ListUserSandboxes(ctx, pgUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user sandboxes: %w", err)
	}

	usage := &QuotaUsage{
		UserID:           userID,
		CurrentSandboxes: int(count),
		Sandboxes:        make([]SandboxResourceUsage, 0, len(dbSandboxes)),
	}

	// Calculate total resource usage
	for _, db := range dbSandboxes {
		if !db.ID.Valid {
			continue
		}

		// Only count running or building sandboxes
		status := SandboxStatusPending
		if db.SandboxStatus != nil {
			status = SandboxStatus(*db.SandboxStatus)
		}

		if status != SandboxStatusRunning && status != SandboxStatusBuilding {
			continue
		}

		sandboxUsage := SandboxResourceUsage{
			AppID:   uuid.UUID(db.ID.Bytes),
			AppName: db.Name,
			Status:  status,
		}

		// Get memory limit from database (stored in container_config)
		// For now, use default values as container_config is not in schema
		// TODO: Store and retrieve actual container resource limits
		sandboxUsage.Memory = s.config.SandboxDefaultMemory
		sandboxUsage.CPU = int64(s.config.SandboxDefaultCPU)

		usage.CurrentTotalMemory += sandboxUsage.Memory
		usage.CurrentTotalCPU += sandboxUsage.CPU
		usage.Sandboxes = append(usage.Sandboxes, sandboxUsage)
	}

	s.logger.Debug("calculated user usage",
		"user_id", userID,
		"sandboxes", usage.CurrentSandboxes,
		"total_memory", usage.CurrentTotalMemory,
		"total_cpu", usage.CurrentTotalCPU)

	return usage, nil
}

// SetUserQuotaOverride sets custom quota limits for a user (admin only).
// This override is stored in-memory and takes precedence over default quotas.
func (s *QuotaService) SetUserQuotaOverride(ctx context.Context, userID uuid.UUID, quota UserQuota) error {
	quota.UserID = userID
	quota.IsOverride = true

	s.mu.Lock()
	s.quotaOverrides[userID] = &quota
	s.mu.Unlock()

	s.logger.Info("set quota override",
		"user_id", userID,
		"max_sandboxes", quota.MaxSandboxes,
		"max_memory_per_sandbox", quota.MaxMemoryPerSandbox,
		"max_total_memory", quota.MaxTotalMemory)

	// TODO: Persist to database for durability across restarts

	return nil
}

// RemoveUserQuotaOverride removes custom quota limits for a user.
func (s *QuotaService) RemoveUserQuotaOverride(ctx context.Context, userID uuid.UUID) error {
	s.mu.Lock()
	delete(s.quotaOverrides, userID)
	s.mu.Unlock()

	s.logger.Info("removed quota override",
		"user_id", userID)

	// TODO: Remove from database

	return nil
}

// getDefaultQuota returns the default quota limits from config.
func (s *QuotaService) getDefaultQuota(userID uuid.UUID) *UserQuota {
	// Default values
	maxSandboxes := 5
	maxMemoryPerSandbox := int64(512 * 1024 * 1024) // 512MB
	maxCPUPerSandbox := int64(50000)                // 50% of 1 CPU
	maxTotalMemory := int64(2 * 1024 * 1024 * 1024) // 2GB total
	maxTotalStorage := int64(5 * 1024 * 1024 * 1024) // 5GB total

	// Override from config if set
	if s.config != nil {
		if s.config.SandboxMaxPerUser > 0 {
			maxSandboxes = s.config.SandboxMaxPerUser
		}
		if s.config.SandboxDefaultMemory > 0 {
			maxMemoryPerSandbox = s.config.SandboxDefaultMemory
		}
		if s.config.SandboxDefaultCPU > 0 {
			maxCPUPerSandbox = int64(s.config.SandboxDefaultCPU)
		}
		if s.config.SandboxMaxTotalMemory > 0 {
			maxTotalMemory = s.config.SandboxMaxTotalMemory
		}
		if s.config.SandboxMaxTotalStorage > 0 {
			maxTotalStorage = s.config.SandboxMaxTotalStorage
		}
	}

	return &UserQuota{
		UserID:              userID,
		MaxSandboxes:        maxSandboxes,
		MaxMemoryPerSandbox: maxMemoryPerSandbox,
		MaxCPUPerSandbox:    maxCPUPerSandbox,
		MaxTotalMemory:      maxTotalMemory,
		MaxTotalStorage:     maxTotalStorage,
		IsOverride:          false,
	}
}

// GetStats returns quota service statistics.
func (s *QuotaService) GetStats() map[string]interface{} {
	s.mu.RLock()
	overrideCount := len(s.quotaOverrides)
	s.mu.RUnlock()

	return map[string]interface{}{
		"quota_overrides_count": overrideCount,
		"default_max_sandboxes": s.config.SandboxMaxPerUser,
		"default_max_memory":    s.config.SandboxDefaultMemory,
	}
}

// ValidateQuotaRequest performs basic validation on a quota request.
func ValidateQuotaRequest(req QuotaRequest) error {
	if req.SandboxCount < 0 {
		return fmt.Errorf("%w: sandbox count must be non-negative", ErrInvalidQuotaRequest)
	}
	if req.MemoryPerSandbox < 0 {
		return fmt.Errorf("%w: memory must be non-negative", ErrInvalidQuotaRequest)
	}
	if req.CPUPerSandbox < 0 {
		return fmt.Errorf("%w: CPU must be non-negative", ErrInvalidQuotaRequest)
	}
	if req.StoragePerSandbox < 0 {
		return fmt.Errorf("%w: storage must be non-negative", ErrInvalidQuotaRequest)
	}

	// Sanity check: memory per sandbox should be reasonable (between 64MB and 8GB)
	if req.MemoryPerSandbox > 0 && (req.MemoryPerSandbox < 64*1024*1024 || req.MemoryPerSandbox > 8*1024*1024*1024) {
		return fmt.Errorf("%w: memory per sandbox must be between 64MB and 8GB", ErrInvalidQuotaRequest)
	}

	// Sanity check: CPU quota should be reasonable (between 10% and 400% of 1 CPU)
	if req.CPUPerSandbox > 0 && (req.CPUPerSandbox < 10000 || req.CPUPerSandbox > 400000) {
		return fmt.Errorf("%w: CPU quota must be between 10000 (10%%) and 400000 (400%%)", ErrInvalidQuotaRequest)
	}

	return nil
}
