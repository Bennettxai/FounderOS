package osa

import (
	"fmt"
)

// NewCloudClient creates an OSA client backed by MIOSA Cloud (api.miosa.ai).
// The cloudConfig must supply a non-empty APIKey. BaseURL defaults to
// https://api.miosa.ai if left blank.
func NewCloudClient(cloudConfig *CloudConfig) (*Client, error) {
	if err := cloudConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid cloud config: %w", err)
	}

	baseURL := cloudConfig.BaseURL
	if baseURL == "" {
		baseURL = "https://api.miosa.ai"
	}

	runtimeConfig := &Config{
		BaseURL:      baseURL,
		SharedSecret: cloudConfig.APIKey,
		Timeout:      cloudConfig.Timeout,
		MaxRetries:   cloudConfig.MaxRetries,
		RetryDelay:   cloudConfig.RetryDelay,
	}

	return newHTTPClient(runtimeConfig, cloudConfig.APIKey), nil
}
