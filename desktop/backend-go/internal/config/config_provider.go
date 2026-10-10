package config

import "log/slog"

// GetActiveProvider returns the currently configured AI provider.
// Falls back to "ollama_local" when the configured provider lacks credentials
// and logs a warning so the misconfiguration is visible.
func (c *Config) GetActiveProvider() string {
	switch c.AIProvider {
	case "ollama_cloud":
		if c.OllamaCloudAPIKey != "" {
			return "ollama_cloud"
		}
		slog.Warn("[Config] AI_PROVIDER=ollama_cloud but OLLAMA_CLOUD_API_KEY is empty, falling back to ollama_local")
		return "ollama_local"
	case "anthropic":
		if c.AnthropicAPIKey != "" {
			return "anthropic"
		}
		slog.Warn("[Config] AI_PROVIDER=anthropic but ANTHROPIC_API_KEY is empty, falling back to ollama_local")
		return "ollama_local"
	case "groq":
		if c.GroqAPIKey != "" {
			return "groq"
		}
		slog.Warn("[Config] AI_PROVIDER=groq but GROQ_API_KEY is empty, falling back to ollama_local")
		return "ollama_local"
	case "ollama_local":
		return "ollama_local"
	default:
		if c.AIProvider != "" {
			slog.Warn("[Config] Unknown AI_PROVIDER value, falling back to ollama_local", "value", c.AIProvider)
		}
		return "ollama_local"
	}
}

// GetModelForProvider delegates to GetActiveModel for backwards compatibility.
func (c *Config) GetModelForProvider() string {
	return c.GetActiveModel()
}

// GetActiveModel returns the appropriate model name based on the active AI provider.
func (c *Config) GetActiveModel() string {
	switch c.GetActiveProvider() {
	case "ollama_cloud":
		if c.OllamaCloudModel != "" {
			return c.OllamaCloudModel
		}
		return "llama3.2"
	case "groq":
		if c.GroqModel != "" {
			return c.GroqModel
		}
		return "llama-3.3-70b-versatile"
	case "anthropic":
		if c.AnthropicModel != "" {
			return c.AnthropicModel
		}
		return "claude-sonnet-4-20250514"
	case "ollama_local":
		fallthrough
	default:
		if c.DefaultModel != "" {
			return c.DefaultModel
		}
		return "llama3.2:latest"
	}
}

// UseOllamaCloud returns true if Ollama Cloud should be used.
func (c *Config) UseOllamaCloud() bool {
	return c.GetActiveProvider() == "ollama_cloud"
}

// UseAnthropic returns true if Anthropic/Claude should be used.
func (c *Config) UseAnthropic() bool {
	return c.GetActiveProvider() == "anthropic"
}

// UseGroq returns true if Groq should be used.
func (c *Config) UseGroq() bool {
	return c.GetActiveProvider() == "groq"
}

// UseOllamaLocal returns true if local Ollama should be used.
func (c *Config) UseOllamaLocal() bool {
	return c.GetActiveProvider() == "ollama_local"
}

// IsProduction returns true if running in production environment.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

// IsCloudDeployment returns true when BusinessOS is running inside a MIOSA
// Firecracker VM (DEPLOYMENT_MODE=cloud). In cloud mode the MIOSA JWT auth
// middleware is active and tenant-scoped behaviour applies.
func (c *Config) IsCloudDeployment() bool {
	return DeploymentMode(c.DeploymentMode) == DeploymentCloud
}

// LocalModelsAllowed returns true if local models can be used.
// In production, this respects the explicit EnableLocalModels flag.
// In development, local models are always allowed.
func (c *Config) LocalModelsAllowed() bool {
	if c.IsProduction() {
		return c.EnableLocalModels
	}
	return true
}
