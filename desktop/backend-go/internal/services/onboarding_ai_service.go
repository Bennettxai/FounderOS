package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// OnboardingAIService handles AI-powered onboarding conversations.
// Supports Grok (x.ai), OpenAI, Anthropic, and Groq as fallback providers.
type OnboardingAIService struct {
	provider   string
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// AIProvider configuration
type AIProviderConfig struct {
	Provider string
	APIKey   string
	Model    string
	BaseURL  string
}

// OnboardingChatMessage for AI requests
type OnboardingChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OnboardingChatCompletionRequest for AI API
type OnboardingChatCompletionRequest struct {
	Model       string                   `json:"model"`
	Messages    []OnboardingChatMessage  `json:"messages"`
	MaxTokens   int                      `json:"max_tokens,omitempty"`
	Temperature float64                  `json:"temperature,omitempty"`
}

// ChatCompletionResponse from AI API
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// OnboardingAIResponse from AI processing
type OnboardingAIResponse struct {
	AgentMessage       string                 `json:"agent_message"`
	ExtractedFields    map[string]interface{} `json:"extracted_fields"`
	ConfidenceScore    float64                `json:"confidence_score"`
	NextQuestionType   string                 `json:"next_question_type"`
	IsComplete         bool                   `json:"is_complete"`
	ShouldShowFallback bool                   `json:"should_show_fallback"`
}

// Onboarding system prompt - designed to be warm, efficient, and extract data reliably
const onboardingSystemPrompt = `You are a friendly AI assistant helping new users set up their BusinessOS workspace.

GOAL: Gather these 5 pieces of information through natural conversation:
1. workspace_name - Company or workspace name
2. business_type - One of: agency, startup, freelance, ecommerce, consulting, other
3. team_size - One of: solo, 2-5, 6-15, 16-50, 50+
4. role - User's role/title
5. challenge - Main problem they want to solve

CURRENT STATE:
- Step: %s
- Collected so far: %s

GUIDELINES:
- Be warm but concise (1-2 sentences max)
- Ask ONE question at a time
- Acknowledge what they shared naturally before asking next question
- If response is unclear, ask a brief clarifying question
- Match the user's tone (casual if they're casual)

EXTRACTION RULES:
- Extract data even if phrased differently (e.g., "just me" = "solo", "small team" = "2-5")
- For business_type, map: "marketing firm" → agency, "online store" → ecommerce, "solo consultant" → consulting
- confidence_score: 1.0 = clear match, 0.7 = reasonable inference, 0.4 = uncertain
- If you can't confidently extract the expected field, set confidence_score below 0.6

RESPOND WITH ONLY THIS JSON (no markdown, no explanation):
{"agent_message":"Your response here","extracted_fields":{"field_name":"value"},"confidence_score":0.9,"next_question_type":"business_type","is_complete":false}

FIELD MAPPINGS:
- company_name → extract to workspace_name
- business_type → map to: agency|startup|freelance|ecommerce|consulting|other  
- team_size → map to: solo|2-5|6-15|16-50|50+
- role → extract as-is
- challenge → extract as-is

Set is_complete=true and next_question_type="complete" only when ALL 5 fields are gathered.`

// NewOnboardingAIService creates a new onboarding AI service
func NewOnboardingAIService() *OnboardingAIService {
	service := &OnboardingAIService{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	// Try providers in order: Grok (x.ai) -> OpenAI -> Anthropic -> Groq
	if apiKey := os.Getenv("XAI_API_KEY"); apiKey != "" {
		service.provider = "xai"
		service.apiKey = apiKey
		service.model = getEnvOrDefault("XAI_MODEL", "grok-beta")
		service.baseURL = getEnvOrDefault("XAI_BASE_URL", "https://api.x.ai/v1")
	} else if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		service.provider = "openai"
		service.apiKey = apiKey
		service.model = getEnvOrDefault("OPENAI_MODEL", "gpt-4o")
		service.baseURL = "https://api.openai.com/v1"
	} else if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		service.provider = "anthropic"
		service.apiKey = apiKey
		service.model = getEnvOrDefault("ANTHROPIC_MODEL", "claude-sonnet-4-20250514")
		service.baseURL = "https://api.anthropic.com/v1"
	} else if apiKey := os.Getenv("GROQ_API_KEY"); apiKey != "" {
		service.provider = "groq"
		service.apiKey = apiKey
		service.model = getEnvOrDefault("GROQ_MODEL", "llama-3.3-70b-versatile")
		service.baseURL = "https://api.groq.com/openai/v1"
	}

	return service
}

// NewOnboardingAIServiceWithConfig creates a service with explicit config
func NewOnboardingAIServiceWithConfig(config AIProviderConfig) *OnboardingAIService {
	return &OnboardingAIService{
		provider:   config.Provider,
		apiKey:     config.APIKey,
		model:      config.Model,
		baseURL:    config.BaseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// ProcessMessage processes a user message and returns AI response with automatic retry
func (s *OnboardingAIService) ProcessMessage(ctx context.Context, userMessage string, currentStep string, extractedData map[string]interface{}, conversationHistory []OnboardingChatMessage) (*OnboardingAIResponse, error) {
	// If no AI provider configured, use deterministic fallback
	if s.apiKey == "" {
		return s.deterministicResponse(userMessage, currentStep, extractedData)
	}

	// Build system prompt with context
	extractedJSON, _ := json.Marshal(extractedData)
	systemPrompt := fmt.Sprintf(onboardingSystemPrompt, currentStep, string(extractedJSON))

	// Build messages
	messages := []OnboardingChatMessage{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, conversationHistory...)
	messages = append(messages, OnboardingChatMessage{Role: "user", Content: userMessage})

	// Call AI API with retry logic
	return s.callWithRetry(ctx, messages)
}

// callWithRetry wraps AI API calls with exponential backoff retry
func (s *OnboardingAIService) callWithRetry(ctx context.Context, messages []OnboardingChatMessage) (*OnboardingAIResponse, error) {
	maxRetries := 3
	baseDelay := 500 * time.Millisecond
	
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Check context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		
		// Call the appropriate API
		var response *OnboardingAIResponse
		var err error
		if s.provider == "anthropic" {
			response, err = s.callAnthropicAPI(ctx, messages)
		} else {
			response, err = s.callOpenAICompatibleAPI(ctx, messages)
		}
		
		if err == nil {
			return response, nil
		}
		
		lastErr = err
		
		// Don't retry on context errors or if it's the last attempt
		if ctx.Err() != nil || attempt == maxRetries-1 {
			break
		}
		
		// Exponential backoff: 500ms, 1s, 2s
		delay := baseDelay * time.Duration(1<<attempt)
		select {
		case <-time.After(delay):
			// Continue to next retry
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	return nil, fmt.Errorf("AI call failed after %d retries: %w", maxRetries, lastErr)
}

// callOpenAICompatibleAPI calls OpenAI-compatible APIs (OpenAI, Grok, Groq)
func (s *OnboardingAIService) callOpenAICompatibleAPI(ctx context.Context, messages []OnboardingChatMessage) (*OnboardingAIResponse, error) {
	reqBody := OnboardingChatCompletionRequest{
		Model:       s.model,
		Messages:    messages,
		MaxTokens:   500,
		Temperature: 0.7,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	// Parse the AI response as JSON
	content := chatResp.Choices[0].Message.Content
	return s.parseAIResponse(content)
}

// callAnthropicAPI calls Anthropic's Claude API
func (s *OnboardingAIService) callAnthropicAPI(ctx context.Context, messages []OnboardingChatMessage) (*OnboardingAIResponse, error) {
	// Extract system message
	var systemPrompt string
	var conversationMessages []OnboardingChatMessage
	for _, msg := range messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else {
			conversationMessages = append(conversationMessages, msg)
		}
	}

	reqBody := map[string]interface{}{
		"model":      s.model,
		"max_tokens": 500,
		"system":     systemPrompt,
		"messages":   conversationMessages,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(body))
	}

	var anthropicResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	return s.parseAIResponse(anthropicResp.Content[0].Text)
}

// parseAIResponse parses the AI response JSON
func (s *OnboardingAIService) parseAIResponse(content string) (*OnboardingAIResponse, error) {
	// Try to extract JSON from the response
	content = strings.TrimSpace(content)
	
	// Handle markdown code blocks
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	} else if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}

	// First try to parse the entire content as JSON
	var response OnboardingAIResponse
	if err := json.Unmarshal([]byte(content), &response); err == nil {
		return &response, nil
	}

	// If that fails, try to extract JSON from anywhere in the response
	// Look for JSON object pattern: starts with { and ends with }
	jsonStart := strings.Index(content, "{")
	jsonEnd := strings.LastIndex(content, "}")
	
	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		jsonContent := content[jsonStart : jsonEnd+1]
		if err := json.Unmarshal([]byte(jsonContent), &response); err == nil {
			return &response, nil
		}
	}

	// If parsing still fails, use the raw content as the message
	return &OnboardingAIResponse{
		AgentMessage:    content,
		ConfidenceScore: 0.5,
	}, nil
}

// deterministicResponse provides a fallback when no AI is configured
func (s *OnboardingAIService) deterministicResponse(userMessage string, currentStep string, extractedData map[string]interface{}) (*OnboardingAIResponse, error) {
	response := &OnboardingAIResponse{
		ExtractedFields: make(map[string]interface{}),
		ConfidenceScore: 1.0,
	}

	switch currentStep {
	case "company_name":
		response.ExtractedFields["workspace_name"] = userMessage
		response.AgentMessage = "What kind of work do you do?"
		response.NextQuestionType = "business_type"

	case "business_type":
		businessType := strings.ToLower(userMessage)
		// Map common responses to categories
		switch {
		case strings.Contains(businessType, "agency"):
			businessType = "agency"
		case strings.Contains(businessType, "startup"):
			businessType = "startup"
		case strings.Contains(businessType, "freelance") || strings.Contains(businessType, "solo"):
			businessType = "freelance"
		case strings.Contains(businessType, "ecommerce") || strings.Contains(businessType, "e-commerce") || strings.Contains(businessType, "shop"):
			businessType = "ecommerce"
		case strings.Contains(businessType, "consult"):
			businessType = "consulting"
		default:
			businessType = "other"
		}
		response.ExtractedFields["business_type"] = businessType
		
		if businessType == "freelance" {
			response.ExtractedFields["team_size"] = "solo"
			response.AgentMessage = "What's your role?"
			response.NextQuestionType = "role"
		} else {
			response.AgentMessage = "How big is your team?"
			response.NextQuestionType = "team_size"
		}

	case "team_size":
		teamSize := strings.ToLower(userMessage)
		switch {
		case strings.Contains(teamSize, "just me") || strings.Contains(teamSize, "solo") || teamSize == "1":
			teamSize = "solo"
		case strings.Contains(teamSize, "2") || strings.Contains(teamSize, "3") || strings.Contains(teamSize, "4") || strings.Contains(teamSize, "5"):
			teamSize = "2-5"
		case strings.Contains(teamSize, "6") || strings.Contains(teamSize, "10") || strings.Contains(teamSize, "15"):
			teamSize = "6-15"
		case strings.Contains(teamSize, "16") || strings.Contains(teamSize, "20") || strings.Contains(teamSize, "50"):
			teamSize = "16-50"
		case strings.Contains(teamSize, "50+") || strings.Contains(teamSize, "100"):
			teamSize = "50+"
		default:
			teamSize = userMessage
		}
		response.ExtractedFields["team_size"] = teamSize
		response.AgentMessage = "What's your role?"
		response.NextQuestionType = "role"

	case "role":
		response.ExtractedFields["role"] = userMessage
		response.AgentMessage = "What's the biggest challenge you're hoping to solve?"
		response.NextQuestionType = "challenge"

	case "challenge":
		response.ExtractedFields["challenge"] = userMessage
		response.AgentMessage = "Perfect! Let's connect your favorite tools."
		response.NextQuestionType = "integrations"

	case "integrations":
		response.AgentMessage = "Great! You're all set up."
		response.NextQuestionType = "complete"
		response.IsComplete = true
	}

	return response, nil
}

// IsConfigured returns whether an AI provider is configured
func (s *OnboardingAIService) IsConfigured() bool {
	return s.apiKey != ""
}

// GetProvider returns the configured provider name
func (s *OnboardingAIService) GetProvider() string {
	if s.provider == "" {
		return "deterministic"
	}
	return s.provider
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
