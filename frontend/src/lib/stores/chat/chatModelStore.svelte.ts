/**
 * chatModelStore.svelte.ts
 *
 * Svelte 5 runes-based singleton store that owns all model selection,
 * discovery, and AI-parameter state for the chat page.
 *
 * Responsibilities:
 *  - Fetch and separate local / cloud models from the backend
 *  - Track which cloud providers are configured
 *  - Persist the selected model and AI parameters to user settings
 *  - Pre-warm Ollama local models for reduced first-message latency
 *  - Sync the COT (chain-of-thought) toggle to the thinking store
 */

import { get } from "svelte/store";
import { api, apiClient } from "$lib/api";
import { handleApiCall } from "$lib/utils/api-handler";
import { thinking } from "$lib/stores/thinking";
import {
  getModelDescription,
  cloudModelsMap,
  modelContextLimits,
  type ModelOption,
} from "./chatActions";

// ─── Cloud-model migration map ────────────────────────────────────────────────
// Old IDs stored in user settings that must be silently upgraded on load.
const cloudModelMigrations: Record<string, string> = {
  "qwen3-coder:480b": "qwen3-coder:480b-cloud",
  "qwen3-coder:30b": "qwen3-coder:30b-cloud",
};

// ─── Store factory ────────────────────────────────────────────────────────────

function createChatModelStore() {
  // ── Reactive state ──────────────────────────────────────────────────────────
  let selectedModel = $state("");
  let installedModels = $state<ModelOption[]>([]);
  let ollamaCloudModels = $state<ModelOption[]>([]);
  let loadingModels = $state(false);
  let activeProvider = $state("ollama_local");
  let configuredProviders = $state<Set<string>>(new Set());

  // Persist warmed-up model IDs across page reloads via localStorage.
  let warmedUpModels = $state<Set<string>>(
    typeof window !== "undefined" && localStorage.getItem("warmedUpModels")
      ? new Set<string>(
          JSON.parse(
            localStorage.getItem("warmedUpModels") ?? "[]",
          ) as string[],
        )
      : new Set<string>(),
  );

  let warmingUpModel = $state<string | null>(null);
  let aiTemperature = $state(0.7);
  let aiMaxTokens = $state(8192);
  let aiTopP = $state(0.9);
  let useCOT = $state(false);
  let showUsageInChat = $state(true);

  // ── Derived state ───────────────────────────────────────────────────────────

  /**
   * Full combined model list: local Ollama + Ollama-cloud + configured
   * cloud-provider models (Groq, Anthropic, etc.).
   */
  const models = $derived.by<ModelOption[]>(() => {
    const all: ModelOption[] = [...installedModels, ...ollamaCloudModels];

    for (const provider of configuredProviders) {
      const providerModels = cloudModelsMap[provider] ?? [];
      for (const model of providerModels) {
        all.push(model);
      }
    }

    return all;
  });

  /** Human-readable display name for the currently selected model. */
  const currentModelName = $derived(
    models.find((m) => m.id === selectedModel)?.name ?? selectedModel,
  );

  /** Context window token limit for the currently selected model. */
  const currentContextLimit = $derived.by<number>(() => {
    // Exact match
    if (modelContextLimits[selectedModel]) {
      return modelContextLimits[selectedModel];
    }

    // Base model (strip tag, e.g. "llama3.2:latest" -> "llama3.2")
    const baseModel = selectedModel.split(":")[0];
    if (modelContextLimits[baseModel]) {
      return modelContextLimits[baseModel];
    }

    // Substring match
    for (const [key, limit] of Object.entries(modelContextLimits)) {
      if (selectedModel.toLowerCase().includes(key.toLowerCase())) {
        return limit;
      }
    }

    return 8192; // Conservative fallback
  });

  // ── Internal helpers ────────────────────────────────────────────────────────

  function persistWarmedUpModels(set: Set<string>): void {
    if (typeof window !== "undefined") {
      localStorage.setItem("warmedUpModels", JSON.stringify([...set]));
    }
  }

  // ── Public methods ──────────────────────────────────────────────────────────

  /**
   * Fetch available providers and local models, then populate installedModels,
   * ollamaCloudModels, configuredProviders, and activeProvider.
   *
   * Validates that the current selectedModel still exists in the combined list;
   * falls back to the first available model when it does not.
   */
  async function loadModels(): Promise<void> {
    loadingModels = true;

    const { error } = await handleApiCall(
      async () => {
        // ── Step 1: Provider list ─────────────────────────────────────────────
        const providersRes = await apiClient.get("/ai/providers");
        if (providersRes.ok) {
          const data = (await providersRes.json()) as {
            active_provider?: string;
            providers?: { id: string; configured: boolean }[];
          };
          activeProvider = data.active_provider ?? "ollama_local";

          const configured = new Set<string>();
          for (const provider of data.providers ?? []) {
            if (provider.configured) {
              configured.add(provider.id);
            }
          }
          configuredProviders = configured;
        }

        // ── Step 2: Local models ──────────────────────────────────────────────
        // A local Ollama runtime only exists in desktop mode. In cloud mode this
        // endpoint 503s (no local runtime), so treat any failure as "no local
        // models" and degrade quietly - never let it abort provider loading.
        try {
          const localRes = await apiClient.get("/ai/models/local");
          if (localRes.ok) {
            const data = (await localRes.json()) as {
              models?: {
                id: string;
                name?: string;
                family?: string;
                size?: string;
                type?: string;
              }[];
            };

            const filtered = (data.models ?? [])
              .filter((m) => {
                // Discard empty cloud-reference stubs
                const size = m.size ?? "";
                return size !== "< 1 KB" && size !== "0 B";
              })
              .map((m): ModelOption => ({
                id: m.id,
                name: m.name ?? m.id,
                description: getModelDescription(m.id, m.family),
                type: (m.type === "cloud" ? "cloud" : "local") as
                  "local" | "cloud",
                size: m.size,
              }));

            installedModels = filtered.filter((m) => m.type === "local");
            ollamaCloudModels = filtered.filter((m) => m.type === "cloud");
          } else {
            // No local runtime available - show no local models, silently.
            installedModels = [];
            ollamaCloudModels = [];
          }
        } catch {
          // Network error reaching local runtime - treat as no local models.
          installedModels = [];
          ollamaCloudModels = [];
        }

        // ── Step 3: Validate selectedModel exists ─────────────────────────────
        if (selectedModel) {
          const allAvailable: ModelOption[] = [
            ...installedModels,
            ...ollamaCloudModels,
          ];
          for (const provider of configuredProviders) {
            allAvailable.push(...(cloudModelsMap[provider] ?? []));
          }

          const exists = allAvailable.some((m) => m.id === selectedModel);
          if (!exists && allAvailable.length > 0) {
            selectedModel = allAvailable[0].id;
          }
        }
      },
      { showErrorToast: false },
    );

    // Model loading is best-effort. Failures (e.g. cloud mode with no local
    // runtime, or a transient 503) degrade to whatever models did load - we do
    // not log to the console so a cloud user isn't flooded with errors.
    void error;

    loadingModels = false;
  }

  /**
   * Load user preferences (model, AI params, usage display) from the backend.
   * Must be called before loadModels so the saved model is respected during
   * the validation step in loadModels.
   */
  async function loadUserSettings(): Promise<void> {
    try {
      const response = await apiClient.get("/settings");
      if (!response.ok) return;

      const settings = (await response.json()) as {
        default_model?: string | null;
        model_settings?: {
          showUsageInChat?: boolean;
          temperature?: number;
          max_tokens?: number;
          top_p?: number;
        };
        custom_settings?: {
          showUsageInChat?: boolean;
        };
      };

      // ── showUsageInChat ───────────────────────────────────────────────────
      if (settings.model_settings?.showUsageInChat !== undefined) {
        showUsageInChat = settings.model_settings.showUsageInChat;
      } else if (settings.custom_settings?.showUsageInChat !== undefined) {
        showUsageInChat = settings.custom_settings.showUsageInChat;
      }

      // ── Default model (with cloud migration) ──────────────────────────────
      if (settings.default_model) {
        let savedModel = settings.default_model;

        // Migrate stale model IDs
        if (cloudModelMigrations[savedModel]) {
          savedModel = cloudModelMigrations[savedModel];
        }

        selectedModel = savedModel;
      }

      // ── AI generation parameters ──────────────────────────────────────────
      if (settings.model_settings) {
        const ms = settings.model_settings;
        if (typeof ms.temperature === "number") aiTemperature = ms.temperature;
        if (typeof ms.max_tokens === "number") aiMaxTokens = ms.max_tokens;
        if (typeof ms.top_p === "number") aiTopP = ms.top_p;
      }
    } catch (err) {
      console.error("[ChatModelStore] loadUserSettings failed:", err);
    }
  }

  /**
   * Persist the selected model ID to the user's backend settings.
   */
  async function saveModelPreference(modelId: string): Promise<void> {
    try {
      await apiClient.put("/settings", { default_model: modelId });
    } catch (err) {
      console.error("[ChatModelStore] saveModelPreference failed:", err);
    }
  }

  /**
   * Select a model, persist the preference, and trigger a pre-warm when on
   * the Ollama local provider.
   */
  function selectModel(modelId: string): void {
    selectedModel = modelId;
    saveModelPreference(modelId);

    if (activeProvider === "ollama_local" && !warmedUpModels.has(modelId)) {
      warmupModel(modelId);
    }
  }

  /**
   * Send a warmup request for an Ollama model to pre-load it into GPU memory.
   * Silently ignores failures - this is best-effort.
   */
  async function warmupModel(modelId: string): Promise<void> {
    if (warmingUpModel === modelId || warmedUpModels.has(modelId)) return;

    warmingUpModel = modelId;

    try {
      const result = await api.warmupModel(modelId);
      if (result.status === "ready" || result.status === "skipped") {
        markModelWarmedUp(modelId);
      }
    } catch (err) {
      // Warmup is best-effort; swallow the error
      console.warn("[ChatModelStore] warmupModel failed (non-critical):", err);
    } finally {
      if (warmingUpModel === modelId) {
        warmingUpModel = null;
      }
    }
  }

  /**
   * Mark a model ID as warmed up in the in-memory set and persist to
   * localStorage so the warm state survives page reloads.
   */
  function markModelWarmedUp(modelId: string): void {
    const next = new Set(warmedUpModels);
    next.add(modelId);
    warmedUpModels = next;
    persistWarmedUpModels(next);
  }

  /**
   * Toggle chain-of-thought reasoning. Persists the new state to the thinking
   * store's backend settings and reverts on error.
   */
  async function toggleCOT(): Promise<void> {
    const prev = useCOT;
    useCOT = !prev;

    try {
      const thinkingState = get(thinking);
      await thinking.updateSettings({
        enabled: useCOT,
        show_in_ui: thinkingState.settings?.show_in_ui ?? false,
        save_traces: thinkingState.settings?.save_traces ?? true,
        max_tokens: thinkingState.settings?.max_tokens ?? 4096,
        default_template_id:
          thinkingState.settings?.default_template_id ?? null,
      });
    } catch (err) {
      // Revert on failure
      useCOT = prev;
      console.error("[ChatModelStore] toggleCOT failed:", err);
    }
  }

  // ── Public surface ──────────────────────────────────────────────────────────

  return {
    // Getters & setters for primitive state
    get selectedModel() {
      return selectedModel;
    },
    set selectedModel(v: string) {
      selectedModel = v;
    },

    get installedModels() {
      return installedModels;
    },
    set installedModels(v: ModelOption[]) {
      installedModels = v;
    },

    get ollamaCloudModels() {
      return ollamaCloudModels;
    },
    set ollamaCloudModels(v: ModelOption[]) {
      ollamaCloudModels = v;
    },

    get loadingModels() {
      return loadingModels;
    },
    set loadingModels(v: boolean) {
      loadingModels = v;
    },

    get activeProvider() {
      return activeProvider;
    },
    set activeProvider(v: string) {
      activeProvider = v;
    },

    get configuredProviders() {
      return configuredProviders;
    },
    set configuredProviders(v: Set<string>) {
      configuredProviders = v;
    },

    get warmedUpModels() {
      return warmedUpModels;
    },
    set warmedUpModels(v: Set<string>) {
      warmedUpModels = v;
      persistWarmedUpModels(v);
    },

    get warmingUpModel() {
      return warmingUpModel;
    },
    set warmingUpModel(v: string | null) {
      warmingUpModel = v;
    },

    get aiTemperature() {
      return aiTemperature;
    },
    set aiTemperature(v: number) {
      aiTemperature = v;
    },

    get aiMaxTokens() {
      return aiMaxTokens;
    },
    set aiMaxTokens(v: number) {
      aiMaxTokens = v;
    },

    get aiTopP() {
      return aiTopP;
    },
    set aiTopP(v: number) {
      aiTopP = v;
    },

    get useCOT() {
      return useCOT;
    },
    set useCOT(v: boolean) {
      useCOT = v;
    },

    get showUsageInChat() {
      return showUsageInChat;
    },
    set showUsageInChat(v: boolean) {
      showUsageInChat = v;
    },

    // Derived (read-only)
    get models() {
      return models;
    },
    get currentModelName() {
      return currentModelName;
    },
    get currentContextLimit() {
      return currentContextLimit;
    },

    // Methods
    loadModels,
    loadUserSettings,
    saveModelPreference,
    selectModel,
    warmupModel,
    markModelWarmedUp,
    toggleCOT,
  };
}

// ─── Singleton export ─────────────────────────────────────────────────────────
export const chatModelStore = createChatModelStore();
