// Package signal defines the Signal Theory 5-tuple S=(M,G,T,F,W) types.
//
// Every communication event in OSA is represented as a Signal:
//   - M (Mode):   operational mode, maps to Beer's VSM Systems 1-5
//   - G (Genre):  communicative purpose (speech act category)
//   - T (Type):   domain-specific signal category
//   - F (Format): serialization format
//   - W (Weight): informational value [0.0, 1.0] (Shannon information content)
//
// Reference: Roberto's Signal Theory paper, Section 3: Signal Formalism
package signal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Mode represents the OSA operational mode — maps to Beer's VSM Systems 1-5.
type Mode string

const (
	ModeExecute  Mode = "EXECUTE"  // VSM System 1: operational implementation
	ModeAssist   Mode = "ASSIST"   // VSM System 2: coordination and support
	ModeAnalyze  Mode = "ANALYZE"  // VSM System 3: monitoring and optimization
	ModeBuild    Mode = "BUILD"    // VSM System 4: strategic adaptation
	ModeMaintain Mode = "MAINTAIN" // VSM System 5: policy and identity
)

// VSMSystem identifies Beer's Viable System Model system level (1-5).
type VSMSystem int

const (
	VSMSystem1Operations   VSMSystem = 1 // Operational implementation (EXECUTE)
	VSMSystem2Coordination VSMSystem = 2 // Anti-oscillation coordination (ASSIST)
	VSMSystem3Optimization VSMSystem = 3 // Internal monitoring and optimization (ANALYZE)
	VSMSystem4Intelligence VSMSystem = 4 // Environmental scanning and adaptation (BUILD)
	VSMSystem5Policy       VSMSystem = 5 // Policy, identity, ultimate authority (MAINTAIN)
)

// Genre represents the communicative purpose of a signal.
// Derived from speech act theory as applied in Signal Theory.
type Genre string

const (
	GenreDirect  Genre = "DIRECT"  // Directive: cause an action
	GenreInform  Genre = "INFORM"  // Assertive: convey information
	GenreCommit  Genre = "COMMIT"  // Commissive: bind the sender
	GenreDecide  Genre = "DECIDE"  // Declarative: change system state
	GenreExpress Genre = "EXPRESS" // Expressive: convey internal state
)

// Format represents the serialization format of a signal.
type Format string

const (
	FormatJSON     Format = "JSON"
	FormatSSE      Format = "SSE"
	FormatMarkdown Format = "MARKDOWN"
	FormatCode     Format = "CODE"
)

// Signal is the 5-tuple S=(M,G,T,F,W) from Signal Theory.
// It is the canonical representation of any communication event in OSA.
type Signal struct {
	ID        string    `json:"id"`
	Mode      Mode      `json:"mode"`    // M: operational mode
	Genre     Genre     `json:"genre"`   // G: communicative purpose
	Type      string    `json:"type"`    // T: signal category (domain-specific)
	Format    Format    `json:"format"`  // F: serialization format
	Weight    float64   `json:"weight"`  // W: informational value [0.0, 1.0]
	Payload   []byte    `json:"payload"` // raw signal content
	CreatedAt time.Time `json:"created_at"`
	TenantID  string    `json:"tenant_id"`
}

// SignalOption is a functional option for NewSignal.
type SignalOption func(*Signal)

// WithTenantID sets the tenant ID on the signal.
func WithTenantID(tenantID string) SignalOption {
	return func(s *Signal) {
		s.TenantID = tenantID
	}
}

// WithPayload sets the raw payload on the signal.
func WithPayload(payload []byte) SignalOption {
	return func(s *Signal) {
		s.Payload = payload
	}
}

// NewSignal constructs a validated Signal 5-tuple.
// Returns an error if any required field is invalid.
func NewSignal(_ context.Context, mode Mode, genre Genre, sigType string, format Format, weight float64, opts ...SignalOption) (*Signal, error) {
	if err := validateMode(mode); err != nil {
		return nil, fmt.Errorf("signal: %w", err)
	}
	if err := validateGenre(genre); err != nil {
		return nil, fmt.Errorf("signal: %w", err)
	}
	if sigType == "" {
		return nil, fmt.Errorf("signal: type must not be empty")
	}
	if err := validateFormat(format); err != nil {
		return nil, fmt.Errorf("signal: %w", err)
	}
	if weight < 0.0 || weight > 1.0 {
		return nil, fmt.Errorf("signal: weight %.4f out of range [0.0, 1.0]", weight)
	}

	s := &Signal{
		ID:        uuid.NewString(),
		Mode:      mode,
		Genre:     genre,
		Type:      sigType,
		Format:    format,
		Weight:    weight,
		CreatedAt: time.Now().UTC(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

// String returns a human-readable representation of the signal.
func (s *Signal) String() string {
	return fmt.Sprintf("Signal{id=%s mode=%s genre=%s type=%s format=%s weight=%.2f}",
		s.ID, s.Mode, s.Genre, s.Type, s.Format, s.Weight)
}

// SignalResult represents the response to a processed signal.
type SignalResult struct {
	SignalID  string    `json:"signal_id"` // ID of the originating signal
	Success   bool      `json:"success"`
	Output    []byte    `json:"output"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// SignalError represents a signal processing failure with failure mode classification.
type SignalError struct {
	SignalID    string  `json:"signal_id"`
	FailureMode string  `json:"failure_mode"` // one of the 11 failure modes
	Message     string  `json:"message"`
	Severity    float64 `json:"severity"` // 0.0-1.0
}

// Error implements the error interface.
func (e *SignalError) Error() string {
	return fmt.Sprintf("signal error [%s] (severity=%.2f): %s", e.FailureMode, e.Severity, e.Message)
}

// MarshalJSON implements json.Marshaler for Signal.
func (s *Signal) MarshalJSON() ([]byte, error) {
	type alias Signal
	return json.Marshal((*alias)(s))
}

// UnmarshalJSON implements json.Unmarshaler for Signal.
func (s *Signal) UnmarshalJSON(data []byte) error {
	type alias Signal
	return json.Unmarshal(data, (*alias)(s))
}

// ValidModes returns all valid Mode values.
func ValidModes() []Mode {
	return []Mode{ModeExecute, ModeAssist, ModeAnalyze, ModeBuild, ModeMaintain}
}

// ValidGenres returns all valid Genre values.
func ValidGenres() []Genre {
	return []Genre{GenreDirect, GenreInform, GenreCommit, GenreDecide, GenreExpress}
}

// ValidFormats returns all valid Format values.
func ValidFormats() []Format {
	return []Format{FormatJSON, FormatSSE, FormatMarkdown, FormatCode}
}

func validateMode(m Mode) error {
	switch m {
	case ModeExecute, ModeAssist, ModeAnalyze, ModeBuild, ModeMaintain:
		return nil
	default:
		return fmt.Errorf("invalid mode %q", m)
	}
}

func validateGenre(g Genre) error {
	switch g {
	case GenreDirect, GenreInform, GenreCommit, GenreDecide, GenreExpress:
		return nil
	default:
		return fmt.Errorf("invalid genre %q", g)
	}
}

func validateFormat(f Format) error {
	switch f {
	case FormatJSON, FormatSSE, FormatMarkdown, FormatCode:
		return nil
	default:
		return fmt.Errorf("invalid format %q", f)
	}
}
