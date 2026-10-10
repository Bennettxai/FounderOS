package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/config"
)

// ProfileAnalyzerAgent analyzes user email metadata to build a profile
// Uses Groq LLM to extract patterns, interests, and tools used
type ProfileAnalyzerAgent struct {
	llmService LLMService
	cfg        *config.Config
}

// EmailMetadataInput represents aggregated email data for analysis
type EmailMetadataInput struct {
	TotalEmails     int                    `json:"total_emails"`
	SenderDomains   map[string]int         `json:"sender_domains"` // domain -> count
	SubjectKeywords []string               `json:"subject_keywords"`
	BodyKeywords    []string               `json:"body_keywords"`
	DetectedTools   map[string]int         `json:"detected_tools"`  // tool -> count
	TopicFrequency  map[string]int         `json:"topic_frequency"` // topic -> count
	EmailDates      []time.Time            `json:"email_dates"`     // For activity pattern detection
	RawMetadata     map[string]interface{} `json:"raw_metadata,omitempty"`
}

// ProfileAnalysisResult represents the analyzed user profile
type ProfileAnalysisResult struct {
	Insights                []string               `json:"insights"`                  // 3 conversational phrases
	Interests               []string               `json:"interests"`                 // Detected interests
	ToolsUsed               []string               `json:"tools_used"`                // Top tools
	ProfileSummary          string                 `json:"profile_summary"`           // Full narrative summary
	WorkPatterns            map[string]interface{} `json:"work_patterns"`             // Activity patterns
	Confidence              float64                `json:"confidence"`                // 0.0 to 1.0
	BusinessType            string                 `json:"business_type"`             // Business type
	TeamSize                string                 `json:"team_size"`                 // Team size
	OwnerRole               string                 `json:"owner_role"`                // Owner role
	MainChallenge           string                 `json:"main_challenge"`            // Main challenge
	RecommendedIntegrations []string               `json:"recommended_integrations"`  // Recommended integrations
}

func NewProfileAnalyzerAgent(cfg *config.Config) *ProfileAnalyzerAgent {
	return &ProfileAnalyzerAgent{
		llmService: NewGroqService(cfg, "llama-3.3-70b-versatile"), // Use Llama 3.3 70B for better reasoning
		cfg:        cfg,
	}
}

// AnalyzeProfile analyzes email metadata and generates user profile
func (a *ProfileAnalyzerAgent) AnalyzeProfile(ctx context.Context, metadata *EmailMetadataInput) (*ProfileAnalysisResult, error) {
	startTime := time.Now()
	slog.Info("ProfileAnalyzerAgent starting analysis",
		"total_emails", metadata.TotalEmails,
		"unique_domains", len(metadata.SenderDomains),
		"detected_tools", len(metadata.DetectedTools),
	)

	// Build analysis prompt with structured input
	prompt := a.buildAnalysisPrompt(metadata)

	// Call Groq LLM for analysis
	messages := []ChatMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	systemPrompt := `You are a user profiling expert analyzing email patterns to understand someone's work style, interests, and tool usage.

Your task is to:
1. Generate 3 short, conversational insight phrases (like "No-code builder energy ✨", "Design tools are your playground", "AI curious, testing new platforms")
2. Identify core interests (max 5)
3. List top tools/platforms used (max 5)
4. Write a concise profile summary (2-3 sentences)

Be specific, personal, and encouraging. Use natural language that feels like a friend describing them.

IMPORTANT: Respond ONLY with valid JSON matching this exact schema:
{
  "insights": ["phrase 1", "phrase 2", "phrase 3"],
  "interests": ["interest1", "interest2", ...],
  "tools_used": ["tool1", "tool2", ...],
  "profile_summary": "A concise 2-3 sentence summary...",
  "work_patterns": {"key": "value"},
  "confidence": 0.85
}

Do NOT include any explanatory text before or after the JSON. ONLY output valid JSON.`

	response, usage, err := a.llmService.ChatCompleteWithUsage(ctx, messages, systemPrompt)
	if err != nil {
		slog.Error("ProfileAnalyzerAgent LLM call failed", "error", err)
		return nil, fmt.Errorf("LLM analysis failed: %w", err)
	}

	// Parse JSON response
	result, err := a.parseAnalysisResponse(response)
	if err != nil {
		slog.Error("ProfileAnalyzerAgent failed to parse response", "error", err, "response", response)
		return nil, fmt.Errorf("failed to parse analysis: %w", err)
	}

	duration := time.Since(startTime)
	slog.Info("ProfileAnalyzerAgent analysis complete",
		"duration_ms", duration.Milliseconds(),
		"insights_count", len(result.Insights),
		"interests_count", len(result.Interests),
		"tools_count", len(result.ToolsUsed),
		"tokens_used", usage.TotalTokens,
		"model", usage.Model,
	)

	return result, nil
}

// buildAnalysisPrompt constructs the prompt from metadata
func (a *ProfileAnalyzerAgent) buildAnalysisPrompt(metadata *EmailMetadataInput) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("Analyze this user's email activity from %d recent emails:\n\n", metadata.TotalEmails))

	// Sender domains
	if len(metadata.SenderDomains) > 0 {
		prompt.WriteString("**Top Email Senders:**\n")
		for domain, count := range metadata.SenderDomains {
			prompt.WriteString(fmt.Sprintf("- %s: %d emails\n", domain, count))
		}
		prompt.WriteString("\n")
	}

	// Detected tools
	if len(metadata.DetectedTools) > 0 {
		prompt.WriteString("**Tools/Platforms Mentioned:**\n")
		for tool, count := range metadata.DetectedTools {
			prompt.WriteString(fmt.Sprintf("- %s: mentioned %d times\n", tool, count))
		}
		prompt.WriteString("\n")
	}

	// Topics
	if len(metadata.TopicFrequency) > 0 {
		prompt.WriteString("**Discussion Topics:**\n")
		for topic, count := range metadata.TopicFrequency {
			prompt.WriteString(fmt.Sprintf("- %s: %d occurrences\n", topic, count))
		}
		prompt.WriteString("\n")
	}

	// Keywords (sample)
	if len(metadata.SubjectKeywords) > 0 {
		prompt.WriteString("**Subject Keywords (sample):**\n")
		sampleSize := minInt(10, len(metadata.SubjectKeywords))
		prompt.WriteString(strings.Join(metadata.SubjectKeywords[:sampleSize], ", "))
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Based on this data, create a personalized profile analysis.")

	return prompt.String()
}

// parseAnalysisResponse parses the LLM JSON response
func (a *ProfileAnalyzerAgent) parseAnalysisResponse(response string) (*ProfileAnalysisResult, error) {
	// Clean response (remove markdown code blocks if present)
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var result ProfileAnalysisResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("JSON unmarshal error: %w", err)
	}

	// Validate required fields
	if len(result.Insights) == 0 {
		return nil, fmt.Errorf("no insights generated")
	}
	if len(result.Interests) == 0 {
		return nil, fmt.Errorf("no interests detected")
	}
	if result.ProfileSummary == "" {
		return nil, fmt.Errorf("no profile summary generated")
	}

	// Ensure exactly 3 insights (truncate or pad)
	if len(result.Insights) > 3 {
		result.Insights = result.Insights[:3]
	} else if len(result.Insights) < 3 {
		// Pad with generic insights if needed
		for len(result.Insights) < 3 {
			result.Insights = append(result.Insights, "Getting-things-done energy 💪")
		}
	}

	return &result, nil
}
