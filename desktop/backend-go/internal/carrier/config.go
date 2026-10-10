// Package carrier implements the CARRIER protocol bridge, connecting the BOS
// SORX engine to the Elixir SorxMain reasoning engine over AMQP 0-9-1.
package carrier

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the CARRIER AMQP client.
type Config struct {
	// URL is the AMQP connection string.
	// Env: CARRIER_AMQP_URL (e.g. amqp://guest:guest@localhost:5672/)
	URL string

	// Exchange is the AMQP topic exchange name.
	// Defaults to "sorx.carrier".
	Exchange string

	// OSInstanceID uniquely identifies this BOS instance.
	// Used to derive the per-OS reply queue name.
	// Env: OS_INSTANCE_ID (required)
	OSInstanceID string

	// SendTimeout is the maximum time to wait for a synchronous response.
	// Env: CARRIER_SEND_TIMEOUT (e.g. "60s"). Defaults to 60s.
	SendTimeout time.Duration

	// Prefetch controls how many unacknowledged messages the broker delivers
	// to the consumer channel at once. Defaults to 10.
	Prefetch int

	// Enabled is a feature flag that disables CARRIER when false.
	// SORX falls back to local LLM calls when disabled.
	// Env: CARRIER_ENABLED (default: false)
	Enabled bool
}

// DefaultConfig returns a Config populated with safe defaults.
// Callers should override URL and OSInstanceID for their environment.
func DefaultConfig() Config {
	return Config{
		URL:         "amqp://guest:guest@localhost:5672/",
		Exchange:    "sorx.carrier",
		SendTimeout: 60 * time.Second,
		Prefetch:    10,
		Enabled:     false,
	}
}

// ConfigFromEnv reads CARRIER configuration from environment variables,
// merging over the supplied base. Missing optional variables retain their
// base values; missing required variables cause an error when Enabled is true.
func ConfigFromEnv(base Config) (Config, error) {
	cfg := base

	if v := os.Getenv("CARRIER_AMQP_URL"); v != "" {
		cfg.URL = v
	}

	if v := os.Getenv("OS_INSTANCE_ID"); v != "" {
		cfg.OSInstanceID = v
	}

	if v := os.Getenv("CARRIER_ENABLED"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return cfg, fmt.Errorf("carrier: invalid CARRIER_ENABLED value %q: %w", v, err)
		}
		cfg.Enabled = enabled
	}

	if v := os.Getenv("CARRIER_SEND_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return cfg, fmt.Errorf("carrier: invalid CARRIER_SEND_TIMEOUT value %q: %w", v, err)
		}
		cfg.SendTimeout = d
	}

	if v := os.Getenv("CARRIER_PREFETCH"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return cfg, fmt.Errorf("carrier: invalid CARRIER_PREFETCH value %q: must be a positive integer", v)
		}
		cfg.Prefetch = n
	}

	if cfg.Enabled && cfg.OSInstanceID == "" {
		return cfg, fmt.Errorf("carrier: OS_INSTANCE_ID is required when CARRIER_ENABLED=true")
	}

	return cfg, nil
}

// replyQueueName returns the per-OS reply queue name derived from the instance ID.
func replyQueueName(osInstanceID string) string {
	return "sorx.responses." + osInstanceID
}
