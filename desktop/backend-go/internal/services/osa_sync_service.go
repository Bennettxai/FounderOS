package services

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	nats "github.com/nats-io/nats.go"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/integrations/osa"
)

// OSASyncService handles bidirectional synchronization between BusinessOS and OSA-5
// This service implements the transactional outbox pattern for reliable event publishing
// and provides idempotent sync operations with automatic retry and conflict resolution.
type OSASyncService struct {
	pool      *pgxpool.Pool
	osaClient *osa.Client
	queries   *sqlc.Queries
	logger    *slog.Logger
	natsConn  *nats.Conn
	natsJS    nats.JetStreamContext
	cfg       *config.Config
}

// NewOSASyncService creates a new OSA sync service with proper initialization
func NewOSASyncService(pool *pgxpool.Pool, cfg *config.Config) (*OSASyncService, error) {
	if pool == nil {
		return nil, fmt.Errorf("database pool is required")
	}
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	// Initialize OSA client
	osaConfig := &osa.Config{
		BaseURL:      cfg.OSABaseURL,
		SharedSecret: cfg.OSASharedSecret,
		Timeout:      30 * time.Second,
		MaxRetries:   3,
		RetryDelay:   2 * time.Second,
	}

	osaClient, err := osa.NewClient(osaConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSA client: %w", err)
	}

	service := &OSASyncService{
		pool:      pool,
		osaClient: osaClient,
		queries:   sqlc.New(pool),
		logger:    slog.Default().With("service", "osa_sync"),
		cfg:       cfg,
	}

	// Initialize NATS connection if enabled
	if cfg.NATSEnabled && cfg.NATSURL != "" {
		if err := service.initNATS(); err != nil {
			slog.Warn("Failed to initialize NATS, will use outbox-only mode",
				"error", err)
		}
	} else {
		slog.Info("NATS disabled, using outbox-only mode")
	}

	return service, nil
}

// initNATS initializes NATS connection and JetStream
func (s *OSASyncService) initNATS() error {
	conn, err := nats.Connect(s.cfg.NATSURL,
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			if err != nil {
				s.logger.Warn("NATS disconnected", "error", err)
			}
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			s.logger.Info("NATS reconnected")
		}),
	)
	if err != nil {
		return fmt.Errorf("connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return fmt.Errorf("create JetStream context: %w", err)
	}

	// Create stream if it doesn't exist
	streamName := "OSA_SYNC"
	_, err = js.StreamInfo(streamName)
	if err == nats.ErrStreamNotFound {
		ttl := time.Duration(s.cfg.NATSTTL) * time.Hour
		_, err = js.AddStream(&nats.StreamConfig{
			Name:     streamName,
			Subjects: []string{"osa.>"},
			MaxAge:   ttl,
			Storage:  nats.FileStorage,
		})
		if err != nil {
			conn.Close()
			return fmt.Errorf("create stream: %w", err)
		}
		s.logger.Info("Created NATS stream",
			"stream", streamName,
			"ttl", ttl)
	}

	s.natsConn = conn
	s.natsJS = js
	s.logger.Info("NATS initialized successfully",
		"url", s.cfg.NATSURL)

	return nil
}

// Close closes the OSA client and cleans up resources
func (s *OSASyncService) Close() error {
	var errs []error

	// Close NATS connection
	if s.natsConn != nil {
		s.natsConn.Close()
		s.logger.Info("NATS connection closed")
	}

	// Close OSA client
	if s.osaClient != nil {
		if err := s.osaClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close OSA client: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	return nil
}

// pgTypeUUID converts uuid.UUID to pgtype.UUID
func pgTypeUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}
}
