package signal

import (
	"context"
	"fmt"
	"time"
)

// FailureSeverity classifies how serious a detected failure is.
type FailureSeverity string

const (
	SeverityCritical FailureSeverity = "CRITICAL" // system cannot continue
	SeverityHigh     FailureSeverity = "HIGH"     // significant degradation
	SeverityMedium   FailureSeverity = "MEDIUM"   // noticeable impairment
	SeverityLow      FailureSeverity = "LOW"      // minor issue, log and continue
)

// FailureReport is the output of a failure mode detector.
type FailureReport struct {
	DetectorName string          // which detector produced this report
	Detected     bool            // whether a failure was detected
	Severity     FailureSeverity // severity if detected
	Description  string          // human-readable description of the failure
	Signal       *Signal         // the signal that triggered the detection
	DetectedAt   time.Time       // when the failure was detected
	Metadata     map[string]any  // detector-specific additional data
}

// FailureDetector is the base interface all 11 failure mode detectors implement.
type FailureDetector interface {
	// Name returns the detector's canonical name.
	Name() string
	// Detect analyzes the signal and returns a FailureReport.
	// Returns a report with Detected=false if no failure is found.
	Detect(ctx context.Context, s *Signal) (FailureReport, error)
}

// --- Shannon Failure Modes (Information Theory) ---

// RoutingFailure detects when a signal cannot be routed to the correct mode.
// In Shannon terms: the signal arrives at the wrong receiver, or no receiver
// is available — equivalent to a communication channel with no destination.
// Theory: Shannon — channel routing and destination resolution.
type RoutingFailure interface {
	FailureDetector
	// SetRouteTable provides the detector with the current valid routes.
	// Called when route configuration changes.
	SetRouteTable(ctx context.Context, routes map[Mode][]Genre) error
}

// BandwidthOverload detects when the system is receiving more signals than it
// can process within acceptable latency bounds. Maps to Shannon channel capacity
// exceeded — the channel cannot carry all the information being sent.
// Theory: Shannon — channel capacity saturation.
type BandwidthOverload interface {
	FailureDetector
	// SetCapacityThreshold configures the maximum acceptable signal rate.
	SetCapacityThreshold(ctx context.Context, maxPerSecond float64) error
}

// FidelityLoss detects when a signal loses informational content during
// transmission or processing. Maps to Shannon noise — the received signal
// has lower information content than the sent signal.
// Theory: Shannon — noise-induced information loss.
type FidelityLoss interface {
	FailureDetector
	// CompareFidelity compares original and processed signals for information loss.
	CompareFidelity(ctx context.Context, original, processed *Signal) (float64, error)
}

// --- Ashby Failure Modes (Cybernetics / Requisite Variety) ---

// GenreMismatch detects when the genre of a signal does not match the
// genre expected by the receiving mode. Ashby: the receiver lacks the
// variety to handle the genre being sent.
// Theory: Ashby — requisite variety mismatch between sender and receiver.
type GenreMismatch interface {
	FailureDetector
	// ExpectedGenres returns the genres the given mode is designed to handle.
	ExpectedGenres(ctx context.Context, mode Mode) ([]Genre, error)
}

// VarietyFailure detects when the system's requisite variety (Ashby's Law)
// falls below the variety of its environment. The system cannot respond
// adequately to the complexity of the signals it receives.
// Theory: Ashby — Law of Requisite Variety violation (R < environment complexity).
type VarietyFailure interface {
	FailureDetector
	// VarietyRatio returns the ratio of system variety to environment variety.
	// A ratio >= 1.0 indicates adequate variety. < 1.0 indicates variety failure.
	VarietyRatio(ctx context.Context) (float64, error)
}

// StructureFailure detects when a signal's structural format (JSON schema,
// field presence, type correctness) does not match the expected structure.
// Ashby: the signal cannot be decoded because its structure is unexpected.
// Theory: Ashby — structural encoding mismatch.
type StructureFailure interface {
	FailureDetector
	// ValidateStructure checks the signal payload against the expected schema.
	ValidateStructure(ctx context.Context, s *Signal, schema []byte) error
}

// --- Beer Failure Modes (Viable System Model) ---

// BridgeFailure detects failures in the communication channel between VSM
// system levels (e.g., System 3 cannot receive reports from System 1).
// Beer: the algedonic channel or command channel is severed.
// Theory: Beer — inter-system communication channel failure.
type BridgeFailure interface {
	FailureDetector
	// CheckBridge verifies the communication channel between two VSM levels.
	CheckBridge(ctx context.Context, from, to VSMSystem) error
}

// Herniation detects when a lower VSM system level is being managed directly
// by a higher-than-one-step-up system, bypassing intermediate systems.
// Beer: System 5 issuing direct orders to System 1, bypassing 3 and 4.
// Theory: Beer — VSM level bypass (management herniation).
type Herniation interface {
	FailureDetector
	// DetectBypass checks whether the signal skips expected VSM system levels.
	DetectBypass(ctx context.Context, s *Signal, expectedPath []VSMSystem) (bool, error)
}

// Decay detects when signal quality degrades over multiple routing hops.
// Beer: the signal loses structural integrity as it passes through system levels,
// arriving at the destination in a degraded or distorted state.
// Theory: Beer — multi-hop signal degradation.
type Decay interface {
	FailureDetector
	// TrackSignal records a signal's state at a routing hop for decay analysis.
	TrackSignal(ctx context.Context, s *Signal, hop int) error
	// DecayRate returns the measured decay rate for a signal over its routing path.
	DecayRate(ctx context.Context, signalID string) (float64, error)
}

// --- Wiener Failure Modes (Feedback / Cybernetics) ---

// FeedbackFailure detects when the feedback loop (SENSOR-COMPARATOR-ACTUATOR)
// is broken. Wiener: the system cannot correct itself because it cannot
// receive or process feedback signals.
// Theory: Wiener — broken cybernetic feedback loop.
type FeedbackFailure interface {
	FailureDetector
	// LoopClosed checks whether the feedback loop for a given signal completed.
	LoopClosed(ctx context.Context, originSignalID string) (bool, error)
	// LoopLatency returns the time taken for the feedback loop to close.
	LoopLatency(ctx context.Context, originSignalID string) (time.Duration, error)
}

// --- Adversarial Failure Modes ---

// AdversarialNoise detects signals that appear valid but contain adversarial
// content designed to manipulate the system's behavior. Applicable across all
// theoretical frameworks — adversarial signals can exploit Shannon, Ashby, and
// Beer failure vectors simultaneously.
// Theory: Adversarial — cross-framework signal manipulation.
type AdversarialNoise interface {
	FailureDetector
	// AnomalyScore returns a score [0.0, 1.0] indicating adversarial likelihood.
	// 0.0 = clean, 1.0 = highly likely adversarial.
	AnomalyScore(ctx context.Context, s *Signal) (float64, error)
}

// FailureDetectorRegistry holds all 11 failure mode detectors.
// Constructed via NewFailureDetectorRegistry -- no global singletons.
type FailureDetectorRegistry struct {
	// Shannon
	RoutingFailure    RoutingFailure
	BandwidthOverload BandwidthOverload
	FidelityLoss      FidelityLoss
	// Ashby
	GenreMismatch    GenreMismatch
	VarietyFailure   VarietyFailure
	StructureFailure StructureFailure
	// Beer
	BridgeFailure BridgeFailure
	Herniation    Herniation
	Decay         Decay
	// Wiener
	FeedbackFailure FeedbackFailure
	// Adversarial
	AdversarialNoise AdversarialNoise
}

// NewFailureDetectorRegistry constructs a registry with the provided detectors.
// Nil detectors are accepted — they are skipped during DetectAll.
func NewFailureDetectorRegistry(
	ctx context.Context,
	routingFailure RoutingFailure,
	bandwidthOverload BandwidthOverload,
	fidelityLoss FidelityLoss,
	genreMismatch GenreMismatch,
	varietyFailure VarietyFailure,
	structureFailure StructureFailure,
	bridgeFailure BridgeFailure,
	herniation Herniation,
	decay Decay,
	feedbackFailure FeedbackFailure,
	adversarialNoise AdversarialNoise,
) (*FailureDetectorRegistry, error) {
	return &FailureDetectorRegistry{
		RoutingFailure:    routingFailure,
		BandwidthOverload: bandwidthOverload,
		FidelityLoss:      fidelityLoss,
		GenreMismatch:     genreMismatch,
		VarietyFailure:    varietyFailure,
		StructureFailure:  structureFailure,
		BridgeFailure:     bridgeFailure,
		Herniation:        herniation,
		Decay:             decay,
		FeedbackFailure:   feedbackFailure,
		AdversarialNoise:  adversarialNoise,
	}, nil
}

// DetectAll runs all registered (non-nil) detectors against a signal and returns reports.
// Nil detectors are skipped gracefully. An error return indicates a
// detector itself failed (not that it detected a failure).
func (r *FailureDetectorRegistry) DetectAll(ctx context.Context, s *Signal) ([]FailureReport, error) {
	var detectors []FailureDetector
	if r.RoutingFailure != nil {
		detectors = append(detectors, r.RoutingFailure)
	}
	if r.BandwidthOverload != nil {
		detectors = append(detectors, r.BandwidthOverload)
	}
	if r.FidelityLoss != nil {
		detectors = append(detectors, r.FidelityLoss)
	}
	if r.GenreMismatch != nil {
		detectors = append(detectors, r.GenreMismatch)
	}
	if r.VarietyFailure != nil {
		detectors = append(detectors, r.VarietyFailure)
	}
	if r.StructureFailure != nil {
		detectors = append(detectors, r.StructureFailure)
	}
	if r.BridgeFailure != nil {
		detectors = append(detectors, r.BridgeFailure)
	}
	if r.Herniation != nil {
		detectors = append(detectors, r.Herniation)
	}
	if r.Decay != nil {
		detectors = append(detectors, r.Decay)
	}
	if r.FeedbackFailure != nil {
		detectors = append(detectors, r.FeedbackFailure)
	}
	if r.AdversarialNoise != nil {
		detectors = append(detectors, r.AdversarialNoise)
	}

	var reports []FailureReport
	for _, d := range detectors {
		report, err := d.Detect(ctx, s)
		if err != nil {
			return nil, fmt.Errorf("failure_modes: detector %s: %w", d.Name(), err)
		}
		reports = append(reports, report)
	}
	return reports, nil
}
