package osa

import (
	"fmt"
	"time"
)

// CloudConfig holds configuration for connecting OSA to MIOSA Cloud.
// It is only used when OSA_MODE=cloud.
type CloudConfig struct {
	// APIKey is the MIOSA API key obtained from app.miosa.ai/settings/api-keys.
	// Required when mode is cloud.
	APIKey string

	// BaseURL overrides the default https://api.miosa.ai endpoint.
	// Leave blank to use the default.
	BaseURL string

	// Timeout for API requests. Defaults to 30s.
	Timeout time.Duration

	// MaxRetries for transient failures. Defaults to 3.
	MaxRetries int

	// RetryDelay between retries. Defaults to 2s.
	RetryDelay time.Duration
}

// DefaultCloudConfig returns a CloudConfig with sensible defaults.
// The APIKey must still be set by the caller.
func DefaultCloudConfig(apiKey string) *CloudConfig {
	return &CloudConfig{
		APIKey:     apiKey,
		BaseURL:    "https://api.miosa.ai",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: 2 * time.Second,
	}
}

// Validate checks that the cloud config is usable.
func (c *CloudConfig) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("MIOSA_API_KEY is required when OSA_MODE=cloud")
	}
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	return nil
}
