# Platform Taxonomy

This document defines canonical names for BusinessOS platform concepts. Use these terms in code, docs, settings, and issue descriptions.

## Platform

- **MIOSA**: the company and ecosystem.
- **MIOSA Platform**: hosted control plane for billing, fleet management, compute, and shared platform services.
- **BusinessOS** or **BOS**: this product and repository.
- **BusinessOS Cloud**: hosted single-tenant BusinessOS running in an isolated microVM. Do not describe it as generic multi-tenant SaaS.
- **Optimal System**: domain-specific operating environment. BusinessOS is one Optimal System.
- **OptimalEngine**: knowledge substrate and signal engine. The Elixir implementation is a reference system; the Go runtime is the production path inside BusinessOS.

## Product Surfaces

- **Module**: user-facing app surface, route, or window. Prefer "module" over "app" in platform architecture docs.
- **CustomModule**: user-authored module definition.
- **ModuleInstallation**: installed copy of a module in a workspace.
- **ModuleInstance**: runnable instance of an installed module.
- **ModuleFile**: source or asset file belonging to a module.

## Agents And Runtime

- **OSA**: Optimal System Agent. Current architecture treats OSA as a separate system reached through the backend-owned HTTP runtime adapter, not as a built-in BusinessOS package.
- **Local reactive agents**: BusinessOS fallback agents such as intent routing, chain-of-thought orchestration, and specialist agents.
- **Custom agent**: user-configurable agent profile with prompt, model preference, and tool settings.
- **RunnableAgent**: Go interface for registry-managed BusinessOS agents.
- **Agent runtime**: execution environment that runs an agent loop, model calls, tool calls, and continuation steps.
- **BusinessOS agent runtime**: the built-in runtime used by chat and local reactive agents.
- **External runtime**: a future runtime adapter such as Claude Code, Codex, or another tool-running agent system.
- **Harness**: evaluation or integration wiring around an agent/runtime. Do not use "harness" as a synonym for runtime.
- **Sandbox**: isolated execution or preview environment for generated modules or code.

## AI Access

- **LLM provider**: model API provider, such as Ollama Cloud, Groq, Anthropic, OpenAI, or xAI.
- **Model**: selected model ID, such as `kimi-k2.6:cloud`.
- **LLM access policy**: resolved provider, model, key source, billing mode, and request-scoped config.
- **Key source**:
  - `platform`: BusinessOS platform credential.
  - `byok`: user-owned credential from Credential Vault.
  - `local`: local runtime with no cloud key.
- **Billing mode**:
  - `platform_credits`: bill against BusinessOS credits.
  - `byok`: user pays the provider directly through their own key.
  - `local`: local runtime, no cloud usage billing.
- **Credential Vault**: encrypted user/provider credential store.
- **AI access settings**: `user_settings.custom_settings.ai_access`, currently `{ provider, billing_mode, runtime }`.

## Current Code Seams

- Backend provider/model catalog: `desktop/backend-go/internal/services/llm_catalog.go`.
- Backend access resolver: `desktop/backend-go/internal/services/llm_access.go`.
- Backend LLM factory: `desktop/backend-go/internal/services/llm.go`.
- AI settings handlers: `desktop/backend-go/internal/handlers/ai_config_*.go`.
- Frontend provider/model helpers: `frontend/src/lib/ai/modelCatalog.ts`.

## Naming Rules

- Use `ollama_local`, `ollama_cloud`, `groq`, and `anthropic` for provider IDs in BusinessOS settings.
- Use `ollama` and `ollama-cloud` only as OSA UI aliases; normalize them before sharing state with BusinessOS AI settings.
- Use `:cloud` and `-cloud` model suffix checks only through a shared helper.
- Prefer "LLM access policy" when describing provider/key/billing resolution.
- Prefer "agent runtime" when describing execution, and "harness" only for integration or evaluation wiring.
