package governance

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/signal"
)

// AlgedonicSeverity classifies the urgency of an algedonic signal.
type AlgedonicSeverity string

const (
	AlgedonicCritical AlgedonicSeverity = "CRITICAL" // Requires immediate attention
	AlgedonicHigh     AlgedonicSeverity = "HIGH"     // Significant issue
	AlgedonicMedium   AlgedonicSeverity = "MEDIUM"   // Notable but not urgent
)

// AlgedonicSignal is an emergency bypass signal from S1 (operations) to S5 (policy).
// In Beer's VSM, the algedonic channel allows lower systems to escalate directly
// to the highest authority when normal channels fail.
type AlgedonicSignal struct {
	Source      string                // Which component fired the signal
	Description string                // Human-readable description
	Severity    AlgedonicSeverity     // Urgency level
	Report      *signal.FailureReport // Optional failure report that triggered this
	Metadata    map[string]any        // Additional context
}

// AlgedonicHandler processes algedonic signals. Start with logging-only handlers.
type AlgedonicHandler interface {
	Handle(ctx context.Context, sig AlgedonicSignal) error
}

// LoggingHandler is the default handler — logs algedonic signals via slog.
type LoggingHandler struct {
	logger *slog.Logger
}

// NewLoggingHandler creates a logging-only algedonic handler.
func NewLoggingHandler(logger *slog.Logger) *LoggingHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &LoggingHandler{logger: logger.With("component", "algedonic")}
}

func (h *LoggingHandler) Handle(_ context.Context, sig AlgedonicSignal) error {
	h.logger.Warn("ALGEDONIC SIGNAL FIRED",
		"source", sig.Source,
		"severity", sig.Severity,
		"description", sig.Description,
	)
	return nil
}

// PostgresHandler persists algedonic signals to the governance_events table.
type PostgresHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewPostgresHandler creates a handler that persists events to the database.
func NewPostgresHandler(pool *pgxpool.Pool, logger *slog.Logger) *PostgresHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &PostgresHandler{pool: pool, logger: logger}
}

func (h *PostgresHandler) Handle(ctx context.Context, sig AlgedonicSignal) error {
	metadata, _ := json.Marshal(sig.Metadata)
	_, err := h.pool.Exec(ctx, `
		INSERT INTO governance_events (event_type, source, severity, description, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`, "algedonic", sig.Source, string(sig.Severity), sig.Description, metadata)
	if err != nil {
		h.logger.Error("failed to persist governance event",
			"error", err, "source", sig.Source)
	}
	return err
}

// AlgedonicChannel is Beer's VSM algedonic bypass channel.
// Routes emergency signals from operations to policy level.
type AlgedonicChannel struct {
	handlers []AlgedonicHandler
	logger   *slog.Logger
}

// NewAlgedonicChannel creates a new AlgedonicChannel with the given handlers.
func NewAlgedonicChannel(logger *slog.Logger, handlers ...AlgedonicHandler) *AlgedonicChannel {
	if logger == nil {
		logger = slog.Default()
	}
	return &AlgedonicChannel{
		handlers: handlers,
		logger:   logger.With("component", "algedonic_channel"),
	}
}

// Fire sends an algedonic signal through all registered handlers.
// Non-blocking: errors are logged but do not halt processing.
func (ac *AlgedonicChannel) Fire(ctx context.Context, sig AlgedonicSignal) {
	ac.logger.Warn("algedonic channel firing",
		"source", sig.Source,
		"severity", sig.Severity,
		"description", sig.Description,
	)

	for _, h := range ac.handlers {
		if err := h.Handle(ctx, sig); err != nil {
			ac.logger.Error("algedonic handler failed",
				"error", err, "source", sig.Source)
		}
	}
}

// FireFromFailure creates and fires an algedonic signal from a failure report.
// Only fires for CRITICAL and HIGH severity failures.
func (ac *AlgedonicChannel) FireFromFailure(ctx context.Context, report signal.FailureReport) {
	if !report.Detected {
		return
	}

	var severity AlgedonicSeverity
	switch report.Severity {
	case signal.SeverityCritical:
		severity = AlgedonicCritical
	case signal.SeverityHigh:
		severity = AlgedonicHigh
	default:
		return // Only escalate CRITICAL and HIGH
	}

	ac.Fire(ctx, AlgedonicSignal{
		Source:      report.DetectorName,
		Description: report.Description,
		Severity:    severity,
		Report:      &report,
		Metadata:    report.Metadata,
	})
}

// LogGovernanceEvent writes an event directly to governance_events.
// Used for non-algedonic governance events (e.g., setpoint adjustments, policy changes).
func LogGovernanceEvent(ctx context.Context, pool *pgxpool.Pool, eventType, source, description string, metadata map[string]any) {
	if pool == nil {
		return
	}
	metadataJSON, _ := json.Marshal(metadata)
	_, err := pool.Exec(ctx, `
		INSERT INTO governance_events (event_type, source, severity, description, metadata)
		VALUES ($1, $2, $3, $4, $5)
	`, eventType, source, "INFO", description, metadataJSON)
	if err != nil {
		slog.Warn("failed to log governance event", "error", err, "type", eventType)
	}
}

// GovernanceEvent represents a row in the governance_events table.
type GovernanceEvent struct {
	ID          string         `json:"id"`
	EventType   string         `json:"event_type"`
	Source      string         `json:"source"`
	Severity    string         `json:"severity"`
	Description string         `json:"description"`
	Metadata    map[string]any `json:"metadata"`
	CreatedAt   time.Time      `json:"created_at"`
}
