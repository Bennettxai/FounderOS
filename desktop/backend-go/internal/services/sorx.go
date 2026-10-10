// Package services provides business logic for BusinessOS.
package services

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
)

// SorxService handles communication with the Sorx skill execution engine.
type SorxService struct {
	pool          *pgxpool.Pool
	cfg           *config.Config
	encryptionKey []byte
	httpClient    *http.Client
}

// NewSorxService creates a new Sorx service.
func NewSorxService(pool *pgxpool.Pool, cfg *config.Config) *SorxService {
	// Derive encryption key from SecretKey (should be 32 bytes for AES-256)
	key := []byte(cfg.SecretKey)
	if len(key) < 32 {
		// Pad key if too short (in production, use proper key derivation)
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	}

	return &SorxService{
		pool:          pool,
		cfg:           cfg,
		encryptionKey: key[:32],
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}
