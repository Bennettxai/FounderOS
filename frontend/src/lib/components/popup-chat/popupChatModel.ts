import { apiClient } from "$lib/api";

export interface LLMModel {
  id: string;
  name: string;
  provider: string;
  description?: string;
  size?: string;
}

export interface PullProgress {
  status: string;
  total?: number;
  completed?: number;
}

export const cloudModelsByProvider: Record<
  string,
  { id: string; name: string; description: string }[]
> = {
  groq: [
    {
      id: "llama-3.3-70b-versatile",
      name: "Llama 3.3 70B",
      description: "Fast 70B",
    },
    {
      id: "llama-3.1-8b-instant",
      name: "Llama 3.1 8B",
      description: "Ultra-fast",
    },
    {
      id: "mixtral-8x7b-32768",
      name: "Mixtral 8x7B",
      description: "32k context",
    },
  ],
  anthropic: [
    {
      id: "claude-sonnet-4-20250514",
      name: "Claude Sonnet 4",
      description: "Best for most tasks",
    },
    {
      id: "claude-opus-4-1-20250805",
      name: "Claude Opus 4",
      description: "Most capable",
    },
  ],
  ollama_cloud: [
    {
      id: "kimi-k2.6:cloud",
      name: "Kimi K2.6",
      description: "256K multimodal agentic coding model",
    },
    { id: "qwen3:480b", name: "Qwen3 480B", description: "Largest Qwen model" },
    { id: "qwen3:235b", name: "Qwen3 235B", description: "Large Qwen model" },
    { id: "qwen3:32b", name: "Qwen3 32B", description: "Qwen3 32B" },
    { id: "llama3.3:70b", name: "Llama 3.3 70B", description: "Latest Llama" },
    {
      id: "deepseek-r1:671b",
      name: "DeepSeek R1 671B",
      description: "Reasoning model",
    },
    {
      id: "deepseek-r1:70b",
      name: "DeepSeek R1 70B",
      description: "Compact reasoning",
    },
    {
      id: "command-a:111b",
      name: "Command A 111B",
      description: "Cohere Command",
    },
  ],
};

export interface ModelState {
  availableModels: LLMModel[];
  localModels: LLMModel[];
  selectedModel: string;
  activeProvider: string;
  configuredProviders: string[];
  isPulling: boolean;
  pullingModel: string;
  pullProgress: PullProgress | null;
  showModelSelector: boolean;
}

export interface ModelStateSetters {
  setAvailableModels: (v: LLMModel[]) => void;
  setLocalModels: (v: LLMModel[]) => void;
  setSelectedModel: (v: string) => void;
  setActiveProvider: (v: string) => void;
  setConfiguredProviders: (v: string[]) => void;
  setIsPulling: (v: boolean) => void;
  setPullingModel: (v: string) => void;
  setPullProgress: (v: PullProgress | null) => void;
  setShowModelSelector: (v: boolean) => void;
  getLocalModels: () => LLMModel[];
  getAvailableModels: () => LLMModel[];
  getSelectedModel: () => string;
  getIsPulling: () => boolean;
}

export async function loadModels(s: ModelStateSetters): Promise<void> {
  try {
    const providersRes = await apiClient.get("/ai/providers");
    if (providersRes.ok) {
      const data = await providersRes.json();
      s.setActiveProvider(data.active_provider || "ollama_local");
      s.setConfiguredProviders(
        (data.providers || [])
          .filter((p: { configured: boolean }) => p.configured)
          .map((p: { id: string }) => p.id),
      );
      if (data.default_model && !s.getSelectedModel()) {
        s.setSelectedModel(data.default_model);
      }
    }

    const response = await apiClient.get("/ai/models");
    if (response.ok) {
      const data = await response.json();
      const models: LLMModel[] = data.models || [];
      s.setAvailableModels(models);
      if (!s.getSelectedModel() && models.length > 0) {
        s.setSelectedModel(models[0].id);
      }
    }

    // Local models only exist when a local runtime (Ollama/OSA) is present. In
    // cloud mode this 503s - treat any failure as "no local models" silently so
    // the cloud user never sees console errors.
    try {
      const localResponse = await apiClient.get("/ai/models/local");
      if (localResponse.ok) {
        const data = await localResponse.json();
        const locals: LLMModel[] = data.models || [];
        s.setLocalModels(locals);
        if (!s.getSelectedModel() && locals.length > 0) {
          s.setSelectedModel(locals[0].id);
        }
      } else {
        s.setLocalModels([]);
      }
    } catch {
      // Local runtime unreachable - no local models available.
      s.setLocalModels([]);
    }
  } catch {
    // Model loading failure is non-fatal
  }
}

export function isModelPulled(
  localModels: LLMModel[],
  modelId: string,
): boolean {
  return localModels.some((m) => m.id === modelId || m.id.startsWith(modelId));
}

export function getModelProvider(
  availableModels: LLMModel[],
  modelId: string,
): string {
  const model = availableModels.find((m) => m.id === modelId);
  return model?.provider || "ollama";
}

export function isCloudModel(
  availableModels: LLMModel[],
  modelId: string,
): boolean {
  const provider = getModelProvider(availableModels, modelId);
  return (
    provider === "groq" ||
    provider === "anthropic" ||
    provider === "ollama_cloud"
  );
}

export async function pullModel(
  modelId: string,
  s: ModelStateSetters,
): Promise<void> {
  if (s.getIsPulling()) return;

  s.setIsPulling(true);
  s.setPullingModel(modelId);
  s.setPullProgress({ status: "Starting..." });

  try {
    const response = await fetch("/api/ai/models/pull", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ model: modelId }),
      credentials: "include",
    });

    if (!response.ok) {
      throw new Error("Failed to pull model");
    }

    const reader = response.body?.getReader();
    if (!reader) throw new Error("No response body");

    const decoder = new TextDecoder();
    let buffer = "";

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        if (line.startsWith("data: ")) {
          try {
            const data = JSON.parse(line.slice(6));
            s.setPullProgress(data);

            if (data.status === "complete" || data.status === "success") {
              s.setSelectedModel(modelId);
              s.setShowModelSelector(false);
              await loadModels(s);
            }
          } catch {
            // JSON parse failure is non-fatal
          }
        }
      }
    }
  } catch {
    s.setPullProgress({ status: "Failed to pull model" });
  } finally {
    s.setIsPulling(false);
    s.setPullingModel("");
    setTimeout(() => s.setPullProgress(null), 2000);
  }
}

export function selectModel(model: LLMModel, s: ModelStateSetters): void {
  if (
    model.provider === "ollama" &&
    !isModelPulled(s.getLocalModels(), model.id)
  ) {
    pullModel(model.id, s);
  } else {
    s.setSelectedModel(model.id);
    s.setShowModelSelector(false);
  }
}
