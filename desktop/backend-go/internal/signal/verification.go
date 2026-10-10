package signal

import (
	"context"
	"fmt"
	"time"
)

const (
	// Verification layer weights (must sum to 1.0)
	WeightFormalValidity     = 0.40
	WeightSemanticValidity   = 0.35
	WeightInfoTheoreticValue = 0.25

	// DefaultVerificationThreshold is the minimum composite score for a signal
	// to be considered valid and routable.
	DefaultVerificationThreshold = 0.6
)

// VerificationLayerScore holds the score and evidence for one verification layer.
type VerificationLayerScore struct {
	Layer       string  `json:"layer"`
	Score       float64 `json:"score"`
	Weight      float64 `json:"weight"`
	Weighted    float64 `json:"weighted"`
	Description string  `json:"description"`
	Passed      bool    `json:"passed"`
}

// VerificationResult is the output of triple-layer verification for a signal.
type VerificationResult struct {
	SignalID       string                   `json:"signal_id"`
	CompositeScore float64                  `json:"composite_score"`
	Passed         bool                     `json:"passed"`
	Threshold      float64                  `json:"threshold"`
	Layers         []VerificationLayerScore `json:"layers"`
	FailureReports []FailureReport          `json:"failure_reports,omitempty"`
	VerifiedAt     time.Time                `json:"verified_at"`
}

// FormalValidator checks structural and schema validity of a signal.
// Layer 1 of triple-layer verification (40% weight).
type FormalValidator interface {
	Validate(ctx context.Context, s *Signal) (VerificationLayerScore, error)
}

// SemanticValidator checks contextual appropriateness of a signal.
// Layer 2 of triple-layer verification (35% weight).
type SemanticValidator interface {
	Validate(ctx context.Context, s *Signal) (VerificationLayerScore, error)
}

// InfoTheoreticValidator checks the signal's informational value.
// Layer 3 of triple-layer verification (25% weight).
type InfoTheoreticValidator interface {
	Validate(ctx context.Context, s *Signal) (VerificationLayerScore, error)
}

// Verifier runs triple-layer verification on signals.
// Constructed via NewVerifier -- no global singletons.
type Verifier struct {
	formal        FormalValidator
	semantic      SemanticValidator
	infoTheoretic InfoTheoreticValidator
	detectors     *FailureDetectorRegistry
	threshold     float64
}

// NewVerifier constructs a Verifier with all three validation layers.
// All validators are required. Uses DefaultVerificationThreshold if threshold is 0.
func NewVerifier(
	_ context.Context,
	formal FormalValidator,
	semantic SemanticValidator,
	infoTheoretic InfoTheoreticValidator,
	detectors *FailureDetectorRegistry,
	threshold float64,
) (*Verifier, error) {
	if formal == nil {
		return nil, fmt.Errorf("verifier: FormalValidator is required")
	}
	if semantic == nil {
		return nil, fmt.Errorf("verifier: SemanticValidator is required")
	}
	if infoTheoretic == nil {
		return nil, fmt.Errorf("verifier: InfoTheoreticValidator is required")
	}
	if detectors == nil {
		return nil, fmt.Errorf("verifier: FailureDetectorRegistry is required")
	}
	if threshold <= 0 {
		threshold = DefaultVerificationThreshold
	}
	return &Verifier{
		formal:        formal,
		semantic:      semantic,
		infoTheoretic: infoTheoretic,
		detectors:     detectors,
		threshold:     threshold,
	}, nil
}

// Verify runs all three verification layers against a signal and returns a
// VerificationResult. The composite score is the weighted sum of layer scores.
func (v *Verifier) Verify(ctx context.Context, s *Signal) (VerificationResult, error) {
	if s == nil {
		return VerificationResult{}, fmt.Errorf("verifier: signal must not be nil")
	}

	result := VerificationResult{
		SignalID:   s.ID,
		Threshold:  v.threshold,
		VerifiedAt: time.Now().UTC(),
	}

	// Layer 1: Formal validity (40%)
	formalScore, err := v.formal.Validate(ctx, s)
	if err != nil {
		return result, fmt.Errorf("verifier: formal layer: %w", err)
	}
	formalScore.Layer = "formal"
	formalScore.Weight = WeightFormalValidity
	formalScore.Weighted = formalScore.Score * WeightFormalValidity
	formalScore.Passed = formalScore.Score >= 0.7

	// Layer 2: Semantic validity (35%)
	semanticScore, err := v.semantic.Validate(ctx, s)
	if err != nil {
		return result, fmt.Errorf("verifier: semantic layer: %w", err)
	}
	semanticScore.Layer = "semantic"
	semanticScore.Weight = WeightSemanticValidity
	semanticScore.Weighted = semanticScore.Score * WeightSemanticValidity
	semanticScore.Passed = semanticScore.Score >= 0.6

	// Layer 3: Info-theoretic value (25%)
	infoScore, err := v.infoTheoretic.Validate(ctx, s)
	if err != nil {
		return result, fmt.Errorf("verifier: info_theoretic layer: %w", err)
	}
	infoScore.Layer = "info_theoretic"
	infoScore.Weight = WeightInfoTheoreticValue
	infoScore.Weighted = infoScore.Score * WeightInfoTheoreticValue
	infoScore.Passed = infoScore.Score >= 0.5

	result.Layers = []VerificationLayerScore{formalScore, semanticScore, infoScore}
	result.CompositeScore = formalScore.Weighted + semanticScore.Weighted + infoScore.Weighted
	result.Passed = result.CompositeScore >= v.threshold

	// Run failure detectors — propagate errors so callers know the result
	// may be incomplete. A silent swallow here would mask detector failures
	// and let a signal pass verification without adversarial checks running.
	failureReports, detectErr := v.detectors.DetectAll(ctx, s)
	if detectErr != nil {
		result.FailureReports = []FailureReport{}
		return result, fmt.Errorf("verifier: failure detector pipeline: %w", detectErr)
	}
	result.FailureReports = failureReports

	return result, nil
}
