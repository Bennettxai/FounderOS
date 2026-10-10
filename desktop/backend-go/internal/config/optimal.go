package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// OptimalConfig holds configuration for the Optimal reasoning engine integration.
// Optimal is the cloud-side SORX orchestration layer that dispatches skills to
// BOS instances and receives action results back via CARRIER.
type OptimalConfig struct {
	// Enabled controls whether this template connects to Optimal.
	// When false, all reasoning happens locally via the on-instance SORX engine.
	// Env: OPTIMAL_ENABLED (default: false)
	Enabled bool

	// Mode determines how the template interacts with Optimal.
	// "full"   — SORX skills execute on Optimal; local actions dispatched back.
	// "hybrid" — Tier 1–2 execute locally; Tier 3–4 routed via Optimal.
	// "local"  — Everything local; Optimal disabled regardless of Enabled flag.
	// Env: OPTIMAL_MODE (default: "hybrid")
	Mode string

	// InstalledModules lists active modules for this workspace.
	// Reported to Optimal during registration so it can tailor skill execution.
	// Env: OPTIMAL_MODULES (comma-separated, e.g. "crm,projects,calendar")
	InstalledModules []string

	// Capabilities lists supported integrations.
	// Reported to Optimal so it knows which execute_action commands this instance
	// can handle (e.g. ["gmail", "slack", "hubspot"]).
	// Env: OPTIMAL_CAPABILITIES (comma-separated, e.g. "gmail,slack,hubspot")
	Capabilities []string

	// HeartbeatInterval is how often to send heartbeats to Optimal.
	// Env: OPTIMAL_HEARTBEAT_INTERVAL (Go duration, e.g. "30s"). Default: 30s.
	HeartbeatInterval time.Duration

	// TemplateType identifies this template type to Optimal.
	// Env: OPTIMAL_TEMPLATE_TYPE (default: "bos")
	TemplateType string
}

// DefaultOptimalConfig returns safe defaults. Optimal is disabled by default;
// operators must explicitly opt in via OPTIMAL_ENABLED=true.
func DefaultOptimalConfig() OptimalConfig {
	return OptimalConfig{
		Enabled:           false,
		Mode:              "hybrid",
		InstalledModules:  []string{},
		Capabilities:      []string{},
		HeartbeatInterval: 30 * time.Second,
		TemplateType:      "bos",
	}
}

// OptimalConfigFromEnv reads Optimal configuration from environment variables,
// merging over the supplied base config. Missing optional variables retain
// their base values. Returns an error for invalid values.
//
// Environment variables:
//
//	OPTIMAL_ENABLED              — bool (true/false/1/0)
//	OPTIMAL_MODE                 — "full" | "hybrid" | "local"
//	OPTIMAL_MODULES              — comma-separated list of module names
//	OPTIMAL_CAPABILITIES         — comma-separated list of integration names
//	OPTIMAL_HEARTBEAT_INTERVAL   — Go duration string (e.g. "30s", "1m")
//	OPTIMAL_TEMPLATE_TYPE        — string (e.g. "bos", "custom_os")
func OptimalConfigFromEnv(base OptimalConfig) (OptimalConfig, error) {
	cfg := base

	if v := os.Getenv("OPTIMAL_ENABLED"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return cfg, fmt.Errorf("optimal: invalid OPTIMAL_ENABLED value %q: %w", v, err)
		}
		cfg.Enabled = enabled
	}

	if v := os.Getenv("OPTIMAL_MODE"); v != "" {
		switch v {
		case "full", "hybrid", "local":
			cfg.Mode = v
		default:
			return cfg, fmt.Errorf("optimal: invalid OPTIMAL_MODE value %q: must be one of full, hybrid, local", v)
		}
	}

	if v := os.Getenv("OPTIMAL_MODULES"); v != "" {
		cfg.InstalledModules = splitTrimmed(v)
	}

	if v := os.Getenv("OPTIMAL_CAPABILITIES"); v != "" {
		cfg.Capabilities = splitTrimmed(v)
	}

	if v := os.Getenv("OPTIMAL_HEARTBEAT_INTERVAL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return cfg, fmt.Errorf("optimal: invalid OPTIMAL_HEARTBEAT_INTERVAL value %q: %w", v, err)
		}
		if d <= 0 {
			return cfg, fmt.Errorf("optimal: OPTIMAL_HEARTBEAT_INTERVAL must be positive, got %q", v)
		}
		cfg.HeartbeatInterval = d
	}

	if v := os.Getenv("OPTIMAL_TEMPLATE_TYPE"); v != "" {
		cfg.TemplateType = v
	}

	// "local" mode is equivalent to disabled regardless of the Enabled flag.
	if cfg.Mode == "local" {
		cfg.Enabled = false
	}

	return cfg, nil
}

// splitTrimmed splits a comma-separated string and trims whitespace from each
// element, dropping empty entries.
func splitTrimmed(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
