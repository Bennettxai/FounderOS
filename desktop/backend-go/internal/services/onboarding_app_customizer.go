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

// AppCustomizerAgent recommends and customizes starter apps based on user profile
// Uses Groq LLM to generate personalized app recommendations
type AppCustomizerAgent struct {
	llmService LLMService
	cfg        *config.Config
}

// StarterAppRecommendation represents a recommended app
type StarterAppRecommendation struct {
	Title                string                 `json:"title"`
	Description          string                 `json:"description"`
	IconEmoji            string                 `json:"icon_emoji"`
	Category             string                 `json:"category"`
	Reasoning            string                 `json:"reasoning"`
	CustomizationPrompt  string                 `json:"customization_prompt"`
	BasedOnInterests     []string               `json:"based_on_interests"`
	BasedOnTools         []string               `json:"based_on_tools"`
	BaseModule           string                 `json:"base_module,omitempty"` // Optional: CRM, Tasks, Projects, etc.
	ModuleCustomizations map[string]interface{} `json:"module_customizations,omitempty"`
	Priority             int                    `json:"priority"` // 1-4 for display order
}

// AppRecommendationsResult contains all recommended apps
type AppRecommendationsResult struct {
	Apps       []StarterAppRecommendation `json:"apps"`
	TotalApps  int                        `json:"total_apps"`
	Confidence float64                    `json:"confidence"`
}

// CoreModule represents available BusinessOS core modules
type CoreModule struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

var coreModules = []CoreModule{
	{Name: "CRM", Description: "Client relationship management", Icon: "👥"},
	{Name: "Tasks", Description: "Task and todo management", Icon: "✅"},
	{Name: "Projects", Description: "Project tracking and management", Icon: "📊"},
	{Name: "Calendar", Description: "Calendar and scheduling", Icon: "📅"},
	{Name: "Notes", Description: "Note-taking and documentation", Icon: "📝"},
	{Name: "Dashboard", Description: "Overview and analytics", Icon: "📈"},
	{Name: "Team", Description: "Team collaboration", Icon: "👨‍👩‍👧‍👦"},
	{Name: "Knowledge", Description: "Knowledge base and docs", Icon: "📚"},
}

func NewAppCustomizerAgent(cfg *config.Config) *AppCustomizerAgent {
	return &AppCustomizerAgent{
		llmService: NewGroqService(cfg, "llama-3.3-70b-versatile"),
		cfg:        cfg,
	}
}

// RecommendApps recommends 3-4 personalized starter apps
func (a *AppCustomizerAgent) RecommendApps(ctx context.Context, profile *ProfileAnalysisResult) (*AppRecommendationsResult, error) {
	startTime := time.Now()
	slog.Info("AppCustomizerAgent recommending apps",
		"interests_count", len(profile.Interests),
		"tools_count", len(profile.ToolsUsed),
	)

	prompt := a.buildRecommendationPrompt(profile)

	messages := []ChatMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	systemPrompt := `You are an app recommendation specialist for BusinessOS, a personalized business operating system.

Based on the user's profile, recommend 3-4 starter apps that will be most valuable to them.

For each app, you can either:
1. Customize an existing core module (CRM, Tasks, Projects, Calendar, Notes, Dashboard, Team, Knowledge)
2. Create a completely new app concept tailored to their needs

Guidelines:
- Be specific and personal - use their actual interests and tools
- Keep app titles short and memorable (2-4 words)
- Use relevant emojis for icons
- Explain WHY this app helps them specifically
- Category: tracker, companion, feedback, daily, workflow, automation, etc.
- Priority 1 = most important

Available core modules: CRM, Tasks, Projects, Calendar, Notes, Dashboard, Team, Knowledge

Respond ONLY with valid JSON matching this schema:
{
  "apps": [
    {
      "title": "App Name",
      "description": "What it does",
      "icon_emoji": "📚",
      "category": "tracker",
      "reasoning": "Why this helps the user",
      "customization_prompt": "Full prompt for AI to build this app",
      "based_on_interests": ["interest1", "interest2"],
      "based_on_tools": ["tool1"],
      "base_module": "Tasks",
      "module_customizations": {"theme": "design-focused"},
      "priority": 1
    }
  ],
  "confidence": 0.9
}

Do NOT include explanatory text. ONLY output valid JSON.`

	response, usage, err := a.llmService.ChatCompleteWithUsage(ctx, messages, systemPrompt)
	if err != nil {
		slog.Error("AppCustomizerAgent LLM call failed", "error", err)
		return nil, fmt.Errorf("LLM recommendation failed: %w", err)
	}

	// Parse JSON response
	result, err := a.parseRecommendationResponse(response)
	if err != nil {
		slog.Error("AppCustomizerAgent failed to parse response", "error", err, "response", response)
		return nil, fmt.Errorf("failed to parse recommendations: %w", err)
	}

	duration := time.Since(startTime)
	slog.Info("AppCustomizerAgent recommendations complete",
		"duration_ms", duration.Milliseconds(),
		"apps_count", result.TotalApps,
		"tokens_used", usage.TotalTokens,
		"model", usage.Model,
	)

	return result, nil
}

// buildRecommendationPrompt constructs the prompt from user profile
func (a *AppCustomizerAgent) buildRecommendationPrompt(profile *ProfileAnalysisResult) string {
	var prompt strings.Builder

	prompt.WriteString("Recommend 3-4 personalized starter apps for this user:\n\n")

	prompt.WriteString("**User Profile:**\n")
	prompt.WriteString(fmt.Sprintf("- Summary: %s\n", profile.ProfileSummary))
	prompt.WriteString("\n")

	if len(profile.Insights) > 0 {
		prompt.WriteString("**Their Vibe:**\n")
		for _, insight := range profile.Insights {
			prompt.WriteString(fmt.Sprintf("- %s\n", insight))
		}
		prompt.WriteString("\n")
	}

	if len(profile.Interests) > 0 {
		prompt.WriteString("**Interests:**\n")
		prompt.WriteString(strings.Join(profile.Interests, ", "))
		prompt.WriteString("\n\n")
	}

	if len(profile.ToolsUsed) > 0 {
		prompt.WriteString("**Tools They Use:**\n")
		prompt.WriteString(strings.Join(profile.ToolsUsed, ", "))
		prompt.WriteString("\n\n")
	}

	prompt.WriteString("Recommend apps that feel personal and valuable to THIS specific user.")

	return prompt.String()
}

// parseRecommendationResponse parses the LLM JSON response
func (a *AppCustomizerAgent) parseRecommendationResponse(response string) (*AppRecommendationsResult, error) {
	// Clean response
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var rawResult struct {
		Apps       []StarterAppRecommendation `json:"apps"`
		Confidence float64                    `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(response), &rawResult); err != nil {
		return nil, fmt.Errorf("JSON unmarshal error: %w", err)
	}

	// Validate
	if len(rawResult.Apps) == 0 {
		return nil, fmt.Errorf("no apps recommended")
	}

	// Ensure 3-4 apps (truncate if more)
	if len(rawResult.Apps) > 4 {
		rawResult.Apps = rawResult.Apps[:4]
	}

	// Set priorities if missing
	for i := range rawResult.Apps {
		if rawResult.Apps[i].Priority == 0 {
			rawResult.Apps[i].Priority = i + 1
		}
	}

	result := &AppRecommendationsResult{
		Apps:       rawResult.Apps,
		TotalApps:  len(rawResult.Apps),
		Confidence: rawResult.Confidence,
	}

	return result, nil
}
