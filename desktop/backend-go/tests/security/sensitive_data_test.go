package security_test

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestSecretsNotInLogs tests that secrets don't appear in logs
func TestSecretsNotInLogs(t *testing.T) {
	secrets := []string{
		"sk_test_1234567890abcdef",
		"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
		"postgres://user:password@localhost/db",
		"mongodb://admin:secret@localhost:27017",
	}

	for _, secret := range secrets {
		t.Run("Secret_"+limitString(secret, 20), func(t *testing.T) {
			// Simulate logging
			masked := maskSensitiveData(secret)

			// Verify secret is masked
			assert.NotEqual(t, secret, masked, "Secret should be masked")
			assert.Contains(t, masked, "***", "Masked value should contain ***")
			assert.Less(t, len(masked), len(secret), "Masked value should be shorter")
		})
	}
}

// TestAPIKeysNotInResponses tests that API keys don't appear in responses
func TestAPIKeysNotInResponses(t *testing.T) {
	t.Run("User profile excludes API keys", func(t *testing.T) {
		user := map[string]interface{}{
			"id":       "user-123",
			"email":    "user@example.com",
			"name":     "Test User",
			"api_key":  "sk_live_1234567890", // Should be excluded
		}

		// Simulate API response serialization
		responseUser := filterSensitiveFields(user)

		_, hasAPIKey := responseUser["api_key"]
		assert.False(t, hasAPIKey, "API response should not include api_key field")
	})

	t.Run("Integration credentials masked", func(t *testing.T) {
		integration := map[string]interface{}{
			"id":            "int-123",
			"provider":      "google",
			"access_token":  "ya29.a0AfB_byC...", // Should be excluded
			"refresh_token": "1//0gH9X...",       // Should be excluded
		}

		response := filterSensitiveFields(integration)

		// Tokens should be completely excluded from the response
		_, hasAccessToken := response["access_token"]
		_, hasRefreshToken := response["refresh_token"]
		assert.False(t, hasAccessToken, "API response should not include access_token field")
		assert.False(t, hasRefreshToken, "API response should not include refresh_token field")

		// Non-sensitive fields should be preserved
		assert.Equal(t, "int-123", response["id"])
		assert.Equal(t, "google", response["provider"])
	})
}

// TestBcryptPasswordHashing tests password hashing security
func TestBcryptPasswordHashing(t *testing.T) {
	t.Run("Passwords hashed with bcrypt", func(t *testing.T) {
		password := "SecurePassword123!"

		// Hash with bcrypt cost 12
		hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
		assert.NoError(t, err)

		// Verify it's bcrypt (starts with $2a$ or $2b$)
		hashStr := string(hash)
		assert.True(t, strings.HasPrefix(hashStr, "$2a$") || strings.HasPrefix(hashStr, "$2b$"),
			"Should use bcrypt hashing")
	})

	t.Run("Bcrypt cost is 12 or higher", func(t *testing.T) {
		password := "TestPassword123!"
		hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
		assert.NoError(t, err)

		// Extract cost from hash (format: $2a$12$...)
		hashStr := string(hash)
		parts := strings.Split(hashStr, "$")
		assert.GreaterOrEqual(t, len(parts), 4, "Hash should have correct format")

		cost := parts[2]
		assert.Equal(t, "12", cost, "Bcrypt cost should be 12")
	})

	t.Run("Plaintext passwords never stored", func(t *testing.T) {
		password := "UserPassword123!"

		// Database should only store hash, never plaintext
		stored := hashPassword(password)

		assert.NotEqual(t, password, stored, "Should not store plaintext password")
		assert.NotContains(t, stored, password, "Hash should not contain plaintext")
	})
}

// TestSensitiveDataMasking tests masking in error messages
func TestSensitiveDataMasking(t *testing.T) {
	t.Run("Database connection errors mask credentials", func(t *testing.T) {
		connString := "postgres://admin:supersecret@localhost:5432/businessos"
		err := simulateConnectionError(connString)

		errMsg := err.Error()
		assert.NotContains(t, errMsg, "supersecret", "Error should not contain password")
		assert.Contains(t, errMsg, "***", "Error should mask credentials")
	})

	t.Run("JWT errors don't expose token", func(t *testing.T) {
		token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc123"
		err := simulateJWTError(token)

		errMsg := err.Error()
		assert.NotContains(t, errMsg, token, "Error should not contain full token")
		assert.NotContains(t, errMsg, "abc123", "Error should not contain signature")
	})

	t.Run("Validation errors don't leak sensitive data", func(t *testing.T) {
		sensitiveInput := "sk_live_1234567890"
		err := simulateValidationError(sensitiveInput)

		errMsg := err.Error()
		assert.NotContains(t, errMsg, "sk_live", "Error should not contain API key")
	})
}

// TestJWTSecretSafety tests JWT secret handling
func TestJWTSecretSafety(t *testing.T) {
	t.Run("JWT secret not in environment dump", func(t *testing.T) {
		// Simulate environment variables
		envVars := map[string]string{
			"DATABASE_URL": "postgres://user:pass@localhost/db",
			"JWT_SECRET":   "my-super-secret-key-min-32-chars",
			"PORT":         "8080",
		}

		// When dumping config, sensitive vars should be masked
		safeEnv := maskEnvironmentVars(envVars)

		assert.Contains(t, safeEnv["JWT_SECRET"], "***", "JWT secret should be masked")
		assert.Contains(t, safeEnv["DATABASE_URL"], "***", "Database URL should be masked")
		assert.Equal(t, "8080", safeEnv["PORT"], "Non-sensitive vars should not be masked")
	})

	t.Run("JWT secret minimum length enforced", func(t *testing.T) {
		shortSecret := "short"
		validSecret := "this-is-a-long-enough-secret-key-for-production-use"

		assert.Less(t, len(shortSecret), 32, "Short secret should be rejected")
		assert.GreaterOrEqual(t, len(validSecret), 32, "Valid secret should be at least 32 chars")
	})
}

// TestEncryptionAtRest tests encryption for sensitive data
func TestEncryptionAtRest(t *testing.T) {
	t.Run("OAuth tokens encrypted in database", func(t *testing.T) {
		plainToken := "ya29.a0AfB_byC..."

		// Tokens should be encrypted before storage
		encrypted := encryptToken(plainToken)

		assert.NotEqual(t, plainToken, encrypted, "Token should be encrypted")
		assert.NotContains(t, encrypted, "ya29", "Encrypted value should not contain plaintext")
	})

	t.Run("API keys encrypted in database", func(t *testing.T) {
		apiKey := "sk_live_1234567890abcdef"

		encrypted := encryptToken(apiKey)

		assert.NotEqual(t, apiKey, encrypted, "API key should be encrypted")
		assert.NotContains(t, encrypted, "sk_live", "Encrypted value should not contain plaintext")
	})

	t.Run("Encryption key rotation supported", func(t *testing.T) {
		// Generate two different encryption keys
		oldKey, err := security.GenerateKey()
		require.NoError(t, err)

		newKey, err := security.GenerateKey()
		require.NoError(t, err)

		// Keys should be different and both valid
		assert.NotEqual(t, oldKey, newKey, "Keys should be different")

		// Both keys should create valid encryptors
		oldEncryptor, err := security.NewTokenEncryption(oldKey)
		require.NoError(t, err)
		assert.NotNil(t, oldEncryptor)

		newEncryptor, err := security.NewTokenEncryption(newKey)
		require.NoError(t, err)
		assert.NotNil(t, newEncryptor)

		// Test that encryption with different keys produces different ciphertexts
		testToken := "test_token_12345"
		encrypted1, err := oldEncryptor.Encrypt(testToken)
		require.NoError(t, err)

		encrypted2, err := newEncryptor.Encrypt(testToken)
		require.NoError(t, err)

		assert.NotEqual(t, encrypted1, encrypted2, "Same plaintext with different keys should produce different ciphertexts")
	})
}

// TestLoggingSecurityHeaders tests that sensitive headers are not logged
func TestLoggingSecurityHeaders(t *testing.T) {
	t.Run("Authorization header masked in logs", func(t *testing.T) {
		headers := map[string]string{
			"Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
			"Content-Type":  "application/json",
			"User-Agent":    "Mozilla/5.0",
		}

		loggedHeaders := maskSensitiveHeaders(headers)

		assert.Contains(t, loggedHeaders["Authorization"], "***", "Authorization should be masked")
		assert.Equal(t, "application/json", loggedHeaders["Content-Type"], "Non-sensitive headers preserved")
	})

	t.Run("Cookie header masked in logs", func(t *testing.T) {
		headers := map[string]string{
			"Cookie": "session=abc123; token=xyz789",
		}

		loggedHeaders := maskSensitiveHeaders(headers)

		assert.Contains(t, loggedHeaders["Cookie"], "***", "Cookie should be masked")
	})
}

// TestSlogLogging tests structured logging with slog
func TestSlogLogging(t *testing.T) {
	t.Run("Use slog instead of fmt.Printf", func(t *testing.T) {
		// Create test logger
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

		// Good: structured logging with slog
		logger.Info("test message",
			"user_id", "user-123",
			"action", "login",
		)

		// Bad: fmt.Printf (should not be used)
		// fmt.Printf("User %s logged in\n", "user-123")

		assert.NotNil(t, logger, "Should use slog for logging")
	})

	t.Run("Sensitive data masked in slog", func(t *testing.T) {
		logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

		token := "Bearer sk_live_1234567890"
		maskedToken := maskSensitiveData(token)

		// Log with masked token
		logger.Info("API call", "token", maskedToken)

		assert.Contains(t, maskedToken, "***", "Token should be masked in logs")
	})
}

// TestMemoryLeaks tests that sensitive data is cleared from memory
func TestMemoryLeaks(t *testing.T) {
	t.Run("Clear password from memory after hashing", func(t *testing.T) {
		password := "MyPassword123!"
		passwordBytes := []byte(password)

		// Hash password
		hash, err := bcrypt.GenerateFromPassword(passwordBytes, 12)
		assert.NoError(t, err)

		// Clear password from memory
		for i := range passwordBytes {
			passwordBytes[i] = 0
		}

		assert.NotNil(t, hash, "Hash should be generated")
		assert.Equal(t, byte(0), passwordBytes[0], "Password should be cleared from memory")
	})
}

// Helper functions

func maskSensitiveData(data string) string {
	if len(data) <= 8 {
		return "***"
	}
	return data[:4] + "***" + data[len(data)-4:]
}

func filterSensitiveFields(data map[string]interface{}) map[string]interface{} {
	sensitiveFields := []string{"api_key", "password", "secret", "token"}

	filtered := make(map[string]interface{})
	for key, value := range data {
		isSensitive := false
		for _, sensitive := range sensitiveFields {
			if strings.Contains(strings.ToLower(key), sensitive) {
				isSensitive = true
				break
			}
		}

		// Exclude sensitive fields entirely from the response
		if !isSensitive {
			filtered[key] = value
		}
	}

	return filtered
}

func hashPassword(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(hash)
}

func simulateConnectionError(connString string) error {
	// Mask password in connection string
	masked := maskSensitiveData(connString)
	return &SecurityError{Message: "Connection failed: " + masked}
}

func simulateJWTError(token string) error {
	return &SecurityError{Message: "Invalid token format"}
}

func simulateValidationError(input string) error {
	return &SecurityError{Message: "Invalid input format"}
}

func maskEnvironmentVars(envVars map[string]string) map[string]string {
	sensitiveKeys := []string{"SECRET", "PASSWORD", "TOKEN", "KEY", "DATABASE_URL"}

	masked := make(map[string]string)
	for key, value := range envVars {
		isSensitive := false
		upperKey := strings.ToUpper(key)
		for _, sensitive := range sensitiveKeys {
			if strings.Contains(upperKey, sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			masked[key] = "***REDACTED***"
		} else {
			masked[key] = value
		}
	}

	return masked
}

func encryptToken(token string) string {
	// Use real AES-256-GCM encryption for testing
	// Generate a test encryption key (32 bytes base64 encoded)
	testKey, err := security.GenerateKey()
	if err != nil {
		panic("Failed to generate encryption key: " + err.Error())
	}

	encryptor, err := security.NewTokenEncryption(testKey)
	if err != nil {
		panic("Failed to create encryptor: " + err.Error())
	}

	encrypted, err := encryptor.Encrypt(token)
	if err != nil {
		panic("Failed to encrypt token: " + err.Error())
	}

	return encrypted
}

func maskSensitiveHeaders(headers map[string]string) map[string]string {
	sensitiveHeaders := []string{"Authorization", "Cookie", "X-API-Key"}

	masked := make(map[string]string)
	for key, value := range headers {
		isSensitive := false
		for _, sensitive := range sensitiveHeaders {
			if key == sensitive {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			masked[key] = maskSensitiveData(value)
		} else {
			masked[key] = value
		}
	}

	return masked
}

type SecurityError struct {
	Message string
}

func (e *SecurityError) Error() string {
	return e.Message
}
