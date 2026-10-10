package sorx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// ============================================================================
// AI Actions (Integrated with Groq LLM API)
// ============================================================================

// callGroqLLM makes a chat completion request to the Groq API.
// Falls back gracefully if no API key is configured.
func callGroqLLM(ctx context.Context, systemPrompt, userMessage string) (string, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GROQ_API_KEY not configured")
	}

	model := os.Getenv("GROQ_MODEL")
	if model == "" {
		model = "llama-3.1-8b-instant"
	}

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"temperature": 0.3,
		"max_tokens":  2048,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Groq API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Groq API returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return result.Choices[0].Message.Content, nil
}

func aiExtractActions(ctx context.Context, ac ActionContext) (interface{}, error) {
	source, _ := ac.Params["source"].(string)
	if source == "" {
		return map[string]interface{}{"actions": []interface{}{}, "count": 0, "error": "empty source"}, nil
	}

	tier := skillTierFromContext(ac)
	slog.Info("aiExtractActions", "user_id", ac.Execution.UserID, "source_len", len(source), "tier", tier)

	// Truncate source if too long
	if len(source) > 4000 {
		source = source[:4000]
	}

	systemPrompt := `You are an action extractor. Given text, extract actionable items.
Return ONLY valid JSON in this format:
{"actions": [{"action": "description", "priority": "high|medium|low", "assignee_hint": "person or role if mentioned"}]}
If no actions found, return: {"actions": []}`

	response, err := routeAICall(ctx, tier, "extract_actions", ac.Execution.UserID, systemPrompt, source)
	if err != nil {
		slog.Warn("aiExtractActions: AI call failed, using fallback", "tier", tier, "error", err)
		// Fallback: simple keyword-based extraction
		actions := extractActionsFallback(source)
		return map[string]interface{}{
			"actions":  actions,
			"count":    len(actions),
			"fallback": true,
		}, nil
	}

	// Parse LLM response
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		// Try to extract JSON from response
		if start := strings.Index(response, "{"); start >= 0 {
			if end := strings.LastIndex(response, "}"); end > start {
				if err := json.Unmarshal([]byte(response[start:end+1]), &parsed); err != nil {
					slog.Warn("aiExtractActions: failed to parse LLM response", "error", err)
					actions := extractActionsFallback(source)
					return map[string]interface{}{"actions": actions, "count": len(actions), "fallback": true}, nil
				}
			}
		}
	}

	actions, _ := parsed["actions"].([]interface{})
	return map[string]interface{}{
		"actions": actions,
		"count":   len(actions),
	}, nil
}

// extractActionsFallback uses simple keyword matching to find action items.
func extractActionsFallback(text string) []map[string]interface{} {
	var actions []map[string]interface{}
	lines := strings.Split(text, "\n")
	actionKeywords := []string{"todo", "fix", "implement", "add", "create", "update", "remove", "deploy", "test", "review"}

	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		for _, keyword := range actionKeywords {
			if strings.Contains(lower, keyword) {
				actions = append(actions, map[string]interface{}{
					"action":   strings.TrimSpace(line),
					"priority": "medium",
				})
				break
			}
		}
	}
	if len(actions) > 10 {
		actions = actions[:10]
	}
	return actions
}

func aiSummarize(ctx context.Context, ac ActionContext) (interface{}, error) {
	text, _ := ac.Params["text"].(string)
	if text == "" {
		return map[string]interface{}{"summary": "", "error": "empty text"}, nil
	}

	tier := skillTierFromContext(ac)
	slog.Info("aiSummarize", "user_id", ac.Execution.UserID, "text_len", len(text), "tier", tier)

	// Truncate if too long
	if len(text) > 6000 {
		text = text[:6000]
	}

	systemPrompt := "You are a concise summarizer. Summarize the following text in 2-3 sentences. Return ONLY the summary, no preamble."

	response, err := routeAICall(ctx, tier, "summarize", ac.Execution.UserID, systemPrompt, text)
	if err != nil {
		slog.Warn("aiSummarize: AI call failed, using fallback", "tier", tier, "error", err)
		// Fallback: first 200 chars
		summary := text
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		return map[string]interface{}{"summary": summary, "fallback": true}, nil
	}

	return map[string]interface{}{
		"summary": strings.TrimSpace(response),
	}, nil
}

func aiClassify(ctx context.Context, ac ActionContext) (interface{}, error) {
	text, _ := ac.Params["text"].(string)
	categories, _ := ac.Params["categories"].([]interface{})

	if text == "" {
		return map[string]interface{}{"category": "unknown", "confidence": 0.0, "error": "empty text"}, nil
	}

	tier := skillTierFromContext(ac)
	slog.Info("aiClassify", "user_id", ac.Execution.UserID, "text_len", len(text), "categories", len(categories), "tier", tier)

	// Truncate if too long
	if len(text) > 4000 {
		text = text[:4000]
	}

	// Build categories string
	catStrings := make([]string, 0, len(categories))
	for _, cat := range categories {
		if s, ok := cat.(string); ok {
			catStrings = append(catStrings, s)
		}
	}
	if len(catStrings) == 0 {
		catStrings = []string{"general", "technical", "business", "personal"}
	}

	systemPrompt := fmt.Sprintf(`You are a text classifier. Classify the given text into ONE of these categories: %s
Return ONLY valid JSON: {"category": "chosen_category", "confidence": 0.0-1.0}`, strings.Join(catStrings, ", "))

	response, err := routeAICall(ctx, tier, "classify", ac.Execution.UserID, systemPrompt, text)
	if err != nil {
		slog.Warn("aiClassify: AI call failed, using fallback", "tier", tier, "error", err)
		// Fallback: return first category with low confidence
		fallbackCat := "general"
		if len(catStrings) > 0 {
			fallbackCat = catStrings[0]
		}
		return map[string]interface{}{"category": fallbackCat, "confidence": 0.3, "fallback": true}, nil
	}

	// Parse response
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		if start := strings.Index(response, "{"); start >= 0 {
			if end := strings.LastIndex(response, "}"); end > start {
				json.Unmarshal([]byte(response[start:end+1]), &parsed)
			}
		}
	}

	if parsed == nil {
		return map[string]interface{}{"category": catStrings[0], "confidence": 0.5, "raw": response}, nil
	}

	return parsed, nil
}
