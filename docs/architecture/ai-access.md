# AI Access Architecture

BusinessOS resolves AI access per user request. The resolver selects provider, model, key source, and billing mode before constructing an LLM service.

## Core Flow

1. Read requested model from the caller.
2. Load `user_settings.custom_settings.ai_access`.
3. Infer provider from model ID when the model clearly belongs to a provider.
4. Resolve key source:
   - local provider uses local runtime.
   - `byok` uses Credential Vault.
   - `platform` uses platform environment credentials.
   - if platform credentials are absent and a BYOK key exists, fall back to BYOK.
5. Return a request-scoped config and `LLMAccessPolicy`.
6. Construct the LLM service from that request-scoped config.

## Backend Modules

- `services/llm_catalog.go`: provider IDs, provider definitions, model catalog, defaults, credential provider IDs, provider inference, and key-source labels.
- `services/llm_access.go`: user-specific LLM access resolution.
- `services/llm.go`: provider service factory.
- `handlers/ai_config_providers.go`: provider status and AI access settings updates.
- `handlers/ai_config_models.go`: model list assembly from local Ollama and configured cloud providers.

## Rules

- Handlers must not own provider IDs, model defaults, model catalog data, or credential provider IDs.
- New cloud providers must be added to the catalog first.
- User-scoped API keys must be stored in Credential Vault with provider ID `ai_<provider>`.
- Agent runtimes should receive request-scoped config from `ResolveLLMAccess`, not global config directly.
- Module-level one-off LLM tasks should use `NewLLMServiceForUser` when a user context exists.

## Known Cleanup Queue

- Migrate legacy chat endpoints, slash commands, custom-agent sandbox tests, summarizer, onboarding LLM tasks, and subconscious classifier onto the same access resolver.
- Add xAI/OpenAI as first-class catalog providers before exposing them in settings.
- Persist usage records with key source and billing mode so platform credits can be enforced.
