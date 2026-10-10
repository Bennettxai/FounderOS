// ─── Engine API types ────────────────────────────────────────────────────────

export interface Workspace {
  id: string;
  name: string;
  tenant?: string;
}

export interface Memory {
  id: string;
  content: string;
  workspace: string;
  citation_uri?: string;
  metadata?: MemoryMetadata;
  created_at?: string;
}

export interface MemoryMetadata {
  url?: string;
  title?: string;
  clipped_at?: string;
  tags?: string[];
  [key: string]: unknown;
}

export interface ClipPayload {
  content: string;
  workspace: string;
  citation_uri?: string;
  metadata: MemoryMetadata;
}

export interface SearchResult {
  id: string;
  content: string;
  score?: number;
  workspace: string;
  metadata?: MemoryMetadata;
}

export interface ProfileSummary {
  workspace: string;
  summary?: string;
  active_wiki_page?: string;
  active_wiki_content?: string;
  memory_count?: number;
}

// ─── Extension settings ───────────────────────────────────────────────────────

export interface ExtensionSettings {
  engineUrl: string;
  defaultWorkspace: string;
  apiKey: string;
  autoTag: boolean;
}

export const DEFAULT_SETTINGS: ExtensionSettings = {
  engineUrl: "http://localhost:4200",
  defaultWorkspace: "",
  apiKey: "",
  autoTag: false,
};

// ─── Message passing ──────────────────────────────────────────────────────────

export type MessageType =
  | "CLIP_PAGE"
  | "CLIP_SELECTION"
  | "CLIP_URL"
  | "GET_SELECTION"
  | "GET_PAGE_INFO";

export interface ExtensionMessage {
  type: MessageType;
  payload?: unknown;
}

export interface PageInfo {
  url: string;
  title: string;
  selectedText: string;
  bodyText: string;
}

// ─── Result type ──────────────────────────────────────────────────────────────

export type Result<T, E = string> =
  | { ok: true; value: T }
  | { ok: false; error: E };
