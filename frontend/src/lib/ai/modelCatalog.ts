export const aiProviderIds = {
  ollamaLocal: "ollama_local",
  ollamaCloud: "ollama_cloud",
  groq: "groq",
  anthropic: "anthropic",
} as const;

export const osaProviderAliases: Record<string, string> = {
  ollama: aiProviderIds.ollamaLocal,
  "ollama-cloud": aiProviderIds.ollamaCloud,
  anthropic: aiProviderIds.anthropic,
};

export function isCloudModelId(modelId: string | undefined): boolean {
  const id = modelId?.toLowerCase() || "";
  return id.includes("-cloud") || id.endsWith(":cloud");
}

export function normalizeAIProviderId(providerId: string): string {
  return osaProviderAliases[providerId] || providerId;
}

export function isLocalProviderId(providerId: string): boolean {
  return normalizeAIProviderId(providerId) === aiProviderIds.ollamaLocal;
}
