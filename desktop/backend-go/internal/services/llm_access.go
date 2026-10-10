package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

const (
	LLMKeySourcePlatform = "platform"
	LLMKeySourceBYOK     = "byok"
	LLMKeySourceLocal    = "local"

	LLMBillingPlatformCredits = "platform_credits"
	LLMBillingBYOK            = "byok"
	LLMBillingLocal           = "local"
)

type LLMAccessPolicy struct {
	Provider            string `json:"provider"`
	Model               string `json:"model"`
	KeySource           string `json:"key_source"`
	BillingMode         string `json:"billing_mode"`
	UsesPlatformCredits bool   `json:"uses_platform_credits"`
}

// ResolveLLMAccess returns a request-scoped config with the provider/model/API key
// selected from user settings, BYOK credentials, platform credentials, and model ID.
func ResolveLLMAccess(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, userID, requestedModel string) (*config.Config, LLMAccessPolicy, error) {
	if cfg == nil {
		return nil, LLMAccessPolicy{}, fmt.Errorf("LLM config is nil")
	}

	resolved := *cfg
	settings := loadAIAccessSettings(ctx, pool, userID)

	provider := strings.TrimSpace(settings.Provider)
	if inferred := InferLLMProviderFromModel(requestedModel); inferred != "" {
		provider = inferred
	}
	if provider == "" {
		provider = cfg.GetActiveProvider()
	}
	if provider == "" {
		provider = LLMProviderOllamaLocal
	}

	model := strings.TrimSpace(requestedModel)
	if model == "" {
		model = modelForProvider(cfg, provider)
	}

	billingPreference := strings.TrimSpace(settings.BillingMode)
	if billingPreference == "" {
		billingPreference = LLMKeySourcePlatform
	}

	resolved.AIProvider = provider
	applyModelToConfig(&resolved, provider, model)

	if provider == LLMProviderOllamaLocal {
		return &resolved, LLMAccessPolicy{
			Provider:    provider,
			Model:       model,
			KeySource:   LLMKeySourceLocal,
			BillingMode: LLMBillingLocal,
		}, nil
	}

	vault := NewCredentialVaultService(pool)
	byokKey, byokOK := getBYOKAPIKey(ctx, vault, userID, provider)
	platformOK := platformProviderConfigured(cfg, provider)

	if billingPreference == LLMKeySourceBYOK {
		if byokOK {
			applyAPIKeyToConfig(&resolved, provider, byokKey)
			return &resolved, LLMAccessPolicy{
				Provider:    provider,
				Model:       model,
				KeySource:   LLMKeySourceBYOK,
				BillingMode: LLMBillingBYOK,
			}, nil
		}
		return nil, LLMAccessPolicy{}, fmt.Errorf("BYOK is selected but no %s API key is saved", provider)
	}

	if platformOK {
		return &resolved, LLMAccessPolicy{
			Provider:            provider,
			Model:               model,
			KeySource:           LLMKeySourcePlatform,
			BillingMode:         LLMBillingPlatformCredits,
			UsesPlatformCredits: true,
		}, nil
	}

	if byokOK {
		applyAPIKeyToConfig(&resolved, provider, byokKey)
		return &resolved, LLMAccessPolicy{
			Provider:    provider,
			Model:       model,
			KeySource:   LLMKeySourceBYOK,
			BillingMode: LLMBillingBYOK,
		}, nil
	}

	if cfg.LocalModelsAllowed() {
		resolved.AIProvider = LLMProviderOllamaLocal
		resolved.DefaultModel = modelForProvider(cfg, LLMProviderOllamaLocal)
		return &resolved, LLMAccessPolicy{
			Provider:    LLMProviderOllamaLocal,
			Model:       resolved.DefaultModel,
			KeySource:   LLMKeySourceLocal,
			BillingMode: LLMBillingLocal,
		}, nil
	}

	return nil, LLMAccessPolicy{}, fmt.Errorf("no credentials available for provider %s", provider)
}

func NewLLMServiceForUser(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config, userID, model string) (LLMService, LLMAccessPolicy, error) {
	resolvedCfg, policy, err := ResolveLLMAccess(ctx, pool, cfg, userID, model)
	if err != nil {
		return nil, LLMAccessPolicy{}, err
	}
	return NewLLMService(resolvedCfg, policy.Model), policy, nil
}

type aiAccessSettings struct {
	Provider    string `json:"provider"`
	BillingMode string `json:"billing_mode"`
	Runtime     string `json:"runtime"`
}

func loadAIAccessSettings(ctx context.Context, pool *pgxpool.Pool, userID string) aiAccessSettings {
	if pool == nil || userID == "" {
		return aiAccessSettings{}
	}
	settings, err := sqlc.New(pool).GetUserSettings(ctx, userID)
	if err != nil || settings.CustomSettings == nil {
		return aiAccessSettings{}
	}
	var custom map[string]json.RawMessage
	if err := json.Unmarshal(settings.CustomSettings, &custom); err != nil {
		return aiAccessSettings{}
	}
	var aiAccess aiAccessSettings
	if raw := custom["ai_access"]; raw != nil {
		_ = json.Unmarshal(raw, &aiAccess)
	}
	return aiAccess
}

func getBYOKAPIKey(ctx context.Context, vault *CredentialVaultService, userID, provider string) (string, bool) {
	if vault == nil || userID == "" {
		return "", false
	}
	key, err := vault.GetAPIKey(ctx, userID, AICredentialProviderID(provider))
	if err == nil && key != nil && key.APIKey != "" {
		return key.APIKey, true
	}
	key, err = vault.GetAPIKey(ctx, userID, provider)
	if err == nil && key != nil && key.APIKey != "" {
		return key.APIKey, true
	}
	return "", false
}

func modelForProvider(cfg *config.Config, provider string) string {
	switch provider {
	case LLMProviderOllamaCloud:
		if cfg.OllamaCloudModel != "" {
			return cfg.OllamaCloudModel
		}
		return DefaultModelForProvider(provider)
	case LLMProviderAnthropic:
		if cfg.AnthropicModel != "" {
			return cfg.AnthropicModel
		}
		return DefaultModelForProvider(provider)
	case LLMProviderGroq:
		if cfg.GroqModel != "" {
			return cfg.GroqModel
		}
		return DefaultModelForProvider(provider)
	default:
		if cfg.DefaultModel != "" {
			return cfg.DefaultModel
		}
		return DefaultModelForProvider(provider)
	}
}

func applyModelToConfig(cfg *config.Config, provider, model string) {
	switch provider {
	case LLMProviderOllamaCloud:
		cfg.OllamaCloudModel = model
	case LLMProviderAnthropic:
		cfg.AnthropicModel = model
	case LLMProviderGroq:
		cfg.GroqModel = model
	default:
		cfg.DefaultModel = model
	}
}

func platformProviderConfigured(cfg *config.Config, provider string) bool {
	switch provider {
	case LLMProviderOllamaCloud:
		return cfg.OllamaCloudAPIKey != ""
	case LLMProviderAnthropic:
		return cfg.AnthropicAPIKey != ""
	case LLMProviderGroq:
		return cfg.GroqAPIKey != ""
	default:
		return provider == LLMProviderOllamaLocal
	}
}

func applyAPIKeyToConfig(cfg *config.Config, provider, apiKey string) {
	switch provider {
	case LLMProviderOllamaCloud:
		cfg.OllamaCloudAPIKey = apiKey
	case LLMProviderAnthropic:
		cfg.AnthropicAPIKey = apiKey
	case LLMProviderGroq:
		cfg.GroqAPIKey = apiKey
	}
}
