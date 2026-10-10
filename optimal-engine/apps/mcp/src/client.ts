/**
 * Thin HTTP wrapper around the Optimal Engine REST API.
 * All tools use these helpers — nothing calls fetch directly.
 */

import { config } from "./config.js";

type HttpMethod = "GET" | "POST";

interface RequestOptions {
  method?: HttpMethod;
  params?: Record<string, string | number | boolean | undefined>;
  body?: unknown;
}

export class EngineError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message);
    this.name = "EngineError";
  }
}

function buildHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "application/json",
  };
  if (config.apiKey) {
    headers["Authorization"] = `Bearer ${config.apiKey}`;
  }
  return headers;
}

function buildUrl(
  path: string,
  params?: Record<string, string | number | boolean | undefined>,
): string {
  const url = new URL(`${config.engineUrl}${path}`);
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== null) {
        url.searchParams.set(key, String(value));
      }
    }
  }
  return url.toString();
}

async function request<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { method = "GET", params, body } = options;
  const url = buildUrl(path, params);

  const response = await fetch(url, {
    method,
    headers: buildHeaders(),
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!response.ok) {
    let message: string;
    try {
      const err = (await response.json()) as {
        error?: string;
        message?: string;
      };
      message = err.error ?? err.message ?? response.statusText;
    } catch {
      message = response.statusText;
    }
    throw new EngineError(response.status, message);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

// ---------------------------------------------------------------------------
// Typed API surface
// ---------------------------------------------------------------------------

export interface RagResponse {
  source: string;
  envelope: unknown;
  trace: unknown;
}

export interface SearchResult {
  query: string;
  results: unknown[];
}

export interface GrepResult {
  query: string;
  workspace_id: string;
  results: unknown[];
}

export interface ProfileResponse {
  [key: string]: unknown;
}

export interface MemoryResponse {
  [key: string]: unknown;
}

export interface WikiPage {
  [key: string]: unknown;
}

export interface Workspace {
  [key: string]: unknown;
}

export interface WorkspaceList {
  workspaces?: Workspace[];
  [key: string]: unknown;
}

export interface RecallResponse {
  [key: string]: unknown;
}

export interface RenderContextResponse {
  spec: {
    name: string;
    title: string;
    audience: string;
    genre: string;
    bandwidth: string;
    constraints: Record<string, string>;
    html_hints: Record<string, unknown>;
    modalities: string[];
  };
  context: {
    body: string;
    format: string;
    sources: string[];
    warnings: string[];
  };
  prompt_guidance: string;
  trace: unknown;
}

export interface RenderSpecsResponse {
  specs: Array<{
    name: string;
    title: string;
    audience: string;
    genre: string;
    bandwidth: string;
    constraints: Record<string, string>;
    html_hints: Record<string, unknown>;
    modalities: string[];
  }>;
}

// ---------------------------------------------------------------------------
// API calls — one function per engine endpoint
// ---------------------------------------------------------------------------

export const engine = {
  ask(body: {
    query: string;
    workspace: string;
    audience?: string;
  }): Promise<RagResponse> {
    return request<RagResponse>("/api/rag", { method: "POST", body });
  },

  search(params: {
    q: string;
    workspace: string;
    limit?: number;
  }): Promise<SearchResult> {
    return request<SearchResult>("/api/search", { params });
  },

  grep(params: {
    q: string;
    workspace: string;
    intent?: string;
    scale?: string;
  }): Promise<GrepResult> {
    return request<GrepResult>("/api/grep", { params });
  },

  profile(params: {
    workspace: string;
    audience?: string;
    bandwidth?: string;
  }): Promise<ProfileResponse> {
    return request<ProfileResponse>("/api/profile", { params });
  },

  addMemory(body: {
    content: string;
    workspace: string;
    is_static?: boolean;
    audience?: string;
    citation_uri?: string;
    source_chunk_id?: string;
  }): Promise<MemoryResponse> {
    return request<MemoryResponse>("/api/memory", { method: "POST", body });
  },

  forgetMemory(
    id: string,
    body: { reason?: string; forget_after?: string },
  ): Promise<void> {
    return request<void>(`/api/memory/${id}/forget`, { method: "POST", body });
  },

  recall(
    action: "actions" | "who" | "when" | "where" | "owns",
    params: {
      topic?: string;
      workspace: string;
      event?: string;
      thing?: string;
      actor?: string;
    },
  ): Promise<RecallResponse> {
    return request<RecallResponse>(`/api/recall/${action}`, { params });
  },

  wikiGet(
    slug: string,
    params: { workspace: string; audience?: string },
  ): Promise<WikiPage> {
    return request<WikiPage>(`/api/wiki/${encodeURIComponent(slug)}`, {
      params,
    });
  },

  workspaces(params?: { tenant?: string }): Promise<WorkspaceList> {
    return request<WorkspaceList>("/api/workspaces", { params });
  },

  renderContext(body: {
    source: string;
    spec: string;
    workspace: string;
  }): Promise<RenderContextResponse> {
    return request<RenderContextResponse>("/api/render/context", {
      method: "POST",
      body,
    });
  },

  renderSpecs(params: { workspace: string }): Promise<RenderSpecsResponse> {
    return request<RenderSpecsResponse>("/api/render/specs", { params });
  },
};
