// Package redis provides Redis connection management for session storage,
// caching, and pub/sub messaging across horizontal scaling.
package redis

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	client *redis.Client
	once   sync.Once
)

// Config holds Redis connection configuration
type Config struct {
	URL             string        // Redis connection URL (redis://host:port/db or rediss://host:port/db for TLS)
	Password        string        // Redis password for authentication (optional)
	TLSEnabled      bool          // Enable TLS for connections (rediss://)
	TLSInsecure     bool          // Skip TLS certificate verification (dev only - NEVER in production)
	MaxRetries      int           // Max retry attempts
	PoolSize        int           // Connection pool size
	MinIdleConns    int           // Minimum idle connections
	ConnMaxIdleTime time.Duration // Max idle time before closing
	ConnMaxLifetime time.Duration // Max connection lifetime
	ReadTimeout     time.Duration // Read timeout
	WriteTimeout    time.Duration // Write timeout
}

// DefaultConfig returns production-ready default configuration
func DefaultConfig() *Config {
	return &Config{
		URL:             "redis://localhost:6379/0",
		Password:        "",
		TLSEnabled:      false,
		TLSInsecure:     false, // Always verify certificates in production
		MaxRetries:      3,
		PoolSize:        50,
		MinIdleConns:    10,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: 30 * time.Minute,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
	}
}

// Connect initializes the Redis client with the given configuration
func Connect(ctx context.Context, cfg *Config) error {
	var connectErr error

	once.Do(func() {
		opts, err := redis.ParseURL(cfg.URL)
		if err != nil {
			connectErr = fmt.Errorf("failed to parse Redis URL: %w", err)
			return
		}

		// Apply security configuration
		if cfg.Password != "" {
			opts.Password = cfg.Password
			slog.Info("Redis password authentication enabled")
		}

		// Configure TLS if enabled
		if cfg.TLSEnabled {
			tlsConfig := &tls.Config{
				MinVersion: tls.VersionTLS12, // Enforce TLS 1.2 minimum
			}

			// WARNING: InsecureSkipVerify should ONLY be used in development
			// In production, always use proper certificate validation
			if cfg.TLSInsecure {
				tlsConfig.InsecureSkipVerify = true
				slog.Warn("Redis TLS enabled with InsecureSkipVerify=true (dev only)")
			} else {
				slog.Info("Redis TLS enabled with certificate validation")
			}

			opts.TLSConfig = tlsConfig
		}

		// Apply connection pool configuration
		opts.MaxRetries = cfg.MaxRetries
		opts.PoolSize = cfg.PoolSize
		opts.MinIdleConns = cfg.MinIdleConns
		opts.ConnMaxIdleTime = cfg.ConnMaxIdleTime
		opts.ConnMaxLifetime = cfg.ConnMaxLifetime
		opts.ReadTimeout = cfg.ReadTimeout
		opts.WriteTimeout = cfg.WriteTimeout

		client = redis.NewClient(opts)

		// Test connection with authentication
		if err := client.Ping(ctx).Err(); err != nil {
			connectErr = fmt.Errorf("failed to ping Redis: %w", err)
			client = nil
			return
		}

		// Sanitize URL for logging (hide password)
		safeURL := sanitizeRedisURL(cfg.URL)
		tlsStatus := "plain"
		if cfg.TLSEnabled {
			tlsStatus = "TLS"
		}
		slog.Info("Redis connected", "url", safeURL, "pool_size", cfg.PoolSize, "protocol", tlsStatus)
	})

	return connectErr
}

// sanitizeRedisURL removes sensitive information from Redis URL for logging
func sanitizeRedisURL(url string) string {
	// Replace password in URLs like redis://:password@host:port/db
	// Keep it simple - just show the protocol and host
	if len(url) > 0 {
		if url[:8] == "rediss://" {
			return "rediss://***:***@***"
		}
		if url[:8] == "redis://" {
			return "redis://***:***@***"
		}
	}
	return "***"
}

// ConnectWithURL is a convenience function that uses default config with custom URL
func ConnectWithURL(ctx context.Context, url string) error {
	cfg := DefaultConfig()
	cfg.URL = url
	return Connect(ctx, cfg)
}

// Client returns the Redis client instance
// Returns nil if not connected
func Client() *redis.Client {
	return client
}

// Close closes the Redis connection
func Close() error {
	if client != nil {
		return client.Close()
	}
	return nil
}

// IsConnected checks if Redis is connected and responsive
func IsConnected(ctx context.Context) bool {
	if client == nil {
		return false
	}
	return client.Ping(ctx).Err() == nil
}

// HealthCheck performs a comprehensive health check
func HealthCheck(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Connected: false,
		Latency:   0,
	}

	if client == nil {
		return status, fmt.Errorf("redis client not initialized")
	}

	// Measure ping latency
	start := time.Now()
	if err := client.Ping(ctx).Err(); err != nil {
		return status, fmt.Errorf("redis ping failed: %w", err)
	}
	status.Latency = time.Since(start)
	status.Connected = true

	// Get pool stats
	poolStats := client.PoolStats()
	status.PoolStats = &PoolStats{
		Hits:       poolStats.Hits,
		Misses:     poolStats.Misses,
		Timeouts:   poolStats.Timeouts,
		TotalConns: poolStats.TotalConns,
		IdleConns:  poolStats.IdleConns,
		StaleConns: poolStats.StaleConns,
	}

	// Get server info
	info, err := client.Info(ctx, "memory", "clients", "stats").Result()
	if err == nil {
		status.ServerInfo = info
	}

	return status, nil
}

// HealthStatus contains Redis health check results
type HealthStatus struct {
	Connected  bool          `json:"connected"`
	Latency    time.Duration `json:"latency_ms"`
	PoolStats  *PoolStats    `json:"pool_stats,omitempty"`
	ServerInfo string        `json:"server_info,omitempty"`
}

// PoolStats contains connection pool statistics
type PoolStats struct {
	Hits       uint32 `json:"hits"`
	Misses     uint32 `json:"misses"`
	Timeouts   uint32 `json:"timeouts"`
	TotalConns uint32 `json:"total_conns"`
	IdleConns  uint32 `json:"idle_conns"`
	StaleConns uint32 `json:"stale_conns"`
}

// MarshalJSON customizes JSON output for HealthStatus
func (h *HealthStatus) MarshalJSON() ([]byte, error) {
	type Alias HealthStatus
	return []byte(fmt.Sprintf(`{"connected":%t,"latency_ms":%d,"pool_stats":%v}`,
		h.Connected,
		h.Latency.Milliseconds(),
		h.PoolStats,
	)), nil
}
