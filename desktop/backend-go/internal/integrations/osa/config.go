package osa

import (
	"fmt"
	"time"
)

// Config holds configuration for the OSA client
type Config struct {
	// BaseURL is the base URL for the OSA API (e.g., "http://localhost:8089")
	BaseURL string

	// SharedSecret is the JWT secret shared between BusinessOS and OSA
	SharedSecret string

	// Timeout for API requests
	Timeout time.Duration

	// MaxRetries for failed requests
	MaxRetries int

	// RetryDelay between retries
	RetryDelay time.Duration
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		BaseURL:      "http://localhost:8089",
		Timeout:      30 * time.Second,
		MaxRetries:   3,
		RetryDelay:   2 * time.Second,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.BaseURL == "" {
		return fmt.Errorf("OSA base URL is required")
	}

	if c.SharedSecret == "" {
		return fmt.Errorf("OSA shared secret is required")
	}

	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}

	return nil
}
