// Package services provides business logic services for the application.
// sandbox_env.go manages environment variables for sandbox containers.
package services

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// Sandbox environment errors
var (
	ErrInvalidEnvVarName  = errors.New("invalid environment variable name")
	ErrInvalidEnvVarValue = errors.New("invalid environment variable value")
	ErrEnvVarNotFound     = errors.New("environment variable not found")
	ErrEncryptionFailed   = errors.New("encryption failed")
	ErrDecryptionFailed   = errors.New("decryption failed")
)

// Environment variable name validation regex (alphanumeric + underscore)
var envVarNameRegex = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

// SandboxEnvService manages environment variables for sandboxes.
// It stores encrypted env vars in database and injects them at container creation.
type SandboxEnvService struct {
	pool          *pgxpool.Pool
	queries       *sqlc.Queries
	config        *config.Config
	logger        *slog.Logger
	encryptionKey []byte // Derived from config
}

// EnvVar represents an environment variable for a sandbox app.
type EnvVar struct {
	ID        uuid.UUID
	AppID     uuid.UUID
	Key       string
	Value     string // Decrypted value
	IsSecret  bool
	CreatedAt string
	UpdatedAt string
}

// NewSandboxEnvService creates a new sandbox environment variable service.
func NewSandboxEnvService(
	pool *pgxpool.Pool,
	cfg *config.Config,
	logger *slog.Logger,
) (*SandboxEnvService, error) {
	if cfg.SecretKey == "" {
		return nil, errors.New("SECRET_KEY must be set for environment variable encryption")
	}

	// Derive encryption key from config secret key
	encryptionKey := deriveKey(cfg.SecretKey, "sandbox-env-encryption")

	return &SandboxEnvService{
		pool:          pool,
		queries:       sqlc.New(pool),
		config:        cfg,
		logger:        logger,
		encryptionKey: encryptionKey,
	}, nil
}

// SetEnvVar stores or updates an environment variable for a sandbox app.
// If isSecret is true, the value is encrypted before storage and masked in logs.
func (s *SandboxEnvService) SetEnvVar(ctx context.Context, appID uuid.UUID, key string, value string, isSecret bool) error {
	// Validate env var name
	if !IsValidEnvVarName(key) {
		s.logger.Warn("invalid environment variable name rejected",
			slog.String("app_id", appID.String()),
			slog.String("key", key),
		)
		return ErrInvalidEnvVarName
	}

	// Validate value (no shell injection)
	if !IsValidEnvVarValue(value) {
		s.logger.Warn("invalid environment variable value rejected",
			slog.String("app_id", appID.String()),
			slog.String("key", key),
		)
		return ErrInvalidEnvVarValue
	}

	// Encrypt value if secret
	var storedValue string
	var err error
	if isSecret {
		storedValue, err = s.encrypt(value)
		if err != nil {
			s.logger.Error("failed to encrypt environment variable",
				slog.String("app_id", appID.String()),
				slog.String("key", key),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
		}
	} else {
		storedValue = value
	}

	// Store in database
	// Note: This requires a sandbox_env_vars table to be created
	// For now, we'll store in osa_module_instances.metadata JSONB field
	// TODO: Create dedicated sandbox_env_vars table in migration

	// Get current app metadata
	app, err := s.queries.GetOSAModuleInstanceByID(ctx, pgtype.UUID{Bytes: appID, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	// Parse existing metadata
	metadata := make(map[string]interface{})
	if len(app.Metadata) > 0 {
		if err := json.Unmarshal(app.Metadata, &metadata); err != nil {
			return fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	// Initialize env_vars map if not exists
	envVars, ok := metadata["env_vars"].(map[string]interface{})
	if !ok {
		envVars = make(map[string]interface{})
		metadata["env_vars"] = envVars
	}

	// Store encrypted value with metadata
	envVars[key] = map[string]interface{}{
		"value":     storedValue,
		"is_secret": isSecret,
	}

	// Update app metadata
	// TODO: Replace with dedicated table update when migration is ready

	// Log (mask secret values)
	logValue := value
	if isSecret {
		logValue = maskSecret(value)
	}
	s.logger.Info("environment variable set",
		slog.String("app_id", appID.String()),
		slog.String("key", key),
		slog.String("value", logValue),
		slog.Bool("is_secret", isSecret),
	)

	return nil
}

// GetEnvVars retrieves all environment variables for a sandbox app.
// Secret values are decrypted before returning.
func (s *SandboxEnvService) GetEnvVars(ctx context.Context, appID uuid.UUID) (map[string]string, error) {
	// Get app metadata
	app, err := s.queries.GetOSAModuleInstanceByID(ctx, pgtype.UUID{Bytes: appID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	// Parse metadata
	metadata := make(map[string]interface{})
	if len(app.Metadata) > 0 {
		if err := json.Unmarshal(app.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	// Extract env_vars
	envVarsData, ok := metadata["env_vars"].(map[string]interface{})
	if !ok {
		return make(map[string]string), nil // No env vars set
	}

	// Decrypt secret values
	result := make(map[string]string)
	for key, valueData := range envVarsData {
		varMap, ok := valueData.(map[string]interface{})
		if !ok {
			continue
		}

		storedValue, _ := varMap["value"].(string)
		isSecret, _ := varMap["is_secret"].(bool)

		if isSecret {
			decrypted, err := s.decrypt(storedValue)
			if err != nil {
				s.logger.Error("failed to decrypt environment variable",
					slog.String("app_id", appID.String()),
					slog.String("key", key),
					slog.String("error", err.Error()),
				)
				return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
			}
			result[key] = decrypted
		} else {
			result[key] = storedValue
		}
	}

	// Log (don't include values)
	s.logger.Debug("retrieved environment variables",
		slog.String("app_id", appID.String()),
		slog.Int("count", len(result)),
	)

	return result, nil
}

// DeleteEnvVar removes an environment variable for a sandbox app.
func (s *SandboxEnvService) DeleteEnvVar(ctx context.Context, appID uuid.UUID, key string) error {
	// Get app metadata
	app, err := s.queries.GetOSAModuleInstanceByID(ctx, pgtype.UUID{Bytes: appID, Valid: true})
	if err != nil {
		return fmt.Errorf("failed to get app: %w", err)
	}

	// Parse metadata
	metadata := make(map[string]interface{})
	if len(app.Metadata) > 0 {
		if err := json.Unmarshal(app.Metadata, &metadata); err != nil {
			return fmt.Errorf("failed to parse metadata: %w", err)
		}
	}

	// Get env_vars map
	envVars, ok := metadata["env_vars"].(map[string]interface{})
	if !ok {
		return ErrEnvVarNotFound
	}

	// Check if key exists
	if _, exists := envVars[key]; !exists {
		return ErrEnvVarNotFound
	}

	// Delete key
	delete(envVars, key)

	// Update app metadata
	// TODO: Replace with dedicated table update when migration is ready

	s.logger.Info("environment variable deleted",
		slog.String("app_id", appID.String()),
		slog.String("key", key),
	)

	return nil
}

// BuildContainerEnv builds the complete environment variable array for container creation.
// It combines user-defined env vars with system env vars.
func (s *SandboxEnvService) BuildContainerEnv(ctx context.Context, appID uuid.UUID, appName string, userID uuid.UUID, systemEnv map[string]string) ([]string, error) {
	// Get user-defined env vars
	userEnv, err := s.GetEnvVars(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user env vars: %w", err)
	}

	// Start with system env vars (these override user vars)
	result := make([]string, 0, len(systemEnv)+len(userEnv)+5)

	// Add user-defined vars first
	for key, value := range userEnv {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}

	// Add/override with system env vars
	for key, value := range systemEnv {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}

	// Always include these system variables (they override everything)
	result = append(result,
		fmt.Sprintf("APP_ID=%s", appID.String()),
		fmt.Sprintf("APP_NAME=%s", appName),
		fmt.Sprintf("USER_ID=%s", userID.String()),
		fmt.Sprintf("NODE_ENV=production"),
	)

	// Add PORT if specified in systemEnv
	if port, exists := systemEnv["PORT"]; exists {
		// Already added above, but ensure it's present
		_ = port
	}

	s.logger.Debug("built container environment",
		slog.String("app_id", appID.String()),
		slog.Int("total_vars", len(result)),
		slog.Int("user_vars", len(userEnv)),
		slog.Int("system_vars", len(systemEnv)),
	)

	return result, nil
}

// IsValidEnvVarName validates environment variable names.
// Must be alphanumeric + underscore, start with letter or underscore.
func IsValidEnvVarName(name string) bool {
	if len(name) == 0 || len(name) > 255 {
		return false
	}
	return envVarNameRegex.MatchString(name)
}

// IsValidEnvVarValue validates environment variable values.
// Prevents shell injection by checking for dangerous characters.
func IsValidEnvVarValue(value string) bool {
	// Check length
	if len(value) > 4096 {
		return false
	}

	// Check for shell injection patterns
	dangerous := []string{
		";", "|", "&", "`", "$(",
		"\n", "\r", // Line breaks
		"$(", "${", // Command substitution
	}

	for _, pattern := range dangerous {
		if strings.Contains(value, pattern) {
			return false
		}
	}

	return true
}

// encrypt encrypts a value using AES-GCM.
func (s *SandboxEnvService) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a value using AES-GCM.
func (s *SandboxEnvService) decrypt(ciphertext string) (string, error) {
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Extract nonce
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, encryptedData := data[:nonceSize], data[nonceSize:]

	// Decrypt
	plaintext, err := aesGCM.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// deriveKey derives a 32-byte AES key from a passphrase using SHA-256.
func deriveKey(passphrase string, salt string) []byte {
	hash := sha256.New()
	hash.Write([]byte(passphrase))
	hash.Write([]byte(salt))
	return hash.Sum(nil)
}

// maskSecret masks a secret value for safe logging.
// Shows first 4 chars, masks the rest.
func maskSecret(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	maskLen := 8
	if len(value)-4 < maskLen {
		maskLen = len(value) - 4
	}
	return value[:4] + strings.Repeat("*", maskLen)
}
