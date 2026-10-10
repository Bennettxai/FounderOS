package config

import (
	"strings"
	"testing"
)

func productionConfigForTest() *Config {
	return &Config{
		Environment:          "production",
		SecretKey:            strings.Repeat("s", 64),
		TokenEncryptionKey:   strings.Repeat("t", 32),
		RedisKeyHMACSecret:   strings.Repeat("r", 32),
		InternalAPISecret:    strings.Repeat("i", 32),
		WebhookSigningSecret: strings.Repeat("w", 32),
		DatabaseURL:          "postgres://user:pass@db.example.com/businessos",
		AllowedOrigins:       []string{"https://app.example.com"},
	}
}

func TestValidateProductionAcceptsCompleteSecureConfig(t *testing.T) {
	cfg := productionConfigForTest()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid production config, got %v", err)
	}
}

func TestValidateProductionRejectsMissingInternalAPISecret(t *testing.T) {
	cfg := productionConfigForTest()
	cfg.InternalAPISecret = ""

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "INTERNAL_API_SECRET") {
		t.Fatalf("expected INTERNAL_API_SECRET error, got %v", err)
	}
}

func TestValidateProductionRejectsWildcardOrigins(t *testing.T) {
	cfg := productionConfigForTest()
	cfg.AllowedOrigins = []string{"*"}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "ALLOWED_ORIGINS") {
		t.Fatalf("expected ALLOWED_ORIGINS error, got %v", err)
	}
}
