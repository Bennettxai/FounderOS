package subconscious

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/signal"
)

// ClassificationResult holds the genre and weight assigned to a signal.
type ClassificationResult struct {
	Genre  signal.Genre
	Weight float64
}

// SignalClassifier classifies user messages into Signal Theory genres and weights.
// Uses a cheap LLM (llama-3.1-8b-instant via Groq) for classification.
// Includes a circuit breaker: after 5 consecutive failures, skips for 5 minutes.
type SignalClassifier struct {
	cfg    *config.Config
	pool   *pgxpool.Pool
	logger *slog.Logger

	// Circuit breaker state
	mu               sync.Mutex
	consecutiveFails int
	cooldownUntil    time.Time
}

const (
	classifierModel       = "llama-3.1-8b-instant"
	classifierMaxFails    = 5
	classifierCooldown    = 5 * time.Minute
	classifierMaxTokens   = 50
	classifierTemperature = 0.0
)

const classifierSystemPrompt = `You are a signal classifier. Given a user message, classify it.

Output EXACTLY two lines:
GENRE: <one of: DIRECT, INFORM, COMMIT, DECIDE, EXPRESS>
WEIGHT: <float 0.0 to 1.0>

Genre definitions:
- DIRECT: User wants an action performed (commands, requests, instructions)
- INFORM: User wants/provides information (questions, explanations, data)
- COMMIT: User is making a commitment or promise (agreements, confirmations)
- DECIDE: User is making a decision or changing state (selections, configurations)
- EXPRESS: User is expressing feelings, opinions, or internal state (feedback, frustration, praise)

Weight = informational density:
- 0.1-0.3: Low info (greetings, acknowledgments, "ok", "thanks")
- 0.4-0.6: Medium info (simple questions, brief instructions)
- 0.7-0.9: High info (detailed requests, complex questions, multi-part instructions)
- 1.0: Maximum info (critical decisions, comprehensive specifications)

No explanation. Just the two lines.`

// NewSignalClassifier creates a new classifier.
func NewSignalClassifier(cfg *config.Config, pool *pgxpool.Pool, logger *slog.Logger) *SignalClassifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &SignalClassifier{
		cfg:    cfg,
		pool:   pool,
		logger: logger.With("component", "signal_classifier"),
	}
}

// Classify determines genre and weight for a user message, then UPDATEs signal_log.
func (c *SignalClassifier) Classify(ctx context.Context, signalLogID, userMessage string) ClassificationResult {
	defaults := ClassificationResult{Genre: signal.GenreInform, Weight: 0.5}

	if c.isInCooldown() {
		return defaults
	}

	if c.cfg.GroqAPIKey == "" {
		return defaults
	}

	groq := services.NewGroqService(c.cfg, classifierModel)
	groq.SetOptions(services.LLMOptions{
		Temperature: classifierTemperature,
		MaxTokens:   classifierMaxTokens,
	})

	messages := []services.ChatMessage{
		{Role: "user", Content: userMessage},
	}

	resp, err := groq.ChatComplete(ctx, messages, classifierSystemPrompt)
	if err != nil {
		c.recordFailure()
		c.logger.Warn("classification failed", "error", err)
		return defaults
	}

	result := parseClassification(resp)
	c.recordSuccess()

	// Update signal_log asynchronously
	if signalLogID != "" {
		go c.updateSignalLog(context.Background(), signalLogID, result)
	}

	return result
}

func (c *SignalClassifier) isInCooldown() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Now().Before(c.cooldownUntil)
}

func (c *SignalClassifier) recordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consecutiveFails++
	if c.consecutiveFails >= classifierMaxFails {
		c.cooldownUntil = time.Now().Add(classifierCooldown)
		c.logger.Warn("classifier circuit breaker tripped",
			"consecutive_fails", c.consecutiveFails,
			"cooldown_until", c.cooldownUntil)
	}
}

func (c *SignalClassifier) recordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consecutiveFails = 0
}

func (c *SignalClassifier) updateSignalLog(ctx context.Context, signalLogID string, result ClassificationResult) {
	_, err := c.pool.Exec(ctx, `
		UPDATE signal_log SET genre = $1, weight = $2 WHERE id = $3
	`, string(result.Genre), result.Weight, signalLogID)
	if err != nil {
		c.logger.Warn("signal_log update failed", "id", signalLogID, "error", err)
	}
}

// parseClassification extracts genre and weight from classifier response.
func parseClassification(response string) ClassificationResult {
	result := ClassificationResult{Genre: signal.GenreInform, Weight: 0.5}

	lines := strings.Split(strings.TrimSpace(response), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)

		if strings.HasPrefix(upper, "GENRE:") {
			genre := strings.TrimSpace(strings.TrimPrefix(upper, "GENRE:"))
			switch genre {
			case "DIRECT":
				result.Genre = signal.GenreDirect
			case "INFORM":
				result.Genre = signal.GenreInform
			case "COMMIT":
				result.Genre = signal.GenreCommit
			case "DECIDE":
				result.Genre = signal.GenreDecide
			case "EXPRESS":
				result.Genre = signal.GenreExpress
			}
		}

		if strings.HasPrefix(upper, "WEIGHT:") {
			weightStr := strings.TrimSpace(strings.TrimPrefix(upper, "WEIGHT:"))
			var w float64
			if _, err := fmt.Sscanf(weightStr, "%f", &w); err == nil {
				if w >= 0 && w <= 1.0 {
					result.Weight = w
				}
			}
		}
	}

	return result
}
