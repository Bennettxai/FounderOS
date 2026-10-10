/**
 * Client for the Optimal Engine HTTP API.
 *
 * Endpoints implemented:
 *   GET  /api/status             — engine readiness
 *   GET  /api/metrics            — counters + histograms
 *   GET  /api/search?q=…         — hybrid search hits
 *   POST /api/rag                — wiki-first retrieval envelope
 *   GET  /api/wiki               — list wiki pages
 *   GET  /api/wiki/:slug         — render a single page
 *
 * In dev the Vite proxy forwards `/api` → http://127.0.0.1:4200.
 * In a Tauri build we use `VITE_ENGINE_URL` (defaults to the same host).
 */

const BASE = import.meta.env.VITE_ENGINE_URL ?? "";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "content-type": "application/json" },
    ...init,
  });

  if (!res.ok) {
    throw new Error(`HTTP ${res.status}: ${await res.text()}`);
  }

  return (await res.json()) as T;
}

export interface StatusPayload {
  status: "up" | "degraded" | "down";
  "ok?": boolean;
  checks: Record<string, string>;
  degraded: string[];
}

export interface RagEnvelope {
  body: string;
  format: "plain" | "markdown" | "claude" | "openai";
  sources: string[];
  warnings: string[];
}

export interface RagResult {
  source: "wiki" | "chunks" | "empty";
  envelope: RagEnvelope;
  // Elixir emits keys verbatim including `?` — these match the wire shape.
  trace: {
    "wiki_hit?": boolean;
    n_candidates: number;
    n_delivered: number;
    "truncated?": boolean;
    elapsed_ms: number;
    intent?: unknown;
  };
}

export interface WikiPageSummary {
  slug: string;
  audience: string;
  version: number;
  last_curated: string | null;
  curated_by: string | null;
  size_bytes: number;
}

export async function status(): Promise<StatusPayload> {
  return request<StatusPayload>("/api/status");
}

export async function search(
  q: string,
  limit = 10,
  workspace?: string | null,
): Promise<{ query: string; results: unknown[] }> {
  const params = new URLSearchParams({ q, limit: String(limit) });
  if (workspace) params.set("workspace", workspace);
  return request(`/api/search?${params}`);
}

export async function ask(
  query: string,
  opts: {
    audience?: string;
    format?: string;
    bandwidth?: string;
    workspace?: string | null;
  } = {},
): Promise<RagResult> {
  return request<RagResult>("/api/rag", {
    method: "POST",
    body: JSON.stringify({
      query,
      audience: opts.audience ?? "default",
      format: opts.format ?? "markdown",
      bandwidth: opts.bandwidth ?? "medium",
      ...(opts.workspace ? { workspace: opts.workspace } : {}),
    }),
  });
}

export async function listWiki(
  opts: { tenant?: string; workspace?: string } = {},
): Promise<{ pages: WikiPageSummary[]; workspace_id?: string }> {
  const params = new URLSearchParams({
    tenant: opts.tenant ?? "default",
    workspace: opts.workspace ?? "default",
  });
  return request(`/api/wiki?${params}`);
}

export async function getWiki(
  slug: string,
  opts: {
    tenant?: string;
    workspace?: string;
    audience?: string;
    format?: string;
  } = {},
): Promise<{
  slug: string;
  body: string;
  warnings: string[];
  workspace_id?: string;
}> {
  const params = new URLSearchParams({
    tenant: opts.tenant ?? "default",
    workspace: opts.workspace ?? "default",
    audience: opts.audience ?? "default",
    format: opts.format ?? "markdown",
  });

  return request(`/api/wiki/${encodeURIComponent(slug)}?${params}`);
}

// ── Organizations + Workspaces (Phase 1.5) ──────────────────────────────

export interface Organization {
  id: string;
  name: string;
  plan: string;
  region: string | null;
  created_at: string | null;
}

export interface Workspace {
  id: string;
  tenant_id: string;
  slug: string;
  name: string;
  description: string | null;
  status: "active" | "archived";
  created_at: string | null;
  archived_at: string | null;
  metadata: Record<string, unknown>;
}

export async function listOrganizations(): Promise<{
  organizations: Organization[];
}> {
  return request("/api/organizations");
}

export async function listWorkspaces(
  tenant = "default",
  status: "active" | "archived" | "all" = "active",
): Promise<{ tenant_id: string; workspaces: Workspace[] }> {
  const params = new URLSearchParams({ tenant, status });
  return request(`/api/workspaces?${params}`);
}

export async function getWorkspace(id: string): Promise<Workspace> {
  return request(`/api/workspaces/${encodeURIComponent(id)}`);
}

export async function createWorkspace(attrs: {
  slug: string;
  name: string;
  description?: string;
  tenant?: string;
}): Promise<Workspace> {
  return request("/api/workspaces", {
    method: "POST",
    body: JSON.stringify(attrs),
  });
}

export async function updateWorkspace(
  id: string,
  attrs: { name?: string; description?: string },
): Promise<Workspace> {
  return request(`/api/workspaces/${encodeURIComponent(id)}`, {
    method: "PATCH",
    body: JSON.stringify(attrs),
  });
}

export async function archiveWorkspace(id: string): Promise<void> {
  await fetch(`/api/workspaces/${encodeURIComponent(id)}/archive`, {
    method: "POST",
  });
}

// ── Subscriptions + proactive surfacing (Phase 15) ──────────────────────

export type SubscriptionScope = "workspace" | "node" | "topic" | "audience";

export type MemoryCategory =
  | "recent_actions"
  | "autobiographical_past"
  | "contacts"
  | "schedules"
  | "ownership"
  | "open_tasks"
  | "tip_of_tongue"
  | "professional_knowledge"
  | "file_locations"
  | "procedures"
  | "event_locations"
  | "factual"
  | "unassigned";

export interface Subscription {
  id: string;
  tenant_id: string;
  workspace_id: string;
  principal_id: string | null;
  scope: SubscriptionScope;
  scope_value: string | null;
  categories: MemoryCategory[];
  activity: string | null;
  status: "active" | "paused";
  created_at: string | null;
}

export interface SurfaceEvent {
  subscription_id: string;
  workspace_id: string;
  trigger: "wiki_updated" | "chunk_indexed" | "test_push";
  envelope: {
    slug: string;
    kind: "wiki_page" | "signal" | "other";
    audience: string;
  };
  category: MemoryCategory;
  score: number;
  pushed_at: string;
}

export async function listSubscriptions(
  workspace = "default",
): Promise<{ workspace_id: string; subscriptions: Subscription[] }> {
  return request(
    `/api/subscriptions?workspace=${encodeURIComponent(workspace)}`,
  );
}

export async function createSubscription(attrs: {
  workspace?: string;
  scope?: SubscriptionScope;
  scope_value?: string | null;
  categories?: MemoryCategory[];
  principal_id?: string | null;
  activity?: string | null;
}): Promise<Subscription> {
  return request("/api/subscriptions", {
    method: "POST",
    body: JSON.stringify(attrs),
  });
}

export async function deleteSubscription(id: string): Promise<void> {
  await fetch(`/api/subscriptions/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}

export async function pauseSubscription(id: string): Promise<void> {
  await fetch(`/api/subscriptions/${encodeURIComponent(id)}/pause`, {
    method: "POST",
  });
}

export async function resumeSubscription(id: string): Promise<void> {
  await fetch(`/api/subscriptions/${encodeURIComponent(id)}/resume`, {
    method: "POST",
  });
}

export async function testSurface(
  subscriptionId: string,
  slug: string,
): Promise<void> {
  await fetch("/api/surface/test", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ subscription: subscriptionId, slug }),
  });
}

/**
 * Opens an SSE connection to /api/surface/stream for the given subscription.
 * Returns the EventSource so the caller can close it. The handler is invoked
 * for each surface event.
 */
export function openSurfaceStream(
  subscriptionId: string,
  onEvent: (e: SurfaceEvent) => void,
): EventSource {
  const url = `/api/surface/stream?subscription=${encodeURIComponent(subscriptionId)}`;
  const es = new EventSource(url);
  es.addEventListener("surface", (msg) => {
    try {
      onEvent(JSON.parse((msg as MessageEvent).data) as SurfaceEvent);
    } catch (err) {
      console.error("[surface] bad payload", err);
    }
  });
  return es;
}

// ── Memory graph (versioned memories + typed relations) ──────────────────────

export interface Memory {
  id: string;
  tenant_id: string;
  workspace_id: string;
  content: string;
  is_static: boolean;
  is_forgotten: boolean;
  forget_after: string | null;
  forget_reason: string | null;
  version: number;
  parent_memory_id: string | null;
  root_memory_id: string | null;
  is_latest: boolean;
  citation_uri: string | null;
  source_chunk_id: string | null;
  audience: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export type MemoryRelationType =
  | "updates"
  | "extends"
  | "derives"
  | "contradicts"
  | "cites";

export interface MemoryRelation {
  source_memory_id: string;
  target_memory_id: string;
  relation: MemoryRelationType;
  created_at: string;
}

export async function listMemories(
  opts: {
    workspace?: string;
    audience?: string;
    includeForgotten?: boolean;
    includeOldVersions?: boolean;
    limit?: number;
  } = {},
): Promise<{ workspace_id: string; count: number; memories: Memory[] }> {
  const params = new URLSearchParams();
  if (opts.workspace) params.set("workspace", opts.workspace);
  if (opts.audience) params.set("audience", opts.audience);
  if (opts.includeForgotten) params.set("include_forgotten", "true");
  if (opts.includeOldVersions) params.set("include_old_versions", "true");
  if (opts.limit !== undefined) params.set("limit", String(opts.limit));
  const qs = params.toString();
  return request(`/api/memory${qs ? `?${qs}` : ""}`);
}

export async function getMemory(id: string): Promise<Memory> {
  return request(`/api/memory/${encodeURIComponent(id)}`);
}

export async function createMemory(attrs: {
  content: string;
  workspace?: string;
  isStatic?: boolean;
  audience?: string;
  citationUri?: string;
  metadata?: Record<string, unknown>;
}): Promise<Memory> {
  return request("/api/memory", {
    method: "POST",
    body: JSON.stringify({
      content: attrs.content,
      ...(attrs.workspace ? { workspace: attrs.workspace } : {}),
      ...(attrs.isStatic !== undefined ? { is_static: attrs.isStatic } : {}),
      ...(attrs.audience ? { audience: attrs.audience } : {}),
      ...(attrs.citationUri ? { citation_uri: attrs.citationUri } : {}),
      ...(attrs.metadata ? { metadata: attrs.metadata } : {}),
    }),
  });
}

export async function forgetMemory(
  id: string,
  opts: { reason?: string; forgetAfter?: string } = {},
): Promise<void> {
  await fetch(`/api/memory/${encodeURIComponent(id)}/forget`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      ...(opts.reason ? { reason: opts.reason } : {}),
      ...(opts.forgetAfter ? { forget_after: opts.forgetAfter } : {}),
    }),
  });
}

export async function updateMemory(
  id: string,
  attrs: { content: string; audience?: string },
): Promise<Memory> {
  return request(`/api/memory/${encodeURIComponent(id)}/update`, {
    method: "POST",
    body: JSON.stringify(attrs),
  });
}

export async function extendMemory(
  id: string,
  attrs: { content: string; audience?: string },
): Promise<Memory> {
  return request(`/api/memory/${encodeURIComponent(id)}/extend`, {
    method: "POST",
    body: JSON.stringify(attrs),
  });
}

export async function deriveMemory(
  id: string,
  attrs: { content: string; audience?: string },
): Promise<Memory> {
  return request(`/api/memory/${encodeURIComponent(id)}/derive`, {
    method: "POST",
    body: JSON.stringify(attrs),
  });
}

export async function memoryVersions(
  id: string,
): Promise<{ memory_id: string; root_id: string; versions: Memory[] }> {
  return request(`/api/memory/${encodeURIComponent(id)}/versions`);
}

export async function memoryRelations(id: string): Promise<{
  memory_id: string;
  inbound: MemoryRelation[];
  outbound: MemoryRelation[];
}> {
  return request(`/api/memory/${encodeURIComponent(id)}/relations`);
}

export async function deleteMemory(id: string): Promise<void> {
  await fetch(`/api/memory/${encodeURIComponent(id)}`, { method: "DELETE" });
}
