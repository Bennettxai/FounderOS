package services

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ModeTransition records a single mode change in a conversation.
type ModeTransition struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	FromMode       string    `json:"from_mode"`
	ToMode         string    `json:"to_mode"`
	Confidence     float64   `json:"confidence"`
	DurationMs     int64     `json:"duration_ms"` // Time spent in previous mode
	CreatedAt      time.Time `json:"created_at"`
}

// lastModeState tracks the last known mode for a conversation.
type lastModeState struct {
	Mode      string
	EnteredAt time.Time
}

const maxLastModeEntries = 500 // Cap in-memory mode cache to prevent unbounded growth

// ModeTransitionService tracks OSA mode transitions per conversation.
type ModeTransitionService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	mu     sync.RWMutex
	// In-memory cache of last known mode per conversation (capped at maxLastModeEntries)
	lastMode map[string]lastModeState
}

// NewModeTransitionService creates a new mode transition tracking service.
func NewModeTransitionService(pool *pgxpool.Pool, logger *slog.Logger) *ModeTransitionService {
	return &ModeTransitionService{
		pool:     pool,
		logger:   logger.With("component", "mode_transition"),
		lastMode: make(map[string]lastModeState),
	}
}

// RecordTransition records a mode transition if the mode changed.
// Returns true if a transition was recorded, false if mode unchanged.
func (s *ModeTransitionService) RecordTransition(ctx context.Context, conversationID, newMode string, confidence float64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	last, exists := s.lastMode[conversationID]

	// Update last known mode
	s.lastMode[conversationID] = lastModeState{
		Mode:      newMode,
		EnteredAt: now,
	}

	// Evict random entries if cache exceeds cap
	if len(s.lastMode) > maxLastModeEntries {
		for key := range s.lastMode {
			if key != conversationID {
				delete(s.lastMode, key)
				break
			}
		}
	}

	// No change — skip recording
	if exists && last.Mode == newMode {
		return false, nil
	}

	// Calculate duration in previous mode
	var durationMs int64
	fromMode := ""
	if exists {
		fromMode = last.Mode
		durationMs = now.Sub(last.EnteredAt).Milliseconds()
	}

	// Persist to DB
	_, err := s.pool.Exec(ctx,
		`INSERT INTO mode_transitions (conversation_id, from_mode, to_mode, confidence, duration_ms, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		conversationID, fromMode, newMode, confidence, durationMs, now)
	if err != nil {
		s.logger.Warn("failed to record mode transition",
			"conversation_id", conversationID,
			"from", fromMode,
			"to", newMode,
			"error", err,
		)
		return false, err
	}

	s.logger.Info("mode transition recorded",
		"conversation_id", conversationID,
		"from", fromMode,
		"to", newMode,
		"confidence", confidence,
		"duration_ms", durationMs,
	)

	return true, nil
}

// GetTransitionHistory returns recent transitions for a conversation.
func (s *ModeTransitionService) GetTransitionHistory(ctx context.Context, conversationID string, limit int) ([]ModeTransition, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.pool.Query(ctx,
		`SELECT id, conversation_id, from_mode, to_mode, confidence, duration_ms, created_at
		 FROM mode_transitions
		 WHERE conversation_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2`,
		conversationID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transitions []ModeTransition
	for rows.Next() {
		var t ModeTransition
		if err := rows.Scan(&t.ID, &t.ConversationID, &t.FromMode, &t.ToMode, &t.Confidence, &t.DurationMs, &t.CreatedAt); err != nil {
			return nil, err
		}
		transitions = append(transitions, t)
	}

	return transitions, rows.Err()
}

// GetModeFlipRate returns the number of mode transitions per minute
// within the given time window for a conversation.
func (s *ModeTransitionService) GetModeFlipRate(ctx context.Context, conversationID string, windowMinutes int) (float64, error) {
	if windowMinutes <= 0 {
		windowMinutes = 5
	}

	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*)
		 FROM mode_transitions
		 WHERE conversation_id = $1
		   AND created_at >= NOW() - INTERVAL '1 minute' * $2`,
		conversationID, windowMinutes).Scan(&count)
	if err != nil {
		return 0, err
	}

	return float64(count) / float64(windowMinutes), nil
}

// GetLastMode returns the last known mode for a conversation from cache.
// Returns empty string if no mode is cached.
func (s *ModeTransitionService) GetLastMode(conversationID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if last, exists := s.lastMode[conversationID]; exists {
		return last.Mode
	}
	return ""
}
