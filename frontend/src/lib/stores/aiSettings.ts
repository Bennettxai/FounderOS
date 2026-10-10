// Shared types and utilities for AI Settings page

// ──────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────

export interface LLMModel {
  id: string;
  name: string;
  provider: string;
  description?: string;
  size?: string;
  family?: string;
}

export interface LLMProvider {
  id: string;
  name: string;
  type: string;
  description: string;
  configured: boolean;
  base_url?: string;
  key_source?: "platform" | "byok" | "both" | "local" | "none" | string;
  platform_configured?: boolean;
  byok_configured?: boolean;
}

export interface PullProgress {
  status: string;
  digest?: string;
  total?: number;
  completed?: number;
}

export interface RecommendedModel {
  name: string;
  description: string;
  ram_required: string;
  speed: string;
  quality: string;
}

export interface SystemInfo {
  total_ram_gb: number;
  available_ram_gb: number;
  platform: string;
  has_gpu: boolean;
  gpu_name?: string;
  recommended_models: RecommendedModel[];
}

export interface OutputStyle {
  id: string;
  name: string;
  display_name: string;
  description?: string;
  is_active: boolean;
  sort_order: number;
}

export interface OutputPreference {
  user_id: string;
  default_style_id?: string;
  default_style_name?: string;
  style_overrides: Record<string, string>;
  custom_instructions?: string;
}

export interface UserFact {
  id: string;
  user_id: string;
  fact_key: string;
  fact_value: string;
  fact_type: string;
  source_memory_id?: string | null;
  confidence_score: number;
  is_active: boolean;
  last_confirmed_at?: string | null;
  created_at: string;
  updated_at: string;
}

export interface UsageStats {
  total_requests: number;
  total_tokens: number;
  total_cost: number;
  input_tokens: number;
  output_tokens: number;
  by_provider: Record<
    string,
    {
      requests: number;
      tokens: number;
      cost: number;
      input_tokens: number;
      output_tokens: number;
    }
  >;
  by_model: Record<
    string,
    {
      requests: number;
      tokens: number;
      input_tokens: number;
      output_tokens: number;
      avg_latency_ms: number;
    }
  >;
  by_agent: Record<string, { requests: number; tokens: number }>;
  recent: { date: string; requests: number; tokens: number; cost: number }[];
  daily_trend: {
    date: string;
    local_requests: number;
    cloud_requests: number;
    local_tokens: number;
    cloud_tokens: number;
  }[];
  session_count: number;
  avg_session_duration_min: number;
  avg_requests_per_session: number;
  local_model_storage_gb: number;
  avg_response_time_ms: number;
  local_power_cost_estimate: number;
  cloud_api_cost: number;
  period_start: string;
  period_end: string;
}

export interface CommandInfo {
  id?: string;
  name: string;
  display_name: string;
  description: string;
  icon: string;
  category: string;
  context_sources: string[];
  is_custom: boolean;
  system_prompt?: string;
  is_builtin_override?: boolean;
}

export interface AgentInfo {
  id: string;
  name: string;
  description: string;
  prompt: string;
  category: "general" | "specialist" | "system";
}

export interface CustomAgent {
  id: string;
  user_id: string;
  name: string;
  display_name: string;
  description: string;
  avatar?: string;
  system_prompt: string;
  model_preference?: string;
  temperature?: number;
  max_tokens?: number;
  capabilities?: string[];
  tools_enabled?: string[];
  context_sources?: string[];
  thinking_enabled?: boolean;
  streaming_enabled?: boolean;
  category?: string;
  is_active: boolean;
  times_used: number;
  last_used_at?: string;
  created_at: string;
  updated_at: string;
}

export type ModelCapability =
  | "vision"
  | "tools"
  | "coding"
  | "reasoning"
  | "rag"
  | "multilingual"
  | "fast";

export interface ModelVariant {
  id: string;
  params: string;
  size: string;
}

export interface AvailableModel {
  id: string;
  name: string;
  description: string;
  size: string;
  params: string;
  capabilities: ModelCapability[];
  provider: "local" | "cloud";
  downloads?: string;
  isInstalled?: boolean;
  variants?: ModelVariant[];
}

// ──────────────────────────────────────────────
// Static data
// ──────────────────────────────────────────────

export const capabilityInfo: Record<
  ModelCapability,
  { label: string; color: string; iconPath: string }
> = {
  vision: {
    label: "Vision",
    color: "bg-purple-100 text-purple-700",
    iconPath:
      "M15 12a3 3 0 11-6 0 3 3 0 016 0zM2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z",
  },
  tools: {
    label: "Tools",
    color: "bg-blue-100 text-blue-700",
    iconPath:
      "M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065zM15 12a3 3 0 11-6 0 3 3 0 016 0z",
  },
  coding: {
    label: "Code",
    color: "bg-green-100 text-green-700",
    iconPath: "M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4",
  },
  reasoning: {
    label: "Reasoning",
    color: "bg-orange-100 text-orange-700",
    iconPath:
      "M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z",
  },
  rag: {
    label: "RAG",
    color: "bg-cyan-100 text-cyan-700",
    iconPath:
      "M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253",
  },
  multilingual: {
    label: "Multi-lang",
    color: "bg-pink-100 text-pink-700",
    iconPath:
      "M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9",
  },
  fast: {
    label: "Fast",
    color: "bg-yellow-100 text-yellow-700",
    iconPath: "M13 10V3L4 14h7v7l9-11h-7z",
  },
};

export const availableModels: AvailableModel[] = [
  // Coding & Tool-focused
  {
    id: "qwen3-coder",
    name: "Qwen3 Coder",
    description: "Best for coding & agentic tasks",
    size: "9GB",
    params: "14B",
    capabilities: ["coding", "tools", "multilingual"],
    provider: "local",
    downloads: "2.1M",
    variants: [
      { id: "qwen3-coder:1.5b", params: "1.5B", size: "1GB" },
      { id: "qwen3-coder:7b", params: "7B", size: "4.5GB" },
      { id: "qwen3-coder:14b", params: "14B", size: "9GB" },
      { id: "qwen3-coder:30b", params: "30B", size: "19GB" },
    ],
  },
  {
    id: "deepseek-coder-v2",
    name: "DeepSeek Coder V2",
    description: "Strong coding assistant",
    size: "10GB",
    params: "16B",
    capabilities: ["coding", "tools"],
    provider: "local",
    downloads: "500K",
    variants: [
      { id: "deepseek-coder-v2:16b", params: "16B", size: "10GB" },
      { id: "deepseek-coder-v2:236b", params: "236B", size: "Cloud" },
    ],
  },
  {
    id: "codellama",
    name: "Code Llama",
    description: "Meta's code-focused model",
    size: "4GB",
    params: "7B",
    capabilities: ["coding"],
    provider: "local",
    downloads: "2M",
    variants: [
      { id: "codellama:7b", params: "7B", size: "4GB" },
      { id: "codellama:13b", params: "13B", size: "7GB" },
      { id: "codellama:34b", params: "34B", size: "19GB" },
      { id: "codellama:70b", params: "70B", size: "39GB" },
    ],
  },
  {
    id: "starcoder2:15b",
    name: "StarCoder2 15B",
    description: "BigCode coding model",
    size: "9GB",
    params: "15B",
    capabilities: ["coding"],
    provider: "local",
    downloads: "300K",
  },

  // Reasoning
  {
    id: "deepseek-r1",
    name: "DeepSeek R1",
    description: "Advanced reasoning model",
    size: "9GB",
    params: "14B",
    capabilities: ["reasoning", "coding", "tools"],
    provider: "local",
    downloads: "1.9M",
    variants: [
      { id: "deepseek-r1:1.5b", params: "1.5B", size: "1GB" },
      { id: "deepseek-r1:7b", params: "7B", size: "4.5GB" },
      { id: "deepseek-r1:8b", params: "8B", size: "5GB" },
      { id: "deepseek-r1:14b", params: "14B", size: "9GB" },
      { id: "deepseek-r1:32b", params: "32B", size: "20GB" },
      { id: "deepseek-r1:70b", params: "70B", size: "43GB" },
    ],
  },

  // Vision models
  {
    id: "llama3.2-vision",
    name: "Llama 3.2 Vision",
    description: "Multimodal vision-language",
    size: "7GB",
    params: "11B",
    capabilities: ["vision", "tools"],
    provider: "local",
    downloads: "1.5M",
    variants: [
      { id: "llama3.2-vision:11b", params: "11B", size: "7GB" },
      { id: "llama3.2-vision:90b", params: "90B", size: "55GB" },
    ],
  },
  {
    id: "llava",
    name: "LLaVA",
    description: "Vision-language model",
    size: "5GB",
    params: "7B",
    capabilities: ["vision"],
    provider: "local",
    downloads: "2M",
    variants: [
      { id: "llava:7b", params: "7B", size: "5GB" },
      { id: "llava:13b", params: "13B", size: "8GB" },
      { id: "llava:34b", params: "34B", size: "20GB" },
    ],
  },
  {
    id: "minicpm-v:8b",
    name: "MiniCPM-V 8B",
    description: "Efficient vision model",
    size: "5GB",
    params: "8B",
    capabilities: ["vision", "fast"],
    provider: "local",
    downloads: "400K",
  },
  {
    id: "moondream",
    name: "Moondream",
    description: "Tiny vision model",
    size: "1.7GB",
    params: "1.8B",
    capabilities: ["vision", "fast"],
    provider: "local",
    downloads: "500K",
  },

  // General purpose
  {
    id: "llama3.3",
    name: "Llama 3.3",
    description: "Latest Meta flagship",
    size: "43GB",
    params: "70B",
    capabilities: ["tools", "coding", "reasoning"],
    provider: "local",
    downloads: "3M",
    variants: [{ id: "llama3.3:70b", params: "70B", size: "43GB" }],
  },
  {
    id: "llama3.2",
    name: "Llama 3.2",
    description: "Fast general purpose",
    size: "2GB",
    params: "3B",
    capabilities: ["tools", "fast"],
    provider: "local",
    downloads: "9M",
    variants: [
      { id: "llama3.2:1b", params: "1B", size: "1.3GB" },
      { id: "llama3.2:3b", params: "3B", size: "2GB" },
      { id: "llama3.2:8b", params: "8B", size: "5GB" },
    ],
  },
  {
    id: "qwen3",
    name: "Qwen3",
    description: "Strong multilingual model",
    size: "5GB",
    params: "8B",
    capabilities: ["tools", "coding", "reasoning", "multilingual"],
    provider: "local",
    downloads: "4.5M",
    variants: [
      { id: "qwen3:0.6b", params: "0.6B", size: "0.5GB" },
      { id: "qwen3:1.7b", params: "1.7B", size: "1.2GB" },
      { id: "qwen3:4b", params: "4B", size: "2.5GB" },
      { id: "qwen3:8b", params: "8B", size: "5GB" },
      { id: "qwen3:14b", params: "14B", size: "9GB" },
      { id: "qwen3:32b", params: "32B", size: "20GB" },
    ],
  },
  {
    id: "mistral",
    name: "Mistral",
    description: "Efficient general model",
    size: "4GB",
    params: "7B",
    capabilities: ["tools", "fast"],
    provider: "local",
    downloads: "4M",
    variants: [{ id: "mistral:7b", params: "7B", size: "4GB" }],
  },
  {
    id: "gemma2",
    name: "Gemma 2",
    description: "Google efficient model",
    size: "2GB",
    params: "2B",
    capabilities: ["fast"],
    provider: "local",
    downloads: "2M",
    variants: [
      { id: "gemma2:2b", params: "2B", size: "2GB" },
      { id: "gemma2:9b", params: "9B", size: "5GB" },
      { id: "gemma2:27b", params: "27B", size: "16GB" },
    ],
  },
  {
    id: "phi3",
    name: "Phi-3",
    description: "Microsoft reasoning model",
    size: "2.2GB",
    params: "3.8B",
    capabilities: ["reasoning", "coding", "fast"],
    provider: "local",
    downloads: "1.5M",
    variants: [
      { id: "phi3:mini", params: "3.8B", size: "2.2GB" },
      { id: "phi3:medium", params: "14B", size: "8GB" },
    ],
  },

  // Embedding/RAG models
  {
    id: "nomic-embed-text",
    name: "Nomic Embed Text",
    description: "Text embeddings for RAG",
    size: "274MB",
    params: "137M",
    capabilities: ["rag"],
    provider: "local",
    downloads: "3M",
  },
  {
    id: "mxbai-embed-large",
    name: "MxBai Embed Large",
    description: "High-quality embeddings",
    size: "670MB",
    params: "335M",
    capabilities: ["rag"],
    provider: "local",
    downloads: "1M",
  },
  {
    id: "bge-m3",
    name: "BGE-M3",
    description: "Multilingual embeddings",
    size: "1.2GB",
    params: "568M",
    capabilities: ["rag", "multilingual"],
    provider: "local",
    downloads: "800K",
  },

  // Cloud models
  {
    id: "kimi-k2.6:cloud",
    name: "Kimi K2.6",
    description: "256K multimodal agentic coding model via Ollama Cloud",
    size: "Cloud",
    params: "-",
    capabilities: ["vision", "tools", "coding", "reasoning"],
    provider: "cloud",
    downloads: "78K",
  },
  {
    id: "qwen3-coder:480b-cloud",
    name: "Qwen3 Coder 480B",
    description: "480B via Ollama Cloud",
    size: "Cloud",
    params: "480B",
    capabilities: ["coding", "tools", "reasoning", "multilingual"],
    provider: "cloud",
    downloads: "500K",
  },
  {
    id: "deepseek-r1:671b",
    name: "DeepSeek R1 671B",
    description: "Full reasoning via cloud",
    size: "Cloud",
    params: "671B",
    capabilities: ["reasoning", "coding", "tools"],
    provider: "cloud",
    downloads: "200K",
  },
];

export const cloudModels: Record<
  string,
  { id: string; name: string; description: string }[]
> = {
  groq: [
    {
      id: "llama-3.3-70b-versatile",
      name: "Llama 3.3 70B",
      description: "Fast 70B model",
    },
    {
      id: "llama-3.1-8b-instant",
      name: "Llama 3.1 8B",
      description: "Ultra-fast responses",
    },
    {
      id: "mixtral-8x7b-32768",
      name: "Mixtral 8x7B",
      description: "32k context window",
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
    {
      id: "qwen3-coder:480b-cloud",
      name: "Qwen3 Coder 480B",
      description: "480B cloud - best quality",
    },
    {
      id: "qwen3-coder:30b",
      name: "Qwen3 Coder 30B",
      description: "30B coding model",
    },
    {
      id: "deepseek-r1:671b",
      name: "DeepSeek R1 671B",
      description: "Full reasoning model",
    },
    {
      id: "deepseek-r1:70b",
      name: "DeepSeek R1 70B",
      description: "Reasoning model",
    },
    {
      id: "llama3.3:70b",
      name: "Llama 3.3 70B",
      description: "Latest Llama model",
    },
    { id: "llama3.2", name: "Llama 3.2", description: "Fast Llama model" },
    { id: "qwen3:8b", name: "Qwen3 8B", description: "Balanced model" },
    { id: "mistral", name: "Mistral", description: "Mistral AI flagship" },
  ],
};

// ──────────────────────────────────────────────
// Pure utility functions
// ──────────────────────────────────────────────

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return bytes + " B";
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
  if (bytes < 1024 * 1024 * 1024)
    return (bytes / (1024 * 1024)).toFixed(1) + " MB";
  return (bytes / (1024 * 1024 * 1024)).toFixed(2) + " GB";
}

export function formatTokens(tokens: number): string {
  if (tokens >= 1000000) return (tokens / 1000000).toFixed(1) + "M";
  if (tokens >= 1000) return (tokens / 1000).toFixed(1) + "K";
  return tokens.toString();
}

export function formatDuration(ms: number): string {
  if (ms < 1000) return ms + "ms";
  return (ms / 1000).toFixed(1) + "s";
}

export function getDateRange(period: string): { start: string; end: string } {
  const end = new Date();
  const start = new Date();
  switch (period) {
    case "today":
      start.setHours(0, 0, 0, 0);
      break;
    case "week":
      start.setDate(end.getDate() - 7);
      break;
    case "month":
      start.setMonth(end.getMonth() - 1);
      break;
    default:
      start.setFullYear(end.getFullYear() - 1);
  }
  return {
    start: start.toISOString().split("T")[0],
    end: end.toISOString().split("T")[0],
  };
}

export function calculateLocalStorageUsage(models: LLMModel[]): number {
  let totalGB = 0;
  models.forEach((m) => {
    const sizeStr = m.size || "0";
    const num = parseFloat(sizeStr);
    if (sizeStr.includes("GB")) totalGB += num;
    else if (sizeStr.includes("MB")) totalGB += num / 1024;
  });
  return Math.round(totalGB * 10) / 10;
}

export function calculateLocalPowerCost(tokens: number): number {
  const kwhUsed = (tokens / 1000) * 0.0003;
  return Math.round(kwhUsed * 0.12 * 100) / 100;
}

export function toggleContextSource(
  sources: string[] | undefined,
  source: string,
): string[] {
  const current = sources || [];
  if (current.includes(source)) {
    return current.filter((s) => s !== source);
  }
  return [...current, source];
}

// SVG icon paths for providers
export const providerIconPaths: Record<string, string> = {
  ollama_local:
    "M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2zM22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z",
  ollama_cloud: "M18 10h-1.26A8 8 0 1 0 9 20h9a5 5 0 0 0 0-10z",
  groq: "M13 2L3 14h9l-1 8 10-12h-9l1-8z",
  anthropic: "M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5",
};

const defaultProviderIconPath =
  "M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73v.27a6 6 0 0 1 6 6v7h1a1 1 0 0 1 0 2H4a1 1 0 0 1 0-2h1v-7a6 6 0 0 1 6-6v-.27c-.6-.34-1-.99-1-1.73a2 2 0 0 1 2-2z";

export function getProviderIconPath(id: string): string {
  return providerIconPaths[id] || defaultProviderIconPath;
}

export function getProviderLabel(id: string): string {
  const labels: Record<string, string> = {
    ollama_local: "Local",
    ollama_cloud: "Cloud",
    groq: "Groq",
    anthropic: "Claude",
  };
  return labels[id] || id;
}

// SVG icon paths for agent categories
export const categoryIconPaths: Record<string, string> = {
  general: "M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5",
  specialist:
    "M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.77-3.77a6 6 0 0 1-7.94 7.94l-6.91 6.91a2.12 2.12 0 0 1-3-3l6.91-6.91a6 6 0 0 1 7.94-7.94l-3.76 3.76z",
  system:
    "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z",
};

const defaultCategoryIconPath =
  "M12 2a2 2 0 0 1 2 2c0 .74-.4 1.39-1 1.73v.27a6 6 0 0 1 6 6v7h1a1 1 0 0 1 0 2H4a1 1 0 0 1 0-2h1v-7a6 6 0 0 1 6-6v-.27c-.6-.34-1-.99-1-1.73a2 2 0 0 1 2-2z";

export function getCategoryIconPath(category: string): string {
  return categoryIconPaths[category] || defaultCategoryIconPath;
}

export function getCategoryLabel(category: string): string {
  const labels: Record<string, string> = {
    general: "General",
    specialist: "Specialist",
    system: "System",
  };
  return labels[category] || category;
}

export const contextSourceOptions = [
  {
    id: "documents",
    label: "Documents",
    desc: "Load content from selected context documents",
  },
  {
    id: "conversations",
    label: "Conversations",
    desc: "Include recent conversation history",
  },
  { id: "artifacts", label: "Artifacts", desc: "Include generated artifacts" },
  { id: "projects", label: "Projects", desc: "Include project details" },
  { id: "clients", label: "Clients", desc: "Include client information" },
  { id: "tasks", label: "Tasks", desc: "Include task list" },
];
