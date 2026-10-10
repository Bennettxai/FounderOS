// Package services provides business logic services for the application.
// port_allocator.go manages dynamic port allocation for sandbox containers.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// Redis key patterns for port allocation
const (
	// sandbox:port:{port} -> app_id (TTL: 24h)
	redisPortKeyPrefix = "sandbox:port:"
	// sandbox:app:{app_id} -> port (TTL: 24h)
	redisAppKeyPrefix = "sandbox:app:"
	// TTL for Redis port allocation keys
	portAllocationTTL = 24 * time.Hour
)

// Errors for port allocation
var (
	ErrPortsExhausted = errors.New("no available ports in configured range")
	ErrPortInUse      = errors.New("port is already in use")
	ErrAppNotFound    = errors.New("app not found or has no allocated port")
	ErrInvalidConfig  = errors.New("invalid port allocator configuration")
)

// SandboxPortAllocator manages dynamic port allocation for sandbox containers.
// It uses Redis for fast distributed lookups with database fallback for persistence.
type SandboxPortAllocator struct {
	redisClient *redis.Client
	pool        *pgxpool.Pool
	queries     *sqlc.Queries
	minPort     int
	maxPort     int
	logger      *slog.Logger
	mu          sync.Mutex

	// In-memory cache for faster local lookups
	// Maps port -> app_id and app_id -> port
	portToApp map[int]uuid.UUID
	appToPort map[uuid.UUID]int
}

// NewSandboxPortAllocator creates a new port allocator with the given configuration.
func NewSandboxPortAllocator(
	redisClient *redis.Client,
	pool *pgxpool.Pool,
	cfg *config.Config,
	logger *slog.Logger,
) (*SandboxPortAllocator, error) {
	if cfg.SandboxPortMin <= 0 || cfg.SandboxPortMax <= 0 {
		return nil, fmt.Errorf("%w: port range must be positive", ErrInvalidConfig)
	}
	if cfg.SandboxPortMin >= cfg.SandboxPortMax {
		return nil, fmt.Errorf("%w: min port must be less than max port", ErrInvalidConfig)
	}
	if cfg.SandboxPortMax-cfg.SandboxPortMin < 10 {
		return nil, fmt.Errorf("%w: port range must have at least 10 ports", ErrInvalidConfig)
	}

	pa := &SandboxPortAllocator{
		redisClient: redisClient,
		pool:        pool,
		queries:     sqlc.New(pool),
		minPort:     cfg.SandboxPortMin,
		maxPort:     cfg.SandboxPortMax,
		logger:      logger.With("service", "port_allocator"),
		portToApp:   make(map[int]uuid.UUID),
		appToPort:   make(map[uuid.UUID]int),
	}

	return pa, nil
}

// Allocate assigns an available port to the given app.
// Uses Redis SETNX for distributed locking, with database persistence.
func (p *SandboxPortAllocator) Allocate(ctx context.Context, appID uuid.UUID) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if app already has a port
	if existingPort, ok := p.appToPort[appID]; ok {
		p.logger.Debug("app already has allocated port", "app_id", appID, "port", existingPort)
		return existingPort, nil
	}

	// Try to find an available port
	for port := p.minPort; port <= p.maxPort; port++ {
		if p.tryAllocatePort(ctx, port, appID) {
			p.logger.Info("port allocated",
				"app_id", appID,
				"port", port,
				"range", fmt.Sprintf("%d-%d", p.minPort, p.maxPort))
			return port, nil
		}
	}

	return 0, ErrPortsExhausted
}

// tryAllocatePort attempts to allocate a specific port to an app.
// Returns true if successful, false if port is already in use.
func (p *SandboxPortAllocator) tryAllocatePort(ctx context.Context, port int, appID uuid.UUID) bool {
	// Check local cache first
	if _, inUse := p.portToApp[port]; inUse {
		return false
	}

	// Try Redis SETNX for distributed lock
	if p.redisClient != nil {
		portKey := fmt.Sprintf("%s%d", redisPortKeyPrefix, port)
		appKey := fmt.Sprintf("%s%s", redisAppKeyPrefix, appID.String())

		// Use SETNX (SET if Not eXists) for atomic allocation
		set, err := p.redisClient.SetNX(ctx, portKey, appID.String(), portAllocationTTL).Result()
		if err != nil {
			p.logger.Warn("Redis SETNX failed, falling back to DB",
				"port", port,
				"error", err)
		} else if !set {
			// Port already allocated in Redis
			return false
		} else {
			// Also set reverse mapping
			if err := p.redisClient.Set(ctx, appKey, port, portAllocationTTL).Err(); err != nil {
				p.logger.Warn("failed to set app->port mapping in Redis",
					"app_id", appID,
					"port", port,
					"error", err)
			}
		}
	}

	// Check database for existing allocation (skip if no pool)
	if p.pool != nil {
		port32 := int32(port)
		dbApp, err := p.queries.GetSandboxByPort(ctx, &port32)
		if err == nil && dbApp.ID.Valid {
			// Port is already allocated in database
			return false
		}
	}

	// Update local cache
	p.portToApp[port] = appID
	p.appToPort[appID] = port

	// Log event to database (skip if no pool)
	if p.pool != nil {
		p.logEvent(ctx, appID, "port_allocated", port, nil)
	}

	return true
}

// Release frees the port allocated to the given app.
func (p *SandboxPortAllocator) Release(ctx context.Context, appID uuid.UUID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	port, ok := p.appToPort[appID]
	if !ok {
		// Try to find in Redis or DB
		var err error
		port, err = p.getPortFromStorage(ctx, appID)
		if err != nil {
			return ErrAppNotFound
		}
	}

	// Remove from Redis
	if p.redisClient != nil {
		portKey := fmt.Sprintf("%s%d", redisPortKeyPrefix, port)
		appKey := fmt.Sprintf("%s%s", redisAppKeyPrefix, appID.String())

		if err := p.redisClient.Del(ctx, portKey, appKey).Err(); err != nil {
			p.logger.Warn("failed to delete Redis keys",
				"app_id", appID,
				"port", port,
				"error", err)
		}
	}

	// Remove from local cache
	delete(p.portToApp, port)
	delete(p.appToPort, appID)

	// Log event (skip if no pool)
	if p.pool != nil {
		p.logEvent(ctx, appID, "port_released", port, nil)
	}

	p.logger.Info("port released", "app_id", appID, "port", port)
	return nil
}

// IsAvailable checks if a specific port is available for allocation.
func (p *SandboxPortAllocator) IsAvailable(ctx context.Context, port int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check range
	if port < p.minPort || port > p.maxPort {
		return false
	}

	// Check local cache
	if _, inUse := p.portToApp[port]; inUse {
		return false
	}

	// Check Redis
	if p.redisClient != nil {
		portKey := fmt.Sprintf("%s%d", redisPortKeyPrefix, port)
		exists, err := p.redisClient.Exists(ctx, portKey).Result()
		if err == nil && exists > 0 {
			return false
		}
	}

	// Check database (skip if no pool)
	if p.pool != nil {
		port32 := int32(port)
		dbApp, err := p.queries.GetSandboxByPort(ctx, &port32)
		if err == nil && dbApp.ID.Valid {
			return false
		}
	}

	return true
}

// GetPortForApp returns the port allocated to the given app.
func (p *SandboxPortAllocator) GetPortForApp(ctx context.Context, appID uuid.UUID) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check local cache
	if port, ok := p.appToPort[appID]; ok {
		return port, nil
	}

	// Check storage
	return p.getPortFromStorage(ctx, appID)
}

// getPortFromStorage retrieves port from Redis or database.
func (p *SandboxPortAllocator) getPortFromStorage(ctx context.Context, appID uuid.UUID) (int, error) {
	// Check Redis first
	if p.redisClient != nil {
		appKey := fmt.Sprintf("%s%s", redisAppKeyPrefix, appID.String())
		portStr, err := p.redisClient.Get(ctx, appKey).Result()
		if err == nil {
			var port int
			if _, err := fmt.Sscanf(portStr, "%d", &port); err == nil {
				return port, nil
			}
		}
	}

	// Check database (skip if no pool)
	if p.pool != nil {
		pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
		info, err := p.queries.GetAppSandboxInfo(ctx, pgAppID)
		if err == nil && info.SandboxPort != nil {
			return int(*info.SandboxPort), nil
		}
	}

	return 0, ErrAppNotFound
}

// RecoverFromDB recovers port allocations from the database on startup.
// This should be called once during server initialization.
func (p *SandboxPortAllocator) RecoverFromDB(ctx context.Context) error {
	// Skip if no database pool
	if p.pool == nil {
		p.logger.Warn("skipping port recovery: no database pool")
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.logger.Info("recovering port allocations from database")

	// Get all running sandboxes
	sandboxes, err := p.queries.ListRunningSandboxes(ctx)
	if err != nil {
		return fmt.Errorf("failed to list running sandboxes: %w", err)
	}

	recovered := 0
	for _, sandbox := range sandboxes {
		if sandbox.SandboxPort == nil || !sandbox.ID.Valid {
			continue
		}

		appID := uuid.UUID(sandbox.ID.Bytes)
		port := int(*sandbox.SandboxPort)

		// Update local cache
		p.portToApp[port] = appID
		p.appToPort[appID] = port

		// Update Redis if available
		if p.redisClient != nil {
			portKey := fmt.Sprintf("%s%d", redisPortKeyPrefix, port)
			appKey := fmt.Sprintf("%s%s", redisAppKeyPrefix, appID.String())

			if err := p.redisClient.Set(ctx, portKey, appID.String(), portAllocationTTL).Err(); err != nil {
				p.logger.Warn("failed to restore port to Redis",
					"port", port,
					"app_id", appID,
					"error", err)
			}
			if err := p.redisClient.Set(ctx, appKey, port, portAllocationTTL).Err(); err != nil {
				p.logger.Warn("failed to restore app mapping to Redis",
					"port", port,
					"app_id", appID,
					"error", err)
			}
		}

		recovered++
	}

	p.logger.Info("port allocations recovered",
		"count", recovered,
		"range", fmt.Sprintf("%d-%d", p.minPort, p.maxPort))

	return nil
}

// GetStats returns statistics about port allocation.
func (p *SandboxPortAllocator) GetStats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	return map[string]interface{}{
		"min_port":      p.minPort,
		"max_port":      p.maxPort,
		"total_ports":   p.maxPort - p.minPort + 1,
		"allocated":     len(p.portToApp),
		"available":     (p.maxPort - p.minPort + 1) - len(p.portToApp),
		"utilization":   float64(len(p.portToApp)) / float64(p.maxPort-p.minPort+1) * 100,
		"redis_enabled": p.redisClient != nil,
	}
}

// RefreshTTL refreshes the TTL for an app's port allocation in Redis.
// Call this periodically for long-running containers to prevent expiration.
func (p *SandboxPortAllocator) RefreshTTL(ctx context.Context, appID uuid.UUID) error {
	if p.redisClient == nil {
		return nil
	}

	p.mu.Lock()
	port, ok := p.appToPort[appID]
	p.mu.Unlock()

	if !ok {
		return ErrAppNotFound
	}

	portKey := fmt.Sprintf("%s%d", redisPortKeyPrefix, port)
	appKey := fmt.Sprintf("%s%s", redisAppKeyPrefix, appID.String())

	if err := p.redisClient.Expire(ctx, portKey, portAllocationTTL).Err(); err != nil {
		return fmt.Errorf("failed to refresh port key TTL: %w", err)
	}
	if err := p.redisClient.Expire(ctx, appKey, portAllocationTTL).Err(); err != nil {
		return fmt.Errorf("failed to refresh app key TTL: %w", err)
	}

	return nil
}

// logEvent logs a port allocation event to the database.
func (p *SandboxPortAllocator) logEvent(ctx context.Context, appID uuid.UUID, eventType string, port int, details map[string]interface{}) {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["port"] = port
	details["min_port"] = p.minPort
	details["max_port"] = p.maxPort

	detailsJSON, _ := encodeJSON(details)

	pgAppID := pgtype.UUID{Bytes: appID, Valid: true}
	_, err := p.queries.InsertSandboxEvent(ctx, sqlc.InsertSandboxEventParams{
		ModuleInstanceID: pgAppID,
		EventType:        eventType,
		ContainerID:      nil,
		Details:          detailsJSON,
	})
	if err != nil {
		p.logger.Warn("failed to log port allocation event",
			"event_type", eventType,
			"app_id", appID,
			"error", err)
	}
}

// encodeJSON converts a map to JSON bytes.
func encodeJSON(data map[string]interface{}) ([]byte, error) {
	if data == nil {
		return nil, nil
	}
	// Simple JSON encoding without importing encoding/json again
	// The sqlc queries expect []byte for JSONB
	result := "{"
	first := true
	for k, v := range data {
		if !first {
			result += ","
		}
		first = false
		result += fmt.Sprintf(`"%s":%v`, k, formatJSONValue(v))
	}
	result += "}"
	return []byte(result), nil
}

func formatJSONValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, val)
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf(`"%v"`, val)
	}
}
