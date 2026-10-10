package services

import (
	"context"
	"log/slog"
	"time"
)

// SessionHealth captures cognitive session metrics for a conversation.
type SessionHealth struct {
	ModeFlipRate     float64       `json:"mode_flip_rate"`    // Transitions per minute
	MessageFrequency float64       `json:"message_frequency"` // Messages per minute (caller provides)
	SessionDuration  time.Duration `json:"session_duration"`  // Time since conversation start
	IsConfused       bool          `json:"is_confused"`       // High flip rate signals confusion
	IsFrustrated     bool          `json:"is_frustrated"`     // Rapid messages signal frustration
	IsDeepFocus      bool          `json:"is_deep_focus"`     // Same mode for extended period
	CurrentMode      string        `json:"current_mode"`      // Current active mode
	DominantMode     string        `json:"dominant_mode"`     // Most time spent in this mode
	Hint             string        `json:"hint"`              // Agent behavior hint
}

// Session health thresholds — tune these to calibrate heuristics.
const (
	flipRateConfusedThreshold   = 3.0              // transitions/min above which user appears confused
	messageFrustrationThreshold = 5.0              // messages/min above which user appears frustrated
	deepFocusMinDuration        = 30 * time.Minute // minimum session length for deep focus detection
	deepFocusMaxFlipRate        = 0.5              // flip rate must be below this for deep focus
)

// SessionHealthService computes cognitive load metrics per conversation.
type SessionHealthService struct {
	transitions *ModeTransitionService
	logger      *slog.Logger
}

// NewSessionHealthService creates a new session health service.
func NewSessionHealthService(transitions *ModeTransitionService, logger *slog.Logger) *SessionHealthService {
	return &SessionHealthService{
		transitions: transitions,
		logger:      logger.With("component", "session_health"),
	}
}

// ComputeHealth calculates session health metrics for a conversation.
// messageCount and sessionStart are provided by the caller from conversation metadata.
func (s *SessionHealthService) ComputeHealth(ctx context.Context, conversationID string, messageCount int, sessionStart time.Time) (*SessionHealth, error) {
	health := &SessionHealth{
		CurrentMode: s.transitions.GetLastMode(conversationID),
	}

	// Session duration
	health.SessionDuration = time.Since(sessionStart)

	// Message frequency (messages per minute)
	if health.SessionDuration > 0 {
		minutes := health.SessionDuration.Minutes()
		if minutes > 0 {
			health.MessageFrequency = float64(messageCount) / minutes
		}
	}

	// Mode flip rate (transitions per minute over last 5 min window)
	flipRate, err := s.transitions.GetModeFlipRate(ctx, conversationID, 5)
	if err != nil {
		s.logger.Warn("failed to get mode flip rate", "error", err)
	} else {
		health.ModeFlipRate = flipRate
	}

	// Determine dominant mode from recent transition history
	history, err := s.transitions.GetTransitionHistory(ctx, conversationID, 20)
	if err == nil && len(history) > 0 {
		health.DominantMode = computeDominantMode(history)
	}

	// Heuristics
	health.IsConfused = health.ModeFlipRate > flipRateConfusedThreshold
	health.IsFrustrated = health.MessageFrequency > messageFrustrationThreshold
	health.IsDeepFocus = !health.IsConfused && health.SessionDuration > deepFocusMinDuration && health.ModeFlipRate < deepFocusMaxFlipRate

	// Generate behavioral hint for agents
	health.Hint = generateHint(health)

	return health, nil
}

// computeDominantMode finds the mode with the most cumulative time.
// DurationMs measures time spent in the *previous* mode (FromMode), not the destination.
func computeDominantMode(history []ModeTransition) string {
	durations := make(map[string]int64)
	for _, t := range history {
		if t.FromMode != "" && t.DurationMs > 0 {
			durations[t.FromMode] += t.DurationMs
		}
	}

	var dominant string
	var maxDuration int64
	for mode, dur := range durations {
		if dur > maxDuration {
			maxDuration = dur
			dominant = mode
		}
	}
	return dominant
}

// generateHint produces a behavioral hint string for agents based on health signals.
func generateHint(h *SessionHealth) string {
	switch {
	case h.IsConfused:
		return "User appears confused (frequent mode switching). Ask clarifying questions before proceeding."
	case h.IsFrustrated:
		return "User is sending messages rapidly. Be concise and action-oriented."
	case h.IsDeepFocus:
		return "User is in deep focus. Provide detailed, thorough responses."
	default:
		return ""
	}
}
