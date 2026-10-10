package services

import "strings"

const (
	LLMProviderOllamaCloud = "ollama_cloud"
	LLMProviderOllamaLocal = "ollama_local"
	LLMProviderGroq        = "groq"
	LLMProviderAnthropic   = "anthropic"
)

type LLMProviderDefinition struct {
	ID           string
	Name         string
	Type         string
	Description  string
	DefaultModel string
}

type LLMModelDefinition struct {
	ID          string
	Name        string
	Provider    string
	Description string
	Family      string
}

var llmProviderDefinitions = []LLMProviderDefinition{
	{
		ID:           LLMProviderOllamaCloud,
		Name:         "Ollama Cloud",
		Type:         "cloud",
		Description:  "Run Llama and other models via Ollama's cloud API",
		DefaultModel: "llama3.2",
	},
	{
		ID:           LLMProviderOllamaLocal,
		Name:         "Ollama (Local)",
		Type:         "local",
		Description:  "Run open-source models locally on your machine",
		DefaultModel: "llama3.2:latest",
	},
	{
		ID:           LLMProviderGroq,
		Name:         "Groq",
		Type:         "cloud",
		Description:  "Ultra-fast inference with Groq's LPU hardware",
		DefaultModel: "llama-3.3-70b-versatile",
	},
	{
		ID:           LLMProviderAnthropic,
		Name:         "Anthropic Claude",
		Type:         "cloud",
		Description:  "Claude AI models from Anthropic",
		DefaultModel: "claude-sonnet-4-5-20250929",
	},
}

var llmModelDefinitions = []LLMModelDefinition{
	{ID: "kimi-k2.6:cloud", Name: "Kimi K2.6", Provider: LLMProviderOllamaCloud, Description: "256K context multimodal agentic model for coding, tools, and long-horizon work", Family: "kimi"},
	{ID: "qwen3-coder:480b-cloud", Name: "Qwen3 Coder 480B", Provider: LLMProviderOllamaCloud, Description: "480B cloud model - best quality for coding", Family: "qwen"},
	{ID: "qwen3-coder:30b", Name: "Qwen3 Coder 30B", Provider: LLMProviderOllamaCloud, Description: "30B coding model", Family: "qwen"},
	{ID: "qwen3:4b", Name: "Qwen 3 4B", Provider: LLMProviderOllamaCloud, Description: "Fast, efficient 4B model", Family: "qwen"},
	{ID: "qwen3:8b", Name: "Qwen 3 8B", Provider: LLMProviderOllamaCloud, Description: "Balanced 8B model", Family: "qwen"},
	{ID: "qwen3:14b", Name: "Qwen 3 14B", Provider: LLMProviderOllamaCloud, Description: "Capable 14B model", Family: "qwen"},
	{ID: "qwen3:30b", Name: "Qwen 3 30B", Provider: LLMProviderOllamaCloud, Description: "Large 30B model", Family: "qwen"},
	{ID: "qwen3:32b", Name: "Qwen 3 32B", Provider: LLMProviderOllamaCloud, Description: "Large 32B model", Family: "qwen"},
	{ID: "llama3.3:70b", Name: "Llama 3.3 70B", Provider: LLMProviderOllamaCloud, Description: "Latest Llama model from Meta", Family: "llama"},
	{ID: "llama3.2", Name: "Llama 3.2", Provider: LLMProviderOllamaCloud, Description: "Fast Llama model", Family: "llama"},
	{ID: "deepseek-r1:671b", Name: "DeepSeek R1 671B", Provider: LLMProviderOllamaCloud, Description: "Full reasoning model - cloud", Family: "deepseek"},
	{ID: "deepseek-r1:70b", Name: "DeepSeek R1 70B", Provider: LLMProviderOllamaCloud, Description: "Reasoning model", Family: "deepseek"},
	{ID: "deepseek-r1:32b", Name: "DeepSeek R1 32B", Provider: LLMProviderOllamaCloud, Description: "Compact reasoning model", Family: "deepseek"},
	{ID: "mistral", Name: "Mistral", Provider: LLMProviderOllamaCloud, Description: "Mistral AI's flagship model", Family: "mistral"},

	{ID: "llama-3.3-70b-versatile", Name: "Llama 3.3 70B Versatile", Provider: LLMProviderGroq, Description: "Fast 70B model for general tasks", Family: "llama"},
	{ID: "llama-3.1-8b-instant", Name: "Llama 3.1 8B Instant", Provider: LLMProviderGroq, Description: "Ultra-fast 8B model", Family: "llama"},
	{ID: "mixtral-8x7b-32768", Name: "Mixtral 8x7B", Provider: LLMProviderGroq, Description: "Mixtral MoE model with 32k context", Family: "mixtral"},
	{ID: "gemma2-9b-it", Name: "Gemma 2 9B", Provider: LLMProviderGroq, Description: "Google's Gemma 2 model", Family: "gemma"},

	{ID: "claude-sonnet-4-5-20250929", Name: "Claude Sonnet 4.5", Provider: LLMProviderAnthropic, Description: "Balanced default for everyday work", Family: "claude-4"},
	{ID: "claude-opus-4-1-20250805", Name: "Claude Opus 4.1", Provider: LLMProviderAnthropic, Description: "Most capable model for complex reasoning", Family: "claude-4"},
	{ID: "claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5", Provider: LLMProviderAnthropic, Description: "Fast, low-cost model for light tasks", Family: "claude-4"},
}

func LLMProviderDefinitions() []LLMProviderDefinition {
	return append([]LLMProviderDefinition(nil), llmProviderDefinitions...)
}

func CloudLLMProviderIDs() []string {
	ids := make([]string, 0, len(llmProviderDefinitions))
	for _, provider := range llmProviderDefinitions {
		if provider.Type == "cloud" {
			ids = append(ids, provider.ID)
		}
	}
	return ids
}

func IsKnownLLMProvider(providerID string) bool {
	_, ok := LLMProviderDefinitionByID(providerID)
	return ok
}

func IsCloudLLMProvider(providerID string) bool {
	def, ok := LLMProviderDefinitionByID(providerID)
	return ok && def.Type == "cloud"
}

func LLMProviderDefinitionByID(providerID string) (LLMProviderDefinition, bool) {
	for _, provider := range llmProviderDefinitions {
		if provider.ID == providerID {
			return provider, true
		}
	}
	return LLMProviderDefinition{}, false
}

func DefaultModelForProvider(providerID string) string {
	if provider, ok := LLMProviderDefinitionByID(providerID); ok {
		return provider.DefaultModel
	}
	return "llama3.2:latest"
}

func AICredentialProviderID(providerID string) string {
	return "ai_" + providerID
}

func LLMModelsForProvider(providerID string) []LLMModelDefinition {
	models := []LLMModelDefinition{}
	for _, model := range llmModelDefinitions {
		if model.Provider == providerID {
			models = append(models, model)
		}
	}
	return models
}

func InferLLMProviderFromModel(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case m == "":
		return ""
	case strings.HasPrefix(m, "claude-"):
		return LLMProviderAnthropic
	case strings.HasPrefix(m, "llama-3.") || strings.HasPrefix(m, "mixtral-") || strings.HasPrefix(m, "gemma2-"):
		return LLMProviderGroq
	case strings.Contains(m, "-cloud") || strings.HasSuffix(m, ":cloud"):
		return LLMProviderOllamaCloud
	default:
		return ""
	}
}

func ProviderKeySource(platformConfigured, byokConfigured bool) string {
	switch {
	case platformConfigured && byokConfigured:
		return "both"
	case byokConfigured:
		return LLMKeySourceBYOK
	case platformConfigured:
		return LLMKeySourcePlatform
	default:
		return "none"
	}
}
