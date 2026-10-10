// ---------------------------------------------------------------------------
// Branded primitives
// ---------------------------------------------------------------------------

export type MemoryId = string & { readonly __brand: "MemoryId" };
export type WorkspaceId = string & { readonly __brand: "WorkspaceId" };
export type SubscriptionId = string & { readonly __brand: "SubscriptionId" };
export type WikiSlug = string & { readonly __brand: "WikiSlug" };

export function asMemoryId(v: string): MemoryId {
  return v as MemoryId;
}
export function asWorkspaceId(v: string): WorkspaceId {
  return v as WorkspaceId;
}
export function asSubscriptionId(v: string): SubscriptionId {
  return v as SubscriptionId;
}
export function asWikiSlug(v: string): WikiSlug {
  return v as WikiSlug;
}

// ---------------------------------------------------------------------------
// Shared enums / literals
// ---------------------------------------------------------------------------

export type Bandwidth = "l0" | "l1" | "full";
export type Audience = string; // workspace-defined; e.g. "public" | "internal"
export type RecallScope = "workspace" | "topic" | "actor" | "global";
export type SubscriptionCategory = string;

// ---------------------------------------------------------------------------
// Client config
// ---------------------------------------------------------------------------

export interface OptimalEngineConfig {
  /** Base URL of the Optimal Engine HTTP API. Defaults to http://localhost:4200 */
  baseUrl?: string;
  /** Optional API key — sent as Bearer token when provided. */
  apiKey?: string;
  /** Default workspace applied to every call unless overridden per-call. */
  workspace?: string;
}

// ---------------------------------------------------------------------------
// Error
// ---------------------------------------------------------------------------

export interface OptimalEngineErrorBody {
  error?: string;
  message?: string;
  code?: string;
}

// ---------------------------------------------------------------------------
// RAG / retrieval
// ---------------------------------------------------------------------------

export interface AskOptions {
  workspace?: string;
  audience?: Audience;
  format?: string;
  bandwidth?: Bandwidth;
}

export interface AskResult {
  answer: string;
  citations?: Citation[];
  envelope?: Record<string, unknown>;
  [key: string]: unknown;
}

export interface Citation {
  id: MemoryId;
  uri?: string;
  score?: number;
  excerpt?: string;
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

export interface SearchOptions {
  workspace?: string;
  limit?: number;
}

export interface SearchResult {
  results: SearchHit[];
  [key: string]: unknown;
}

export interface SearchHit {
  id: MemoryId;
  content: string;
  score?: number;
  audience?: Audience;
  metadata?: Record<string, unknown>;
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Grep
// ---------------------------------------------------------------------------

export interface GrepOptions {
  workspace?: string;
  intent?: string;
  scale?: string;
  modality?: string;
  limit?: number;
  literal?: boolean;
}

export interface GrepResult {
  results: GrepHit[];
  [key: string]: unknown;
}

export interface GrepHit {
  id: MemoryId;
  content: string;
  score?: number;
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Profile
// ---------------------------------------------------------------------------

export interface ProfileOptions {
  workspace?: string;
  audience?: Audience;
  bandwidth?: Bandwidth;
  node?: string;
}

export interface ProfileResult {
  static?: Record<string, unknown>;
  dynamic?: Record<string, unknown>;
  curated?: Record<string, unknown>;
  activity?: Record<string, unknown>;
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Recall
// ---------------------------------------------------------------------------

export interface RecallActionsOptions {
  topic: string;
  actor?: string;
  since?: string;
  workspace?: string;
}

export interface RecallWhoOptions {
  topic: string;
  role?: string;
  workspace?: string;
}

export interface RecallWhenOptions {
  event: string;
  workspace?: string;
}

export interface RecallWhereOptions {
  thing: string;
  workspace?: string;
}

export interface RecallOwnsOptions {
  actor: string;
  workspace?: string;
}

export interface RecallResult {
  items: RecallItem[];
  [key: string]: unknown;
}

export interface RecallItem {
  id?: MemoryId;
  content?: string;
  timestamp?: string;
  actor?: string;
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Memory
// ---------------------------------------------------------------------------

export interface CreateMemoryInput {
  content: string;
  workspace?: string;
  isStatic?: boolean;
  audience?: Audience;
  citationUri?: string;
  metadata?: Record<string, unknown>;
}

export interface ListMemoriesOptions {
  workspace?: string;
  audience?: Audience;
  includeForgotten?: boolean;
  limit?: number;
}

export interface ForgetMemoryInput {
  reason?: string;
  forgetAfter?: string;
}

export interface UpdateMemoryInput {
  content: string;
  [key: string]: unknown;
}

export interface ExtendMemoryInput {
  content: string;
  [key: string]: unknown;
}

export interface DeriveMemoryInput {
  content: string;
  [key: string]: unknown;
}

export interface Memory {
  id: MemoryId;
  content: string;
  workspace?: string;
  is_static?: boolean;
  audience?: Audience;
  citation_uri?: string;
  metadata?: Record<string, unknown>;
  forgotten_at?: string;
  inserted_at?: string;
  updated_at?: string;
  [key: string]: unknown;
}

export interface MemoryVersion {
  id: string;
  memory_id: MemoryId;
  content: string;
  inserted_at: string;
  [key: string]: unknown;
}

export interface MemoryRelation {
  id: string;
  source_id: MemoryId;
  target_id: MemoryId;
  type: string;
  [key: string]: unknown;
}

export interface MemoryListResult {
  memories: Memory[];
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Workspaces
// ---------------------------------------------------------------------------

export interface ListWorkspacesOptions {
  tenant?: string;
}

export interface CreateWorkspaceInput {
  slug: string;
  name: string;
  description?: string;
  tenant?: string;
}

export interface Workspace {
  id: WorkspaceId;
  slug: string;
  name: string;
  description?: string;
  tenant?: string;
  inserted_at?: string;
  updated_at?: string;
  [key: string]: unknown;
}

export interface WorkspaceConfig {
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Wiki
// ---------------------------------------------------------------------------

export interface ListWikiOptions {
  workspace?: string;
}

export interface GetWikiOptions {
  workspace?: string;
  audience?: Audience;
  format?: string;
}

export interface WikiArticle {
  slug: WikiSlug;
  title?: string;
  content?: string;
  audience?: Audience;
  [key: string]: unknown;
}

export interface WikiContradiction {
  id: string;
  memory_ids: MemoryId[];
  description?: string;
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Subscriptions / surface
// ---------------------------------------------------------------------------

export interface ListSubscriptionsOptions {
  workspace?: string;
}

export interface CreateSubscriptionInput {
  workspace: string;
  scope: RecallScope;
  scopeValue?: string;
  categories?: SubscriptionCategory[];
}

export interface Subscription {
  id: SubscriptionId;
  workspace: string;
  scope: RecallScope;
  scope_value?: string;
  categories?: SubscriptionCategory[];
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Status / architectures
// ---------------------------------------------------------------------------

export interface StatusResult {
  status: "ok" | "degraded" | "down";
  version?: string;
  [key: string]: unknown;
}

export interface ArchitectureEntry {
  id: string;
  name: string;
  description?: string;
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// Render pipeline
// ---------------------------------------------------------------------------

export interface RenderSpec {
  name: string;
  title: string;
  audience: string;
  genre: string;
  bandwidth: string;
  constraints: Record<string, string>;
  html_hints: Record<string, unknown>;
  modalities: string[];
}

export interface RenderContextOptions {
  workspace?: string;
}

export interface RenderContextResult {
  spec: RenderSpec;
  context: {
    body: string;
    format: string;
    sources: string[];
    warnings: string[];
  };
  prompt_guidance: string;
  trace: Record<string, unknown>;
}

export interface ListRenderSpecsOptions {
  workspace?: string;
}

export interface RenderSpecsResult {
  specs: RenderSpec[];
}
