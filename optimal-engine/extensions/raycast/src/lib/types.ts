// ---------------------------------------------------------------------------
// Branded ID types
// ---------------------------------------------------------------------------

export type MemoryId = string & { __brand: "MemoryId" };
export type WorkspaceId = string & { __brand: "WorkspaceId" };

// ---------------------------------------------------------------------------
// Raycast preferences
// ---------------------------------------------------------------------------

export interface Preferences {
  engineUrl: string;
  workspace: string;
  apiKey: string;
}

// ---------------------------------------------------------------------------
// API response shapes
// ---------------------------------------------------------------------------

export interface SearchResult {
  id: MemoryId;
  slug: string;
  content: string;
  audience: string;
  score: number;
  citation_uri: string | null;
  metadata: Record<string, unknown>;
  workspace: string;
}

export interface SearchResponse {
  results: SearchResult[];
  total: number;
}

export interface Workspace {
  id: WorkspaceId;
  name: string;
  slug: string;
  tenant: string;
}

export interface WorkspacesResponse {
  workspaces: Workspace[];
}

export interface MemoryInput {
  content: string;
  workspace: string;
  is_static: boolean;
  audience: string;
  citation_uri?: string;
  metadata?: Record<string, unknown>;
}

export interface MemoryCreatedResponse {
  id: MemoryId;
  slug: string;
  workspace: string;
}

export interface RagRequest {
  query: string;
  workspace: string;
  audience?: string;
  format?: "markdown" | "plain";
  bandwidth?: "low" | "medium" | "high";
}

export interface RagSource {
  id: MemoryId;
  slug: string;
  score: number;
  citation_uri: string | null;
  snippet: string;
}

export interface RagEnvelope {
  body: string;
  sources: RagSource[];
  workspace: string;
  query: string;
}

// ---------------------------------------------------------------------------
// Generic API result wrapper (never use `any`)
// ---------------------------------------------------------------------------

export type ApiResult<T> =
  | { ok: true; data: T }
  | { ok: false; error: string; status: number };
