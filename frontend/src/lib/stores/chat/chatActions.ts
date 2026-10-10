/**
 * chatActions.ts — Pure utility functions and constant data for the chat page.
 * No reactive dependencies. All functions are deterministic and side-effect-free.
 */

// ─── Types ────────────────────────────────────────────────────────────────────

export type ModelCapability =
  | "vision"
  | "tools"
  | "coding"
  | "reasoning"
  | "rag"
  | "multilingual"
  | "fast";

export interface ParsedPart {
  type: "text" | "artifact";
  text?: string;
  artifact?: { title: string; type: string; content: string };
}

// ─── Constant Data ────────────────────────────────────────────────────────────

/** Capability badge colors and labels used in the model selector. */
export const capabilityInfo: Record<
  ModelCapability,
  { label: string; color: string; icon: string }
> = {
  vision: {
    label: "Vision",
    color: "bg-purple-100 text-purple-700",
    icon: "👁️",
  },
  tools: { label: "Tools", color: "bg-blue-100 text-blue-700", icon: "🔧" },
  coding: { label: "Code", color: "bg-green-100 text-green-700", icon: "💻" },
  reasoning: {
    label: "Reasoning",
    color: "bg-orange-100 text-orange-700",
    icon: "🧠",
  },
  rag: { label: "RAG", color: "bg-cyan-100 text-cyan-700", icon: "📚" },
  multilingual: {
    label: "Multi-lang",
    color: "bg-pink-100 text-pink-700",
    icon: "🌐",
  },
  fast: { label: "Fast", color: "bg-yellow-100 text-yellow-700", icon: "⚡" },
};

export interface ModelOption {
  id: string;
  name: string;
  description: string;
  type: "cloud" | "local";
  size?: string;
  capabilities?: ModelCapability[];
}

/** Cloud model definitions grouped by provider. */
export const cloudModelsMap: Record<string, ModelOption[]> = {
  groq: [
    {
      id: "llama-3.3-70b-versatile",
      name: "Llama 3.3 70B",
      description: "Fast 70B model",
      type: "cloud",
      capabilities: ["tools", "coding", "fast"],
    },
    {
      id: "llama-3.1-8b-instant",
      name: "Llama 3.1 8B",
      description: "Ultra-fast",
      type: "cloud",
      capabilities: ["fast"],
    },
    {
      id: "mixtral-8x7b-32768",
      name: "Mixtral 8x7B",
      description: "32k context",
      type: "cloud",
      capabilities: ["coding", "fast"],
    },
  ],
  anthropic: [
    {
      id: "claude-sonnet-4-20250514",
      name: "Claude Sonnet 4",
      description: "Best for most tasks",
      type: "cloud",
      capabilities: ["vision", "tools", "coding", "reasoning"],
    },
    {
      id: "claude-opus-4-1-20250805",
      name: "Claude Opus 4",
      description: "Most capable",
      type: "cloud",
      capabilities: ["vision", "tools", "coding", "reasoning"],
    },
  ],
  ollama_cloud: [
    {
      id: "kimi-k2.6:cloud",
      name: "Kimi K2.6",
      description: "256K context multimodal agentic model",
      type: "cloud",
      capabilities: ["vision", "tools", "coding", "reasoning"],
    },
    {
      id: "qwen3-coder:480b-cloud",
      name: "Qwen3 Coder 480B",
      description: "480B cloud - best quality",
      type: "cloud",
      capabilities: ["tools", "coding", "reasoning", "multilingual"],
    },
    {
      id: "qwen3-coder:30b",
      name: "Qwen3 Coder 30B",
      description: "30B coding model",
      type: "cloud",
      capabilities: ["tools", "coding", "multilingual"],
    },
    {
      id: "qwen3:4b",
      name: "Qwen3 4B",
      description: "Fast, efficient",
      type: "cloud",
      capabilities: ["fast", "multilingual"],
    },
    {
      id: "qwen3:8b",
      name: "Qwen3 8B",
      description: "Balanced performance",
      type: "cloud",
      capabilities: ["tools", "multilingual"],
    },
    {
      id: "qwen3:14b",
      name: "Qwen3 14B",
      description: "Capable mid-size",
      type: "cloud",
      capabilities: ["tools", "coding", "multilingual"],
    },
    {
      id: "qwen3:30b",
      name: "Qwen3 30B",
      description: "Large model",
      type: "cloud",
      capabilities: ["tools", "coding", "reasoning", "multilingual"],
    },
    {
      id: "qwen3:32b",
      name: "Qwen3 32B",
      description: "Large model",
      type: "cloud",
      capabilities: ["tools", "coding", "reasoning", "multilingual"],
    },
    {
      id: "llama3.3:70b",
      name: "Llama 3.3 70B",
      description: "Latest Llama model",
      type: "cloud",
      capabilities: ["tools", "coding", "reasoning"],
    },
    {
      id: "llama3.2",
      name: "Llama 3.2",
      description: "Fast Llama model",
      type: "cloud",
      capabilities: ["fast", "tools"],
    },
    {
      id: "llama3.2-vision",
      name: "Llama 3.2 Vision",
      description: "Multimodal Llama",
      type: "cloud",
      capabilities: ["vision", "tools"],
    },
    {
      id: "deepseek-r1:671b",
      name: "DeepSeek R1 671B",
      description: "Full reasoning - cloud",
      type: "cloud",
      capabilities: ["reasoning", "coding", "tools"],
    },
    {
      id: "deepseek-r1:70b",
      name: "DeepSeek R1 70B",
      description: "Reasoning model",
      type: "cloud",
      capabilities: ["reasoning", "coding"],
    },
    {
      id: "deepseek-r1:32b",
      name: "DeepSeek R1 32B",
      description: "Compact reasoning",
      type: "cloud",
      capabilities: ["reasoning", "coding", "fast"],
    },
    {
      id: "llava:34b",
      name: "LLaVA 34B",
      description: "Vision-language model",
      type: "cloud",
      capabilities: ["vision"],
    },
    {
      id: "minicpm-v",
      name: "MiniCPM-V",
      description: "Efficient vision model",
      type: "cloud",
      capabilities: ["vision", "fast"],
    },
    {
      id: "nomic-embed-text",
      name: "Nomic Embed",
      description: "Text embeddings for RAG",
      type: "cloud",
      capabilities: ["rag"],
    },
    {
      id: "mxbai-embed-large",
      name: "MxBai Embed Large",
      description: "High-quality embeddings",
      type: "cloud",
      capabilities: ["rag"],
    },
    {
      id: "mistral",
      name: "Mistral",
      description: "Mistral AI model",
      type: "cloud",
      capabilities: ["tools", "coding", "fast"],
    },
  ],
};

/** Context window limits for different models (in tokens). */
export const modelContextLimits: Record<string, number> = {
  "kimi-k2.6:cloud": 256000,
  "kimi-k2.6": 256000,
  "llama3.2": 128000,
  "llama3.2:1b": 128000,
  "llama3.2:3b": 128000,
  "llama3.1": 128000,
  "llama3.1:8b": 128000,
  "llama3.1:70b": 128000,
  llama3: 8192,
  mistral: 32768,
  "mistral:7b": 32768,
  codellama: 16384,
  "qwen2.5": 128000,
  "qwen2.5:7b": 128000,
  "qwen2.5:14b": 128000,
  "qwen2.5:32b": 128000,
  qwen3: 40960,
  "qwen3:0.6b": 32768,
  "qwen3:1.7b": 32768,
  "qwen3:4b": 32768,
  "qwen3:8b": 40960,
  "qwen3:14b": 40960,
  "qwen3:30b": 40960,
  "qwen3:32b": 40960,
  "qwen3:235b": 40960,
  "qwen3:480b": 40960,
  phi3: 128000,
  gemma2: 8192,
  "deepseek-coder": 16384,
  "deepseek-r1": 128000,
  "gpt-4": 8192,
  "gpt-4-turbo": 128000,
  "gpt-4o": 128000,
  "gpt-3.5-turbo": 16384,
  "claude-3-opus": 200000,
  "claude-3-sonnet": 200000,
  "claude-3-haiku": 200000,
  "llama-3.2-3b-preview": 128000,
  "llama-3.2-11b-vision-preview": 128000,
  "llama-3.2-90b-vision-preview": 128000,
};

/** Quick action prompts shown on the empty state. */
export const quickActions = [
  "Write a business proposal",
  "Analyze my data",
  "Plan my week",
];

/** Rotating typewriter suggestions for the greeting. */
export const greetingSuggestions = [
  "streamline your workflow",
  "automate repetitive tasks",
  "create a business proposal",
  "analyze your metrics",
  "draft a client email",
  "plan your week ahead",
  "optimize your processes",
];

// ─── Pure Functions ───────────────────────────────────────────────────────────

/**
 * Extracts `<think>` / `<thinking>` content from a message, returning
 * the thinking portion and the remaining main content separately.
 */
export function extractThinking(content: string): {
  thinking: string;
  mainContent: string;
} {
  if (!content) return { thinking: "", mainContent: "" };
  const thinkingMatch = content.match(
    /<think[a-z]*\s*>([\s\S]*?)<\/think[a-z]*\s*>/,
  );
  if (thinkingMatch) {
    const thinking = thinkingMatch[1].trim();
    const mainContent = content
      .replace(/<think[a-z]*\s*>[\s\S]*?<\/think[a-z]*\s*>/, "")
      .trim();
    return { thinking, mainContent };
  }
  return { thinking: "", mainContent: content };
}

/**
 * Renders markdown text to XSS-safe HTML using a custom lightweight renderer.
 * Strips any remaining `<think*>` tags before processing.
 */
export function renderMarkdown(text: string): string {
  if (!text) return "";

  // Strip thinking tags and their content
  text = text
    .replace(/<think[a-z]*\s*>[\s\S]*?<\/think[a-z]*\s*>/gi, "")
    .trim();
  text = text.replace(/<\/?[a-z]*think[a-z]*\s*>/gi, "");

  let html = text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(
      /```(\w+)?\n([\s\S]*?)```/g,
      '<pre class="chat-code-block"><code>$2</code></pre>',
    )
    .replace(/`([^`]+)`/g, '<code class="chat-inline-code">$1</code>')
    .replace(/\*\*([^*]+)\*\*/g, '<strong class="chat-bold">$1</strong>')
    .replace(/\*([^*]+)\*/g, '<em class="chat-italic">$1</em>')
    .replace(
      /^(\d+)\.\s+([A-Z][^\n]+)$/gm,
      '<div class="chat-section-divider"></div><h2 class="chat-section-header"><span class="chat-section-number">$1.</span> $2</h2>',
    )
    .replace(
      /^([A-Z])\.\s+(.+)$/gm,
      '<h3 class="chat-subsection-header"><span class="chat-subsection-letter">$1.</span> $2</h3>',
    )
    .replace(/^### (.+)$/gm, '<h4 class="chat-h4">$1</h4>')
    .replace(/^## (.+)$/gm, '<h3 class="chat-h3">$1</h3>')
    .replace(/^# (.+)$/gm, '<h2 class="chat-h2">$1</h2>')
    .replace(
      /^([IVX]+)\.\s+(.+)$/gm,
      '<div class="chat-section-divider"></div><h2 class="chat-section-header">$1. $2</h2>',
    )
    .replace(
      /^(Outcome|Example|Includes|Note|Important|Key|Summary):\s*/gm,
      '<p class="chat-label"><strong>$1:</strong> ',
    )
    .replace(
      /^(\d+)\.\s+\*\*(.+?)\*\*:?\s*(.*)$/gm,
      '<div class="chat-list-item"><span class="chat-list-number">$1.</span><div class="chat-list-content"><strong class="chat-bold">$2</strong>$3</div></div>',
    )
    .replace(
      /^(\d+)\.\s+(.+)$/gm,
      '<div class="chat-list-item"><span class="chat-list-number">$1.</span><span class="chat-list-content">$2</span></div>',
    )
    .replace(
      /^[\t\s]{2,}[\u2022\-\*]\s+(.+)$/gm,
      '<div class="chat-nested-bullet"><span class="chat-bullet">&bull;</span><span>$1</span></div>',
    )
    .replace(
      /^[\u2022\-\*]\s+(.+)$/gm,
      '<div class="chat-bullet-item"><span class="chat-bullet">&bull;</span><span>$1</span></div>',
    )
    .replace(/^---+$/gm, '<div class="chat-section-divider"></div>')
    .replace(
      /\[([^\]]+)\]\(([^)]+)\)/g,
      '<a href="$2" class="chat-link" target="_blank" rel="noopener">$1</a>',
    )
    .replace(/\n\n/g, '</p><p class="chat-paragraph">')
    .replace(/\n/g, "<br />");

  if (
    !html.startsWith("<h") &&
    !html.startsWith("<pre") &&
    !html.startsWith("<div")
  ) {
    html = '<p class="chat-paragraph">' + html + "</p>";
  }

  return html;
}

/** Returns the SVG path string for a given artifact type icon. */
export function getArtifactIcon(type: string): string {
  switch (type) {
    case "proposal":
      return "M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z";
    case "sop":
      return "M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4";
    case "framework":
      return "M4 5a1 1 0 011-1h14a1 1 0 011 1v2a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM4 13a1 1 0 011-1h6a1 1 0 011 1v6a1 1 0 01-1 1H5a1 1 0 01-1-1v-6zM16 13a1 1 0 011-1h2a1 1 0 011 1v6a1 1 0 01-1 1h-2a1 1 0 01-1-1v-6z";
    case "agenda":
      return "M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z";
    case "report":
      return "M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z";
    case "plan":
      return "M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2";
    default:
      return "M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z";
  }
}

/** Returns Tailwind color classes for a given artifact type. */
export function getArtifactColor(type: string): string {
  switch (type) {
    case "proposal":
      return "text-blue-500 bg-blue-50";
    case "sop":
      return "text-green-500 bg-green-50";
    case "framework":
      return "text-purple-500 bg-purple-50";
    case "agenda":
      return "text-orange-500 bg-orange-50";
    case "report":
      return "text-red-500 bg-red-50";
    case "plan":
      return "text-teal-500 bg-teal-50";
    default:
      return "text-gray-500 bg-gray-50";
  }
}

/**
 * Detects capabilities for a model based on its ID string.
 * Returns an array of capability keys.
 */
export function getModelCapabilities(modelId: string): ModelCapability[] {
  const id = modelId.toLowerCase();
  const caps: ModelCapability[] = [];

  if (
    id.includes("vision") ||
    id.includes("llava") ||
    id.includes("minicpm-v") ||
    id.includes("bakllava") ||
    id.includes("moondream") ||
    id.includes("kimi-k2.6")
  ) {
    caps.push("vision");
  }
  if (
    id.includes("code") ||
    id.includes("coder") ||
    id.includes("deepseek") ||
    id.includes("kimi-k2.6") ||
    id.includes("qwen3-coder") ||
    id.includes("starcoder") ||
    id.includes("codellama")
  ) {
    caps.push("coding");
  }
  if (
    id.includes("deepseek-r1") ||
    id.includes("kimi-k2.6") ||
    id.includes("o1") ||
    id.includes("reasoning") ||
    id.includes(":70b") ||
    id.includes(":72b") ||
    id.includes(":480b")
  ) {
    caps.push("reasoning");
  }
  if (
    id.includes("qwen") ||
    id.includes("llama3") ||
    id.includes("mistral") ||
    id.includes("claude") ||
    id.includes("gpt")
  ) {
    caps.push("tools");
  }
  if (
    id.includes("qwen") ||
    id.includes("aya") ||
    id.includes("multilingual")
  ) {
    caps.push("multilingual");
  }
  if (
    id.includes("embed") ||
    id.includes("nomic") ||
    id.includes("mxbai") ||
    id.includes("bge") ||
    id.includes("e5")
  ) {
    caps.push("rag");
  }
  if (
    id.includes(":1b") ||
    id.includes(":3b") ||
    id.includes(":4b") ||
    id.includes("instant") ||
    id.includes("mini") ||
    id.includes("tiny")
  ) {
    caps.push("fast");
  }

  return caps;
}

/** Returns a human-readable description for a model ID. */
export function getModelDescription(modelId: string, family?: string): string {
  const id = modelId.toLowerCase();
  if (id.includes("llama3.2-vision")) return "Multimodal vision model";
  if (id.includes("kimi-k2.6")) return "256K multimodal agentic coding model";
  if (id.includes("llama3.2")) return "Fast and capable, great all-rounder";
  if (id.includes("llama3.1")) return "Powerful general purpose model";
  if (id.includes("llama3.3")) return "Latest Llama, best quality";
  if (id.includes("llama")) return "Meta AI general purpose";
  if (id.includes("mistral")) return "Excellent for general tasks";
  if (id.includes("mixtral")) return "MoE model, very capable";
  if (id.includes("codellama")) return "Optimized for code tasks";
  if (id.includes("deepseek-coder") || id.includes("deepseek:coder"))
    return "Strong coding assistant";
  if (id.includes("deepseek-r1")) return "Advanced reasoning model";
  if (id.includes("deepseek")) return "Capable reasoning model";
  if (id.includes("qwen3-coder")) return "Qwen coding specialist";
  if (id.includes("qwen2.5") || id.includes("qwen2"))
    return "Strong reasoning and math";
  if (id.includes("qwen3")) return "Latest Qwen, very capable";
  if (id.includes("qwen")) return "Alibaba multilingual model";
  if (id.includes("llava")) return "Vision-language model";
  if (id.includes("minicpm-v")) return "Efficient vision model";
  if (id.includes("nomic-embed") || id.includes("mxbai-embed"))
    return "Embedding model for RAG";
  if (id.includes("phi3") || id.includes("phi-3"))
    return "Microsoft efficient model";
  if (id.includes("phi")) return "Microsoft small but capable";
  if (id.includes("gemma2") || id.includes("gemma:2"))
    return "Google efficient model";
  if (id.includes("gemma")) return "Google lightweight model";
  if (id.includes("bakllava")) return "Vision model for images";
  if (id.includes("vicuna")) return "Fine-tuned for chat";
  if (id.includes("wizard")) return "Instruction-following model";
  if (id.includes("openchat")) return "Optimized for conversation";
  if (id.includes("neural-chat")) return "Intel optimized chat";
  if (id.includes("starling")) return "Strong reasoning ability";
  if (id.includes("yi")) return "01.AI bilingual model";
  if (id.includes("command")) return "Cohere instruction model";
  if (id.includes("dolphin")) return "Uncensored assistant";
  if (id.includes("orca")) return "Reasoning focused model";
  if (id.includes("nous")) return "Research-focused model";
  if (id.includes("solar")) return "Upstage efficient model";
  if (id.includes("zephyr")) return "HuggingFace chat model";
  if (family) return family;
  return "Local AI model";
}

/** Returns a time-of-day-aware greeting string. `userName` is passed by the caller. */
export function getTimeBasedGreeting(userName: string): string {
  const hour = new Date().getHours();
  if (hour >= 0 && hour < 5) return `Up late, ${userName}?`;
  if (hour >= 5 && hour < 12) return `Good morning, ${userName}`;
  if (hour >= 12 && hour < 17) return `Good afternoon, ${userName}`;
  if (hour >= 17 && hour < 21) return `Good evening, ${userName}`;
  return `Working late, ${userName}?`;
}

/**
 * Parses `\`\`\`artifact` blocks from a message content string.
 * Returns both the cleaned content (blocks removed) and extracted artifact objects.
 * Also auto-detects document-like content and promotes it to an artifact.
 */
export function parseArtifactsFromContent(content: string): {
  cleanContent: string;
  artifacts: { title: string; type: string; content: string }[];
} {
  const artifacts: { title: string; type: string; content: string }[] = [];
  let cleanContent = content;

  // Remove orchestrator meta tags
  cleanContent = cleanContent.replace(/\[DELEGATE:\w+\]\s*/gi, "");
  cleanContent = cleanContent.replace(
    /^(Task|Context|Orchestrator context):\s*.*$/gim,
    "",
  );

  const artifactRegex = /```artifact\s*\n([\s\S]*?)\n```/g;
  let match;
  while ((match = artifactRegex.exec(content)) !== null) {
    try {
      const artifactData = JSON.parse(match[1].trim());
      if (artifactData.title && artifactData.type && artifactData.content) {
        artifacts.push({
          title: artifactData.title,
          type: artifactData.type,
          content: artifactData.content
            .replace(/\\n/g, "\n")
            .replace(/\\"/g, '"')
            .replace(/\\\\/g, "\\"),
        });
      }
    } catch {
      console.error("Failed to parse artifact JSON");
    }
  }

  // Remove artifact blocks from displayed content
  cleanContent = cleanContent
    .replace(/```artifact\s*\n[\s\S]*?\n```/g, "")
    .trim();

  // Auto-detect structured document content
  const hasHeadings = /^#{1,3}\s+.+$|^[IVX]+\.\s+.+$|^\d+\.\s+[A-Z]/m.test(
    cleanContent,
  );
  const hasMultipleSections =
    (cleanContent.match(/^(?:#{1,3}\s+|[IVX]+\.\s+|\d+\.\s+[A-Z])/gm) || [])
      .length >= 2;
  const isLongContent = cleanContent.length > 500;

  if (
    artifacts.length === 0 &&
    hasHeadings &&
    hasMultipleSections &&
    isLongContent
  ) {
    const titleMatch =
      cleanContent.match(/^#{1,3}\s+(.+)$/m) ||
      cleanContent.match(/^(?:Task:\s*)?(.+?)(?:\n|$)/);
    const title = titleMatch
      ? titleMatch[1]
          .trim()
          .replace(/^(Create|Write|Draft|Make)\s+(a\s+)?/i, "")
      : "Generated Document";

    let type = "document";
    const lowerContent = cleanContent.toLowerCase();
    if (
      lowerContent.includes("standard operating procedure") ||
      lowerContent.includes("sop")
    )
      type = "sop";
    else if (lowerContent.includes("proposal")) type = "proposal";
    else if (lowerContent.includes("framework")) type = "framework";
    else if (lowerContent.includes("plan") || lowerContent.includes("roadmap"))
      type = "plan";
    else if (
      lowerContent.includes("report") ||
      lowerContent.includes("analysis")
    )
      type = "report";

    artifacts.push({
      title: title.substring(0, 100),
      type,
      content: cleanContent,
    });
    cleanContent = "";
  }

  return { cleanContent, artifacts };
}

/** Formats a byte count as a human-readable string (B / KB / MB). */
export function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + " B";
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
  return (bytes / (1024 * 1024)).toFixed(1) + " MB";
}

/**
 * Parses a message content string into an ordered list of text and artifact parts.
 * Used by the message renderer to interleave text with artifact cards.
 */
export function parseMessageContent(content: string): ParsedPart[] {
  const parts: ParsedPart[] = [];
  const pattern = /```artifact\s*\n([\s\S]*?)\n```/g;
  let lastIndex = 0;
  let match;

  while ((match = pattern.exec(content)) !== null) {
    if (match.index > lastIndex) {
      const textBefore = content.slice(lastIndex, match.index).trim();
      if (textBefore) parts.push({ type: "text", text: textBefore });
    }

    try {
      const jsonStr = match[1].trim();
      const artifactData = JSON.parse(jsonStr);
      if (artifactData.title && artifactData.type && artifactData.content) {
        parts.push({
          type: "artifact",
          artifact: {
            title: artifactData.title,
            type: artifactData.type,
            content: artifactData.content
              .replace(/\\n/g, "\n")
              .replace(/\\"/g, '"')
              .replace(/\\\\/g, "\\"),
          },
        });
      }
    } catch {
      // Incomplete artifact JSON during streaming — skip silently
    }

    lastIndex = match.index + match[0].length;
  }

  if (lastIndex < content.length) {
    const remainingText = content.slice(lastIndex).trim();
    if (remainingText) {
      const hasArtifactStart = remainingText.includes("```artifact");
      const hasCompleteArtifactBlock = /```artifact\s*\n[\s\S]*?\n```/.test(
        remainingText,
      );
      if (hasArtifactStart && !hasCompleteArtifactBlock) {
        const beforeArtifact = remainingText.split("```artifact")[0].trim();
        if (beforeArtifact) parts.push({ type: "text", text: beforeArtifact });
      } else {
        parts.push({ type: "text", text: remainingText });
      }
    }
  }

  if (parts.length === 0) {
    if (content.includes("```artifact")) {
      const beforeArtifact = content.split("```artifact")[0].trim();
      if (beforeArtifact) return [{ type: "text", text: beforeArtifact }];
      return [];
    }
    return [{ type: "text", text: content }];
  }

  return parts;
}

/** Formats an ISO date string to a human-readable relative time. */
export function formatTime(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const diffHours = Math.floor(
    (now.getTime() - date.getTime()) / (1000 * 60 * 60),
  );
  if (diffHours < 1) return "Just now";
  if (diffHours < 24) return `${diffHours}h ago`;
  return date.toLocaleDateString();
}

/** Formats a raw token count as a compact string (e.g. 1.5K, 128K, 2M). */
export function formatTokenCount(tokens: number): string {
  if (tokens >= 1_000_000) return (tokens / 1_000_000).toFixed(1) + "M";
  if (tokens >= 1000)
    return (tokens / 1000).toFixed(tokens >= 10_000 ? 0 : 1) + "K";
  return tokens.toString();
}

/**
 * Keyword-scoring heuristic to find the best team member to assign a task to,
 * based on the task title/description and each member's role.
 */
export function findBestAssignee(
  task: { title: string; description?: string },
  availableTeamMembers: { id: string; name: string; role: string }[],
): { id: string; name: string; role: string } | undefined {
  const combined = (task.title + " " + (task.description || "")).toLowerCase();

  const roleKeywords: Record<string, string[]> = {
    developer: [
      "code",
      "implement",
      "build",
      "develop",
      "api",
      "frontend",
      "backend",
      "database",
      "bug",
      "fix",
      "feature",
      "technical",
      "integration",
    ],
    designer: [
      "design",
      "ui",
      "ux",
      "mockup",
      "wireframe",
      "visual",
      "layout",
      "style",
      "brand",
    ],
    "project manager": [
      "coordinate",
      "schedule",
      "timeline",
      "milestone",
      "meeting",
      "stakeholder",
      "plan",
      "track",
      "report",
    ],
    ceo: [
      "strategy",
      "vision",
      "decision",
      "executive",
      "leadership",
      "partnership",
      "investor",
    ],
    cto: [
      "architecture",
      "infrastructure",
      "security",
      "scalability",
      "technical strategy",
      "technology",
    ],
    marketing: [
      "marketing",
      "campaign",
      "content",
      "social",
      "seo",
      "advertising",
      "promotion",
      "brand",
    ],
    sales: [
      "sales",
      "client",
      "customer",
      "deal",
      "proposal",
      "pitch",
      "revenue",
      "lead",
    ],
    operations: [
      "operations",
      "process",
      "workflow",
      "efficiency",
      "sop",
      "documentation",
    ],
    qa: ["test", "quality", "qa", "bug", "verification", "validation"],
    devops: [
      "deploy",
      "ci/cd",
      "infrastructure",
      "monitoring",
      "server",
      "cloud",
      "kubernetes",
      "docker",
    ],
  };

  let bestMatch: {
    member: (typeof availableTeamMembers)[0];
    score: number;
  } | null = null;

  for (const member of availableTeamMembers) {
    const memberRole = member.role.toLowerCase();
    let score = 0;

    for (const [role, keywords] of Object.entries(roleKeywords)) {
      if (memberRole.includes(role)) {
        for (const keyword of keywords) {
          if (combined.includes(keyword)) score += 10;
        }
      }
    }

    if (combined.includes(memberRole)) score += 20;

    if (score > 0 && (!bestMatch || score > bestMatch.score)) {
      bestMatch = { member, score };
    }
  }

  return bestMatch?.member;
}

/** Gets the context window limit for a model ID, falling back to 8192. */
export function getContextLimit(selectedModel: string): number {
  if (modelContextLimits[selectedModel])
    return modelContextLimits[selectedModel];
  const baseModel = selectedModel.split(":")[0];
  if (modelContextLimits[baseModel]) return modelContextLimits[baseModel];
  for (const [key, limit] of Object.entries(modelContextLimits)) {
    if (selectedModel.toLowerCase().includes(key.toLowerCase())) return limit;
  }
  return 8192;
}
