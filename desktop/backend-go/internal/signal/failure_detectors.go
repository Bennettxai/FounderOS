package signal

import (
	"context"
	"sync"
	"time"
)

// --- BandwidthOverloadDetector (Shannon) ---

// BandwidthOverloadDetector detects when signal rate exceeds capacity.
type BandwidthOverloadDetector struct {
	mu            sync.RWMutex
	maxPerSecond  float64
	signalTimes   []time.Time
	windowSeconds float64
}

// NewBandwidthOverloadDetector creates a detector with the given capacity threshold.
func NewBandwidthOverloadDetector(maxPerSecond float64) *BandwidthOverloadDetector {
	if maxPerSecond <= 0 {
		maxPerSecond = 10.0
	}
	return &BandwidthOverloadDetector{
		maxPerSecond:  maxPerSecond,
		windowSeconds: 60.0,
	}
}

func (d *BandwidthOverloadDetector) Name() string { return "bandwidth_overload" }

func (d *BandwidthOverloadDetector) Detect(_ context.Context, _ *Signal) (FailureReport, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	now := time.Now().UTC()
	cutoff := now.Add(-time.Duration(d.windowSeconds) * time.Second)

	var count int
	for _, t := range d.signalTimes {
		if t.After(cutoff) {
			count++
		}
	}

	rate := float64(count) / d.windowSeconds
	if rate > d.maxPerSecond {
		return FailureReport{
			DetectorName: d.Name(),
			Detected:     true,
			Severity:     SeverityHigh,
			Description:  "Signal rate exceeds channel capacity (Shannon bandwidth overload)",
			DetectedAt:   now,
			Metadata:     map[string]any{"rate": rate, "max": d.maxPerSecond},
		}, nil
	}
	return FailureReport{DetectorName: d.Name()}, nil
}

func (d *BandwidthOverloadDetector) SetCapacityThreshold(_ context.Context, maxPerSecond float64) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.maxPerSecond = maxPerSecond
	return nil
}

// RecordSignal records a signal timestamp for rate calculation.
func (d *BandwidthOverloadDetector) RecordSignal() {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now().UTC()
	d.signalTimes = append(d.signalTimes, now)
	// Prune old entries
	cutoff := now.Add(-time.Duration(d.windowSeconds) * time.Second)
	kept := d.signalTimes[:0]
	for _, t := range d.signalTimes {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	d.signalTimes = kept
}

// --- FeedbackFailureDetector (Wiener) ---

// FeedbackFailureDetector checks if feedback loops are closing.
type FeedbackFailureDetector struct {
	mu     sync.RWMutex
	loops  map[string]*feedbackLoop
	maxAge time.Duration
}

type feedbackLoop struct {
	opened   time.Time
	closed   bool
	closedAt time.Time
}

// NewFeedbackFailureDetector creates a detector with the given max loop age.
func NewFeedbackFailureDetector(maxAge time.Duration) *FeedbackFailureDetector {
	if maxAge == 0 {
		maxAge = 5 * time.Minute
	}
	return &FeedbackFailureDetector{
		loops:  make(map[string]*feedbackLoop),
		maxAge: maxAge,
	}
}

func (d *FeedbackFailureDetector) Name() string { return "feedback_failure" }

func (d *FeedbackFailureDetector) Detect(_ context.Context, s *Signal) (FailureReport, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Check overall closure rate
	now := time.Now().UTC()
	cutoff := now.Add(-d.maxAge)
	var total, closed int
	for _, l := range d.loops {
		if l.opened.After(cutoff) {
			total++
			if l.closed {
				closed++
			}
		}
	}

	if total < 5 {
		// Not enough data
		return FailureReport{DetectorName: d.Name()}, nil
	}

	rate := float64(closed) / float64(total)
	if rate < 0.5 {
		return FailureReport{
			DetectorName: d.Name(),
			Detected:     true,
			Severity:     SeverityHigh,
			Description:  "Feedback loop closure rate below 50% (Wiener feedback failure)",
			DetectedAt:   now,
			Metadata:     map[string]any{"closure_rate": rate, "total": total, "closed": closed},
		}, nil
	}
	return FailureReport{DetectorName: d.Name()}, nil
}

func (d *FeedbackFailureDetector) LoopClosed(_ context.Context, originSignalID string) (bool, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if l, ok := d.loops[originSignalID]; ok {
		return l.closed, nil
	}
	return false, nil
}

func (d *FeedbackFailureDetector) LoopLatency(_ context.Context, originSignalID string) (time.Duration, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if l, ok := d.loops[originSignalID]; ok && l.closed {
		return l.closedAt.Sub(l.opened), nil
	}
	return 0, nil
}

// OpenLoop records the start of a feedback loop.
func (d *FeedbackFailureDetector) OpenLoop(signalID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.loops[signalID] = &feedbackLoop{opened: time.Now().UTC()}
	d.prune()
}

// CloseLoop records the completion of a feedback loop.
func (d *FeedbackFailureDetector) CloseLoop(signalID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if l, ok := d.loops[signalID]; ok {
		l.closed = true
		l.closedAt = time.Now().UTC()
	}
}

func (d *FeedbackFailureDetector) prune() {
	cutoff := time.Now().UTC().Add(-2 * d.maxAge)
	for id, l := range d.loops {
		if l.opened.Before(cutoff) {
			delete(d.loops, id)
		}
	}
}

// --- RoutingFailureDetector (Shannon) ---

// RoutingFailureDetector checks signal bounce rate for routing failures.
type RoutingFailureDetector struct {
	mu          sync.RWMutex
	routes      map[Mode][]Genre
	bounceCount int
	totalCount  int
	window      []routingEvent
}

type routingEvent struct {
	bounced bool
	at      time.Time
}

// NewRoutingFailureDetector creates a routing failure detector.
func NewRoutingFailureDetector() *RoutingFailureDetector {
	return &RoutingFailureDetector{
		routes: make(map[Mode][]Genre),
	}
}

func (d *RoutingFailureDetector) Name() string { return "routing_failure" }

func (d *RoutingFailureDetector) Detect(_ context.Context, _ *Signal) (FailureReport, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	now := time.Now().UTC()
	cutoff := now.Add(-5 * time.Minute)
	var total, bounced int
	for _, e := range d.window {
		if e.at.After(cutoff) {
			total++
			if e.bounced {
				bounced++
			}
		}
	}

	if total < 5 {
		return FailureReport{DetectorName: d.Name()}, nil
	}

	rate := float64(bounced) / float64(total)
	if rate > 0.3 {
		return FailureReport{
			DetectorName: d.Name(),
			Detected:     true,
			Severity:     SeverityMedium,
			Description:  "Signal bounce rate >30% indicates routing failure (Shannon)",
			DetectedAt:   now,
			Metadata:     map[string]any{"bounce_rate": rate, "total": total},
		}, nil
	}
	return FailureReport{DetectorName: d.Name()}, nil
}

func (d *RoutingFailureDetector) SetRouteTable(_ context.Context, routes map[Mode][]Genre) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.routes = routes
	return nil
}

// RecordRouting records a routing event (bounced or not).
func (d *RoutingFailureDetector) RecordRouting(bounced bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.window = append(d.window, routingEvent{bounced: bounced, at: time.Now().UTC()})
	// Prune old events
	cutoff := time.Now().UTC().Add(-10 * time.Minute)
	kept := d.window[:0]
	for _, e := range d.window {
		if e.at.After(cutoff) {
			kept = append(kept, e)
		}
	}
	d.window = kept
}

// --- GenreMismatchDetector (Ashby) ---

// GenreMismatchDetector checks genre recognition accuracy.
type GenreMismatchDetector struct {
	mu       sync.RWMutex
	events   []genreEvent
	expected map[Mode][]Genre
}

type genreEvent struct {
	matched bool
	at      time.Time
}

// NewGenreMismatchDetector creates a genre mismatch detector.
func NewGenreMismatchDetector() *GenreMismatchDetector {
	return &GenreMismatchDetector{
		expected: map[Mode][]Genre{
			ModeExecute:  {GenreDirect, GenreCommit},
			ModeAssist:   {GenreInform, GenreDirect, GenreExpress},
			ModeAnalyze:  {GenreInform, GenreDecide},
			ModeBuild:    {GenreDirect, GenreCommit},
			ModeMaintain: {GenreDecide, GenreInform},
		},
	}
}

func (d *GenreMismatchDetector) Name() string { return "genre_mismatch" }

func (d *GenreMismatchDetector) Detect(_ context.Context, _ *Signal) (FailureReport, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	now := time.Now().UTC()
	cutoff := now.Add(-5 * time.Minute)
	var total, matched int
	for _, e := range d.events {
		if e.at.After(cutoff) {
			total++
			if e.matched {
				matched++
			}
		}
	}

	if total < 5 {
		return FailureReport{DetectorName: d.Name()}, nil
	}

	rate := float64(matched) / float64(total)
	if rate < 0.7 {
		return FailureReport{
			DetectorName: d.Name(),
			Detected:     true,
			Severity:     SeverityMedium,
			Description:  "Genre recognition <70% indicates Ashby variety failure",
			DetectedAt:   now,
			Metadata:     map[string]any{"recognition_rate": rate, "total": total},
		}, nil
	}
	return FailureReport{DetectorName: d.Name()}, nil
}

func (d *GenreMismatchDetector) ExpectedGenres(_ context.Context, mode Mode) ([]Genre, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.expected[mode], nil
}

// RecordRecognition records whether a genre was correctly recognized.
func (d *GenreMismatchDetector) RecordRecognition(matched bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.events = append(d.events, genreEvent{matched: matched, at: time.Now().UTC()})
	cutoff := time.Now().UTC().Add(-10 * time.Minute)
	kept := d.events[:0]
	for _, e := range d.events {
		if e.at.After(cutoff) {
			kept = append(kept, e)
		}
	}
	d.events = kept
}

// --- BridgeFailureDetector (Beer) ---

// BridgeFailureDetector checks VSM inter-system communication.
type BridgeFailureDetector struct {
	mu          sync.RWMutex
	transitions []modeTransition
}

type modeTransition struct {
	from Mode
	to   Mode
	at   time.Time
}

// NewBridgeFailureDetector creates a bridge failure detector.
func NewBridgeFailureDetector() *BridgeFailureDetector {
	return &BridgeFailureDetector{}
}

func (d *BridgeFailureDetector) Name() string { return "bridge_failure" }

func (d *BridgeFailureDetector) Detect(_ context.Context, _ *Signal) (FailureReport, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	now := time.Now().UTC()
	cutoff := now.Add(-10 * time.Minute)

	// Detect rapid mode oscillation (bridge instability)
	var recent []modeTransition
	for _, t := range d.transitions {
		if t.at.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) < 3 {
		return FailureReport{DetectorName: d.Name()}, nil
	}

	// Check for oscillation: A→B→A pattern within 2 minutes
	for i := 2; i < len(recent); i++ {
		if recent[i].from == recent[i-2].from &&
			recent[i].to == recent[i-2].to &&
			recent[i].at.Sub(recent[i-2].at) < 2*time.Minute {
			return FailureReport{
				DetectorName: d.Name(),
				Detected:     true,
				Severity:     SeverityMedium,
				Description:  "Mode oscillation detected — VSM bridge instability (Beer)",
				DetectedAt:   now,
				Metadata: map[string]any{
					"from": string(recent[i].from),
					"to":   string(recent[i].to),
				},
			}, nil
		}
	}

	return FailureReport{DetectorName: d.Name()}, nil
}

func (d *BridgeFailureDetector) CheckBridge(_ context.Context, _, _ VSMSystem) error {
	return nil // Always returns nil — no inter-process bridges to check yet
}

// RecordTransition records a mode transition event.
func (d *BridgeFailureDetector) RecordTransition(from, to Mode) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.transitions = append(d.transitions, modeTransition{from: from, to: to, at: time.Now().UTC()})
	// Prune
	cutoff := time.Now().UTC().Add(-15 * time.Minute)
	kept := d.transitions[:0]
	for _, t := range d.transitions {
		if t.at.After(cutoff) {
			kept = append(kept, t)
		}
	}
	d.transitions = kept
}
