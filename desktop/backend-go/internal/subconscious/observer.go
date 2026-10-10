package subconscious

import (
	"context"
	"log/slog"
	"time"
)

// ObserveInput is the data needed to observe a single conversation turn.
type ObserveInput struct {
	UserID          string
	UserMessage     string
	PreviousUserMsg string
	AssistantResp   string
	SignalLogID     string
	AgentType       string
	ConversationID  string
	Latency         time.Duration // Time from message receipt to first SSE token (L6 TimeToDecide)
}

// Observer is the main entry point for the Subconscious Observer pipeline.
// It orchestrates: classify → extract patterns → emit metrics → accumulate blocks.
// All observation is async (fire-and-forget goroutine).
type Observer struct {
	classifier  *SignalClassifier
	extractor   *PatternExtractor
	emitter     *MetricEmitter
	accumulator *BlockAccumulator
	logger      *slog.Logger
}

// NewObserver creates a new Observer.
func NewObserver(
	classifier *SignalClassifier,
	extractor *PatternExtractor,
	emitter *MetricEmitter,
	accumulator *BlockAccumulator,
	logger *slog.Logger,
) *Observer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Observer{
		classifier:  classifier,
		extractor:   extractor,
		emitter:     emitter,
		accumulator: accumulator,
		logger:      logger.With("component", "subconscious_observer"),
	}
}

// Observe fires the full observation pipeline asynchronously.
// Zero latency on the hot path — launches a goroutine and returns immediately.
func (o *Observer) Observe(input ObserveInput) {
	go o.observe(context.Background(), input)
}

// ObserveSync runs the pipeline synchronously. Used for testing.
func (o *Observer) ObserveSync(ctx context.Context, input ObserveInput) {
	o.observe(ctx, input)
}

// observe runs the full pipeline synchronously (called in a goroutine).
func (o *Observer) observe(ctx context.Context, input ObserveInput) {
	defer func() {
		if r := recover(); r != nil {
			o.logger.Error("observer panic recovered", "panic", r)
		}
	}()

	// 1. Classify the signal's genre and weight
	classification := ClassificationResult{}
	if o.classifier != nil {
		classification = o.classifier.Classify(ctx, input.SignalLogID, input.UserMessage)
	}

	// 2. Extract patterns (heuristic, no LLM)
	patterns := ExtractedPatterns{}
	if o.extractor != nil {
		patterns = o.extractor.Extract(input.UserMessage, input.PreviousUserMsg, input.AssistantResp)
	}

	// 3. Emit metrics → proxy metrics + AutopoieticMonitor
	if o.emitter != nil {
		o.emitter.Emit(ctx, patterns, classification, input.Latency)
	}

	// 4. Accumulate memory blocks → subconscious_blocks + SelfImprovementEngine
	if o.accumulator != nil {
		o.accumulator.Accumulate(ctx, Observation{
			UserID:         input.UserID,
			UserMessage:    input.UserMessage,
			Response:       input.AssistantResp,
			Patterns:       patterns,
			Classification: classification,
			AgentType:      input.AgentType,
			ConversationID: input.ConversationID,
		})
	}

	o.logger.Info("observation complete",
		"user_id", input.UserID,
		"genre", classification.Genre,
		"weight", classification.Weight,
		"re_encoding", patterns.IsReEncoding,
		"re_encoding_sim", patterns.ReEncodingSim,
		"frustration", patterns.IsFrustration,
		"feedback_closed", patterns.IsFeedbackClosed,
		"bounce", patterns.IsBounce,
		"has_preference", patterns.HasPreference,
	)
}
