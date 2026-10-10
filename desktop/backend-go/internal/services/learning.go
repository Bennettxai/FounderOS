package services

import (
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// LearningService handles self-learning, feedback processing, and personalization
type LearningService struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewLearningService creates a new learning service
func NewLearningService(pool *pgxpool.Pool) *LearningService {
	return &LearningService{
		pool:   pool,
		logger: slog.Default().With("service", "learning"),
	}
}

// ============================================================================
// Types
// ============================================================================

// LearningEvent represents something the system learned
type LearningEvent struct {
	ID                     uuid.UUID  `json:"id"`
	UserID                 string     `json:"user_id"`
	LearningType           string     `json:"learning_type"`
	LearningContent        string     `json:"learning_content"`
	LearningSummary        string     `json:"learning_summary,omitempty"`
	SourceType             string     `json:"source_type"`
	SourceID               *uuid.UUID `json:"source_id,omitempty"`
	SourceContext          string     `json:"source_context,omitempty"`
	ConfidenceScore        float64    `json:"confidence_score"`
	TimesApplied           int        `json:"times_applied"`
	LastAppliedAt          *time.Time `json:"last_applied_at,omitempty"`
	SuccessfulApplications int        `json:"successful_applications"`
	CreatedMemoryID        *uuid.UUID `json:"created_memory_id,omitempty"`
	CreatedFactKey         string     `json:"created_fact_key,omitempty"`
	Category               string     `json:"category,omitempty"`
	Tags                   []string   `json:"tags,omitempty"`
	WasValidated           bool       `json:"was_validated"`
	ValidatedAt            *time.Time `json:"validated_at,omitempty"`
	ValidationResult       string     `json:"validation_result,omitempty"`
	IsActive               bool       `json:"is_active"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// BehaviorPattern represents an observed user behavior pattern
type BehaviorPattern struct {
	ID                           uuid.UUID `json:"id"`
	UserID                       string    `json:"user_id"`
	PatternType                  string    `json:"pattern_type"`
	PatternKey                   string    `json:"pattern_key"`
	PatternValue                 string    `json:"pattern_value"`
	PatternDescription           string    `json:"pattern_description,omitempty"`
	ObservationCount             int       `json:"observation_count"`
	FirstObservedAt              time.Time `json:"first_observed_at"`
	LastObservedAt               time.Time `json:"last_observed_at"`
	ConfidenceScore              float64   `json:"confidence_score"`
	MinObservationsForConfidence int       `json:"min_observations_for_confidence"`
	IsApplied                    bool      `json:"is_applied"`
	AppliedInPrompt              bool      `json:"applied_in_prompt"`
	IsActive                     bool      `json:"is_active"`
	CreatedAt                    time.Time `json:"created_at"`
	UpdatedAt                    time.Time `json:"updated_at"`
}

// FeedbackEntry represents user feedback on AI output
type FeedbackEntry struct {
	ID                  uuid.UUID  `json:"id"`
	UserID              string     `json:"user_id"`
	TargetType          string     `json:"target_type"`
	TargetID            uuid.UUID  `json:"target_id"`
	FeedbackType        string     `json:"feedback_type"`
	FeedbackValue       string     `json:"feedback_value,omitempty"`
	Rating              *int       `json:"rating,omitempty"`
	ConversationID      *uuid.UUID `json:"conversation_id,omitempty"`
	AgentType           string     `json:"agent_type,omitempty"`
	FocusMode           string     `json:"focus_mode,omitempty"`
	OriginalContent     string     `json:"original_content,omitempty"`
	ExpectedContent     string     `json:"expected_content,omitempty"`
	WasProcessed        bool       `json:"was_processed"`
	ProcessedAt         *time.Time `json:"processed_at,omitempty"`
	ResultingLearningID *uuid.UUID `json:"resulting_learning_id,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// PersonalizationProfile contains aggregated user preferences
type PersonalizationProfile struct {
	ID                    uuid.UUID      `json:"id"`
	UserID                string         `json:"user_id"`
	PreferredTone         string         `json:"preferred_tone"`
	PreferredVerbosity    string         `json:"preferred_verbosity"`
	PreferredFormat       string         `json:"preferred_format"`
	PrefersExamples       bool           `json:"prefers_examples"`
	PrefersAnalogies      bool           `json:"prefers_analogies"`
	PrefersCodeSamples    bool           `json:"prefers_code_samples"`
	PrefersVisualAids     bool           `json:"prefers_visual_aids"`
	ExpertiseAreas        []string       `json:"expertise_areas,omitempty"`
	LearningAreas         []string       `json:"learning_areas,omitempty"`
	CommonTopics          []string       `json:"common_topics,omitempty"`
	Timezone              string         `json:"timezone,omitempty"`
	PreferredWorkingHours map[string]any `json:"preferred_working_hours,omitempty"`
	MostActiveHours       []int          `json:"most_active_hours,omitempty"`
	TotalConversations    int            `json:"total_conversations"`
	TotalFeedbackGiven    int            `json:"total_feedback_given"`
	PositiveFeedbackRatio float64        `json:"positive_feedback_ratio"`
	ProfileCompleteness   float64        `json:"profile_completeness"`
	LastProfileUpdate     time.Time      `json:"last_profile_update"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

// FeedbackInput represents input for recording feedback
type FeedbackInput struct {
	UserID          string
	TargetType      string // 'message', 'artifact', 'memory', 'suggestion', 'agent_response'
	TargetID        uuid.UUID
	FeedbackType    string // 'thumbs_up', 'thumbs_down', 'correction', 'comment', 'rating'
	FeedbackValue   string
	Rating          *int
	ConversationID  *uuid.UUID
	AgentType       string
	FocusMode       string
	OriginalContent string
	ExpectedContent string
}
