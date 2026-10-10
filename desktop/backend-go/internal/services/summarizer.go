package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
)

// SummarizerService provides LLM-powered hierarchical summarization
type SummarizerService struct {
	pool   *pgxpool.Pool
	cfg    *config.Config
	logger *slog.Logger
	llm    LLMService
}

// NewSummarizerService creates a new summarizer service
func NewSummarizerService(pool *pgxpool.Pool, cfg *config.Config) *SummarizerService {
	return &SummarizerService{
		pool:   pool,
		cfg:    cfg,
		logger: slog.Default().With("service", "summarizer"),
		llm:    NewLLMService(cfg, cfg.DefaultModel),
	}
}

// SummarizeConversation creates a concise summary of a message list
func (s *SummarizerService) SummarizeConversation(ctx context.Context, messages []ChatMessage) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	// Format conversation for the LLM
	var sb strings.Builder
	for _, msg := range messages {
		sb.WriteString(fmt.Sprintf("%s: %s\n\n", strings.ToUpper(msg.Role), msg.Content))
	}

	prompt := fmt.Sprintf(`Summarize the following conversation in a concise but comprehensive way. 
Focus on:
1. Main objective or problem discussed.
2. Key decisions made.
3. Pending actions or next steps.
4. Important entities (files, projects, clients) mentioned.

Conversation:
%s

Return ONLY the summary text, no preamble.`, sb.String())

	// Use a faster model if possible for summarization
	summary, err := s.llm.ChatComplete(ctx, []ChatMessage{}, prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(summary), nil
}

// HierarchicalSummarize compresses a long conversation by keeping recent messages and summarizing older ones
func (s *SummarizerService) HierarchicalSummarize(ctx context.Context, messages []ChatMessage, recentCount int) ([]ChatMessage, string, error) {
	if len(messages) <= recentCount {
		return messages, "", nil
	}

	// Split into older and recent messages
	older := messages[:len(messages)-recentCount]
	recent := messages[len(messages)-recentCount:]

	// Summarize the older part
	summary, err := s.SummarizeConversation(ctx, older)
	if err != nil {
		return messages, "", err
	}

	// Create a new message list: [System Summary, Recent Messages]
	summaryMsg := ChatMessage{
		Role:    "system",
		Content: fmt.Sprintf("PREVIOUS CONVERSATION SUMMARY:\n%s", summary),
	}

	result := append([]ChatMessage{summaryMsg}, recent...)

	return result, summary, nil
}
