package signal

import (
	"context"
	"math"
	"testing"
	"time"
)

// =============================================================================
// Agent B Tests — Proxy Metrics (B-01)
// =============================================================================

func TestNewProxyMetricsRegistry_NilCollector(t *testing.T) {
	ctx := context.Background()

	// Helper: create a full set of non-nil mock collectors
	mocks := func() (ActionCompletionRate, ReEncodingFrequency, TimeToDecide, SignalBounceRate, GenreRecognitionRate, FeedbackLoopClosure) {
		return &stubActionCompletion{}, &stubReEncoding{}, &stubTimeToDecide{},
			&stubSignalBounce{}, &stubGenreRecognition{}, &stubFeedbackClosure{}
	}

	tests := []struct {
		name    string
		modify  func(ActionCompletionRate, ReEncodingFrequency, TimeToDecide, SignalBounceRate, GenreRecognitionRate, FeedbackLoopClosure) (ActionCompletionRate, ReEncodingFrequency, TimeToDecide, SignalBounceRate, GenreRecognitionRate, FeedbackLoopClosure)
		wantErr string
	}{
		{
			name: "nil ActionCompletionRate",
			modify: func(a ActionCompletionRate, b ReEncodingFrequency, c TimeToDecide, d SignalBounceRate, e GenreRecognitionRate, f FeedbackLoopClosure) (ActionCompletionRate, ReEncodingFrequency, TimeToDecide, SignalBounceRate, GenreRecognitionRate, FeedbackLoopClosure) {
				return nil, b, c, d, e, f
			},
			wantErr: "ActionCompletionRate",
		},
		{
			name: "nil ReEncodingFrequency",
			modify: func(a ActionCompletionRate, b ReEncodingFrequency, c TimeToDecide, d SignalBounceRate, e GenreRecognitionRate, f FeedbackLoopClosure) (ActionCompletionRate, ReEncodingFrequency, TimeToDecide, SignalBounceRate, GenreRecognitionRate, FeedbackLoopClosure) {
				return a, nil, c, d, e, f
			},
			wantErr: "ReEncodingFrequency",
		},
		{
			name: "nil TimeToDecide",
			modify: func(a ActionCompletionRate, b ReEncodingFrequency, c TimeToDecide, d SignalBounceRate, e GenreRecognitionRate, f FeedbackLoopClosure) (ActionCompletionRate, ReEncodingFrequency, TimeToDecide, SignalBounceRate, GenreRecognitionRate, FeedbackLoopClosure) {
				return a, b, nil, d, e, f
			},
			wantErr: "TimeToDecide",
		},
		{
			name: "nil SignalBounceRate",
			modify: func(a ActionCompletionRate, b ReEncodingFrequency, c TimeToDecide, d SignalBounceRate, e GenreRecognitionRate, f FeedbackLoopClosure) (ActionCompletionRate, ReEncodingFrequency, TimeToDecide, SignalBounceRate, GenreRecognitionRate, FeedbackLoopClosure) {
				return a, b, c, nil, e, f
			},
			wantErr: "SignalBounceRate",
		},
		{
			name: "nil GenreRecognitionRate",
			modify: func(a ActionCompletionRate, b ReEncodingFrequency, c TimeToDecide, d SignalBounceRate, e GenreRecognitionRate, f FeedbackLoopClosure) (ActionCompletionRate, ReEncodingFrequency, TimeToDecide, SignalBounceRate, GenreRecognitionRate, FeedbackLoopClosure) {
				return a, b, c, d, nil, f
			},
			wantErr: "GenreRecognitionRate",
		},
		{
			name: "nil FeedbackLoopClosure",
			modify: func(a ActionCompletionRate, b ReEncodingFrequency, c TimeToDecide, d SignalBounceRate, e GenreRecognitionRate, f FeedbackLoopClosure) (ActionCompletionRate, ReEncodingFrequency, TimeToDecide, SignalBounceRate, GenreRecognitionRate, FeedbackLoopClosure) {
				return a, b, c, d, e, nil
			},
			wantErr: "FeedbackLoopClosure",
		},
		{
			name: "all valid",
			modify: func(a ActionCompletionRate, b ReEncodingFrequency, c TimeToDecide, d SignalBounceRate, e GenreRecognitionRate, f FeedbackLoopClosure) (ActionCompletionRate, ReEncodingFrequency, TimeToDecide, SignalBounceRate, GenreRecognitionRate, FeedbackLoopClosure) {
				return a, b, c, d, e, f
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b, c, d, e, f := mocks()
			a, b, c, d, e, f = tt.modify(a, b, c, d, e, f)
			reg, err := NewProxyMetricsRegistry(ctx, a, b, c, d, e, f)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				if reg != nil {
					t.Fatal("expected nil registry on error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if reg == nil {
					t.Fatal("expected non-nil registry")
				}
			}
		})
	}
}

// =============================================================================
// Agent B Tests — Failure Modes (B-02)
// =============================================================================

func TestNewFailureDetectorRegistry_AllNil(t *testing.T) {
	ctx := context.Background()

	// All nil — should succeed (nil-safe registry skips nil detectors in DetectAll)
	reg, err := NewFailureDetectorRegistry(ctx, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("expected no error for all-nil registry, got: %v", err)
	}

	// DetectAll with all-nil detectors should return empty reports
	reports, err := reg.DetectAll(ctx, &Signal{})
	if err != nil {
		t.Fatalf("expected no error from DetectAll on nil registry, got: %v", err)
	}
	if len(reports) != 0 {
		t.Fatalf("expected 0 reports, got %d", len(reports))
	}
}

func TestFailureReport_Fields(t *testing.T) {
	now := time.Now()
	sig := &Signal{ID: "test-signal", Mode: ModeExecute, Genre: GenreDirect}
	report := FailureReport{
		DetectorName: "TestDetector",
		Detected:     true,
		Severity:     SeverityHigh,
		Description:  "test failure",
		Signal:       sig,
		DetectedAt:   now,
		Metadata:     map[string]any{"key": "value"},
	}

	if report.DetectorName != "TestDetector" {
		t.Errorf("DetectorName = %q, want %q", report.DetectorName, "TestDetector")
	}
	if !report.Detected {
		t.Error("Detected = false, want true")
	}
	if report.Severity != SeverityHigh {
		t.Errorf("Severity = %q, want %q", report.Severity, SeverityHigh)
	}
	if report.Description != "test failure" {
		t.Errorf("Description = %q, want %q", report.Description, "test failure")
	}
	if report.Signal != sig {
		t.Error("Signal pointer mismatch")
	}
	if !report.DetectedAt.Equal(now) {
		t.Error("DetectedAt mismatch")
	}
	if report.Metadata["key"] != "value" {
		t.Error("Metadata mismatch")
	}
}

func TestDetectorNames(t *testing.T) {
	args := makeFullDetectorArgs()
	ctx := context.Background()
	reg, err := NewFailureDetectorRegistry(ctx,
		args.routing, args.bandwidth, args.fidelity,
		args.genre, args.variety, args.structure,
		args.bridge, args.herniation, args.decay,
		args.feedback, args.adversarial,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify each detector has a non-empty Name()
	detectors := []FailureDetector{
		reg.RoutingFailure,
		reg.BandwidthOverload,
		reg.FidelityLoss,
		reg.GenreMismatch,
		reg.VarietyFailure,
		reg.StructureFailure,
		reg.BridgeFailure,
		reg.Herniation,
		reg.Decay,
		reg.FeedbackFailure,
		reg.AdversarialNoise,
	}
	for _, d := range detectors {
		if d.Name() == "" {
			t.Errorf("detector %T has empty Name()", d)
		}
	}
}

func TestDetectAll_AllClean(t *testing.T) {
	args := makeFullDetectorArgs()
	ctx := context.Background()
	reg, err := NewFailureDetectorRegistry(ctx,
		args.routing, args.bandwidth, args.fidelity,
		args.genre, args.variety, args.structure,
		args.bridge, args.herniation, args.decay,
		args.feedback, args.adversarial,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sig := &Signal{ID: "clean-signal", Mode: ModeAssist, Genre: GenreInform}
	reports, err := reg.DetectAll(ctx, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reports) != 11 {
		t.Fatalf("expected 11 reports, got %d", len(reports))
	}
	for _, r := range reports {
		if r.Detected {
			t.Errorf("detector %s unexpectedly detected a failure", r.DetectorName)
		}
	}
}

// =============================================================================
// Agent B Tests — Verification (B-03)
// =============================================================================

func TestNewVerifier_NilValidators(t *testing.T) {
	ctx := context.Background()
	reg := makeTestDetectorRegistry(t)

	tests := []struct {
		name      string
		formal    FormalValidator
		semantic  SemanticValidator
		info      InfoTheoreticValidator
		detectors *FailureDetectorRegistry
		wantErr   string
	}{
		{
			name:      "nil FormalValidator",
			formal:    nil,
			semantic:  &stubSemanticValidator{score: 0.8},
			info:      &stubInfoValidator{score: 0.7},
			detectors: reg,
			wantErr:   "FormalValidator",
		},
		{
			name:      "nil SemanticValidator",
			formal:    &stubFormalValidator{score: 0.9},
			semantic:  nil,
			info:      &stubInfoValidator{score: 0.7},
			detectors: reg,
			wantErr:   "SemanticValidator",
		},
		{
			name:      "nil InfoTheoreticValidator",
			formal:    &stubFormalValidator{score: 0.9},
			semantic:  &stubSemanticValidator{score: 0.8},
			info:      nil,
			detectors: reg,
			wantErr:   "InfoTheoreticValidator",
		},
		{
			name:      "nil FailureDetectorRegistry",
			formal:    &stubFormalValidator{score: 0.9},
			semantic:  &stubSemanticValidator{score: 0.8},
			info:      &stubInfoValidator{score: 0.7},
			detectors: nil,
			wantErr:   "FailureDetectorRegistry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewVerifier(ctx, tt.formal, tt.semantic, tt.info, tt.detectors, 0)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestVerifier_CompositeScore(t *testing.T) {
	ctx := context.Background()
	reg := makeTestDetectorRegistry(t)

	// formal=0.9, semantic=0.8, info=0.7
	// composite = 0.40*0.9 + 0.35*0.8 + 0.25*0.7 = 0.36 + 0.28 + 0.175 = 0.815
	v, err := NewVerifier(ctx,
		&stubFormalValidator{score: 0.9},
		&stubSemanticValidator{score: 0.8},
		&stubInfoValidator{score: 0.7},
		reg, 0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sig := &Signal{ID: "test-composite", Mode: ModeExecute, Genre: GenreDirect}
	result, err := v.Verify(ctx, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := 0.40*0.9 + 0.35*0.8 + 0.25*0.7
	if math.Abs(result.CompositeScore-expected) > 0.0001 {
		t.Errorf("CompositeScore = %f, want %f", result.CompositeScore, expected)
	}
	if result.SignalID != "test-composite" {
		t.Errorf("SignalID = %q, want %q", result.SignalID, "test-composite")
	}
	if len(result.Layers) != 3 {
		t.Fatalf("expected 3 layers, got %d", len(result.Layers))
	}

	// Verify individual layer weights
	if result.Layers[0].Layer != "formal" {
		t.Errorf("layer 0 = %q, want formal", result.Layers[0].Layer)
	}
	if math.Abs(result.Layers[0].Weight-WeightFormalValidity) > 0.0001 {
		t.Errorf("formal weight = %f, want %f", result.Layers[0].Weight, WeightFormalValidity)
	}
	if result.Layers[1].Layer != "semantic" {
		t.Errorf("layer 1 = %q, want semantic", result.Layers[1].Layer)
	}
	if result.Layers[2].Layer != "info_theoretic" {
		t.Errorf("layer 2 = %q, want info_theoretic", result.Layers[2].Layer)
	}
}

func TestVerifier_PassThreshold(t *testing.T) {
	ctx := context.Background()
	reg := makeTestDetectorRegistry(t)

	// All high scores — composite well above 0.6
	v, err := NewVerifier(ctx,
		&stubFormalValidator{score: 1.0},
		&stubSemanticValidator{score: 1.0},
		&stubInfoValidator{score: 1.0},
		reg, 0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sig := &Signal{ID: "pass-signal", Mode: ModeAssist}
	result, err := v.Verify(ctx, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Passed {
		t.Errorf("expected Passed=true, got false (composite=%f, threshold=%f)",
			result.CompositeScore, result.Threshold)
	}
	if result.Threshold != DefaultVerificationThreshold {
		t.Errorf("Threshold = %f, want %f", result.Threshold, DefaultVerificationThreshold)
	}
}

func TestVerifier_FailThreshold(t *testing.T) {
	ctx := context.Background()
	reg := makeTestDetectorRegistry(t)

	// All low scores — composite below 0.6
	// composite = 0.40*0.1 + 0.35*0.1 + 0.25*0.1 = 0.04 + 0.035 + 0.025 = 0.1
	v, err := NewVerifier(ctx,
		&stubFormalValidator{score: 0.1},
		&stubSemanticValidator{score: 0.1},
		&stubInfoValidator{score: 0.1},
		reg, 0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sig := &Signal{ID: "fail-signal", Mode: ModeBuild}
	result, err := v.Verify(ctx, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Passed {
		t.Errorf("expected Passed=false, got true (composite=%f, threshold=%f)",
			result.CompositeScore, result.Threshold)
	}
}

func TestVerifier_NilSignal(t *testing.T) {
	ctx := context.Background()
	reg := makeTestDetectorRegistry(t)

	v, err := NewVerifier(ctx,
		&stubFormalValidator{score: 1.0},
		&stubSemanticValidator{score: 1.0},
		&stubInfoValidator{score: 1.0},
		reg, 0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = v.Verify(ctx, nil)
	if err == nil {
		t.Fatal("expected error for nil signal")
	}
	if !contains(err.Error(), "nil") {
		t.Errorf("expected error to mention nil, got: %v", err)
	}
}

func TestVerifier_DefaultThreshold(t *testing.T) {
	ctx := context.Background()
	reg := makeTestDetectorRegistry(t)

	// threshold=0 should use DefaultVerificationThreshold
	v, err := NewVerifier(ctx,
		&stubFormalValidator{score: 0.9},
		&stubSemanticValidator{score: 0.8},
		&stubInfoValidator{score: 0.7},
		reg, 0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sig := &Signal{ID: "default-threshold", Mode: ModeExecute}
	result, err := v.Verify(ctx, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Threshold != DefaultVerificationThreshold {
		t.Errorf("Threshold = %f, want default %f", result.Threshold, DefaultVerificationThreshold)
	}
}

func TestVerifier_CustomThreshold(t *testing.T) {
	ctx := context.Background()
	reg := makeTestDetectorRegistry(t)

	v, err := NewVerifier(ctx,
		&stubFormalValidator{score: 0.9},
		&stubSemanticValidator{score: 0.8},
		&stubInfoValidator{score: 0.7},
		reg, 0.9,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sig := &Signal{ID: "custom-threshold", Mode: ModeAnalyze}
	result, err := v.Verify(ctx, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Threshold != 0.9 {
		t.Errorf("Threshold = %f, want 0.9", result.Threshold)
	}
	// composite = 0.815, threshold = 0.9 → should fail
	if result.Passed {
		t.Errorf("expected Passed=false with custom threshold 0.9 (composite=%f)",
			result.CompositeScore)
	}
}

func TestWeightsSumToOne(t *testing.T) {
	sum := WeightFormalValidity + WeightSemanticValidity + WeightInfoTheoreticValue
	if math.Abs(sum-1.0) > 0.0001 {
		t.Errorf("weights sum to %f, want 1.0", sum)
	}
}

// =============================================================================
// Test Helpers — Stubs and Utilities
// =============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- Metric Stubs (for B-01 tests) ---

type stubMetricBase struct{ name string }

func (s *stubMetricBase) Name() string { return s.name }
func (s *stubMetricBase) Collect(_ context.Context) (MetricValue, error) {
	return MetricValue{Name: s.name, Value: 0, Timestamp: time.Now()}, nil
}
func (s *stubMetricBase) Register(_ context.Context, _ CollectionPoint) error { return nil }

type stubActionCompletion struct{ stubMetricBase }

func (s *stubActionCompletion) Name() string { return "action_completion_rate" }
func (s *stubActionCompletion) Record(_ context.Context, _ *Signal, _ bool) error {
	return nil
}
func (s *stubActionCompletion) Rate(_ context.Context, _ time.Duration) (float64, error) {
	return 0.95, nil
}

type stubReEncoding struct{ stubMetricBase }

func (s *stubReEncoding) Name() string { return "re_encoding_frequency" }
func (s *stubReEncoding) Increment(_ context.Context, _ *Signal, _ string) error {
	return nil
}
func (s *stubReEncoding) Frequency(_ context.Context, _ time.Duration) (float64, error) {
	return 0.05, nil
}

type stubTimeToDecide struct{ stubMetricBase }

func (s *stubTimeToDecide) Name() string { return "time_to_decide" }
func (s *stubTimeToDecide) RecordLatency(_ context.Context, _ *Signal, _ time.Duration) error {
	return nil
}
func (s *stubTimeToDecide) Percentile(_ context.Context, _ int, _ time.Duration) (time.Duration, error) {
	return 100 * time.Millisecond, nil
}

type stubSignalBounce struct{ stubMetricBase }

func (s *stubSignalBounce) Name() string { return "signal_bounce_rate" }
func (s *stubSignalBounce) RecordBounce(_ context.Context, _ *Signal, _, _ Mode) error {
	return nil
}
func (s *stubSignalBounce) Rate(_ context.Context, _ time.Duration) (float64, error) {
	return 0.02, nil
}

type stubGenreRecognition struct{ stubMetricBase }

func (s *stubGenreRecognition) Name() string { return "genre_recognition_rate" }
func (s *stubGenreRecognition) RecordRecognition(_ context.Context, _ *Signal, _ bool) error {
	return nil
}
func (s *stubGenreRecognition) Rate(_ context.Context, _ time.Duration) (float64, error) {
	return 0.98, nil
}

type stubFeedbackClosure struct{ stubMetricBase }

func (s *stubFeedbackClosure) Name() string { return "feedback_loop_closure" }
func (s *stubFeedbackClosure) RecordFeedback(_ context.Context, _ *Signal, _ bool) error {
	return nil
}
func (s *stubFeedbackClosure) ClosureRate(_ context.Context, _ time.Duration) (float64, error) {
	return 0.90, nil
}

// --- Failure Detector Stubs (for B-02 tests) ---

type stubDetector struct{ name string }

func (s *stubDetector) Name() string { return s.name }
func (s *stubDetector) Detect(_ context.Context, sig *Signal) (FailureReport, error) {
	return FailureReport{
		DetectorName: s.name,
		Detected:     false,
		Signal:       sig,
		DetectedAt:   time.Now(),
	}, nil
}

type stubRouting struct{ stubDetector }

func (s *stubRouting) SetRouteTable(_ context.Context, _ map[Mode][]Genre) error { return nil }

type stubBandwidth struct{ stubDetector }

func (s *stubBandwidth) SetCapacityThreshold(_ context.Context, _ float64) error { return nil }

type stubFidelity struct{ stubDetector }

func (s *stubFidelity) CompareFidelity(_ context.Context, _, _ *Signal) (float64, error) {
	return 1.0, nil
}

type stubGenre struct{ stubDetector }

func (s *stubGenre) ExpectedGenres(_ context.Context, _ Mode) ([]Genre, error) {
	return []Genre{GenreDirect, GenreInform}, nil
}

type stubVariety struct{ stubDetector }

func (s *stubVariety) VarietyRatio(_ context.Context) (float64, error) { return 1.5, nil }

type stubStructure struct{ stubDetector }

func (s *stubStructure) ValidateStructure(_ context.Context, _ *Signal, _ []byte) error {
	return nil
}

type stubBridge struct{ stubDetector }

func (s *stubBridge) CheckBridge(_ context.Context, _, _ VSMSystem) error { return nil }

type stubHerniation struct{ stubDetector }

func (s *stubHerniation) DetectBypass(_ context.Context, _ *Signal, _ []VSMSystem) (bool, error) {
	return false, nil
}

type stubDecay struct{ stubDetector }

func (s *stubDecay) TrackSignal(_ context.Context, _ *Signal, _ int) error { return nil }
func (s *stubDecay) DecayRate(_ context.Context, _ string) (float64, error) {
	return 0.0, nil
}

type stubFeedback struct{ stubDetector }

func (s *stubFeedback) LoopClosed(_ context.Context, _ string) (bool, error) { return true, nil }
func (s *stubFeedback) LoopLatency(_ context.Context, _ string) (time.Duration, error) {
	return 50 * time.Millisecond, nil
}

type stubAdversarial struct{ stubDetector }

func (s *stubAdversarial) AnomalyScore(_ context.Context, _ *Signal) (float64, error) {
	return 0.0, nil
}

// detectorArgs holds all 11 detector stubs for test construction.
type detectorArgs struct {
	routing     RoutingFailure
	bandwidth   BandwidthOverload
	fidelity    FidelityLoss
	genre       GenreMismatch
	variety     VarietyFailure
	structure   StructureFailure
	bridge      BridgeFailure
	herniation  Herniation
	decay       Decay
	feedback    FeedbackFailure
	adversarial AdversarialNoise
}

func makeFullDetectorArgs() *detectorArgs {
	return &detectorArgs{
		routing:     &stubRouting{stubDetector{name: "RoutingFailure"}},
		bandwidth:   &stubBandwidth{stubDetector{name: "BandwidthOverload"}},
		fidelity:    &stubFidelity{stubDetector{name: "FidelityLoss"}},
		genre:       &stubGenre{stubDetector{name: "GenreMismatch"}},
		variety:     &stubVariety{stubDetector{name: "VarietyFailure"}},
		structure:   &stubStructure{stubDetector{name: "StructureFailure"}},
		bridge:      &stubBridge{stubDetector{name: "BridgeFailure"}},
		herniation:  &stubHerniation{stubDetector{name: "Herniation"}},
		decay:       &stubDecay{stubDetector{name: "Decay"}},
		feedback:    &stubFeedback{stubDetector{name: "FeedbackFailure"}},
		adversarial: &stubAdversarial{stubDetector{name: "AdversarialNoise"}},
	}
}

func setDetectorNil(args *detectorArgs, name string) {
	switch name {
	case "RoutingFailure":
		args.routing = nil
	case "BandwidthOverload":
		args.bandwidth = nil
	case "FidelityLoss":
		args.fidelity = nil
	case "GenreMismatch":
		args.genre = nil
	case "VarietyFailure":
		args.variety = nil
	case "StructureFailure":
		args.structure = nil
	case "BridgeFailure":
		args.bridge = nil
	case "Herniation":
		args.herniation = nil
	case "Decay":
		args.decay = nil
	case "FeedbackFailure":
		args.feedback = nil
	case "AdversarialNoise":
		args.adversarial = nil
	}
}

func makeTestDetectorRegistry(t *testing.T) *FailureDetectorRegistry {
	t.Helper()
	args := makeFullDetectorArgs()
	reg, err := NewFailureDetectorRegistry(context.Background(),
		args.routing, args.bandwidth, args.fidelity,
		args.genre, args.variety, args.structure,
		args.bridge, args.herniation, args.decay,
		args.feedback, args.adversarial,
	)
	if err != nil {
		t.Fatalf("failed to create detector registry: %v", err)
	}
	return reg
}

// --- Verification Stubs (for B-03 tests) ---

type stubFormalValidator struct{ score float64 }

func (s *stubFormalValidator) Validate(_ context.Context, _ *Signal) (VerificationLayerScore, error) {
	return VerificationLayerScore{Score: s.score}, nil
}

type stubSemanticValidator struct{ score float64 }

func (s *stubSemanticValidator) Validate(_ context.Context, _ *Signal) (VerificationLayerScore, error) {
	return VerificationLayerScore{Score: s.score}, nil
}

type stubInfoValidator struct{ score float64 }

func (s *stubInfoValidator) Validate(_ context.Context, _ *Signal) (VerificationLayerScore, error) {
	return VerificationLayerScore{Score: s.score}, nil
}
