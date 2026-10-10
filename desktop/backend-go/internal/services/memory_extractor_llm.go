package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// llmExtractedMemory represents the JSON structure from LLM response
type llmExtractedMemory struct {
	Type       string  `json:"type"`
	Summary    string  `json:"summary"`
	Content    string  `json:"content"`
	Importance int     `json:"importance"`
	Confidence float64 `json:"confidence"`
}

// ExtractWithLLM performs LLM-enhanced memory extraction
// This captures nuanced information that pattern-based extraction might miss
func (s *MemoryExtractorService) ExtractWithLLM(ctx context.Context, userID string, messages []Message, opts *ExtractionOptions) (*ExtractionResult, error) {
	startTime := time.Now()

	if s.llmService == nil {
		// Fall back to pattern-based extraction if no LLM service
		return s.ExtractFromConversation(ctx, userID, messages, opts)
	}

	if opts == nil {
		opts = DefaultExtractionOptions()
	}

	result := &ExtractionResult{
		Memories:        make([]ExtractedMemory, 0),
		ByType:          make(map[string]int),
		SourceProcessed: "conversation_llm",
	}

	// Build conversation text for LLM
	var conversationText strings.Builder
	for _, msg := range messages {
		conversationText.WriteString(fmt.Sprintf("[%s]: %s\n\n", msg.Role, msg.Content))
	}

	// Create extraction prompt
	systemPrompt := `You are a memory extraction assistant. Analyze the conversation and extract important information that should be remembered for future reference.

Extract the following types of memories:
1. **Facts**: Concrete information, names, dates, technical details, configurations
2. **Preferences**: User likes, dislikes, working style, communication preferences
3. **Decisions**: Choices made, selected approaches, agreed-upon solutions
4. **Tasks**: Action items, TODOs, things to implement or fix
5. **Insights**: Learnings, discoveries, important realizations
6. **Errors/Solutions**: Problems encountered and how they were solved

For each extracted memory, provide:
- type: one of [fact, preference, decision, task, insight, error, solution]
- summary: brief one-line summary (max 100 chars)
- content: full context (max 300 chars)
- importance: 1-10 scale (10 = critical)
- confidence: 0.0-1.0 how confident you are this should be remembered

IMPORTANT: Focus on information that would be valuable in future conversations.
Skip generic statements and focus on specific, actionable, or unique information.

Respond ONLY with a JSON array of extracted memories. Example:
[
  {"type": "fact", "summary": "Project uses PostgreSQL with pgvector", "content": "The database is PostgreSQL with pgvector extension for embeddings", "importance": 7, "confidence": 0.9},
  {"type": "decision", "summary": "Chose Svelte over React", "content": "Decided to use SvelteKit for the frontend due to better performance", "importance": 8, "confidence": 0.85}
]

If no meaningful memories can be extracted, return an empty array: []`

	chatMessages := []ChatMessage{
		{Role: "user", Content: conversationText.String()},
	}

	// Call LLM for extraction
	response, err := s.llmService.ChatComplete(ctx, chatMessages, systemPrompt)
	if err != nil {
		s.logger.Warn("LLM extraction failed, falling back to pattern-based", "error", err)
		return s.ExtractFromConversation(ctx, userID, messages, opts)
	}

	// Parse LLM response
	llmMemories := s.parseLLMResponse(response, userID)
	result.Memories = append(result.Memories, llmMemories...)

	// Also run pattern-based extraction to catch things LLM might miss
	patternResult, _ := s.ExtractFromConversation(ctx, userID, messages, opts)
	if patternResult != nil {
		// Merge pattern-based results, avoiding duplicates
		for _, pm := range patternResult.Memories {
			if !s.isDuplicate(pm, result.Memories) {
				result.Memories = append(result.Memories, pm)
			}
		}
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

	s.logger.Info("LLM-enhanced extraction completed",
		"total", result.TotalExtracted,
		"duration", result.ProcessingTime)

	return result, nil
}

// parseLLMResponse parses the LLM JSON response into ExtractedMemory structs
func (s *MemoryExtractorService) parseLLMResponse(response, userID string) []ExtractedMemory {
	memories := make([]ExtractedMemory, 0)

	// Clean response - remove markdown code blocks if present
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	// Try to find JSON array in response
	startIdx := strings.Index(response, "[")
	endIdx := strings.LastIndex(response, "]")
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		s.logger.Warn("could not find JSON array in LLM response")
		return memories
	}
	response = response[startIdx : endIdx+1]

	var llmMemories []llmExtractedMemory
	if err := json.Unmarshal([]byte(response), &llmMemories); err != nil {
		s.logger.Warn("failed to parse LLM response", "error", err)
		return memories
	}

	// Convert to ExtractedMemory format
	for _, lm := range llmMemories {
		memType := s.mapLLMType(lm.Type)
		if memType == "" {
			continue
		}

		// Validate
		if len(lm.Summary) < 5 || lm.Confidence < 0.3 {
			continue
		}

		memories = append(memories, ExtractedMemory{
			ID:          uuid.New().String(),
			UserID:      userID,
			Type:        memType,
			Content:     lm.Content,
			Summary:     lm.Summary,
			Source:      MemorySourceConversation,
			Confidence:  lm.Confidence,
			Tags:        s.extractTags(lm.Summary + " " + lm.Content),
			Entities:    s.extractEntitiesFromText(lm.Content),
			Importance:  lm.Importance,
			Metadata:    map[string]interface{}{"extraction_method": "llm"},
			ExtractedAt: time.Now(),
		})
	}

	return memories
}

// mapLLMType maps LLM type strings to MemoryType
func (s *MemoryExtractorService) mapLLMType(llmType string) MemoryType {
	switch strings.ToLower(llmType) {
	case "fact":
		return MemoryTypeFact
	case "preference":
		return MemoryTypePreference
	case "decision":
		return MemoryTypeDecision
	case "task":
		return MemoryTypeTask
	case "insight":
		return MemoryTypeInsight
	case "error":
		return MemoryTypeError
	case "solution":
		return MemoryTypeSolution
	case "reminder":
		return MemoryTypeReminder
	case "contact":
		return MemoryTypeContact
	case "code":
		return MemoryTypeCode
	default:
		return MemoryTypeFact // Default to fact
	}
}
