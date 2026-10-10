// Knowledge (engine) API client - reads the active workspace's Optimal Engine
// knowledge base with canonical operating metadata. Goes through request<T>()
// which handles the API base, session cookie, CSRF, and the X-Workspace-ID
// header (so results are always scoped to the selected workspace).
import { request } from "./base";

// One canonical knowledge document as exposed by the engine. Mirrors the
// metadata a workspace needs to trust a document: source, status, target module,
// owner, and whether it is canonical vs draft / internal vs external.
export interface EngineKnowledgeItem {
  id: string;
  title: string;
  abstract: string;
  source: string;
  status: string;
  target_module: string;
  owner: string;
  canonical: boolean;
  is_external: boolean;
  updated_at: string;
}

export interface EngineKnowledgeResponse {
  items: EngineKnowledgeItem[];
  // false when the active workspace is not linked to a local Optimal Engine.
  engine_linked: boolean;
}

// Fetch the active workspace's engine KB docs. Optional full-text query (q).
// skipCache so a fresh search always hits the network rather than reusing a
// previous query's cached payload.
export async function getEngineKnowledge(
  q?: string,
): Promise<EngineKnowledgeResponse> {
  const query = q ? `?q=${encodeURIComponent(q)}` : "";
  return request<EngineKnowledgeResponse>(`/knowledge/engine${query}`, {
    skipCache: true,
  });
}
