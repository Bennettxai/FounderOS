package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// MemoryExtractorService automatically extracts memories from conversations and content
type MemoryExtractorService struct {
	pool       *pgxpool.Pool
	logger     *slog.Logger
	llmService LLMService // Optional LLM service for enhanced extraction
}

// ExtractedMemory represents a memory extracted from content
type ExtractedMemory struct {
	ID          string                 `json:"id"`
	UserID      string                 `json:"user_id"`
	Type        MemoryType             `json:"type"`
	Content     string                 `json:"content"`
	Summary     string                 `json:"summary"`
	Source      MemorySource           `json:"source"`
	SourceID    string                 `json:"source_id,omitempty"`
	Confidence  float64                `json:"confidence"`
	Tags        []string               `json:"tags"`
	Entities    []string               `json:"entities"`
	RelatedTo   []string               `json:"related_to"`
	Context     string                 `json:"context,omitempty"`
	Importance  int                    `json:"importance"` // 1-10
	Metadata    map[string]interface{} `json:"metadata"`
	ExtractedAt time.Time              `json:"extracted_at"`
}

// MemoryType represents the type of extracted memory
type MemoryType string

const (
	MemoryTypeFact       MemoryType = "fact"
	MemoryTypePreference MemoryType = "preference"
	MemoryTypeDecision   MemoryType = "decision"
	MemoryTypeTask       MemoryType = "task"
	MemoryTypeReminder   MemoryType = "reminder"
	MemoryTypeInsight    MemoryType = "insight"
	MemoryTypeContact    MemoryType = "contact"
	MemoryTypeEvent      MemoryType = "event"
	MemoryTypeNote       MemoryType = "note"
	MemoryTypeCode       MemoryType = "code"
	MemoryTypeError      MemoryType = "error"
	MemoryTypeSolution   MemoryType = "solution"
)

// MemorySource represents the source of the memory
type MemorySource string

const (
	MemorySourceConversation MemorySource = "conversation"
	MemorySourceVoiceNote    MemorySource = "voice_note"
	MemorySourceDocument     MemorySource = "document"
	MemorySourceCode         MemorySource = "code"
	MemorySourceManual       MemorySource = "manual"
	MemorySourceImport       MemorySource = "import"
)

// ExtractionResult contains the results of memory extraction
type ExtractionResult struct {
	Memories        []ExtractedMemory `json:"memories"`
	TotalExtracted  int               `json:"total_extracted"`
	ByType          map[string]int    `json:"by_type"`
	ProcessingTime  string            `json:"processing_time"`
	SourceProcessed string            `json:"source_processed"`
}

// ExtractionOptions configures memory extraction behavior
type ExtractionOptions struct {
	ExtractFacts       bool    `json:"extract_facts"`
	ExtractPreferences bool    `json:"extract_preferences"`
	ExtractDecisions   bool    `json:"extract_decisions"`
	ExtractTasks       bool    `json:"extract_tasks"`
	ExtractInsights    bool    `json:"extract_insights"`
	ExtractContacts    bool    `json:"extract_contacts"`
	ExtractCode        bool    `json:"extract_code"`
	MinConfidence      float64 `json:"min_confidence"`
	MaxMemories        int     `json:"max_memories"`
}

// DefaultExtractionOptions returns default extraction options
func DefaultExtractionOptions() *ExtractionOptions {
	return &ExtractionOptions{
		ExtractFacts:       true,
		ExtractPreferences: true,
		ExtractDecisions:   true,
		ExtractTasks:       true,
		ExtractInsights:    true,
		ExtractContacts:    true,
		ExtractCode:        true,
		MinConfidence:      0.5,
		MaxMemories:        50,
	}
}

// NewMemoryExtractorService creates a new memory extractor service
func NewMemoryExtractorService(pool *pgxpool.Pool, embeddingService *EmbeddingService) *MemoryExtractorService {
	return &MemoryExtractorService{
		pool:   pool,
		logger: slog.Default(),
	}
}

// SetLLMService sets the LLM service for enhanced extraction
func (s *MemoryExtractorService) SetLLMService(llm LLMService) {
	s.llmService = llm
}

// ExtractFromConversation extracts memories from a conversation
func (s *MemoryExtractorService) ExtractFromConversation(ctx context.Context, userID string, messages []Message, opts *ExtractionOptions) (*ExtractionResult, error) {
	startTime := time.Now()

	if opts == nil {
		opts = DefaultExtractionOptions()
	}

	result := &ExtractionResult{
		Memories:        make([]ExtractedMemory, 0),
		ByType:          make(map[string]int),
		SourceProcessed: "conversation",
	}

	// Combine all message content for analysis
	var fullContent strings.Builder
	for _, msg := range messages {
		fullContent.WriteString(fmt.Sprintf("[%s]: %s\n\n", msg.Role, msg.Content))
	}
	content := fullContent.String()

	// Extract different types of memories
	if opts.ExtractFacts {
		facts := s.extractFacts(userID, content, messages)
		result.Memories = append(result.Memories, facts...)
	}

	if opts.ExtractPreferences {
		prefs := s.extractPreferences(userID, content, messages)
		result.Memories = append(result.Memories, prefs...)
	}

	if opts.ExtractDecisions {
		decisions := s.extractDecisionsFromContent(userID, content, messages)
		result.Memories = append(result.Memories, decisions...)
	}

	if opts.ExtractTasks {
		tasks := s.extractTasks(userID, content, messages)
		result.Memories = append(result.Memories, tasks...)
	}

	if opts.ExtractInsights {
		insights := s.extractInsights(userID, content, messages)
		result.Memories = append(result.Memories, insights...)
	}

	if opts.ExtractContacts {
		contacts := s.extractContacts(userID, content)
		result.Memories = append(result.Memories, contacts...)
	}

	if opts.ExtractCode {
		codeMemories := s.extractCodeMemories(userID, content, messages)
		result.Memories = append(result.Memories, codeMemories...)
	}

	// Filter by confidence
	filtered := make([]ExtractedMemory, 0)
	for _, m := range result.Memories {
		if m.Confidence >= opts.MinConfidence {
			filtered = append(filtered, m)
		}
	}
	result.Memories = filtered

	// Limit results
	if opts.MaxMemories > 0 && len(result.Memories) > opts.MaxMemories {
		result.Memories = result.Memories[:opts.MaxMemories]
	}

	// Calculate stats
	result.TotalExtracted = len(result.Memories)
	for _, m := range result.Memories {
		result.ByType[string(m.Type)]++
	}
	result.ProcessingTime = time.Since(startTime).String()

	// Save extracted memories
	for _, memory := range result.Memories {
		if err := s.saveMemory(ctx, &memory); err != nil {
			s.logger.Warn("failed to save extracted memory", "error", err)
		}
	}

	return result, nil
}

// ExtractFromVoiceNote extracts memories from transcribed voice note
func (s *MemoryExtractorService) ExtractFromVoiceNote(ctx context.Context, userID, transcript string, opts *ExtractionOptions) (*ExtractionResult, error) {
	startTime := time.Now()

	if opts == nil {
		opts = DefaultExtractionOptions()
	}

	result := &ExtractionResult{
		Memories:        make([]ExtractedMemory, 0),
		ByType:          make(map[string]int),
		SourceProcessed: "voice_note",
	}

	// Convert transcript to pseudo-messages for processing
	messages := []Message{{
		Role:      "user",
		Content:   transcript,
		Timestamp: time.Now(),
	}}

	// Extract memories (similar to conversation but with voice-specific patterns)
	if opts.ExtractFacts {
		facts := s.extractFacts(userID, transcript, messages)
		for i := range facts {
			facts[i].Source = MemorySourceVoiceNote
		}
		result.Memories = append(result.Memories, facts...)
	}

	if opts.ExtractTasks {
		tasks := s.extractTasksFromVoice(userID, transcript)
		result.Memories = append(result.Memories, tasks...)
	}

	if opts.ExtractInsights {
		insights := s.extractInsights(userID, transcript, messages)
		for i := range insights {
			insights[i].Source = MemorySourceVoiceNote
		}
		result.Memories = append(result.Memories, insights...)
	}

	// Extract reminders (voice notes often contain reminders)
	reminders := s.extractReminders(userID, transcript)
	result.Memories = append(result.Memories, reminders...)

	// Filter and limit
	filtered := make([]ExtractedMemory, 0)
	for _, m := range result.Memories {
		if m.Confidence >= opts.MinConfidence {
			filtered = append(filtered, m)
		}
	}
	result.Memories = filtered

	if opts.MaxMemories > 0 && len(result.Memories) > opts.MaxMemories {
		result.Memories = result.Memories[:opts.MaxMemories]
	}

	// Calculate stats
	result.TotalExtracted = len(result.Memories)
	for _, m := range result.Memories {
		result.ByType[string(m.Type)]++
	}
	result.ProcessingTime = time.Since(startTime).String()

	// Save memories
	for _, memory := range result.Memories {
		if err := s.saveMemory(ctx, &memory); err != nil {
			s.logger.Warn("failed to save extracted memory", "error", err)
		}
	}

	return result, nil
}
