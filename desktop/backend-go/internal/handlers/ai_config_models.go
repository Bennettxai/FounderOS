package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/rhl/businessos-backend/internal/utils"
)

// GetLocalModels returns models available from local Ollama instance
func (h *AIConfigHandler) GetLocalModels(c *gin.Context) {
	models, err := h.fetchOllamaModels()
	if err != nil {
		slog.Debug("local Ollama models unavailable", "error", err)
		c.JSON(http.StatusOK, gin.H{
			"models":    []LLMModel{},
			"provider":  "ollama",
			"base_url":  h.cfg.OllamaLocalURL,
			"available": false,
			"message":   "Ollama is not running locally.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"models":    models,
		"provider":  "ollama",
		"base_url":  h.cfg.OllamaLocalURL,
		"available": true,
	})
}

// GetAllModels returns all available models from all configured providers
func (h *AIConfigHandler) GetAllModels(c *gin.Context) {
	var allModels []LLMModel
	user := middleware.GetCurrentUser(c)
	byokConfigured := map[string]bool{}
	if user != nil {
		vault := services.NewCredentialVaultService(h.pool)
		for _, provider := range services.CloudLLMProviderIDs() {
			ok, _ := vault.CredentialExists(c.Request.Context(), user.ID, services.AICredentialProviderID(provider))
			byokConfigured[provider] = ok
		}
	}

	// Fetch local Ollama models
	ollamaModels, err := h.fetchOllamaModels()
	if err == nil {
		allModels = append(allModels, ollamaModels...)
	}

	for _, provider := range services.CloudLLMProviderIDs() {
		if !h.platformProviderConfigured(provider) && !byokConfigured[provider] {
			continue
		}
		for _, model := range services.LLMModelsForProvider(provider) {
			allModels = append(allModels, LLMModel{
				ID:          model.ID,
				Name:        model.Name,
				Provider:    model.Provider,
				Description: model.Description,
				Family:      model.Family,
			})
		}
	}

	// Get user's default model from their settings (if authenticated)
	defaultModel := h.cfg.DefaultModel
	activeProvider := h.cfg.GetActiveProvider()
	if user != nil {
		queries := sqlc.New(h.pool)
		settings, err := queries.GetUserSettings(c.Request.Context(), user.ID)
		if err == nil && settings.DefaultModel != nil && *settings.DefaultModel != "" {
			defaultModel = *settings.DefaultModel
		}
		if err == nil && settings.CustomSettings != nil {
			if provider := aiAccessProvider(settings.CustomSettings); provider != "" {
				activeProvider = provider
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"models":          allModels,
		"active_provider": activeProvider,
		"default_model":   defaultModel,
	})
}

// PullModel triggers pulling a model from Ollama registry with streaming progress
func (h *AIConfigHandler) PullModel(c *gin.Context) {
	var req struct {
		Model string `json:"model" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Make request to Ollama to pull the model
	pullReq := map[string]interface{}{
		"name":   req.Model,
		"stream": true,
	}
	body, err := json.Marshal(pullReq)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "marshal pull request", err)
		return
	}

	resp, err := http.Post(h.cfg.OllamaLocalURL+"/api/pull", "application/json",
		io.NopCloser(strings.NewReader(string(body))))
	if err != nil {
		utils.ServiceUnavailable(slog.Default(), "Ollama").
			WithMessage("Make sure Ollama is running locally").
			Respond(c)
		return
	}
	defer resp.Body.Close()

	// Set SSE headers for streaming
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Stream the response from Ollama
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		utils.RespondInternalError(c, slog.Default(), "initialize streaming", nil)
		return
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var progress map[string]interface{}
		if err := decoder.Decode(&progress); err != nil {
			if err == io.EOF {
				break
			}
			break
		}

		// Send progress event
		data, err := json.Marshal(progress)
		if err != nil {
			// Send error event
			errorData, _ := json.Marshal(map[string]interface{}{
				"status": "error",
				"error":  "failed to marshal progress",
			})
			fmt.Fprintf(c.Writer, "data: %s\n\n", errorData)
			flusher.Flush()
			break
		}
		fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		flusher.Flush()

		// Check if done
		if status, ok := progress["status"].(string); ok && status == "success" {
			break
		}
	}

	// Send completion event
	fmt.Fprintf(c.Writer, "data: {\"status\":\"complete\",\"model\":\"%s\"}\n\n", req.Model)
	flusher.Flush()
}

// WarmupModel sends a minimal request to load the model into memory.
// This significantly reduces first-message latency for Ollama models.
func (h *AIConfigHandler) WarmupModel(c *gin.Context) {
	var req struct {
		Model string `json:"model" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondInvalidRequest(c, slog.Default(), err)
		return
	}

	// Check if this is a local Ollama model (skip warmup for cloud providers)
	provider := h.cfg.GetActiveProvider()
	if provider != "ollama_local" {
		// For cloud providers, just return success (no warmup needed)
		c.JSON(http.StatusOK, gin.H{
			"status":   "skipped",
			"model":    req.Model,
			"provider": provider,
			"message":  "Cloud providers don't require warmup",
		})
		return
	}

	// Send minimal request to Ollama to load model into memory
	warmupReq := map[string]interface{}{
		"model":    req.Model,
		"messages": []map[string]string{{"role": "user", "content": "Hi"}},
		"stream":   false,
		"options": map[string]interface{}{
			"num_predict": 1, // Generate minimal tokens
		},
	}
	body, err := json.Marshal(warmupReq)
	if err != nil {
		utils.RespondInternalError(c, slog.Default(), "marshal warmup request", err)
		return
	}

	client := &http.Client{Timeout: 120 * time.Second} // Allow time for model loading
	resp, err := client.Post(h.cfg.OllamaLocalURL+"/api/chat", "application/json",
		strings.NewReader(string(body)))
	if err != nil {
		utils.ServiceUnavailable(slog.Default(), "Ollama").
			WithMessage("Make sure Ollama is running locally").
			Respond(c)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		utils.InternalError(slog.Default(), "Model warmup failed").
			WithDetails("response", string(respBody)).
			Respond(c)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ready",
		"model":    req.Model,
		"provider": provider,
		"message":  "Model loaded into memory",
	})
}

// fetchOllamaModels fetches available models from local Ollama
func (h *AIConfigHandler) fetchOllamaModels() ([]LLMModel, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(h.cfg.OllamaLocalURL + "/api/tags")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	var tagsResp OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tagsResp); err != nil {
		return nil, err
	}

	models := make([]LLMModel, 0, len(tagsResp.Models))
	for _, m := range tagsResp.Models {
		size := formatSize(m.Size)
		models = append(models, LLMModel{
			ID:          m.Name,
			Name:        m.Name,
			Provider:    "ollama",
			Description: m.Details.ParameterSize + " parameters",
			Size:        size,
			Family:      m.Details.Family,
		})
	}

	return models, nil
}

// isOllamaAvailable checks if Ollama is running
func (h *AIConfigHandler) isOllamaAvailable() bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(h.cfg.OllamaLocalURL + "/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
