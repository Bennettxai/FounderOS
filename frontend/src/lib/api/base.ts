import {
  API_VERSION,
  LOCAL_BACKEND_URL,
  LOCAL_OSA_URL,
  PRODUCTION_BACKEND_URL,
  getApiBaseUrl as resolveApiBaseUrl,
  getBackendUrl as resolveBackendUrl,
} from "$lib/config/runtime";
import { mirrorToEngine } from "$lib/optimal-engine/connect";

// Writes to these surfaces carry knowledge, user intent, narrative work, or
// source context that should be remembered by the workspace's Optimal Engine.
// Relational app state still belongs in BusinessOS, but memory-bearing writes
// must be mirrored so search and recall work later.
const MIRRORED_CONTENT_ROOTS = new Set([
  "apps",
  "calendar",
  "campaigns",
  "clients",
  "content",
  "contexts",
  "crm",
  "daily",
  "deals",
  "deliverables",
  "inbox",
  "knowledge",
  "memories",
  "offers",
  "personas",
  "projects",
  "relationships",
  "rhythm",
  "sites",
  "tasks",
  "voice-notes",
  "wiki",
]);

function endpointSegments(endpoint: string): string[] {
  return endpoint
    .replace(/^\//, "")
    .split("?")[0]
    .split("/")
    .filter(Boolean);
}

function shouldMirrorContentWrite(endpoint: string): boolean {
  const segments = endpointSegments(endpoint);
  if (!segments.length) return false;
  if (segments[0] === "modules" && segments[2] === "records") return true;
  if (segments[0] === "dashboard" && segments[1] === "tasks") return true;
  if (segments[0] === "workspaces" && segments[2] === "memories") return true;
  if (segments[0] === "workspaces" && segments[2] === "knowledge") return true;
  return MIRRORED_CONTENT_ROOTS.has(segments[0]);
}

function moduleFromEndpoint(endpoint: string): string {
  const segments = endpointSegments(endpoint);
  if (segments[0] === "modules" && segments[2] === "records") {
    return `module:${segments[1] || "unknown"}`;
  }
  if (segments[0] === "dashboard" && segments[1] === "tasks") return "tasks";
  if (segments[0] === "workspaces" && segments[2] === "memories") {
    return "workspace-memory";
  }
  if (segments[0] === "workspaces" && segments[2] === "knowledge") {
    return "workspace-knowledge";
  }
  return segments[0] || "content";
}

function extractContent(obj: unknown): string {
  if (!obj || typeof obj !== "object") return "";
  const o = obj as Record<string, unknown>;
  const inner = (o.context ?? o.data) as Record<string, unknown> | undefined;
  const direct = [
    o.content,
    o.body,
    o.description,
    o.summary,
    o.notes,
    o.title,
    o.name,
    o.url,
  ]
    .filter((v) => typeof v === "string" && v.trim())
    .join(" ");
  const fromInner =
    inner && typeof inner === "object" && !Array.isArray(inner)
      ? Object.values(inner)
          .filter((v) => typeof v === "string" && v.trim())
          .join(" ")
      : "";
  return `${direct} ${fromInner}`.trim();
}

function extractId(data: unknown, body: unknown): string {
  const d = data as Record<string, unknown> | null;
  const inner = d?.context as Record<string, unknown> | undefined;
  return String(
    d?.id ?? inner?.id ?? (body as Record<string, unknown>)?.id ?? "",
  );
}

// Fire-and-forget mirror of a content write into the local engine. Never throws.
function maybeMirrorContentWrite(
  method: string,
  endpoint: string,
  body: unknown,
  data: unknown,
): void {
  try {
    if (method.toUpperCase() === "DELETE") return;
    if (!shouldMirrorContentWrite(endpoint)) return;
    if (typeof localStorage === "undefined") return;
    const workspaceId = localStorage.getItem("businessos_current_workspace_id");
    if (!workspaceId) return;
    const content = extractContent(data) || extractContent(body);
    if (!content) return;
    const id = extractId(data, body);
    const mod = moduleFromEndpoint(endpoint);
    mirrorToEngine(workspaceId, {
      content,
      citation: `businessos://${mod}/${id}`,
      metadata: { businessos_module: mod, businessos_id: id },
    });
  } catch {
    /* mirroring must never affect the request */
  }
}

export {
  API_VERSION,
  LOCAL_BACKEND_URL,
  LOCAL_OSA_URL,
  PRODUCTION_BACKEND_URL,
};

export const getApiBaseUrl = () => resolveApiBaseUrl(API_VERSION);
export const API_BASE = getApiBaseUrl();

let csrfTokenCache: string | null = null;
let csrfInitPromise: Promise<void> | null = null;

export function getActiveWorkspaceHeaders(): Record<string, string> {
  if (typeof localStorage === "undefined") return {};
  const workspaceId = localStorage.getItem("businessos_current_workspace_id");
  return workspaceId ? { "X-Workspace-ID": workspaceId } : {};
}

// Get CSRF token from cookie
export function getCSRFToken(): string | null {
  if (typeof document === "undefined") return csrfTokenCache;

  const cookies = document.cookie.split(";");
  for (const cookie of cookies) {
    const trimmed = cookie.trim();
    const eqIndex = trimmed.indexOf("=");
    if (eqIndex === -1) continue;

    const name = trimmed.substring(0, eqIndex);
    const value = trimmed.substring(eqIndex + 1); // Get everything after first =

    if (name === "csrf_token") {
      return value;
    }
  }
  // Packaged Electron is served from app://localhost while its API lives on
  // app.businessos.dev. The backend cookie is deliberately scoped to the API
  // host, so the renderer cannot read it through document.cookie. Cache the
  // token returned by the CSRF endpoint for the matching request header.
  return csrfTokenCache;
}

// Initialize CSRF token by calling the backend endpoint
// This should be called before any state-changing requests (POST, PUT, DELETE)
export async function initCSRF(): Promise<void> {
  if (typeof window === "undefined") {
    return;
  }

  const readableToken = getCSRFToken();
  if (readableToken) return;
  if (csrfInitPromise) return csrfInitPromise;

  csrfInitPromise = (async () => {
    try {
      // getApiBaseUrl handles all three surfaces correctly: relative URLs for
      // web, local OSA for local mode, and the cloud API for packaged Electron.
      const response = await fetch(`${getApiBaseUrl()}/auth/csrf`, {
        method: "GET",
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        if (typeof data?.csrf_token === "string" && data.csrf_token) {
          csrfTokenCache = data.csrf_token;
        }
      }
    } catch {
      // Callers surface the protected request failure with the backend's error.
    } finally {
      csrfInitPromise = null;
    }
  })();

  return csrfInitPromise;
}

// Add CSRF token to headers for state-changing requests
function addCSRFToken(
  method: string,
  headers: Record<string, string>,
): Record<string, string> {
  const stateChangingMethods = ["POST", "PUT", "PATCH", "DELETE"];
  if (stateChangingMethods.includes(method.toUpperCase())) {
    const csrfToken = getCSRFToken();
    if (csrfToken) {
      headers["X-CSRF-Token"] = csrfToken;
    }
  }
  return headers;
}

/**
 * Get the backend server base URL (without /api suffix)
 * Use this for image URLs and other non-API resources
 */
export const getBackendUrl = () => resolveBackendUrl();

export interface RequestOptions {
  method?: string;
  body?: unknown;
  headers?: Record<string, string>;
  timeout?: number; // Timeout in milliseconds
  skipCache?: boolean; // Bypass the GET response cache
  skipAuthRedirect?: boolean; // Don't auto-redirect to /login on 401
}

// ============================================================================
// Pagination Types
// ============================================================================

export interface PaginationParams {
  page?: number;
  page_size?: number;
}

export interface PaginationMeta {
  page: number;
  page_size: number;
  total_items: number;
  total_pages: number;
  has_more: boolean;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: PaginationMeta;
}

/**
 * Build a query string from pagination params, appending to an existing endpoint.
 * Skips undefined values.
 */
export function buildPaginatedEndpoint(
  endpoint: string,
  params: PaginationParams,
): string {
  const qs = new URLSearchParams();
  if (params.page !== undefined) qs.set("page", String(params.page));
  if (params.page_size !== undefined)
    qs.set("page_size", String(params.page_size));
  const queryString = qs.toString();
  return queryString ? `${endpoint}?${queryString}` : endpoint;
}

/**
 * Make a paginated GET request.
 *
 * Handles both paginated (new backend format) and non-paginated (legacy) responses
 * for backward compatibility. When the response is a plain array or an object
 * without a `pagination` field, it wraps it into a PaginatedResponse so callers
 * always receive the same shape.
 *
 * @param endpoint - API path (e.g. "/osa/module-instances")
 * @param params   - Pagination params (page, page_size)
 * @param dataKey  - Key in non-paginated responses that holds the array (e.g. "apps")
 */
export async function requestPaginated<T>(
  endpoint: string,
  params: PaginationParams = {},
  dataKey?: string,
): Promise<PaginatedResponse<T>> {
  const paginatedEndpoint = buildPaginatedEndpoint(endpoint, params);

  // Use skipCache:true for paginated requests so page changes always hit the
  // network rather than returning stale cached results.
  const raw = await request<unknown>(paginatedEndpoint, { skipCache: true });

  // New paginated format: { data: T[], pagination: { ... } }
  if (
    raw !== null &&
    typeof raw === "object" &&
    !Array.isArray(raw) &&
    "pagination" in raw &&
    "data" in raw
  ) {
    return raw as PaginatedResponse<T>;
  }

  // Legacy array format: T[]
  if (Array.isArray(raw)) {
    return {
      data: raw as T[],
      pagination: {
        page: 1,
        page_size: raw.length,
        total_items: raw.length,
        total_pages: 1,
        has_more: false,
      },
    };
  }

  // Legacy object format: { [dataKey]: T[], ... }
  if (
    raw !== null &&
    typeof raw === "object" &&
    dataKey &&
    dataKey in (raw as Record<string, unknown>)
  ) {
    const items = ((raw as Record<string, unknown>)[dataKey] as T[]) ?? [];
    return {
      data: items,
      pagination: {
        page: 1,
        page_size: items.length,
        total_items: items.length,
        total_pages: 1,
        has_more: false,
      },
    };
  }

  // Fallback: treat entire response as a single-page result
  return {
    data: [],
    pagination: {
      page: 1,
      page_size: 0,
      total_items: 0,
      total_pages: 0,
      has_more: false,
    },
  };
}

// ============================================================================
// GET Request Cache — prevents re-fetching the same data during tab switches
// ============================================================================
const GET_CACHE_TTL_MS = 30_000; // 30 seconds
const getCache = new Map<string, { data: unknown; expiry: number }>();
const inflightRequests = new Map<string, Promise<unknown>>();

function getCacheKey(
  endpoint: string,
  headers: Record<string, string> = {},
): string {
  const workspaceId =
    headers["X-Workspace-ID"] ??
    (typeof localStorage !== "undefined"
      ? localStorage.getItem("businessos_current_workspace_id")
      : null) ??
    "global";
  return `${workspaceId}:${endpoint}`;
}

/**
 * Invalidate all cached GET responses (call after mutations)
 */
export function invalidateApiCache(): void {
  getCache.clear();
}

/**
 * Invalidate cached GET responses matching a prefix
 */
export function invalidateApiCacheByPrefix(prefix: string): void {
  for (const key of getCache.keys()) {
    if (key.startsWith(prefix)) {
      getCache.delete(key);
    }
  }
}

function requestErrorMessage(payload: unknown): string {
  if (typeof payload === "string" && payload.trim()) return payload;
  if (!payload || typeof payload !== "object") return "Request failed";

  const error = payload as Record<string, unknown>;
  for (const key of ["detail", "message", "error"]) {
    const value = error[key];
    if (typeof value === "string" && value.trim()) return value;
    if (value && typeof value === "object") {
      const nested = value as Record<string, unknown>;
      if (typeof nested.message === "string" && nested.message.trim()) {
        return nested.message;
      }
      if (typeof nested.detail === "string" && nested.detail.trim()) {
        return nested.detail;
      }
    }
  }

  return "Request failed";
}

export async function request<T>(
  endpoint: string,
  options: RequestOptions = {},
): Promise<T> {
  const {
    method = "GET",
    body,
    headers = {},
    timeout = 10_000,
    skipCache = false,
    skipAuthRedirect = false,
  } = options;
  const cacheKey = getCacheKey(endpoint, headers);

  // --- GET cache: return cached data if fresh ---
  const isGet = method.toUpperCase() === "GET";
  if (isGet && !skipCache) {
    const cached = getCache.get(cacheKey);
    if (cached && Date.now() < cached.expiry) {
      return cached.data as T;
    }

    // Deduplicate in-flight GET requests to the same endpoint
    const inflight = inflightRequests.get(cacheKey);
    if (inflight) {
      return inflight as Promise<T>;
    }
  }

  if (body && !headers["Content-Type"]) {
    headers["Content-Type"] = "application/json";
  }

  // Add CSRF token for state-changing requests
  const finalHeaders = addCSRFToken(method, headers);

  // BusinessOSSync: scope every request to the active workspace so the backend
  // returns workspace-shared data and stamps new docs with it. Read straight
  // from localStorage to avoid a circular import with the workspaces store.
  if (typeof localStorage !== "undefined") {
    const workspaceId = localStorage.getItem("businessos_current_workspace_id");
    if (workspaceId && !finalHeaders["X-Workspace-ID"]) {
      finalHeaders["X-Workspace-ID"] = workspaceId;
    }
  }

  const baseUrl = getApiBaseUrl();

  // Create abort controller for timeout
  const controller = new AbortController();
  let timeoutId: NodeJS.Timeout | number | undefined;

  if (timeout && timeout > 0) {
    timeoutId = setTimeout(() => controller.abort(), timeout);
  }

  const doFetch = async (): Promise<T> => {
    try {
      const response = await fetch(`${baseUrl}${endpoint}`, {
        method,
        headers: finalHeaders,
        credentials: "include",
        body: body ? JSON.stringify(body) : undefined,
        signal: controller.signal,
      });

      if (!response.ok) {
        // Handle 401 for non-auth endpoints: session expired, redirect to login
        if (
          response.status === 401 &&
          !endpoint.includes("/auth/") &&
          !skipAuthRedirect
        ) {
          const { clearSession } = await import("$lib/auth-client");
          const { goto } = await import("$app/navigation");
          clearSession();
          goto("/login");
          throw new Error("Session expired");
        }

        const error = await response
          .json()
          .catch(() => ({ detail: "Request failed" }));
        const errorMessage = requestErrorMessage(error);
        throw new Error(`${errorMessage} (HTTP ${response.status})`);
      }

      const data: T = await response.json();

      // Cache successful GET responses
      if (isGet && !skipCache) {
        getCache.set(cacheKey, {
          data,
          expiry: Date.now() + GET_CACHE_TTL_MS,
        });
      }

      // Invalidate cache on mutations so subsequent GETs get fresh data
      if (!isGet) {
        invalidateApiCache();
        // Mirror content writes into the active workspace's local engine
        // (desktop + localhost engines only). Fire-and-forget.
        maybeMirrorContentWrite(method, endpoint, body, data);
      }

      return data;
    } catch (error) {
      if (error instanceof Error && error.name === "AbortError") {
        throw new Error(`Request timeout after ${timeout}ms`);
      }
      throw error;
    } finally {
      if (timeoutId !== undefined) {
        clearTimeout(timeoutId as number);
      }
      if (isGet && !skipCache) {
        inflightRequests.delete(cacheKey);
      }
    }
  };

  // Deduplicate in-flight GET requests
  if (isGet && !skipCache) {
    const promise = doFetch();
    inflightRequests.set(cacheKey, promise);
    return promise;
  }

  return doFetch();
}

// For raw response access (like original apiClient)
export const raw = {
  async get(endpoint: string): Promise<Response> {
    return fetch(`${getApiBaseUrl()}${endpoint}`, {
      method: "GET",
      credentials: "include",
    });
  },
  async post(endpoint: string, body?: unknown): Promise<Response> {
    const headers = addCSRFToken("POST", {
      "Content-Type": "application/json",
    });
    return fetch(`${getApiBaseUrl()}${endpoint}`, {
      method: "POST",
      headers,
      credentials: "include",
      body: body ? JSON.stringify(body) : undefined,
    });
  },
  async postFormData(endpoint: string, formData: FormData): Promise<Response> {
    const headers = addCSRFToken("POST", {});
    return fetch(`${getApiBaseUrl()}${endpoint}`, {
      method: "POST",
      headers,
      credentials: "include",
      body: formData,
    });
  },
  async put(endpoint: string, body?: unknown): Promise<Response> {
    const headers = addCSRFToken("PUT", { "Content-Type": "application/json" });
    return fetch(`${getApiBaseUrl()}${endpoint}`, {
      method: "PUT",
      headers,
      credentials: "include",
      body: body ? JSON.stringify(body) : undefined,
    });
  },
  async delete(endpoint: string): Promise<Response> {
    const headers = addCSRFToken("DELETE", {});
    return fetch(`${getApiBaseUrl()}${endpoint}`, {
      method: "DELETE",
      headers,
      credentials: "include",
    });
  },
};
