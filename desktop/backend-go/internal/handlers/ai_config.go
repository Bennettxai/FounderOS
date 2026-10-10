package handlers

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
)

// AIConfigHandler handles AI provider configuration and model management
type AIConfigHandler struct {
	pool *pgxpool.Pool
	cfg  *config.Config
}

// NewAIConfigHandler creates a new AIConfigHandler
func NewAIConfigHandler(pool *pgxpool.Pool, cfg *config.Config) *AIConfigHandler {
	return &AIConfigHandler{pool: pool, cfg: cfg}
}

// LLMProvider represents an LLM provider configuration
type LLMProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"` // "local" or "cloud"
	Description string `json:"description"`
	Configured  bool   `json:"configured"`
	BaseURL     string `json:"base_url,omitempty"`
	KeySource   string `json:"key_source,omitempty"` // "platform", "byok", "local", "none"
	PlatformKey bool   `json:"platform_configured"`
	BYOKKey     bool   `json:"byok_configured"`
}

// LLMModel represents an available model
type LLMModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"` // "ollama", "anthropic", etc.
	Description string `json:"description,omitempty"`
	Size        string `json:"size,omitempty"`
	Family      string `json:"family,omitempty"`
}

// OllamaTagsResponse represents the response from Ollama's /api/tags endpoint
type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

// OllamaModel represents a model in Ollama's response
type OllamaModel struct {
	Name       string `json:"name"`
	Model      string `json:"model"`
	ModifiedAt string `json:"modified_at"`
	Size       int64  `json:"size"`
	Details    struct {
		Family            string   `json:"family"`
		Families          []string `json:"families"`
		ParameterSize     string   `json:"parameter_size"`
		QuantizationLevel string   `json:"quantization_level"`
	} `json:"details"`
}

// SystemInfo represents system hardware information
type SystemInfo struct {
	TotalRAM          int64              `json:"total_ram_gb"`
	AvailableRAM      int64              `json:"available_ram_gb"`
	Platform          string             `json:"platform"`
	HasGPU            bool               `json:"has_gpu"`
	GPUName           string             `json:"gpu_name,omitempty"`
	RecommendedModels []RecommendedModel `json:"recommended_models"`
}

// RecommendedModel represents a model recommendation with resource requirements
type RecommendedModel struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	RAMRequired string `json:"ram_required"`
	Speed       string `json:"speed"`
	Quality     string `json:"quality"`
}

// AgentInfo represents an AI agent with its prompt
type AgentInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
	Category    string `json:"category"`
}
