package config

import "testing"

func TestGetActiveModelAnthropicDefault(t *testing.T) {
	cfg := &Config{AIProvider: "anthropic", AnthropicAPIKey: "sk-test"}

	if got := cfg.GetActiveModel(); got != "claude-sonnet-5" {
		t.Fatalf("GetActiveModel() = %q, want claude-sonnet-5", got)
	}
}
