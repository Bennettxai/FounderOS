package security

import (
	"fmt"
	"strings"
)

// ForbiddenSecretValues are placeholder values that MUST NOT be used in production
var ForbiddenSecretValues = []string{
	"change-this",
	"your-secret-key",
	"generate-with-openssl",
	"test-secret",
	"example-key",
	"replace-me",
	"change_me",
	"changeme",
	"placeholder",
	"todo-change",
	"fixme",
	"default-secret",
	"insecure-key",
}

// ValidateProductionSecrets checks that critical secrets don't contain forbidden placeholder values
// Returns error if any secret contains a forbidden value (case-insensitive)
func ValidateProductionSecrets(jwtSecret, tokenEncryptionKey, redisHMACKey string) error {
	secrets := map[string]string{
		"JWT_SECRET":           jwtSecret,
		"TOKEN_ENCRYPTION_KEY": tokenEncryptionKey,
		"REDIS_HMAC_KEY":       redisHMACKey,
	}

	for name, value := range secrets {
		if value == "" {
			return fmt.Errorf("SECURITY: Production secret %s is EMPTY. Set a secure random value", name)
		}

		// Check minimum length (32 characters recommended for cryptographic keys)
		if len(value) < 32 {
			return fmt.Errorf("SECURITY: Production secret %s is too short (%d chars). Use at least 32 characters", name, len(value))
		}

		// Check for forbidden placeholder values
		valueLower := strings.ToLower(value)
		for _, forbidden := range ForbiddenSecretValues {
			if strings.Contains(valueLower, forbidden) {
				return fmt.Errorf("SECURITY: Production secret %s contains forbidden placeholder '%s'. Generate a secure random value", name, forbidden)
			}
		}
	}

	return nil
}

// GenerateSecretInstructions returns instructions for generating secure secrets
func GenerateSecretInstructions() string {
	return `
Generate secure secrets using one of these methods:

1. Using openssl (recommended):
   openssl rand -base64 32

2. Using Go:
   go run -c 'package main; import ("crypto/rand"; "encoding/base64"; "fmt"); func main() { b := make([]byte, 32); rand.Read(b); fmt.Println(base64.StdEncoding.EncodeToString(b)) }'

3. Using Node.js:
   node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"

Set these in your .env file:
   JWT_SECRET=<generated-value>
   TOKEN_ENCRYPTION_KEY=<generated-value>
   REDIS_HMAC_KEY=<generated-value>
`
}

// ValidateAndFail validates security configuration and fails fast in production
// This is called at server startup to catch configuration issues immediately
func ValidateAndFail(environment, jwtSecret, tokenEncryptionKey, redisHMACKey string) error {
	// In production, enforce strict validation
	if environment == "production" {
		return ValidateProductionSecrets(jwtSecret, tokenEncryptionKey, redisHMACKey)
	}

	// In development, just log warnings (don't block startup)
	return nil
}

// WarnDevelopmentInsecure returns warnings for insecure development configuration
func WarnDevelopmentInsecure(tokenEncryptionKey, redisHMACKey string) []string {
	warnings := []string{}

	if tokenEncryptionKey == "" {
		warnings = append(warnings, "TOKEN_ENCRYPTION_KEY not set - OAuth tokens will be stored in PLAINTEXT")
	}

	if redisHMACKey == "" {
		warnings = append(warnings, "REDIS_HMAC_KEY not set - session cache keys not protected with HMAC")
	}

	return warnings
}
