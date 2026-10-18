package services

import (
	"testing"

	"github.com/rhl/businessos-backend/internal/config"
)

func TestNewAnthropicServiceDefaultModel(t *testing.T) {
	svc := NewAnthropicService(&config.Config{}, "")

	if svc.model != "claude-sonnet-5" {
		t.Fatalf("model = %q, want claude-sonnet-5", svc.model)
	}
}
