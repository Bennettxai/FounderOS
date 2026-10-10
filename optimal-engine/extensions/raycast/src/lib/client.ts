import type {
  ApiResult,
  MemoryCreatedResponse,
  MemoryInput,
  RagEnvelope,
  RagRequest,
  SearchResponse,
  WorkspacesResponse,
} from "./types";
import { getPreferences } from "./preferences";

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

function buildHeaders(): Record<string, string> {
  const { apiKey } = getPreferences();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "application/json",
  };
  if (apiKey) {
    headers["Authorization"] = `Bearer ${apiKey}`;
  }
  return headers;
}

async function request<T>(
  path: string,
  init: RequestInit = {},
): Promise<ApiResult<T>> {
  const { engineUrl } = getPreferences();
  const url = `${engineUrl}${path}`;

  let response: Response;
  try {
    response = await fetch(url, {
      ...init,
      headers: {
        ...buildHeaders(),
        ...(init.headers as Record<string, string> | undefined),
      },
    });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return { ok: false, error: `Network error: ${message}`, status: 0 };
  }

  if (!response.ok) {
    let errorText = response.statusText;
    try {
      const body = (await response.json()) as {
        error?: string;
        message?: string;
      };
      errorText = body.error ?? body.message ?? errorText;
    } catch {
      // ignore parse errors — keep statusText
    }
    return { ok: false, error: errorText, status: response.status };
  }

  try {
    const data = (await response.json()) as T;
    return { ok: true, data };
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return {
      ok: false,
      error: `Failed to parse response: ${message}`,
      status: response.status,
    };
  }
}

// ---------------------------------------------------------------------------
// Public API surface
// ---------------------------------------------------------------------------

/**
 * GET /api/search?q=&workspace=&limit=
 */
export async function searchMemory(
  query: string,
  workspace: string,
  limit = 20,
): Promise<ApiResult<SearchResponse>> {
  const params = new URLSearchParams({
    q: query,
    workspace,
    limit: String(limit),
  });
  return request<SearchResponse>(`/api/search?${params.toString()}`);
}

/**
 * POST /api/memory
 */
export async function addMemory(
  input: MemoryInput,
): Promise<ApiResult<MemoryCreatedResponse>> {
  return request<MemoryCreatedResponse>("/api/memory", {
    method: "POST",
    body: JSON.stringify(input),
  });
}

/**
 * POST /api/rag
 */
export async function askEngine(
  req: RagRequest,
): Promise<ApiResult<RagEnvelope>> {
  return request<RagEnvelope>("/api/rag", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

/**
 * GET /api/workspaces?tenant=
 */
export async function fetchWorkspaces(
  tenant?: string,
): Promise<ApiResult<WorkspacesResponse>> {
  const params = new URLSearchParams();
  if (tenant) params.set("tenant", tenant);
  const qs = params.toString();
  return request<WorkspacesResponse>(`/api/workspaces${qs ? `?${qs}` : ""}`);
}
