// Package security provides cryptographic utilities for securing sensitive data
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"
)

// TokenEncryption handles encryption/decryption of OAuth tokens and other sensitive data
// Uses AES-256-GCM for authenticated encryption
type TokenEncryption struct {
	key   []byte
	gcm   cipher.AEAD
	mutex sync.RWMutex
}

var (
	// ErrInvalidKey is returned when the encryption key is invalid
	ErrInvalidKey = errors.New("invalid encryption key: must be 32 bytes (256 bits)")

	// ErrDecryptionFailed is returned when decryption fails
	ErrDecryptionFailed = errors.New("decryption failed: ciphertext is invalid or tampered")

	// ErrCiphertextTooShort is returned when ciphertext is shorter than nonce size
	ErrCiphertextTooShort = errors.New("ciphertext too short")

	// globalEncryption is the singleton instance
	globalEncryption *TokenEncryption
	once             sync.Once
)

// NewTokenEncryption creates a new TokenEncryption instance
// keyBase64 must be a base64-encoded 32-byte key (use: openssl rand -base64 32)
func NewTokenEncryption(keyBase64 string) (*TokenEncryption, error) {
	if keyBase64 == "" {
		return nil, ErrInvalidKey
	}

	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 key: %w", err)
	}

	if len(key) != 32 {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &TokenEncryption{
		key: key,
		gcm: gcm,
	}, nil
}

// InitGlobalEncryption initializes the global encryption instance
// This should be called once at application startup
func InitGlobalEncryption(keyBase64 string) error {
	var initErr error
	once.Do(func() {
		enc, err := NewTokenEncryption(keyBase64)
		if err != nil {
			initErr = err
			return
		}
		globalEncryption = enc
	})
	return initErr
}

// GetGlobalEncryption returns the global encryption instance
// Returns nil if not initialized
func GetGlobalEncryption() *TokenEncryption {
	return globalEncryption
}

// Encrypt encrypts plaintext and returns base64-encoded ciphertext
// The ciphertext includes the nonce prepended to it
func (te *TokenEncryption) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	te.mutex.RLock()
	defer te.mutex.RUnlock()

	// Generate random nonce
	nonce := make([]byte, te.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce
	ciphertext := te.gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Return base64 encoded for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// EncryptBytes encrypts plaintext and returns raw bytes
// Used for BYTEA columns in PostgreSQL
func (te *TokenEncryption) EncryptBytes(plaintext string) ([]byte, error) {
	if plaintext == "" {
		return nil, nil
	}

	te.mutex.RLock()
	defer te.mutex.RUnlock()

	nonce := make([]byte, te.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	return te.gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// Decrypt decrypts base64-encoded ciphertext and returns plaintext
func (te *TokenEncryption) Decrypt(ciphertextBase64 string) (string, error) {
	if ciphertextBase64 == "" {
		return "", nil
	}

	te.mutex.RLock()
	defer te.mutex.RUnlock()

	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", fmt.Errorf("invalid base64 ciphertext: %w", err)
	}

	return te.decryptBytes(ciphertext)
}

// DecryptBytes decrypts raw bytes and returns plaintext
// Used for BYTEA columns in PostgreSQL
func (te *TokenEncryption) DecryptBytes(ciphertext []byte) (string, error) {
	if len(ciphertext) == 0 {
		return "", nil
	}

	te.mutex.RLock()
	defer te.mutex.RUnlock()

	return te.decryptBytes(ciphertext)
}

func (te *TokenEncryption) decryptBytes(ciphertext []byte) (string, error) {
	nonceSize := te.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrCiphertextTooShort
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := te.gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// GenerateKey generates a new random 32-byte key and returns it base64 encoded
// Use this to generate keys for TOKEN_ENCRYPTION_KEY env var
func GenerateKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// --- Convenience functions using global encryption ---

// EncryptToken encrypts a token using the global encryption instance
func EncryptToken(token string) (string, error) {
	if globalEncryption == nil {
		// In development without encryption key, store plaintext
		// This allows the app to work without encryption configured
		return token, nil
	}
	return globalEncryption.Encrypt(token)
}

// DecryptToken decrypts a token using the global encryption instance
func DecryptToken(encryptedToken string) (string, error) {
	if globalEncryption == nil {
		// In development without encryption key, assume plaintext
		return encryptedToken, nil
	}
	return globalEncryption.Decrypt(encryptedToken)
}

// EncryptTokenBytes encrypts a token to bytes using the global encryption instance
func EncryptTokenBytes(token string) ([]byte, error) {
	if globalEncryption == nil {
		return []byte(token), nil
	}
	return globalEncryption.EncryptBytes(token)
}

// DecryptTokenBytes decrypts bytes to a token using the global encryption instance
func DecryptTokenBytes(encryptedToken []byte) (string, error) {
	if globalEncryption == nil {
		return string(encryptedToken), nil
	}
	return globalEncryption.DecryptBytes(encryptedToken)
}

// IsEncryptionEnabled returns true if encryption is configured
func IsEncryptionEnabled() bool {
	return globalEncryption != nil
}
