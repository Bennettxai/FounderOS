/**
 * Typed client for the workspace / signal / activity / architecture
 * endpoints the /workspace, /activity, and /architectures routes consume.
 *
 * Matches the shapes emitted by OptimalEngine.API.Router exactly.
 */

export interface WorkspaceNode {
  id: string;
  slug: string;
  name: string;
  kind: string;
  parent_id: string | null;
  style: "internal" | "external" | string;
  status: string;
  signal_count: number;
}

export interface SignalChunk {
  id: string;
  parent_id: string | null;
  scale: "document" | "section" | "paragraph" | "sentence" | string;
  modality: string;
  length_bytes: number;
}

export interface SignalEntity {
  name: string;
  type: string;
}

export interface SignalDetail {
  id: string;
  uri: string;
  title: string;
  node: string;
  genre: string;
  modified_at: string;
  architecture_id: string | null;
  signal_dimensions: {
    mode: string;
    genre: string;
    type: string;
    format: string;
    structure: string;
  };
  sn_ratio: number;
  content: string;
  l0_abstract: string;
  l1_overview: string;
  chunks: SignalChunk[];
  entities: SignalEntity[];
  classification: {
    mode: string;
    genre: string;
    type: string;
    format: string;
    structure: string;
    sn_ratio: number | null;
    confidence: number | null;
  } | null;
  intent: { intent: string; confidence: number } | null;
  clusters: {
    id: string;
    theme: string;
    intent_dominant: string;
    weight: number;
  }[];
  citations: {
    wiki_slug: string;
    audience: string;
    claim_hash: string;
    last_verified: string;
  }[];
}

export interface ActivityEvent {
  id: number;
  tenant_id: string;
  ts: string;
  principal: string;
  kind: string;
  target_uri: string | null;
  latency_ms: number | null;
  metadata: Record<string, unknown>;
}

export interface ArchitectureSummary {
  id: string;
  name: string;
  version: number;
  description: string | null;
  modality_primary: string;
  granularity: string[];
  field_count: number;
}

export interface ArchitectureField {
  name: string;
  modality: string;
  dims: (number | "any")[];
  required: boolean;
  processor: string | null;
  description: string | null;
}

export interface ArchitectureDetail {
  id: string;
  name: string;
  version: number;
  description: string | null;
  modality_primary: string;
  granularity: string[];
  retention: string;
  fields: ArchitectureField[];
}

export interface ProcessorSummary {
  id: string;
  modality: string;
  emits: string[];
}

async function j<T>(path: string): Promise<T> {
  const res = await fetch(path, { headers: { accept: "application/json" } });
  if (!res.ok) throw new Error(`${path}: HTTP ${res.status}`);
  return (await res.json()) as T;
}

export async function listWorkspace(
  workspace?: string | null,
): Promise<WorkspaceNode[]> {
  const qs = workspace
    ? "?" + new URLSearchParams({ workspace }).toString()
    : "";
  const { nodes } = await j<{ nodes: WorkspaceNode[] }>(`/api/workspace${qs}`);
  return nodes;
}

export async function getSignal(id: string): Promise<SignalDetail> {
  return j<SignalDetail>(`/api/signals/${encodeURIComponent(id)}`);
}

export async function getActivity(
  opts: { limit?: number; kind?: string; workspace?: string | null } = {},
): Promise<ActivityEvent[]> {
  const params = new URLSearchParams();
  if (opts.limit) params.set("limit", String(opts.limit));
  if (opts.kind) params.set("kind", opts.kind);
  if (opts.workspace) params.set("workspace", opts.workspace);
  const qs = params.toString();
  const { events } = await j<{ events: ActivityEvent[] }>(
    `/api/activity${qs ? "?" + qs : ""}`,
  );
  return events;
}

export async function listArchitectures(): Promise<{
  architectures: ArchitectureSummary[];
  processors: ProcessorSummary[];
}> {
  return j("/api/architectures");
}

export async function getArchitecture(id: string): Promise<ArchitectureDetail> {
  return j<ArchitectureDetail>(`/api/architectures/${encodeURIComponent(id)}`);
}

/**
 * Turn a flat workspace list into a parent→children tree.
 * Unknown parent_id values become top-level.
 */
export function toTree(
  nodes: WorkspaceNode[],
): (WorkspaceNode & { children: WorkspaceNode[] })[] {
  const byId = new Map<string, WorkspaceNode & { children: WorkspaceNode[] }>();
  for (const n of nodes) byId.set(n.id, { ...n, children: [] });

  const roots: (WorkspaceNode & { children: WorkspaceNode[] })[] = [];
  for (const n of byId.values()) {
    if (n.parent_id && byId.has(n.parent_id)) {
      byId.get(n.parent_id)!.children.push(n);
    } else {
      roots.push(n);
    }
  }
  return roots;
}

/** Signals for one node — uses /api/optimal/nodes/:slug/files. */
export async function getNodeFiles(slug: string): Promise<
  {
    name: string;
    path: string;
    is_dir: boolean;
    size: number;
    genre: string | null;
    modified_at: string | null;
  }[]
> {
  const { files } = await j<{ files: any[] }>(
    `/api/optimal/nodes/${encodeURIComponent(slug)}/files`,
  );
  return files;
}
